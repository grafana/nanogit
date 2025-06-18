package nanogit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

// FlatTreeEntry represents a single entry in a flattened Git tree structure.
// Unlike TreeEntry, this includes the full path from the repository root,
// making it suitable for operations that need to work with complete file paths.
//
// A flattened tree contains all files and directories recursively, with each
// entry having its complete path from the repository root.
type FlatTreeEntry struct {
	// Name is the base filename (e.g., "file.txt")
	Name string
	// Path is the full path from repository root (e.g., "dir/subdir/file.txt")
	Path string
	// Mode is the file mode in octal (e.g., 0o100644 for regular files, 0o40000 for directories)
	Mode uint32
	// Hash is the SHA-1 hash of the object
	Hash hash.Hash
	// Type is the type of Git object (blob for files, tree for directories)
	Type protocol.ObjectType
}

// FlatTree represents a recursive, flattened view of a Git tree structure.
// This provides a complete list of all files and directories in the tree,
// with each entry containing its full path from the repository root.
//
// This is useful for operations that need to:
//   - List all files in a repository
//   - Search for specific files by path
//   - Compare entire directory structures
//   - Generate file listings for display
type FlatTree struct {
	// Entries contains all files and directories in the tree (recursive)
	Entries []FlatTreeEntry
	// Hash is the SHA-1 hash of the root tree object
	Hash hash.Hash
}

// TreeEntry represents a single entry in a Git tree object.
// This contains only direct children of the tree (non-recursive),
// similar to what you'd see with 'ls' in a directory.
type TreeEntry struct {
	// Name is the filename or directory name
	Name string
	// Mode is the file mode in octal (e.g., 0o100644 for files, 0o40000 for directories)
	Mode uint32
	// Hash is the SHA-1 hash of the object
	Hash hash.Hash
	// Type is the type of Git object (blob for files, tree for directories)
	Type protocol.ObjectType
}

// Tree represents a single Git tree object containing direct children only.
// This provides a non-recursive view of a directory, showing only the
// immediate files and subdirectories within it.
//
// This is useful for operations that need to:
//   - Browse directory contents one level at a time
//   - Implement tree navigation interfaces
//   - Work with specific directory levels
//   - Minimize memory usage when not all files are needed
type Tree struct {
	// Entries contains the direct children of this tree (non-recursive)
	Entries []TreeEntry
	// Hash is the SHA-1 hash of this tree object
	Hash hash.Hash
}

// GetFlatTree retrieves a complete, recursive view of all files and directories
// in a Git tree structure. This method flattens the entire tree hierarchy into
// a single list where each entry contains its full path from the repository root.
//
// The method can accept either a tree hash directly or a commit hash (in which
// case it will extract the tree from the commit).
//
// Parameters:
//   - ctx: Context for the operation
//   - h: Hash of the commit object
//
// Returns:
//   - *FlatTree: Complete recursive listing of all files and directories
//   - error: Error if the hash is invalid, object not found, or processing fails
//
// Example:
//
//	flatTree, err := client.GetFlatTree(ctx, commitHash)
//	for _, entry := range flatTree.Entries {
//	    fmt.Printf("%s (%s)\n", entry.Path, entry.Type)
//	}
func (c *httpClient) GetFlatTree(ctx context.Context, commitHash hash.Hash) (*FlatTree, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Get flat tree",
		"commit_hash", commitHash.String())

	ctx, _ = storage.FromContextOrInMemory(ctx)

	allTreeObjects, rootTree, err := c.fetchAllTreeObjects(ctx, commitHash)
	if err != nil {
		return nil, fmt.Errorf("fetch tree objects for commit %s: %w", commitHash.String(), err)
	}

	flatTree, err := c.flatten(ctx, rootTree, allTreeObjects)
	if err != nil {
		return nil, fmt.Errorf("flatten tree %s: %w", rootTree.Hash.String(), err)
	}

	logger.Debug("Flat tree retrieved",
		"commit_hash", commitHash.String(),
		"tree_hash", rootTree.Hash.String(),
		"entry_count", len(flatTree.Entries))
	return flatTree, nil
}

