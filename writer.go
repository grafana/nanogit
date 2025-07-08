package nanogit

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

// ErrWriterCleanedUp is returned when trying to use a writer after cleanup has been called.
var ErrWriterCleanedUp = errors.New("writer has been cleaned up and can no longer be used")

// entryPool manages reusable slices for PackfileTreeEntry to reduce allocations.
// This addresses the frequent slice allocations in updateTreeEntry.
type entryPool struct {
	pool [][]protocol.PackfileTreeEntry
}

// newEntryPool creates a new entry pool.
func newEntryPool() *entryPool {
	return &entryPool{
		pool: make([][]protocol.PackfileTreeEntry, 0, 16), // Start with some capacity
	}
}

// get retrieves a slice with at least the specified capacity.
// If a suitable slice exists in the pool, it's reused; otherwise a new one is allocated.
func (p *entryPool) get(capacity int) []protocol.PackfileTreeEntry {
	// Look for a suitable slice in the pool
	for i, slice := range p.pool {
		if cap(slice) >= capacity {
			// Remove from pool and return
			p.pool[i] = p.pool[len(p.pool)-1]
			p.pool = p.pool[:len(p.pool)-1]
			// Reset length but keep capacity
			return slice[:0]
		}
	}
	// No suitable slice found, allocate new one
	return make([]protocol.PackfileTreeEntry, 0, capacity)
}

// put returns a slice to the pool for reuse.
func (p *entryPool) put(slice []protocol.PackfileTreeEntry) {
	if cap(slice) > 0 && len(p.pool) < 32 { // Limit pool size to avoid memory leaks
		p.pool = append(p.pool, slice[:0]) // Reset length but keep capacity
	}
}

// NewStagedWriter creates a new StagedWriter for staging changes to a Git reference.
// It initializes the writer with the current state of the specified reference,
// allowing you to stage multiple changes (create/update/delete blobs and trees)
// before committing and pushing them as a single atomic operation.
//
// The writer maintains an in-memory representation of the repository state and
// tracks all changes until they are committed and pushed.
//
// Example usage:
//
//	writer, err := client.NewStagedWriter(ctx, ref)
//	if err != nil {
//	    return err
//	}
//
//	// Stage multiple changes
//	writer.CreateBlob(ctx, "new.txt", []byte("content"))
//	writer.UpdateBlob(ctx, "existing.txt", []byte("updated"))
//	writer.DeleteBlob(ctx, "old.txt")
//
//	// Commit all changes at once
//	commit, err := writer.Commit(ctx, "Update files", author, committer)
//	if err != nil {
//	    return err
//	}
//
//	// Push to remote
//	return writer.Push(ctx)
func (c *httpClient) NewStagedWriter(ctx context.Context, ref Ref, options ...WriterOption) (StagedWriter, error) {
	// Apply writer options
	opts, err := applyWriterOptions(options)
	if err != nil {
		return nil, fmt.Errorf("apply writer options: %w", err)
	}

	logger := log.FromContext(ctx)
	logger.Debug("Initialize staged writer",
		"ref_name", ref.Name,
		"ref_hash", ref.Hash.String(),
		"storage_mode", opts.StorageMode)

	ctx, objStorage := storage.FromContextOrInMemory(ctx)

	// Get essential objects - fetch commit, root tree, and flat tree
	commit, err := c.getCommit(ctx, ref.Hash, false)
	if err != nil {
		return nil, fmt.Errorf("get commit %s: %w", ref.Hash.String(), err)
	}

	treeObj, err := c.getTree(ctx, commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("get tree %s: %w", commit.Tree.String(), err)
	}

	// Get the flat tree representation for efficient path-based operations
	currentTree, err := c.GetFlatTree(ctx, commit.Hash)
	if err != nil {
		return nil, fmt.Errorf("get flat tree for commit %s: %w", commit.Hash.String(), err)
	}

	// Build tree entries map from flat tree
	entries := make(map[string]*FlatTreeEntry, len(currentTree.Entries))
	for _, entry := range currentTree.Entries {
		entries[entry.Path] = &entry
	}

	logger.Debug("Staged writer ready",
		"ref_name", ref.Name,
		"commit_hash", commit.Hash.String(),
		"tree_hash", treeObj.Hash.String(),
		"tree_entries", len(entries))

	// Convert writer storage mode to protocol storage mode
	var protocolStorageMode protocol.PackfileStorageMode
	switch opts.StorageMode {
	case PackfileStorageMemory:
		protocolStorageMode = protocol.PackfileStorageMemory
	case PackfileStorageDisk:
		protocolStorageMode = protocol.PackfileStorageDisk
	case PackfileStorageAuto:
		protocolStorageMode = protocol.PackfileStorageAuto
	default:
		protocolStorageMode = protocol.PackfileStorageAuto
	}

	writer := protocol.NewPackfileWriter(crypto.SHA1, protocolStorageMode)
	return &stagedWriter{
		client:      c,
		ref:         ref,
		writer:      writer,
		lastCommit:  commit,
		lastTree:    treeObj,
		objStorage:  objStorage,
		treeEntries: entries,
		storageMode: protocolStorageMode,
		entryPool:   newEntryPool(), // Initialize the entry pool for memory optimization
		dirtyPaths:  make(map[string]bool), // Initialize dirty paths tracking for deferred tree building
	}, nil
}

