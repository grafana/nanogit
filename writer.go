package nanogit

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

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
func (c *httpClient) NewStagedWriter(ctx context.Context, ref Ref) (StagedWriter, error) {
	commit, err := c.GetCommit(ctx, ref.Hash)
	if err != nil {
		return nil, fmt.Errorf("getting root tree: %w", err)
	}

	treeObj, err := c.getSingleObject(ctx, commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("getting tree object: %w", err)
	}

	if treeObj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("root is not a tree")
	}

	cache := make(map[string]*protocol.PackfileObject)
	cache[treeObj.Hash.String()] = treeObj

	currentTree, err := c.GetFlatTree(ctx, commit.Hash)
	if err != nil {
		return nil, fmt.Errorf("getting current tree: %w", err)
	}

	entries := make(map[string]*FlatTreeEntry, len(currentTree.Entries))
	for _, entry := range currentTree.Entries {
		entries[entry.Path] = &entry
	}

	// Create a packfile writer
	writer := protocol.NewPackfileWriter(crypto.SHA1)
	return &stagedWriter{
		httpClient: c,
		ref:        ref,
		writer:     writer,
		lastCommit: commit,
		lastTree:   treeObj,
		// TODO: I think we only need one
		treeCache:   cache,
		treeEntries: entries,
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
	*httpClient
	// Git reference being modified
	ref Ref
	// Packfile writer for creating objects
	writer *protocol.PackfileWriter
	// Last commit on the reference
	lastCommit *Commit
	// Root tree object from last commit
	lastTree *protocol.PackfileObject
	// Cache of fetched tree objects
	treeCache map[string]*protocol.PackfileObject
	// Flat mapping of paths to tree entries
	treeEntries map[string]*FlatTreeEntry
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
	entry, exists := w.treeEntries[path]
	if !exists {
		return false, nil
	}

	// Check if the entry is actually a blob
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
	if obj, ok := w.treeEntries[path]; ok {
		return nil, NewObjectAlreadyExistsError(obj.Hash)
	}

	// Create the blob for the file content
	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return nil, fmt.Errorf("create blob: %w", err)
	}

	w.logger.Debug("created blob", "hash", blobHash.String())
	w.treeEntries[path] = &FlatTreeEntry{
		Path: path,
		Hash: blobHash,
		Type: protocol.ObjectTypeBlob,
		Mode: 0o100644,
	}

	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash); err != nil {
		return nil, fmt.Errorf("add new blob to tree: %w", err)
	}

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
	if w.treeEntries[path] == nil {
		return nil, NewPathNotFoundError(path)
	}

	// Create the blob for the file content
	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return nil, fmt.Errorf("create blob: %w", err)
	}

	w.logger.Debug("created blob", "hash", blobHash.String())
	w.treeEntries[path] = &FlatTreeEntry{
		Path: path,
		Hash: blobHash,
		Type: protocol.ObjectTypeBlob,
	}

	// Add the new entry
	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash); err != nil {
		return nil, fmt.Errorf("update tree with updated blob: %w", err)
	}

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
	existing, ok := w.treeEntries[path]
	if !ok {
		return nil, NewPathNotFoundError(path)
	}

	if existing.Type != protocol.ObjectTypeBlob {
		return nil, NewUnexpectedObjectTypeError(existing.Hash, protocol.ObjectTypeBlob, existing.Type)
	}

	blobHash := existing.Hash
	w.logger.Debug("deleting blob", "path", path)

	// Remove the entry from our tracking
	delete(w.treeEntries, path)

	// Update the tree structure to remove the entry
	if err := w.removeBlobFromTree(ctx, path); err != nil {
		return nil, fmt.Errorf("remove blob from tree: %w", err)
	}

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
	existing, ok := w.treeEntries[path]
	if !ok {
		return nil, NewPathNotFoundError(path)
	}

	if existing.Type != protocol.ObjectTypeTree {
		return nil, NewUnexpectedObjectTypeError(existing.Hash, protocol.ObjectTypeTree, existing.Type)
	}
	treeHash := existing.Hash

	w.logger.Debug("deleting tree", "path", path)

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
		w.logger.Debug("removing entry", "path", entryPath)
		delete(w.treeEntries, entryPath)
	}

	// Update the tree structure to remove the directory entry
	if err := w.removeTreeFromTree(ctx, path); err != nil {
		return nil, fmt.Errorf("remove tree from entire tree: %w", err)
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
	// Check if there are any changes to commit
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
		return nil, fmt.Errorf("create commit: %w", err)
	}

	w.lastCommit = &Commit{
		Hash:      commitHash,
		Tree:      w.lastTree.Hash,
		Parent:    w.lastCommit.Hash,
		Author:    author,
		Committer: committer,
		Message:   message,
	}

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
	// Check if there are any objects to push
	if !w.writer.HasObjects() {
		return ErrNothingToPush
	}

	// TODO: write in chunks and not having all bytes in memory
	// Write the packfile
	packfile, err := w.writer.WritePackfile(w.ref.Name, w.ref.Hash)
	if err != nil {
		return fmt.Errorf("write packfile: %w", err)
	}

	// Send the packfile to the server
	if _, err := w.receivePack(ctx, packfile); err != nil {
		return fmt.Errorf("send packfile: %w", err)
	}

	// Reset things to accumulate things for next push
	w.writer = protocol.NewPackfileWriter(crypto.SHA1)

	return nil
}