// fetchAllTreeObjects collects all tree objects needed for the flat tree by starting with
// an initial request and iteratively fetching missing tree objects in batches.
func (c *httpClient) fetchAllTreeObjects(ctx context.Context, commitHash hash.Hash) (storage.PackfileStorage, *protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Fetch tree objects",
		"commit_hash", commitHash.String())

	var totalRequests int
	var totalObjectsFetched int

	ctx, allObjects := storage.FromContextOrInMemory(ctx)
	totalRequests++

	initialObjects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:   true,
		NoBlobFilter: true,
		Want:         []hash.Hash{commitHash},
		Shallow:      true,
		Deepen:       1,
		Done:         true,
	})
	if err != nil {
		// TODO: handle this at the client level
		if strings.Contains(err.Error(), "not our ref") {
			return nil, nil, NewObjectNotFoundError(commitHash)
		}

		return nil, nil, fmt.Errorf("fetch commit tree %s: %w", commitHash.String(), err)
	}

	commitObj, exists := initialObjects[commitHash.String()]
	if !exists {
		return nil, nil, NewObjectNotFoundError(commitHash)
	}

	if commitObj.Type != protocol.ObjectTypeCommit {
		return nil, nil, NewUnexpectedObjectTypeError(commitHash, protocol.ObjectTypeCommit, commitObj.Type)
	}

	totalObjectsFetched = len(initialObjects)

	var commitCount, treeCount, blobCount, otherCount int
	for _, obj := range initialObjects {
		switch obj.Type {
		case protocol.ObjectTypeCommit:
			commitCount++
		case protocol.ObjectTypeTree:
			treeCount++
		case protocol.ObjectTypeBlob:
			blobCount++
		case protocol.ObjectTypeRefDelta, protocol.ObjectTypeOfsDelta, protocol.ObjectTypeTag, protocol.ObjectTypeReserved, protocol.ObjectTypeInvalid:
			otherCount++
		default:
			otherCount++
		}
	}

	logger.Debug("Initial fetch completed",
		"commit_hash", commitHash.String(),
		"object_count", len(initialObjects),
		"commit_count", commitCount,
		"tree_count", treeCount,
		"blob_count", blobCount,
		"other_count", otherCount)

	rootTree, err := c.findRootTree(ctx, commitHash, allObjects)
	if err != nil {
		return nil, nil, fmt.Errorf("find root tree for commit %s: %w", commitHash.String(), err)
	}

	pending := []hash.Hash{}
	retries := []hash.Hash{}
	if rootTree == nil {
		pending = append(pending, commitObj.Commit.Tree)
	}

	processedTrees := make(map[string]bool)
	requestedHashes := make(map[string]bool)

	pending, err = c.collectMissingTreeHashes(ctx, initialObjects, allObjects, pending, processedTrees, requestedHashes)
	if err != nil {
		return nil, nil, fmt.Errorf("collect missing trees: %w", err)
	}

	logger.Debug("Initial tree analysis completed",
		"pending_count", len(pending),
		"processed_count", len(processedTrees))

	const (
		batchSize      = 10
		retryBatchSize = 10
		maxRetries     = 3
		maxBatches     = 1000
	)

	retryCount := make(map[string]int)
	var batchNumber int

	for len(pending) > 0 || len(retries) > 0 {
		batchNumber++

		if batchNumber > maxBatches {
			logger.Error("Maximum batch limit exceeded",
				"max_batches", maxBatches,
				"pending_count", len(pending),
				"retry_count", len(retries),
				"processed_count", len(processedTrees),
				"total_objects", allObjects.Len())
			return nil, nil, fmt.Errorf("exceeded maximum batch limit (%d), possible infinite loop", maxBatches)
		}

		var currentBatch []hash.Hash
		var batchType string

		if len(retries) > 0 {
			batchType = "retry"
			currentBatch = retries
			if len(retries) > retryBatchSize {
				currentBatch = retries[:retryBatchSize]
				retries = retries[retryBatchSize:]
			} else {
				retries = nil
			}
		} else {
			batchType = "normal"
			currentBatch = pending
			if len(pending) > batchSize {
				currentBatch = pending[:batchSize]
				pending = pending[batchSize:]
			} else {
				pending = nil
			}
		}

		logger.Debug("Process batch",
			"batch_number", batchNumber,
			"batch_type", batchType,
			"batch_size", len(currentBatch),
			"pending_count", len(pending),
			"retry_count", len(retries))

		totalRequests++
		objects, err := c.Fetch(ctx, client.FetchOptions{
			NoProgress:   true,
			NoBlobFilter: true,
			Want:         currentBatch,
			Done:         true,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("fetch tree batch: %w", err)
		}

		totalObjectsFetched += len(objects)

		var requestedReceived, additionalReceived int
		for _, requestedHash := range currentBatch {
			if _, exists := objects[requestedHash.String()]; exists {
				requestedReceived++
			}
		}
		additionalReceived = len(objects) - requestedReceived

		logger.Debug("Batch completed",
			"batch_number", batchNumber,
			"requested_count", len(currentBatch),
			"received_count", len(objects),
			"requested_received", requestedReceived,
			"additional_received", additionalReceived)

		for _, requestedHash := range currentBatch {
			if _, exists := objects[requestedHash.String()]; exists {
				continue
			}

			hashStr := requestedHash.String()
			retryCount[hashStr]++

			if retryCount[hashStr] > maxRetries {
				logger.Error("Object not returned after max retries",
					"hash", hashStr,
					"max_retries", maxRetries,
					"total_requests", totalRequests)
				return nil, nil, fmt.Errorf("object %s not returned after %d attempts: %w", hashStr, maxRetries, ErrObjectNotFound)
			}

			if !requestedHashes[hashStr] {
				retries = append(retries, requestedHash)
				requestedHashes[hashStr] = true
			}
		}

		pending, err = c.collectMissingTreeHashes(ctx, objects, allObjects, pending, processedTrees, requestedHashes)
		if err != nil {
			return nil, nil, fmt.Errorf("collect missing trees from batch: %w", err)
		}
	}

	logger.Debug("Tree collection completed",
		"commit_hash", commitHash.String(),
		"total_requests", totalRequests,
		"total_objects", totalObjectsFetched,
		"total_batches", batchNumber)

	return allObjects, rootTree, nil
}