// stagedWriter implements the StagedWriter interface.
// It maintains the state of staged changes for a Git reference, including:
//   - A packfile writer for creating new Git objects
//   - Cache of tree objects to avoid redundant fetches
//   - Mapping of file paths to their tree entries
//   - Reference to the last commit and tree state
//
// The writer operates by maintaining an in-memory representation of the
// repository state and building up a packfile of new objects as changes
// are staged. When committed, all changes are applied atomically.
type stagedWriter struct {
	// Embedded HTTP client for Git operations
	client *httpClient
	// Git reference being modified
	ref Ref
	// Packfile writer for creating objects
	writer *protocol.PackfileWriter
	// Last commit on the reference
	lastCommit *Commit
	// Root tree object from last commit
	lastTree *protocol.PackfileObject
	// Cache of fetched tree objects
	objStorage storage.PackfileStorage
	// Flat mapping of paths to tree entries
	treeEntries map[string]*FlatTreeEntry
	// Storage mode for packfile writer
	storageMode protocol.PackfileStorageMode
	// Memory optimization: pool for reusing tree entry slices
	entryPool *entryPool
	// Track if cleanup has been called
	isCleanedUp bool
	// Deferred tree building optimization: track which directory paths need tree rebuilding
	dirtyPaths map[string]bool
}

// checkCleanupState returns an error if the writer has been cleaned up.
func (w *stagedWriter) checkCleanupState() error {
	if w.isCleanedUp {
		return ErrWriterCleanedUp
	}
	return nil
}


// BlobExists checks if a blob exists at the given path in the repository.
// This method verifies the existence of a file by checking the tree entries
// that have been loaded into memory.
//
// Parameters:
//   - ctx: Context for the operation
//   - path: File path to check (e.g., "docs/readme.md")
//
// Returns:
//   - bool: True if the blob exists at the specified path
//   - error: Error if the check fails
//
// Example:
//
//	exists, err := writer.BlobExists(ctx, "src/main.go")
func (w *stagedWriter) BlobExists(ctx context.Context, path string) (bool, error) {
	if err := w.checkCleanupState(); err != nil {
		return false, err
	}

	if path == "" {
		return false, ErrEmptyPath
	}


	logger := log.FromContext(ctx)
	logger.Debug("Check blob existence", "path", path)

	entry, exists := w.treeEntries[path]
	if !exists {
		return false, nil
	}

	return entry.Type == protocol.ObjectTypeBlob, nil
}

// CreateBlob creates a new blob object at the specified path with the given content.
// The path can include directory separators ("/") to create nested directory structures.
// If intermediate directories don't exist, they will be created automatically.
//
// This operation stages the blob creation but does not immediately commit it.
// You must call Commit() and Push() to persist the changes.
//
// Parameters:
//   - ctx: Context for the operation
//   - path: File path where the blob should be created (e.g., "docs/readme.md")
//   - content: Raw content of the file as bytes
//
// Returns:
//   - hash.Hash: The SHA-1 hash of the created blob object
//   - error: Error if the path already exists or if blob creation fails
//
// Example:
//
//	hash, err := writer.CreateBlob(ctx, "src/main.go", []byte("package main\n"))
func (w *stagedWriter) CreateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	if err := w.checkCleanupState(); err != nil {
		return hash.Zero, err
	}

	if path == "" {
		return hash.Zero, ErrEmptyPath
	}


	logger := log.FromContext(ctx)
	logger.Debug("Create blob",
		"path", path,
		"content_size", len(content))

	if obj, ok := w.treeEntries[path]; ok {
		return hash.Zero, NewObjectAlreadyExistsError(obj.Hash)
	}

	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return hash.Zero, fmt.Errorf("create blob at %q: %w", path, err)
	}

	w.treeEntries[path] = &FlatTreeEntry{
		Path: path,
		Hash: blobHash,
		Type: protocol.ObjectTypeBlob,
		Mode: 0o100644,
	}

	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash); err != nil {
		return hash.Zero, fmt.Errorf("update tree structure for %q: %w", path, err)
	}

	logger.Debug("Blob created",
		"path", path,
		"blob_hash", blobHash.String())

	return blobHash, nil
}

