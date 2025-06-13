package nanogit

import (
	"context"
	"fmt"
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
	// Ensure storage as it's a complex operation with multiple calls
	// and we may get more objects in the same request than expected in some responses
	ctx, _ = storage.FromContextOrInMemory(ctx)

	allTreeObjects, treeHash, err := c.fetchAllTreeObjects(ctx, commitHash)
	if err != nil {
		return nil, err
	}

	return c.flatten(treeHash, allTreeObjects)
}

// fetchAllTreeObjects collects all tree objects needed for the flat tree by starting with
// an initial request and iteratively fetching missing tree objects in batches.
func (c *httpClient) fetchAllTreeObjects(ctx context.Context, commitHash hash.Hash) (storage.PackfileStorage, hash.Hash, error) {
	logger := log.FromContext(ctx)
	// Track essential metrics
	var totalRequests int
	var totalObjectsFetched int

	ctx, allObjects := storage.FromContextOrInMemory(ctx)
	totalRequests++

	// Get all commit tree objects
	initialObjects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:   true,
		NoBlobFilter: true,
		Want:         []hash.Hash{commitHash},
		Shallow:      true,
		Deepen:       1,
		Done:         true,
	})
	if err != nil {
		return nil, hash.Zero, fmt.Errorf("get commit tree: %w", err)
	}

	commitObj, exists := initialObjects[commitHash.String()]
	if !exists {
		return nil, hash.Zero, NewObjectNotFoundError(commitHash)
	}

	if commitObj.Type != protocol.ObjectTypeCommit {
		return nil, hash.Zero, NewUnexpectedObjectTypeError(commitHash, protocol.ObjectTypeCommit, commitObj.Type)
	}

	totalObjectsFetched = len(initialObjects)

	// Debug: analyze what types of objects we got
	// TODO: Move this to a separate function
	var commitCount, treeCount, blobCount, otherCount int
	for _, obj := range initialObjects {
		switch obj.Type {
		case protocol.ObjectTypeCommit:
			commitCount++
		case protocol.ObjectTypeTree:
			treeCount++
		case protocol.ObjectTypeBlob:
			blobCount++
		case protocol.ObjectTypeInvalid, protocol.ObjectTypeTag, protocol.ObjectTypeReserved, protocol.ObjectTypeOfsDelta, protocol.ObjectTypeRefDelta:
			otherCount++
		default:
			otherCount++
		}
	}

	logger.Debug("initial targeted request completed",
		"objects_returned", len(initialObjects),
		"target_hash", commitHash.String(),
		"commits", commitCount,
		"trees", treeCount,
		"blobs", blobCount,
		"other", otherCount)

	tree, treeHash, err := c.findRootTree(ctx, commitHash, allObjects)
	if err != nil {
		return nil, hash.Zero, err
	}

	pending := []hash.Hash{}
	retries := []hash.Hash{}
	if tree == nil {
		pending = append(pending, treeHash)
	}

	// Global tracking to prevent infinite loops
	processedTrees := make(map[string]bool)  // Trees we've already analyzed for dependencies
	requestedHashes := make(map[string]bool) // Hashes we've already requested or are pending/retrying

	pending, err = c.collectMissingTreeHashes(ctx, initialObjects, allObjects, pending, processedTrees, requestedHashes)
	if err != nil {
		return nil, hash.Zero, err
	}

	logger.Debug("initial dependency analysis completed",
		"initial_pending", len(pending),
		"processed_trees", len(processedTrees),
		"requested_hashes", len(requestedHashes))

	// Batch sizes
	const batchSize = 10
	const retryBatchSize = 10 // TODO: revisit this. Use individual requests for retries

	// Track retry attempts to prevent infinite loops
	retryCount := make(map[string]int)
	const maxRetries = 3

	var batchNumber int
	const maxBatches = 1000 // Safeguard against infinite loops

	for len(pending) > 0 || len(retries) > 0 {
		batchNumber++

		// Safeguard against infinite loops
		if batchNumber > maxBatches {
			logger.Error("exceeded maximum batch limit, possible infinite loop",
				"max_batches", maxBatches,
				"remaining_pending", len(pending),
				"remaining_retries", len(retries),
				"processed_trees", len(processedTrees),
				"requested_hashes", len(requestedHashes),
				"total_objects", allObjects.Len())
			return nil, hash.Zero, fmt.Errorf("exceeded maximum batch limit (%d), possible infinite loop detected", maxBatches)
		}

		var currentBatch []hash.Hash
		var batchType string

		// Process retries first with smaller batches
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
			// Process normal pending with larger batches
			currentBatch = pending
			if len(pending) > batchSize {
				currentBatch = pending[:batchSize]
				pending = pending[batchSize:]
			} else {
				pending = nil
			}
		}

		logger.Debug("processing batch",
			"batch_number", batchNumber,
			"batch_type", batchType,
			"batch_size", len(currentBatch),
			"remaining_pending", len(pending),
			"remaining_retries", len(retries))

		totalRequests++
		objects, err := c.Fetch(ctx, client.FetchOptions{
			NoProgress:   true,
			NoBlobFilter: true,
			Want:         currentBatch,
			Done:         true,
		})
		if err != nil {
			return nil, hash.Zero, fmt.Errorf("getting objects: %w", err)
		}

		totalObjectsFetched += len(objects)

		// Analyze what we got back vs what we requested
		var requestedReceived, additionalReceived int
		for _, requestedHash := range currentBatch {
			if _, exists := objects[requestedHash.String()]; exists {
				requestedReceived++
			}
		}
		additionalReceived = len(objects) - requestedReceived

		logger.Debug("batch completed",
			"batch_number", batchNumber,
			"requested", len(currentBatch),
			"received", len(objects),
			"requested_received", requestedReceived,
			"additional_received", additionalReceived,
			"total_objects", totalObjectsFetched,
			"total_requests", totalRequests)

		// Check which objects were actually returned
		for _, requestedHash := range currentBatch {
			if _, exists := objects[requestedHash.String()]; exists {
				continue
			}

			hashStr := requestedHash.String()
			retryCount[hashStr]++

			// Log retry attempts for debugging
			logger.Warn("object not returned by server",
				"hash", hashStr,
				"attempt", retryCount[hashStr],
				"max_retries", maxRetries)

			// If we've retried this object too many times, give up
			if retryCount[hashStr] > maxRetries {
				logger.Error("object persistently not returned by server",
					"hash", hashStr,
					"attempts", maxRetries,
					"total_requests", totalRequests,
					"total_objects_fetched", totalObjectsFetched)
				return nil, hash.Zero, fmt.Errorf("object %s not returned after %d attempts (batch sizes: %d then %d): %w", hashStr, maxRetries, batchSize, retryBatchSize, ErrObjectNotFound)
			}

			// Add missing objects to retries list if not already added
			// We use requestedHashes to track this since it's more efficient than searching the retries slice
			if !requestedHashes[hashStr] {
				retries = append(retries, requestedHash)
				requestedHashes[hashStr] = true
			}
		}

		// Process any new tree dependencies from successful objects
		pending, err = c.collectMissingTreeHashes(ctx, objects, allObjects, pending, processedTrees, requestedHashes)
		if err != nil {
			return nil, hash.Zero, err
		}
	}

	// Log final summary
	logger.Info("tree object collection completed",
		"target_hash", commitHash.String(),
		"total_requests", totalRequests,
		"total_objects_fetched", totalObjectsFetched,
		"total_batches", batchNumber)

	return allObjects, treeHash, nil
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

	// We could have the children of the trees we already have, so we need to add them to the allObjects map
	for _, obj := range objects {
		if obj.Type != protocol.ObjectTypeTree {
			continue
		}

		// Skip if we've already processed this tree for dependencies
		if processedTrees[obj.Hash.String()] {
			continue
		}
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
			entryHash, err := hash.FromHex(entry.Hash)
			if err != nil {
				return nil, fmt.Errorf("parsing child hash %s: %w", entry.Hash, err)
			}

			// Skip if we already have this object
			if _, exists := allObjects.Get(entryHash); exists {
				continue
			}

			// If it's a file, we can ignore it
			if entry.FileMode&0o40000 == 0 {
				continue
			}

			// Skip if we've already requested this hash
			if requestedHashes[entry.Hash] {
				continue
			}

			// If it's a directory, we need to add it to the pending list
			childHash, err := hash.FromHex(entry.Hash)
			if err != nil {
				return nil, fmt.Errorf("parsing child hash %s: %w", entry.Hash, err)
			}

			pending = append(pending, childHash)
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
func (c *httpClient) findRootTree(ctx context.Context, targetHash hash.Hash, allObjects storage.PackfileStorage) (*protocol.PackfileObject, hash.Hash, error) {
	logger := log.FromContext(ctx)
	// Find our target object in the response
	obj, exists := allObjects.Get(targetHash)
	if !exists {
		return nil, hash.Zero, NewObjectNotFoundError(targetHash)
	}

	var tree *protocol.PackfileObject
	var treeHash hash.Hash

	if obj.Type == protocol.ObjectTypeCommit {
		// Extract tree hash from commit
		var err error
		treeHash, err = hash.FromHex(obj.Commit.Tree.String())
		if err != nil {
			return nil, hash.Zero, fmt.Errorf("parsing tree hash: %w", err)
		}

		// Check if the tree object is already in our available objects
		if treeObj, exists := allObjects.Get(treeHash); exists {
			tree = treeObj
		} else {
			// Tree not in available objects, we'll fetch it later
			tree = nil
		}

		logger.Debug("resolved commit to tree",
			"commit_hash", targetHash.String(),
			"tree_hash", treeHash.String(),
			"tree_available", tree != nil)
	} else {
		return nil, hash.Zero, NewUnexpectedObjectTypeError(targetHash, protocol.ObjectTypeCommit, obj.Type)
	}

	return tree, treeHash, nil
}

// flatten converts collected tree objects into a flat tree structure using breadth-first traversal.
func (c *httpClient) flatten(treeHash hash.Hash, allTreeObjects storage.PackfileStorage) (*FlatTree, error) {
	// Get the root tree object
	rootTree, exists := allTreeObjects.Get(treeHash)
	if !exists {
		return nil, fmt.Errorf("root tree %s not found in collected objects", treeHash.String())
	}

	// Build flat entries iteratively using breadth-first traversal
	var entries []FlatTreeEntry

	// Queue for processing: each item contains the tree object and its base path
	type queueItem struct {
		tree     *protocol.PackfileObject
		basePath string
	}

	queue := []queueItem{{tree: rootTree, basePath: ""}}

	// Process the queue iteratively
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Process all entries in this tree
		for _, entry := range current.tree.Tree {
			entryHash, err := hash.FromHex(entry.Hash)
			if err != nil {
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
				childTree, exists := allTreeObjects.Get(entryHash)
				if !exists {
					return nil, fmt.Errorf("tree object %s not found in collection", entry.Hash)
				}
				queue = append(queue, queueItem{
					tree:     childTree,
					basePath: entryPath,
				})
			}
		}
	}

	return &FlatTree{
		Entries: entries,
		Hash:    treeHash,
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
	tree, err := c.getTree(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("get tree object: %w", err)
	}

	return packfileObjectToTree(tree)
}

func (c *httpClient) getTree(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	objects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:   true,
		NoBlobFilter: true,
		Want:         []hash.Hash{want},
		Done:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching tree objects: %w", err)
	}

	if len(objects) == 0 {
		return nil, NewObjectNotFoundError(want)
	}

	// TODO: can we do in the fetch?
	for _, obj := range objects {
		if obj.Type != protocol.ObjectTypeTree {
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeTree, obj.Type)
		}
	}

	// Due to Git protocol limitations, when fetching a tree object, we receive all tree objects
	// in the path. We must filter the response to extract only the requested tree.
	if obj, ok := objects[want.String()]; ok {
		return obj, nil
	}

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
	// Ensure storage as it's a complex operation with multiple calls
	// and we may get more objects in the same request than expected in some responses
	ctx, _ = storage.FromContextOrInMemory(ctx)

	if path == "" || path == "." {
		// Return the root tree
		return c.GetTree(ctx, rootHash)
	}

	// Split the path into parts
	parts := strings.Split(path, "/")
	currentHash := rootHash

	// Navigate through each part of the path
	for i, part := range parts {
		if part == "" {
			continue // Skip empty parts (e.g., from leading/trailing slashes)
		}
		currentPath := strings.Join(parts[:i+1], "/")

		// Get the current tree
		currentTree, err := c.GetTree(ctx, currentHash)
		if err != nil {
			return nil, fmt.Errorf("get tree %s: %w", currentHash, err)
		}

		// Find the entry with the matching name
		found := false
		for _, entry := range currentTree.Entries {
			if entry.Name == part {
				if entry.Type != protocol.ObjectTypeTree {
					return nil, fmt.Errorf("path component '%s' is not a directory: %w", currentPath, NewUnexpectedObjectTypeError(entry.Hash, protocol.ObjectTypeTree, entry.Type))
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

	// Get the final tree
	finalTree, err := c.GetTree(ctx, currentHash)
	if err != nil {
		return nil, fmt.Errorf("get final tree %s: %w", currentHash, err)
	}

	return finalTree, nil
}