// collectMissingTreeHashes processes tree objects and collects missing child tree hashes.
// It iterates through the provided objects, optionally adds them to allObjects if addToCollection is true,
// and identifies any missing child tree objects that need to be fetched.
func (c *httpClient) collectMissingTreeHashes(ctx context.Context, objects map[string]*protocol.PackfileObject, allObjects storage.PackfileStorage, pending []hash.Hash, processedTrees map[string]bool, requestedHashes map[string]bool) ([]hash.Hash, error) {
	logger := log.FromContext(ctx)
	var treesProcessed int
	var newTreesFound int

	// Mark current pending hashes as requested
	for _, h := range pending {
		requestedHashes[h.String()] = true
	}

	for _, obj := range objects {
		if obj.Type != protocol.ObjectTypeTree {
			continue
		}

		// Skip if we've already processed this tree for dependencies
		if processedTrees[obj.Hash.String()] {
			continue
		}

		treesProcessed++
		processedTrees[obj.Hash.String()] = true

		for _, entry := range obj.Tree {
			// If it's a file, we can ignore it
			if entry.FileMode&0o40000 == 0 {
				continue
			}

			entryHash, err := hash.FromHex(entry.Hash)
			if err != nil {
				return nil, fmt.Errorf("parsing child hash %s: %w", entry.Hash, err)
			}

			// Skip if we already have this object
			if _, exists := allObjects.GetByType(entryHash, protocol.ObjectTypeTree); exists {
				continue
			}

			// Skip if we've already requested this hash
			if requestedHashes[entry.Hash] {
				continue
			}

			pending = append(pending, entryHash)
			requestedHashes[entry.Hash] = true
			newTreesFound++
		}
	}

	if newTreesFound > 0 {
		logger.Debug("discovered tree dependencies",
			"trees_processed", treesProcessed,
			"new_trees_found", newTreesFound,
			"total_pending", len(pending))
	}

	return pending, nil
}

// findRootTree locates the root tree object from the target hash and available objects.
// It handles both commit and tree target objects, extracting the tree hash and object as needed.
func (c *httpClient) findRootTree(ctx context.Context, commitHash hash.Hash, allObjects storage.PackfileStorage) (*protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)
	obj, exists := allObjects.GetByType(commitHash, protocol.ObjectTypeCommit)
	if !exists {
		return nil, NewObjectNotFoundError(commitHash)
	}

	// Extract tree hash from commit
	treeHash, err := hash.FromHex(obj.Commit.Tree.String())
	if err != nil {
		return nil, fmt.Errorf("parsing tree hash: %w", err)
	}

	// Check if the tree object is already in our available objects
	treeObj, exists := allObjects.GetByType(treeHash, protocol.ObjectTypeTree)
	if !exists {
		return nil, NewObjectNotFoundError(treeHash)
	}

	logger.Debug("resolved commit to tree",
		"commit_hash", commitHash.String(),
		"tree_hash", treeHash.String(),
		"tree_available", treeObj != nil)

	return treeObj, nil
}