// UpdateBlob updates the content of an existing blob at the specified path.
// The blob must already exist at the given path, otherwise an error is returned.
//
// This operation stages the blob update but does not immediately commit it.
// You must call Commit() and Push() to persist the changes.
//
// Parameters:
//   - ctx: Context for the operation
//   - path: File path of the existing blob to update
//   - content: New content for the file as bytes
//
// Returns:
//   - hash.Hash: The SHA-1 hash of the updated blob object
//   - error: Error if the path doesn't exist or if blob update fails
//
// Example:
//
//	hash, err := writer.UpdateBlob(ctx, "README.md", []byte("Updated content"))
func (w *stagedWriter) UpdateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	if err := w.checkCleanupState(); err != nil {
		return hash.Zero, err
	}

	if path == "" {
		return hash.Zero, ErrEmptyPath
	}


	logger := log.FromContext(ctx)
	logger.Debug("Update blob",
		"path", path,
		"content_size", len(content))

	if w.treeEntries[path] == nil {
		return hash.Zero, NewPathNotFoundError(path)
	}

	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return hash.Zero, fmt.Errorf("create blob at %q: %w", path, err)
	}

	w.treeEntries[path] = &FlatTreeEntry{
		Path: path,
		Hash: blobHash,
		Type: protocol.ObjectTypeBlob,
		Mode: 0o100644,
	}

	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash); err != nil {
		return hash.Zero, fmt.Errorf("update tree structure for %q: %w", path, err)
	}

	logger.Debug("Blob updated",
		"path", path,
		"blob_hash", blobHash.String())

	return blobHash, nil
}

// DeleteBlob removes a blob (file) at the specified path from the repository.
// The blob must exist and must be a file (not a directory), otherwise an error is returned.
// If removing the blob leaves empty parent directories, those directories will also be removed.
//
// This operation stages the blob deletion but does not immediately commit it.
// You must call Commit() and Push() to persist the changes.
//
// Parameters:
//   - ctx: Context for the operation
//   - path: File path of the blob to delete
//
// Returns:
//   - hash.Hash: The SHA-1 hash of the deleted blob object
//   - error: Error if the path doesn't exist, is not a blob, or deletion fails
//
// Example:
//
//	hash, err := writer.DeleteBlob(ctx, "old-file.txt")
func (w *stagedWriter) DeleteBlob(ctx context.Context, path string) (hash.Hash, error) {
	if err := w.checkCleanupState(); err != nil {
		return hash.Zero, err
	}

	if path == "" {
		return hash.Zero, ErrEmptyPath
	}


	logger := log.FromContext(ctx)
	logger.Debug("Delete blob",
		"path", path)

	existing, ok := w.treeEntries[path]
	if !ok {
		return hash.Zero, NewPathNotFoundError(path)
	}

	if existing.Type != protocol.ObjectTypeBlob {
		return hash.Zero, NewUnexpectedObjectTypeError(existing.Hash, protocol.ObjectTypeBlob, existing.Type)
	}

	blobHash := existing.Hash
	delete(w.treeEntries, path)

	if err := w.removeBlobFromTree(ctx, path); err != nil {
		return hash.Zero, fmt.Errorf("remove blob from tree at %q: %w", path, err)
	}

	logger.Debug("Blob deleted",
		"path", path,
		"blob_hash", blobHash.String())

	return blobHash, nil
}

// GetTree retrieves the tree object at the specified path from the repository.
// The tree represents a directory structure containing files and subdirectories.
// The path must exist and must be a directory (tree), otherwise an error is returned.
//
// This operation retrieves the tree from memory if it has been staged,
// or from the repository if it hasn't been modified.
//
// Parameters:
//   - ctx: Context for the operation
//   - path: Directory path to retrieve
//
// Returns:
//   - *Tree: The tree object containing directory entries
//   - error: Error if the path doesn't exist, is not a tree, or retrieval fails
//
// Example:
//
//	tree, err := writer.GetTree(ctx, "src")
//	if err != nil {
//	    return fmt.Errorf("failed to get tree: %w", err)
//	}
//	for _, entry := range tree.Entries {
//	    fmt.Printf("Found %s: %s\n", entry.Type, entry.Name)
//	}
func (w *stagedWriter) GetTree(ctx context.Context, path string) (*Tree, error) {
	if err := w.checkCleanupState(); err != nil {
		return nil, err
	}


	existing, ok := w.treeEntries[path]
	if !ok {
		return nil, NewPathNotFoundError(path)
	}

	if existing.Type != protocol.ObjectTypeTree {
		return nil, NewUnexpectedObjectTypeError(existing.Hash, protocol.ObjectTypeTree, existing.Type)
	}

	// Get all entries that are direct children of this path
	pathPrefix := path + "/"
	var entries []TreeEntry

	for entryPath, entry := range w.treeEntries {
		if entryPath == path {
			continue // Skip the tree itself
		}

		// Check if this is a direct child (no intermediate slashes)
		if strings.HasPrefix(entryPath, pathPrefix) {
			remainingPath := entryPath[len(pathPrefix):]
			if !strings.Contains(remainingPath, "/") {
				entries = append(entries, TreeEntry{
					Name: remainingPath,
					Type: entry.Type,
					Hash: entry.Hash,
					Mode: entry.Mode,
				})
			}
		}
	}

	return &Tree{
		Hash:    existing.Hash,
		Entries: entries,
	}, nil
}