// addMissingOrStaleTreeEntries updates the tree structure to include a new or updated blob.
// This method handles the complex tree manipulation required when adding files to Git:
//   - Creates missing intermediate directories as needed
//   - Updates existing tree objects to include the new blob
//   - Properly handles nested directory structures
//   - Maintains proper Git tree object format and hashing
//
// The method works by traversing the path from the deepest directory up to the root,
// creating or updating tree objects as necessary to accommodate the new blob.
func (w *stagedWriter) addMissingOrStaleTreeEntries(ctx context.Context, path string, blobHash hash.Hash) error {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Get the file name and directory parts
	fileName := pathParts[len(pathParts)-1]
	dirParts := pathParts[:len(pathParts)-1]

	current := protocol.PackfileTreeEntry{
		FileMode: 0o100644,
		FileName: fileName,
		Hash:     blobHash.String(),
	}

	// Iterate bottom up checking if the existing tree.
	// if it does not exist or hash is different, create it with the previous tree entry.
	// if it exists and hash is the same, continue.
	// Add the file to the tree
	for i := len(dirParts) - 1; i >= 0; i-- {
		currentPath := strings.Join(dirParts[:i+1], "/")
		// Check if not a tree
		existingObj, exists := w.treeEntries[currentPath]
		if exists && existingObj.Type != protocol.ObjectTypeTree {
			return NewUnexpectedObjectTypeError(existingObj.Hash, protocol.ObjectTypeTree, existingObj.Type)
		}

		// Create new tree
		if !exists {
			// Create new tree
			treeObj, err := protocol.BuildTreeObject(crypto.SHA1, []protocol.PackfileTreeEntry{current})
			if err != nil {
				return fmt.Errorf("create tree for %s: %w", currentPath, err)
			}

			w.writer.AddObject(treeObj)
			w.logger.Debug("add new tree object", "path", currentPath, "hash", treeObj.Hash.String(), "child", current.Hash, "child_path", current.FileName)

			// Add this tree to the parent's entries
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     treeObj.Hash.String(),
			}

			w.treeCache[treeObj.Hash.String()] = &treeObj
			w.treeEntries[currentPath] = &FlatTreeEntry{
				Path: currentPath,
				Hash: treeObj.Hash,
				Type: protocol.ObjectTypeTree,
				Mode: 0o40000,
			}
		} else {
			// If tree exists, add our entries to it
			existingTree, ok := w.treeCache[existingObj.Hash.String()]
			if !ok {
				w.logger.Info("fetch tree not found in cache", "path", currentPath, "hash", existingObj.Hash.String())
				var err error
				existingTree, err = w.getSingleObject(ctx, existingObj.Hash)
				if err != nil {
					return fmt.Errorf("get existing tree %s: %w", currentPath, err)
				}
				w.treeCache[existingObj.Hash.String()] = existingTree
				w.logger.Info("tree object found in remote", "path", currentPath, "hash", existingObj.Hash.String(), "entries", len(existingTree.Tree))
			} else {
				w.logger.Debug("tree object found in cache", "path", currentPath, "hash", existingObj.Hash.String(), "entries", len(existingTree.Tree))
			}

			newObj, err := w.updateTreeEntry(existingTree, current)
			if err != nil {
				return fmt.Errorf("update tree for %s: %w", currentPath, err)
			}

			w.logger.Debug("add updated tree object", "path", currentPath, "hash", newObj.Hash.String(), "children", len(existingTree.Tree)+1)
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     newObj.Hash.String(),
			}

			w.treeCache[newObj.Hash.String()] = newObj
			w.treeEntries[currentPath] = &FlatTreeEntry{
				Path: currentPath,
				Hash: newObj.Hash,
				Type: protocol.ObjectTypeTree,
			}
		}
	}

	if len(w.lastTree.Tree) == 0 {
		newRoot, err := protocol.BuildTreeObject(crypto.SHA1, []protocol.PackfileTreeEntry{current})
		if err != nil {
			return fmt.Errorf("build new root tree: %w", err)
		}

		w.writer.AddObject(newRoot)
		w.lastTree = &newRoot
		w.treeCache[newRoot.Hash.String()] = &newRoot

		return nil
	}

	newRootObj, err := w.updateTreeEntry(w.lastTree, current)
	if err != nil {
		return fmt.Errorf("update root tree: %w", err)
	}
	w.treeCache[newRootObj.Hash.String()] = newRootObj
	w.lastTree = newRootObj

	return nil
}