// flatten converts collected tree objects into a flat tree structure using breadth-first traversal.
func (c *httpClient) flatten(ctx context.Context, rootTree *protocol.PackfileObject, allTreeObjects storage.PackfileStorage) (*FlatTree, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Flatten tree", "treeHash", rootTree.Hash.String())

	// Build flat entries iteratively using breadth-first traversal
	var entries []FlatTreeEntry

	// Queue for processing: each item contains the tree object and its base path
	type queueItem struct {
		tree     *protocol.PackfileObject
		basePath string
	}

	queue := []queueItem{{tree: rootTree, basePath: ""}}
	logger.Debug("Traverse tree breadth-first for pending objects", "queueSize", len(queue))

	// Process the queue iteratively
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Process all entries in this tree
		for _, entry := range current.tree.Tree {
			entryHash, err := hash.FromHex(entry.Hash)
			if err != nil {
				logger.Debug("Failed to parse entry hash",
					"hash", entry.Hash,
					"error", err)
				return nil, fmt.Errorf("parsing entry hash %s: %w", entry.Hash, err)
			}

			// Build the full path for this entry
			entryPath := entry.FileName
			if current.basePath != "" {
				entryPath = current.basePath + "/" + entry.FileName
			}

			// Determine the type based on the mode
			entryType := protocol.ObjectTypeBlob
			if entry.FileMode&0o40000 != 0 {
				entryType = protocol.ObjectTypeTree
			}

			// Add this entry to results
			entries = append(entries, FlatTreeEntry{
				Name: entry.FileName,
				Path: entryPath,
				Mode: uint32(entry.FileMode),
				Hash: entryHash,
				Type: entryType,
			})

			// If this is a tree, add it to the queue for processing
			if entryType == protocol.ObjectTypeTree {
				childTree, exists := allTreeObjects.GetByType(entryHash, protocol.ObjectTypeTree)
				if !exists {
					logger.Debug("Child tree not found",
						"hash", entry.Hash,
						"path", entryPath)
					return nil, fmt.Errorf("tree object %s not found in collection", entry.Hash)
				}
				queue = append(queue, queueItem{
					tree:     childTree,
					basePath: entryPath,
				})
			}
		}

		logger.Debug("Queue progress",
			"remaining", len(queue),
			"processedEntries", len(entries))
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	logger.Debug("Tree flattening completed",
		"treeHash", rootTree.Hash.String(),
		"totalEntries", len(entries))
	return &FlatTree{
		Entries: entries,
		Hash:    rootTree.Hash,
	}, nil
}

// GetTree retrieves a single Git tree object showing only direct children.
// This method provides a non-recursive view of a directory, similar to running
// 'ls' in a Unix directory - you see only the immediate contents, not subdirectories.
//
// The method can accept either a tree hash directly or a commit hash (in which
// case it will extract the tree from the commit).
//
// Parameters:
//   - ctx: Context for the operation
//   - h: Hash of either a tree object
//
// Returns:
//   - *Tree: Tree object containing direct children only
//   - error: Error if the hash is invalid, object not found, or processing fails
//
// Example:
//
//	tree, err := client.GetTree(ctx, treeHash)
//	for _, entry := range tree.Entries {
//	    if entry.Type == protocol.ObjectTypeTree {
//	        fmt.Printf("📁 %s/\n", entry.Name)
//	    } else {
//	        fmt.Printf("📄 %s\n", entry.Name)
//	    }
//	}
func (c *httpClient) GetTree(ctx context.Context, h hash.Hash) (*Tree, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Get tree",
		"tree_hash", h.String())

	tree, err := c.getTree(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("get tree object %s: %w", h.String(), err)
	}

	result, err := packfileObjectToTree(tree)
	if err != nil {
		return nil, fmt.Errorf("convert tree object %s: %w", h.String(), err)
	}

	logger.Debug("Tree retrieved",
		"tree_hash", h.String(),
		"entry_count", len(result.Entries))
	return result, nil
}

func (c *httpClient) getTree(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Fetch tree object", "hash", want.String())

	objects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:   true,
		NoBlobFilter: true,
		Want:         []hash.Hash{want},
		Done:         true,
	})
	if err != nil {
		// TODO: handle this at the client level
		if strings.Contains(err.Error(), "not our ref") {
			return nil, NewObjectNotFoundError(want)
		}

		logger.Debug("Failed to fetch tree objects", "hash", want.String(), "error", err)
		return nil, fmt.Errorf("fetching tree objects: %w", err)
	}

	if len(objects) == 0 {
		logger.Debug("No objects returned", "hash", want.String())
		return nil, NewObjectNotFoundError(want)
	}

	// TODO: can we do in the fetch?
	for _, obj := range objects {
		if obj.Type != protocol.ObjectTypeTree {
			logger.Debug("Unexpected object type",
				"hash", want.String(),
				"expectedType", protocol.ObjectTypeTree,
				"actualType", obj.Type)
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeTree, obj.Type)
		}
	}

	// Due to Git protocol limitations, when fetching a tree object, we receive all tree objects
	// in the path. We must filter the response to extract only the requested tree.
	if obj, ok := objects[want.String()]; ok {
		logger.Debug("Tree object found", "hash", want.String())
		return obj, nil
	}

	logger.Debug("Tree object not found in response", "hash", want.String())
	return nil, NewObjectNotFoundError(want)
}