// DeleteTree removes an entire directory tree at the specified path from the repository.
// This operation recursively deletes all files and subdirectories within the specified path.
// The path must exist and must be a directory (tree), otherwise an error is returned.
//
// This is equivalent to `rm -rf <path>` in Unix systems.
//
// This operation stages the tree deletion but does not immediately commit it.
// You must call Commit() and Push() to persist the changes.
//
// Parameters:
//   - ctx: Context for the operation
//   - path: Directory path to delete recursively
//
// Returns:
//   - hash.Hash: The SHA-1 hash of the deleted tree object
//   - error: Error if the path doesn't exist, is not a tree, or deletion fails
//
// Example:
//
//	hash, err := writer.DeleteTree(ctx, "old-directory")
func (w *stagedWriter) DeleteTree(ctx context.Context, path string) (hash.Hash, error) {
	if err := w.checkCleanupState(); err != nil {
		return hash.Zero, err
	}

	logger := log.FromContext(ctx)
	if path == "" || path == "." {
		emptyHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeTree, []byte{})
		if err != nil {
			return hash.Zero, fmt.Errorf("create empty tree: %w", err)
		}

		emptyTree := protocol.PackfileObject{
			Hash: emptyHash,
			Type: protocol.ObjectTypeTree,
			Tree: []protocol.PackfileTreeEntry{},
		}

		w.writer.AddObject(emptyTree)
		w.objStorage.Add(&emptyTree)
		w.treeEntries[""] = &FlatTreeEntry{
			Path: "",
			Hash: emptyHash,
			Type: protocol.ObjectTypeTree,
			Mode: 0o40000,
		}
		w.lastTree = &emptyTree

		return emptyHash, nil
	}


	existing, ok := w.treeEntries[path]
	if !ok {
		return hash.Zero, NewPathNotFoundError(path)
	}

	if existing.Type != protocol.ObjectTypeTree {
		return hash.Zero, NewUnexpectedObjectTypeError(existing.Hash, protocol.ObjectTypeTree, existing.Type)
	}
	treeHash := existing.Hash

	logger.Debug("deleting tree", "path", path)

	// Find and remove all entries that start with this path
	pathPrefix := path + "/"
	var entriesToDelete []string

	for entryPath := range w.treeEntries {
		if entryPath == path || strings.HasPrefix(entryPath, pathPrefix) {
			entriesToDelete = append(entriesToDelete, entryPath)
		}
	}

	// Remove all entries under this tree
	for _, entryPath := range entriesToDelete {
		logger.Debug("removing entry", "path", entryPath)
		delete(w.treeEntries, entryPath)
	}

	// Update the tree structure to remove the directory entry
	if err := w.removeTreeFromTree(ctx, path); err != nil {
		return hash.Zero, fmt.Errorf("remove tree from entire tree: %w", err)
	}

	return treeHash, nil
}