// updateTreeEntry creates a new tree object by adding or updating an entry in an existing tree.
// This method takes an existing tree object and either adds a new entry or updates an existing
// entry with the same filename. It maintains proper Git tree object sorting and formatting.
//
// Parameters:
//   - obj: The existing tree object to modify
//   - current: The tree entry to add or update
//
// Returns:
//   - *protocol.PackfileObject: New tree object with the updated entry
//   - error: Error if tree creation fails
func (w *stagedWriter) updateTreeEntry(obj *protocol.PackfileObject, current protocol.PackfileTreeEntry) (*protocol.PackfileObject, error) {
	if obj == nil {
		return nil, errors.New("object is nil")
	}

	if obj.Type != protocol.ObjectTypeTree {
		return nil, NewUnexpectedObjectTypeError(obj.Hash, protocol.ObjectTypeTree, obj.Type)
	}

	// Create a new slice for the updated entries
	combinedEntries := make([]protocol.PackfileTreeEntry, 0, len(obj.Tree))

	// Add all entries except the one we're updating
	for _, entry := range obj.Tree {
		if entry.FileName != current.FileName {
			combinedEntries = append(combinedEntries, entry)
		}
	}

	// Add the new/updated entry
	combinedEntries = append(combinedEntries, current)

	newObj, err := protocol.BuildTreeObject(crypto.SHA1, combinedEntries)
	if err != nil {
		return nil, fmt.Errorf("build tree object: %w", err)
	}

	w.writer.AddObject(newObj)

	return &newObj, nil
}