// packfileObjectToTree converts a packfile object to a tree object.
// It returns the direct children of the tree.
func packfileObjectToTree(obj *protocol.PackfileObject) (*Tree, error) {
	if obj.Type != protocol.ObjectTypeTree {
		return nil, NewUnexpectedObjectTypeError(obj.Hash, protocol.ObjectTypeTree, obj.Type)
	}

	// Convert PackfileTreeEntry to TreeEntry (direct children only)
	entries := make([]TreeEntry, len(obj.Tree))
	for i, entry := range obj.Tree {
		entryHash, err := hash.FromHex(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("parsing hash: %w", err)
		}

		// Determine the type based on the mode
		entryType := protocol.ObjectTypeBlob
		if entry.FileMode&0o40000 != 0 {
			entryType = protocol.ObjectTypeTree
		}

		entries[i] = TreeEntry{
			Name: entry.FileName,
			Mode: uint32(entry.FileMode),
			Hash: entryHash,
			Type: entryType,
		}
	}

	return &Tree{
		Entries: entries,
		Hash:    obj.Hash,
	}, nil
}

// GetTreeByPath retrieves a tree object at a specific path by navigating through
// the directory structure. This method efficiently traverses the tree hierarchy
// to find the directory at the specified path without fetching unnecessary data.
//
// The path should use forward slashes ("/") as separators, similar to Unix paths.
// Empty path or "." returns the root tree.
//
// Parameters:
//   - ctx: Context for the operation
//   - rootHash: Hash of the root tree to start navigation from
//   - path: Directory path to navigate to (e.g., "src/main" or "docs/api")
//
// Returns:
//   - *Tree: Tree object at the specified path
//   - error: Error if path doesn't exist, contains non-directories, or navigation fails
//
// Example:
//
//	// Get the tree for the "src/components" directory
//	tree, err := client.GetTreeByPath(ctx, rootHash, "src/components")
//	if err != nil {
//	    return fmt.Errorf("directory not found: %w", err)
//	}
//
//	// List all files in that directory
//	for _, entry := range tree.Entries {
//	    fmt.Printf("%s\n", entry.Name)
//	}
func (c *httpClient) GetTreeByPath(ctx context.Context, rootHash hash.Hash, path string) (*Tree, error) {
	// If the path is "." or empty, return the root tree
	if path == "" || path == "." {
		return c.GetTree(ctx, rootHash)
	}

	logger := log.FromContext(ctx)
	logger.Debug("Get tree by path",
		"root_hash", rootHash.String(),
		"path", path)

	ctx, _ = storage.FromContextOrInMemory(ctx)

	parts := strings.Split(path, "/")
	currentHash := rootHash

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, errors.New("path component is empty")
		}

		currentPath := strings.Join(parts[:i+1], "/")

		logger.Debug("Navigate directory",
			"depth", i+1,
			"dir_name", part,
			"current_path", currentPath)

		currentTree, err := c.GetTree(ctx, currentHash)
		if err != nil {
			return nil, fmt.Errorf("get tree at %q: %w", currentPath, err)
		}

		found := false
		for _, entry := range currentTree.Entries {
			if entry.Name == part {
				if entry.Type != protocol.ObjectTypeTree {
					return nil, fmt.Errorf("path component %q is not a directory: %w", currentPath, NewUnexpectedObjectTypeError(entry.Hash, protocol.ObjectTypeTree, entry.Type))
				}
				currentHash = entry.Hash
				found = true
				break
			}
		}

		if !found {
			return nil, NewPathNotFoundError(currentPath)
		}
	}

	finalTree, err := c.GetTree(ctx, currentHash)
	if err != nil {
		return nil, fmt.Errorf("get final tree at %q: %w", path, err)
	}

	logger.Debug("Tree found by path",
		"path", path,
		"tree_hash", currentHash.String(),
		"entry_count", len(finalTree.Entries))
	return finalTree, nil
}