// Commit creates a new commit object with all the staged changes and the specified metadata.
// This operation takes all the changes that have been staged via CreateBlob, UpdateBlob,
// DeleteBlob, and DeleteTree operations and creates a single commit containing all of them.
//
// The commit is created in memory but not yet pushed to the remote repository.
// You must call Push() to send the commit to the remote.
//
// Parameters:
//   - ctx: Context for the operation
//   - message: Commit message describing the changes
//   - author: Information about who authored the changes
//   - committer: Information about who created the commit (often same as author)
//
// Returns:
//   - *Commit: The created commit object with hash and metadata
//   - error: Error if commit creation fails
//
// Example:
//
//	author := nanogit.Author{
//	    Name:  "John Doe",
//	    Email: "john@example.com",
//	    Time:  time.Now(),
//	}
//	commit, err := writer.Commit(ctx, "Add new features", author, author)
func (w *stagedWriter) Commit(ctx context.Context, message string, author Author, committer Committer) (*Commit, error) {
	if err := w.checkCleanupState(); err != nil {
		return nil, err
	}

	if message == "" {
		return nil, ErrEmptyCommitMessage
	}

	if author.Name == "" || author.Email == "" {
		return nil, NewAuthorError("author", "missing name or email")
	}

	if committer.Name == "" || committer.Email == "" {
		return nil, NewAuthorError("committer", "missing name or email")
	}

	logger := log.FromContext(ctx)
	logger.Debug("Create commit",
		"message", message,
		"author_name", author.Name,
		"committer_name", committer.Name)

	// Build all pending trees before creating the commit
	// This optimizes performance by deferring tree building until commit time
	if err := w.buildPendingTrees(ctx); err != nil {
		return nil, fmt.Errorf("build pending trees: %w", err)
	}

	if !w.writer.HasObjects() {
		return nil, ErrNothingToCommit
	}

	authorIdentity := protocol.Identity{
		Name:      author.Name,
		Email:     author.Email,
		Timestamp: author.Time.Unix(),
		Timezone:  author.Time.Format("-0700"),
	}

	committerIdentity := protocol.Identity{
		Name:      committer.Name,
		Email:     committer.Email,
		Timestamp: committer.Time.Unix(),
		Timezone:  committer.Time.Format("-0700"),
	}

	commitHash, err := w.writer.AddCommit(w.lastTree.Hash, w.lastCommit.Hash, &authorIdentity, &committerIdentity, message)
	if err != nil {
		return nil, fmt.Errorf("create commit object: %w", err)
	}

	w.lastCommit = &Commit{
		Hash:      commitHash,
		Tree:      w.lastTree.Hash,
		Parent:    w.lastCommit.Hash,
		Author:    author,
		Committer: committer,
		Message:   message,
	}

	logger.Debug("Commit created",
		"commit_hash", commitHash.String(),
		"tree_hash", w.lastTree.Hash.String(),
		"parent_hash", w.lastCommit.Parent.String())

	return w.lastCommit, nil
}

// Push sends all staged changes and commits to the remote Git repository.
// This operation packages all the staged objects into a Git packfile and
// transmits it to the remote repository using the Git protocol.
//
// After a successful push, the writer is reset and can be used to stage
// additional changes for future commits.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - error: Error if the push operation fails
//
// Example:
//
//	err := writer.Push(ctx)
//	if err != nil {
//	    log.Printf("Failed to push changes: %v", err)
//	}
func (w *stagedWriter) Push(ctx context.Context) error {
	if err := w.checkCleanupState(); err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	logger.Debug("Push changes",
		"ref_name", w.ref.Name,
		"from_hash", w.ref.Hash.String(),
		"to_hash", w.lastCommit.Hash.String())

	if !w.writer.HasObjects() {
		return ErrNothingToPush
	}

	// Create a pipe to stream packfile data directly from WritePackfile to ReceivePack
	pipeReader, pipeWriter := io.Pipe()

	// Channel to capture any error from WritePackfile goroutine
	writeErrChan := make(chan error, 1)

	// Start WritePackfile in a goroutine, writing to the pipe
	go func() {
		defer func() {
			_ = pipeWriter.Close() // Best effort close in goroutine
		}()
		err := w.writer.WritePackfile(pipeWriter, w.ref.Name, w.ref.Hash)
		writeErrChan <- err
	}()

	// Call ReceivePack with the pipe reader (this will stream the data and parse the response)
	err := w.client.ReceivePack(ctx, pipeReader)
	if err != nil {
		_ = pipeReader.Close() // Best effort close since we're already handling an error
		return fmt.Errorf("send packfile to remote: %w", err)
	}

	// Check for any error from the WritePackfile goroutine
	if writeErr := <-writeErrChan; writeErr != nil {
		return fmt.Errorf("write packfile for ref %q: %w", w.ref.Name, writeErr)
	}

	logger.Debug("Packfile streamed successfully")

	w.writer = protocol.NewPackfileWriter(crypto.SHA1, w.storageMode)
	w.ref.Hash = w.lastCommit.Hash

	logger.Debug("Push completed",
		"ref_name", w.ref.Name,
		"new_hash", w.lastCommit.Hash.String())

	return nil
}

// addMissingOrStaleTreeEntries marks directory paths as dirty for deferred tree building.
// This method handles the tree structure updates required when adding files to Git:
//   - Marks all parent directories as dirty for later tree rebuilding
//   - Creates missing intermediate directory entries in treeEntries map
//   - Defers actual tree object creation until commit time for better performance
//
// The method works by traversing the path from the file up to the root,
// marking each directory path as dirty so trees can be built efficiently at commit time.
func (w *stagedWriter) addMissingOrStaleTreeEntries(ctx context.Context, path string, blobHash hash.Hash) error {
	logger := log.FromContext(ctx)
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	// Get the file name and directory parts  
	dirParts := pathParts[:len(pathParts)-1]

	// Mark all parent directories as dirty for deferred tree building
	for i := 0; i < len(dirParts); i++ {
		currentPath := strings.Join(dirParts[:i+1], "/")
		
		// Check if not a tree
		existingObj, exists := w.treeEntries[currentPath]
		if exists && existingObj.Type != protocol.ObjectTypeTree {
			return NewUnexpectedObjectTypeError(existingObj.Hash, protocol.ObjectTypeTree, existingObj.Type)
		}

		// Create directory entry if it doesn't exist
		if !exists {
			w.treeEntries[currentPath] = &FlatTreeEntry{
				Path: currentPath,
				Hash: hash.Zero, // Will be calculated during tree building
				Type: protocol.ObjectTypeTree,
				Mode: 0o40000,
			}
			logger.Debug("created directory entry", "path", currentPath)
		}
		
		// Mark this directory path as dirty
		w.dirtyPaths[currentPath] = true
		logger.Debug("marked path as dirty", "path", currentPath)
	}
	
	// Mark root as dirty if file is in root directory
	if len(dirParts) == 0 {
		w.dirtyPaths[""] = true
		logger.Debug("marked root as dirty for root-level file")
	} else {
		// Always mark root as dirty when any nested directory changes
		w.dirtyPaths[""] = true
		logger.Debug("marked root as dirty for nested file")
	}

	return nil
}