// removeBlobFromTree removes a blob entry from the tree structure and updates all parent trees.
// This method handles the complex tree manipulation required when deleting files from Git:
//   - Removes the blob from its immediate parent directory
//   - Updates all ancestor directories with new tree hashes
//   - Properly handles nested directory structures
//   - Maintains Git tree object integrity throughout the hierarchy
//
// The method works by traversing from the immediate parent directory up to the root,
// updating each tree object to reflect the removal of the blob or updated child tree.
func (w *stagedWriter) removeBlobFromTree(ctx context.Context, path string) error {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Get the file name and directory parts
	fileName := pathParts[len(pathParts)-1]
	dirParts := pathParts[:len(pathParts)-1]

	// First, remove the file from its immediate parent directory
	if len(dirParts) == 0 {
		// File is in root directory
		newRootObj, err := w.removeTreeEntry(w.lastTree, fileName)
		if err != nil {
			return fmt.Errorf("remove file from root tree: %w", err)
		}
		w.lastTree = newRootObj
		w.treeCache[newRootObj.Hash.String()] = newRootObj
		w.logger.Debug("removed file from root", "file", fileName, "new_root_hash", newRootObj.Hash.String())
		return nil
	}

	// File is in a subdirectory - we need to update the tree hierarchy
	// Start from the immediate parent and work up to root
	var updatedChildHash hash.Hash

	for i := len(dirParts) - 1; i >= 0; i-- {
		currentPath := strings.Join(dirParts[:i+1], "/")

		// Get the tree we need to modify
		existingObj, exists := w.treeEntries[currentPath]
		if !exists {
			return fmt.Errorf("parent directory %s does not exist: %w", currentPath, NewPathNotFoundError(currentPath))
		}

		if existingObj.Type != protocol.ObjectTypeTree {
			return fmt.Errorf("parent path is not a tree: %w", NewUnexpectedObjectTypeError(existingObj.Hash, protocol.ObjectTypeTree, existingObj.Type))
		}

		// Get tree object from cache or fetch it
		treeObj, ok := w.treeCache[existingObj.Hash.String()]
		if !ok {
			var err error
			treeObj, err = w.getSingleObject(ctx, existingObj.Hash)
			if err != nil {
				return fmt.Errorf("get tree %s: %w", currentPath, err)
			}
			w.treeCache[existingObj.Hash.String()] = treeObj
		}

		var newObj *protocol.PackfileObject
		var err error

		if i == len(dirParts)-1 {
			// This is the immediate parent - remove the file
			newObj, err = w.removeTreeEntry(treeObj, fileName)
			if err != nil {
				return fmt.Errorf("remove file from tree %s: %w", currentPath, err)
			}
			w.logger.Debug("removed file from parent tree", "path", currentPath, "file", fileName, "new_hash", newObj.Hash.String())
		} else {
			// This is an ancestor directory - update with new child hash
			childDirName := dirParts[i+1]
			childEntry := protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: childDirName,
				Hash:     updatedChildHash.String(),
			}
			newObj, err = w.updateTreeEntry(treeObj, childEntry)
			if err != nil {
				return fmt.Errorf("update tree %s with new child: %w", currentPath, err)
			}
			w.logger.Debug("updated parent tree with new child", "path", currentPath, "child", childDirName, "child_hash", updatedChildHash.String(), "new_hash", newObj.Hash.String())
		}

		// Store the new tree hash for the next iteration
		updatedChildHash = newObj.Hash

		// Update our references
		w.treeCache[newObj.Hash.String()] = newObj
		w.treeEntries[currentPath] = &FlatTreeEntry{
			Path: currentPath,
			Hash: newObj.Hash,
			Type: protocol.ObjectTypeTree,
		}
	}

	// Finally, update the root tree with the new top-level directory hash
	rootDirName := dirParts[0]
	rootDirEntry := protocol.PackfileTreeEntry{
		FileMode: 0o40000, // Directory mode
		FileName: rootDirName,
		Hash:     updatedChildHash.String(),
	}

	newRootObj, err := w.updateTreeEntry(w.lastTree, rootDirEntry)
	if err != nil {
		return fmt.Errorf("update root tree: %w", err)
	}

	w.lastTree = newRootObj
	w.treeCache[newRootObj.Hash.String()] = newRootObj
	w.logger.Debug("updated root tree", "dir", rootDirName, "dir_hash", updatedChildHash.String(), "new_root_hash", newRootObj.Hash.String())

	return nil
}