// updateTreeEntry creates a new tree object by adding or updating an entry in an existing tree.
// This method takes an existing tree object and either adds a new entry or updates an existing
// entry with the same filename. It maintains proper Git tree object sorting and formatting.
//
// Parameters:
//   - ctx: Context for the operation
//   - obj: The existing tree object to modify
//   - current: The tree entry to add or update
//
// Returns:
//   - *protocol.PackfileObject: New tree object with the updated entry
//   - error: Error if tree creation fails
func (w *stagedWriter) updateTreeEntry(ctx context.Context, treeObj *protocol.PackfileObject, current protocol.PackfileTreeEntry) (*protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Update tree entry",
		"fileName", current.FileName,
		"fileMode", fmt.Sprintf("%o", current.FileMode),
	)

	// Use pooled slice to reduce allocations - get appropriate capacity
	var capacity int
	found := false
	for _, entry := range treeObj.Tree {
		if entry.FileName == current.FileName {
			found = true
			break
		}
	}
	if found {
		// Replacing existing entry - same capacity
		capacity = len(treeObj.Tree)
	} else {
		// Adding new entry - need +1 capacity
		capacity = len(treeObj.Tree) + 1
	}
	
	combinedEntries := w.entryPool.get(capacity)

	// Add all entries except the one we're updating
	for _, entry := range treeObj.Tree {
		if entry.FileName != current.FileName {
			combinedEntries = append(combinedEntries, entry)
		}
	}

	// Add the new/updated entry
	combinedEntries = append(combinedEntries, current)
	
	// CRITICAL: Make a copy of the slice for BuildTreeObject since it sorts in-place
	// and we don't want to corrupt the pooled slice
	entriesCopy := make([]protocol.PackfileTreeEntry, len(combinedEntries))
	copy(entriesCopy, combinedEntries)
	
	// Return slice to pool immediately after copying, before BuildTreeObject
	w.entryPool.put(combinedEntries)
	
	newObj, err := protocol.BuildTreeObject(crypto.SHA1, entriesCopy)
	if err != nil {
		return nil, fmt.Errorf("build tree object: %w", err)
	}

	w.writer.AddObject(newObj)
	w.objStorage.Add(&newObj)
	w.objStorage.Delete(treeObj.Hash)

	logger.Debug("Tree entry updated",
		"fileName", current.FileName,
		"oldEntryCount", len(treeObj.Tree),
		"newEntryCount", len(combinedEntries),
		"newHash", newObj.Hash.String())

	return &newObj, nil
}

// removeBlobFromTree marks directory paths as dirty for deferred tree building after blob removal.
// This method handles marking the tree structure for updates when deleting files from Git:
//   - Marks all parent directories as dirty for later tree rebuilding
//   - Defers actual tree object rebuilding until commit time for better performance
//
// The method works by traversing the path from the file up to the root,
// marking each directory path as dirty so trees can be rebuilt efficiently at commit time.
func (w *stagedWriter) removeBlobFromTree(ctx context.Context, path string) error {
	logger := log.FromContext(ctx)
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Get the directory parts
	dirParts := pathParts[:len(pathParts)-1]

	// Mark all parent directories as dirty for deferred tree building
	for i := 0; i < len(dirParts); i++ {
		currentPath := strings.Join(dirParts[:i+1], "/")
		
		// Verify the directory exists
		existingObj, exists := w.treeEntries[currentPath]
		if !exists {
			return fmt.Errorf("parent directory %s does not exist: %w", currentPath, NewPathNotFoundError(currentPath))
		}

		if existingObj.Type != protocol.ObjectTypeTree {
			return fmt.Errorf("parent path is not a tree: %w", NewUnexpectedObjectTypeError(existingObj.Hash, protocol.ObjectTypeTree, existingObj.Type))
		}
		
		// Mark this directory path as dirty
		w.dirtyPaths[currentPath] = true
		logger.Debug("marked path as dirty for blob removal", "path", currentPath)
	}
	
	// Always mark root as dirty when any file is removed
	w.dirtyPaths[""] = true
	logger.Debug("marked root as dirty for blob removal")

	return nil
}