// removeTreeFromTree removes a directory tree from the tree structure and updates all parent trees.
// This method handles the complex tree manipulation required when deleting directories from Git:
//   - Removes the directory from its immediate parent
//   - Updates all ancestor directories with new tree hashes
//   - Properly handles nested directory structures
//   - Maintains Git tree object integrity throughout the hierarchy
//
// This is similar to removeBlobFromTree but handles directory removal instead of file removal.
func (w *stagedWriter) removeTreeFromTree(ctx context.Context, path string) error {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Get the directory name and parent directory parts
	dirName := pathParts[len(pathParts)-1]
	parentParts := pathParts[:len(pathParts)-1]

	// First, remove the directory from its immediate parent
	if len(parentParts) == 0 {
		// Directory is in root
		newRootObj, err := w.removeTreeEntry(w.lastTree, dirName)
		if err != nil {
			return fmt.Errorf("remove directory from root tree: %w", err)
		}
		w.lastTree = newRootObj
		w.treeCache[newRootObj.Hash.String()] = newRootObj
		w.logger.Debug("removed directory from root", "dir", dirName, "new_root_hash", newRootObj.Hash.String())
		return nil
	}

	// Directory is in a subdirectory - we need to update the tree hierarchy
	// Start from the immediate parent and work up to root
	var updatedChildHash hash.Hash

	for i := len(parentParts) - 1; i >= 0; i-- {
		currentPath := strings.Join(parentParts[:i+1], "/")

		// Get the tree we need to modify
		existingObj, exists := w.treeEntries[currentPath]
		if !exists {
			return fmt.Errorf("parent directory %s does not exist: %w", currentPath, NewPathNotFoundError(currentPath))
		}

		if existingObj.Type != protocol.ObjectTypeTree {
			return fmt.Errorf("parent path is not a tree: %w", NewUnexpectedObjectTypeError(existingObj.Hash, protocol.ObjectTypeTree, existingObj.Type))
		}

		// Get tree object from cache or fetch it
		treeObj, ok := w.treeCache[existingObj.Hash.String()]
		if !ok {
			var err error
			treeObj, err = w.getSingleObject(ctx, existingObj.Hash)
			if err != nil {
				return fmt.Errorf("get tree %s: %w", currentPath, err)
			}
			w.treeCache[existingObj.Hash.String()] = treeObj
		}

		var newObj *protocol.PackfileObject
		var err error

		if i == len(parentParts)-1 {
			// This is the immediate parent - remove the directory
			newObj, err = w.removeTreeEntry(treeObj, dirName)
			if err != nil {
				return fmt.Errorf("remove directory from tree %s: %w", currentPath, err)
			}
			w.logger.Debug("removed directory from parent tree", "path", currentPath, "dir", dirName, "new_hash", newObj.Hash.String())
		} else {
			// This is an ancestor directory - update with new child hash
			childDirName := parentParts[i+1]
			childEntry := protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: childDirName,
				Hash:     updatedChildHash.String(),
			}
			newObj, err = w.updateTreeEntry(treeObj, childEntry)
			if err != nil {
				return fmt.Errorf("update tree %s with new child: %w", currentPath, err)
			}
			w.logger.Debug("updated parent tree with new child", "path", currentPath, "child", childDirName, "child_hash", updatedChildHash.String(), "new_hash", newObj.Hash.String())
		}

		// Store the new tree hash for the next iteration
		updatedChildHash = newObj.Hash

		// Update our references
		w.treeCache[newObj.Hash.String()] = newObj
		w.treeEntries[currentPath] = &FlatTreeEntry{
			Path: currentPath,
			Hash: newObj.Hash,
			Type: protocol.ObjectTypeTree,
		}
	}

	// Finally, update the root tree with the new top-level directory hash
	rootDirName := parentParts[0]
	rootDirEntry := protocol.PackfileTreeEntry{
		FileMode: 0o40000, // Directory mode
		FileName: rootDirName,
		Hash:     updatedChildHash.String(),
	}

	newRootObj, err := w.updateTreeEntry(w.lastTree, rootDirEntry)
	if err != nil {
		return fmt.Errorf("update root tree: %w", err)
	}

	w.lastTree = newRootObj
	w.treeCache[newRootObj.Hash.String()] = newRootObj
	w.logger.Debug("updated root tree", "dir", rootDirName, "dir_hash", updatedChildHash.String(), "new_root_hash", newRootObj.Hash.String())

	return nil
}

// removeTreeEntry creates a new tree object by removing a specific entry from an existing tree.
// This is a lower-level helper method that handles the actual removal of an entry from a
// Git tree object, creating a new tree object with the filtered entries.
//
// Parameters:
//   - obj: The tree object to modify
//   - targetFileName: The filename of the entry to remove
//
// Returns:
//   - *protocol.PackfileObject: New tree object without the specified entry
//   - error: Error if tree creation fails
//
// Note: If the target entry is not found, the original object is returned unchanged.
func (w *stagedWriter) removeTreeEntry(obj *protocol.PackfileObject, targetFileName string) (*protocol.PackfileObject, error) {
	if obj == nil {
		return nil, errors.New("object is nil")
	}

	if obj.Type != protocol.ObjectTypeTree {
		return nil, NewUnexpectedObjectTypeError(obj.Hash, protocol.ObjectTypeTree, obj.Type)
	}

	// Create a new slice excluding the target entry
	filteredEntries := make([]protocol.PackfileTreeEntry, 0, len(obj.Tree))
	found := false

	for _, entry := range obj.Tree {
		if entry.FileName != targetFileName {
			filteredEntries = append(filteredEntries, entry)
		} else {
			found = true
		}
	}

	if !found {
		// Entry not found in tree, but this might be okay for intermediate trees
		// Return the original object unchanged
		return obj, nil
	}

	// Build new tree object with the filtered entries
	newObj, err := protocol.BuildTreeObject(crypto.SHA1, filteredEntries)
	if err != nil {
		return nil, fmt.Errorf("build tree object: %w", err)
	}

	w.writer.AddObject(newObj)

	return &newObj, nil
}