// removeTreeFromTree marks directory paths as dirty for deferred tree building after tree removal.
// This method handles marking the tree structure for updates when deleting directories from Git:
//   - Marks all parent directories as dirty for later tree rebuilding
//   - Defers actual tree object rebuilding until commit time for better performance
//
// This is similar to removeBlobFromTree but handles directory removal instead of file removal.
func (w *stagedWriter) removeTreeFromTree(ctx context.Context, path string) error {
	logger := log.FromContext(ctx)
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	// Get the parent directory parts
	parentParts := pathParts[:len(pathParts)-1]

	// Mark all parent directories as dirty for deferred tree building
	for i := 0; i < len(parentParts); i++ {
		currentPath := strings.Join(parentParts[:i+1], "/")
		
		// Verify the directory exists
		existingObj, exists := w.treeEntries[currentPath]
		if !exists {
			return fmt.Errorf("parent directory %s does not exist: %w", currentPath, NewPathNotFoundError(currentPath))
		}

		if existingObj.Type != protocol.ObjectTypeTree {
			return fmt.Errorf("parent path is not a tree: %w", NewUnexpectedObjectTypeError(existingObj.Hash, protocol.ObjectTypeTree, existingObj.Type))
		}
		
		// Mark this directory path as dirty
		w.dirtyPaths[currentPath] = true
		logger.Debug("marked path as dirty for tree removal", "path", currentPath)
	}
	
	// Always mark root as dirty when any directory is removed
	w.dirtyPaths[""] = true
	logger.Debug("marked root as dirty for tree removal")

	return nil
}

// removeTreeEntry creates a new tree object by removing a specific entry from an existing tree.
// This is a lower-level helper method that handles the actual removal of an entry from a
// Git tree object, creating a new tree object with the filtered entries.
//
// Parameters:
//   - ctx: Context for the operation
//   - obj: The tree object to modify
//   - targetFileName: The filename of the entry to remove
//
// Returns:
//   - *protocol.PackfileObject: New tree object without the specified entry
//   - error: Error if tree creation fails
//
// Note: If the target entry is not found, the original object is returned unchanged.
func (w *stagedWriter) removeTreeEntry(ctx context.Context, treeObj *protocol.PackfileObject, targetFileName string) (*protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)

	logger.Debug("Remove tree entry",
		"targetFileName", targetFileName,
		"treeHash", treeObj.Hash.String())

	// Create a new slice excluding the target entry
	filteredEntries := make([]protocol.PackfileTreeEntry, 0, len(treeObj.Tree))
	found := false

	for _, entry := range treeObj.Tree {
		if entry.FileName != targetFileName {
			filteredEntries = append(filteredEntries, entry)
		} else {
			found = true
		}
	}

	if !found {
		logger.Debug("Entry not found in tree",
			"targetFileName", targetFileName,
			"treeHash", treeObj.Hash.String())
		// Entry not found in tree, but this might be okay for intermediate trees
		// Return the original object unchanged
		return treeObj, nil
	}

	// Build new tree object with the filtered entries
	newObj, err := protocol.BuildTreeObject(crypto.SHA1, filteredEntries)
	if err != nil {
		return nil, fmt.Errorf("build tree object: %w", err)
	}

	w.writer.AddObject(newObj)
	w.objStorage.Add(&newObj)
	w.objStorage.Delete(treeObj.Hash)

	logger.Debug("Tree entry removed",
		"targetFileName", targetFileName,
		"oldEntryCount", len(treeObj.Tree),
		"newEntryCount", len(filteredEntries),
		"newHash", newObj.Hash.String())

	return &newObj, nil
}

// buildPendingTrees builds all dirty tree objects in topological order (deepest first).
// This method is called at commit time to efficiently build all trees that need updating.
// It builds trees bottom-up to ensure parent trees can reference their children's hashes.
func (w *stagedWriter) buildPendingTrees(ctx context.Context) error {
	if len(w.dirtyPaths) == 0 {
		return nil // No dirty paths, nothing to build
	}
	
	logger := log.FromContext(ctx)
	logger.Debug("Building pending trees", "dirty_path_count", len(w.dirtyPaths))

	// Step 1: Collect all dirty paths and sort them by depth (deepest first)
	var dirtyPathList []string
	for path := range w.dirtyPaths {
		dirtyPathList = append(dirtyPathList, path)
	}
	
	// Sort by depth (deepest first) - deeper paths have more "/" separators
	sort.Slice(dirtyPathList, func(i, j int) bool {
		depthI := strings.Count(dirtyPathList[i], "/")
		depthJ := strings.Count(dirtyPathList[j], "/")
		if depthI != depthJ {
			return depthI > depthJ // Deeper paths first
		}
		// Same depth, sort alphabetically for consistency
		// Root directory ("") should always be last
		if dirtyPathList[i] == "" {
			return false
		}
		if dirtyPathList[j] == "" {
			return true
		}
		return dirtyPathList[i] < dirtyPathList[j]
	})
	
	logger.Debug("Sorted dirty paths", "paths", dirtyPathList)

	// Step 2: Build trees from deepest to shallowest
	for _, path := range dirtyPathList {
		// Skip if this path was already processed (can happen with complex operations)
		if _, exists := w.treeEntries[path]; !exists && path != "" {
			continue // Directory was deleted
		}
		
		if err := w.buildSingleTree(ctx, path); err != nil {
			return fmt.Errorf("build tree for path %q: %w", path, err)
		}
	}
	
	// Step 3: Clear dirty paths since all trees have been built
	w.dirtyPaths = make(map[string]bool)
	logger.Debug("Finished building pending trees")
	
	return nil
}

// buildSingleTree builds a single tree object for the given directory path.
// It collects all direct children (files and subdirectories) and creates a tree object.
func (w *stagedWriter) buildSingleTree(ctx context.Context, dirPath string) error {
	logger := log.FromContext(ctx)
	
	// Collect all direct children of this directory
	var entries []protocol.PackfileTreeEntry
	pathPrefix := dirPath
	if pathPrefix != "" {
		pathPrefix += "/"
	}
	
	// Find all direct children (files and subdirectories)
	for entryPath, entry := range w.treeEntries {
		if entryPath == dirPath {
			continue // Skip the directory itself
		}
		
		var isDirectChild bool
		var childName string
		
		if dirPath == "" {
			// Root directory: direct children have no "/" in their path
			if !strings.Contains(entryPath, "/") {
				isDirectChild = true
				childName = entryPath
			}
		} else {
			// Non-root directory: direct children start with dirPath + "/"
			if strings.HasPrefix(entryPath, pathPrefix) {
				remainingPath := entryPath[len(pathPrefix):]
				if !strings.Contains(remainingPath, "/") {
					isDirectChild = true
					childName = remainingPath
				}
			}
		}
		
		if isDirectChild {
			entries = append(entries, protocol.PackfileTreeEntry{
				FileMode: entry.Mode,
				FileName: childName,
				Hash:     entry.Hash.String(),
			})
		}
	}
	
	// Handle empty directories (they shouldn't exist in Git)
	if len(entries) == 0 {
		logger.Debug("Removing empty directory", "path", dirPath)
		// Empty directories don't exist in Git - remove them from treeEntries
		if dirPath != "" {
			delete(w.treeEntries, dirPath)
		}
		return nil
	}
	
	// Build the tree object
	treeObj, err := protocol.BuildTreeObject(crypto.SHA1, entries)
	if err != nil {
		return fmt.Errorf("build tree object for %q: %w", dirPath, err)
	}
	
	// Add to writer and storage
	w.writer.AddObject(treeObj)
	w.objStorage.Add(&treeObj)
	
	// Update the tree entry with the calculated hash
	if dirPath == "" {
		// This is the root tree
		w.lastTree = &treeObj
		logger.Debug("Built root tree", "hash", treeObj.Hash.String(), "entry_count", len(entries))
	} else {
		// Update the directory entry
		if dirEntry, exists := w.treeEntries[dirPath]; exists {
			dirEntry.Hash = treeObj.Hash
		}
		logger.Debug("Built directory tree", "path", dirPath, "hash", treeObj.Hash.String(), "entry_count", len(entries))
	}
	
	return nil
}

// Cleanup releases all resources held by the writer and clears staged changes.
// This method:
//   - Cleans up the underlying PackfileWriter (removes temp files)
//   - Clears all staged tree entries from memory
//   - Resets the writer state
//
// After calling Cleanup, the writer should not be used for further operations.
func (w *stagedWriter) Cleanup(ctx context.Context) error {
	if w.isCleanedUp {
		return ErrWriterCleanedUp
	}

	logger := log.FromContext(ctx)
	logger.Debug("Cleaning up staged writer")

	// Clean up the packfile writer (removes temp files)
	if err := w.writer.Cleanup(); err != nil {
		return fmt.Errorf("cleanup packfile writer: %w", err)
	}

	// Clear all staged changes from memory
	w.treeEntries = make(map[string]*FlatTreeEntry)
	w.dirtyPaths = make(map[string]bool)

	// Reset writer state
	w.writer = protocol.NewPackfileWriter(crypto.SHA1, w.storageMode)

	// Mark as cleaned up to prevent further use
	w.isCleanedUp = true

	logger.Debug("Staged writer cleanup completed")
	return nil
}
