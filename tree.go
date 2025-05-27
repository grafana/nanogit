package nanogit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
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
//   - h: Hash of either a tree object or commit object
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
func (c *httpClient) GetFlatTree(ctx context.Context, h hash.Hash) (*FlatTree, error) {
	// Make initial getObjects call - this could potentially return many objects we need
	initialObjects, err := c.getObjects(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("getting initial objects: %w", err)
	}

	// Find our target object in the response
	obj, exists := initialObjects[h.String()]
	if !exists {
		return nil, fmt.Errorf("object %s not found: %w", h.String(), ErrRefNotFound)
	}

	var tree *protocol.PackfileObject
	var treeHash hash.Hash

	if obj.Type == protocol.ObjectTypeCommit {
		// Extract tree hash from commit
		treeHash, err = hash.FromHex(obj.Commit.Tree.String())
		if err != nil {
			return nil, fmt.Errorf("parsing tree hash: %w", err)
		}

		// Check if the tree object is already in our initial response
		if treeObj, exists := initialObjects[treeHash.String()]; exists {
			tree = treeObj
		} else {
			// Tree not in initial response, we'll fetch it in buildFlatTreeIteratively
			tree = nil
		}
	} else if obj.Type == protocol.ObjectTypeTree {
		tree = obj
		treeHash = h
	} else {
		return nil, errors.New("target object is not a commit or tree")
	}

	return c.buildFlatTreeIteratively(ctx, treeHash, tree, initialObjects)
}

// buildFlatTreeIteratively builds a flat tree by iteratively collecting all tree objects
// using batched requests until all children are resolved.
func (c *httpClient) buildFlatTreeIteratively(ctx context.Context, rootHash hash.Hash, rootTree *protocol.PackfileObject, seedObjects map[string]*protocol.PackfileObject) (*FlatTree, error) {
	const batchSize = 50 // Reasonable batch size for tree objects

	// Start with seed objects from initial request
	allTrees := make(map[string]*protocol.PackfileObject)
	for k, v := range seedObjects {
		if v.Type == protocol.ObjectTypeTree {
			allTrees[k] = v
		}
	}

	// If we don't have the root tree, we need to fetch it
	if rootTree == nil {
		if seedTree, exists := allTrees[rootHash.String()]; exists {
			rootTree = seedTree
		} else {
			// Need to fetch the root tree
			obj, err := c.getSingleObject(ctx, rootHash)
			if err != nil {
				return nil, fmt.Errorf("getting root tree: %w", err)
			}
			rootTree = obj
			allTrees[rootHash.String()] = rootTree
		}
	} else {
		// Ensure root tree is in our collection
		allTrees[rootHash.String()] = rootTree
	}

	// Keep track of what tree hashes we need to process
	toProcess := []hash.Hash{rootHash}
	processed := make(map[string]bool)

	// Iteratively discover and fetch all tree objects
	for len(toProcess) > 0 {
		currentBatch := toProcess
		toProcess = nil

		// Find tree children that we haven't collected yet
		var missingHashes []hash.Hash
		for _, treeHash := range currentBatch {
			hashStr := treeHash.String()
			if processed[hashStr] {
				continue
			}
			processed[hashStr] = true

			treeObj, exists := allTrees[hashStr]
			if !exists {
				// This shouldn't happen if we're tracking correctly
				continue
			}

			// Examine this tree's children
			for _, entry := range treeObj.Tree {
				if entry.FileMode&0o40000 != 0 { // Directory
					childHashStr := entry.Hash
					if _, alreadyHave := allTrees[childHashStr]; !alreadyHave {
						childHash, err := hash.FromHex(childHashStr)
						if err != nil {
							return nil, fmt.Errorf("parsing child hash %s: %w", childHashStr, err)
						}
						missingHashes = append(missingHashes, childHash)
					}
				}
			}
		}

		// If we have missing tree objects, fetch them in batches
		if len(missingHashes) > 0 {
			c.logger.Debug("Fetching missing tree objects", "count", len(missingHashes))

			// Process missing hashes in batches
			for i := 0; i < len(missingHashes); i += batchSize {
				end := i + batchSize
				if end > len(missingHashes) {
					end = len(missingHashes)
				}
				batch := missingHashes[i:end]

				// Try to fetch this batch
				fetchedObjects, err := c.getObjects(ctx, batch...)
				if err != nil {
					c.logger.Debug("Batch fetch failed, falling back to individual requests", "error", err)
					// Fall back to individual requests
					for _, h := range batch {
						obj, err := c.getSingleObject(ctx, h)
						if err != nil {
							return nil, fmt.Errorf("fetching tree object %s: %w", h, err)
						}
						allTrees[h.String()] = obj
						toProcess = append(toProcess, h)
					}
				} else {
					// Add successfully fetched objects
					for k, v := range fetchedObjects {
						allTrees[k] = v
						// Convert back to hash for processing queue
						h, err := hash.FromHex(k)
						if err != nil {
							return nil, fmt.Errorf("parsing fetched hash %s: %w", k, err)
						}
						toProcess = append(toProcess, h)
					}

					// Check for missing objects and fetch individually
					for _, h := range batch {
						if _, exists := fetchedObjects[h.String()]; !exists {
							c.logger.Debug("Object missing from batch, fetching individually", "hash", h.String())
							obj, err := c.getSingleObject(ctx, h)
							if err != nil {
								return nil, fmt.Errorf("fetching missing tree object %s: %w", h, err)
							}
							allTrees[h.String()] = obj
							toProcess = append(toProcess, h)
						}
					}
				}
			}
		}
	}

	// Now that we have all tree objects, build the flat structure iteratively
	entries, err := c.buildFlatEntriesIteratively(rootTree, "", allTrees)
	if err != nil {
		return nil, fmt.Errorf("building flat entries: %w", err)
	}

	return &FlatTree{
		Entries: entries,
		Hash:    rootHash,
	}, nil
}

// buildFlatEntriesIteratively builds flat tree entries using a breadth-first iterative approach
func (c *httpClient) buildFlatEntriesIteratively(rootTree *protocol.PackfileObject, rootPath string, allTrees map[string]*protocol.PackfileObject) ([]FlatTreeEntry, error) {
	var result []FlatTreeEntry

	// Queue for processing: each item contains the tree object and its base path
	type queueItem struct {
		tree     *protocol.PackfileObject
		basePath string
	}

	queue := []queueItem{{tree: rootTree, basePath: rootPath}}

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
			result = append(result, FlatTreeEntry{
				Name: entry.FileName,
				Path: entryPath,
				Mode: uint32(entry.FileMode),
				Hash: entryHash,
				Type: entryType,
			})

			// If this is a tree, add it to the queue for processing
			if entryType == protocol.ObjectTypeTree {
				childTree, exists := allTrees[entry.Hash]
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

	return result, nil
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
//   - h: Hash of either a tree object or commit object
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
	obj, err := c.getSingleObject(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	var tree *protocol.PackfileObject
	if obj.Type == protocol.ObjectTypeCommit && obj.Hash.Is(h) {
		// If it's a commit, get the tree hash from it
		treeHash, err := hash.FromHex(obj.Commit.Tree.String())
		if err != nil {
			return nil, fmt.Errorf("parsing tree hash: %w", err)
		}

		treeObj, err := c.getSingleObject(ctx, treeHash)
		if err != nil {
			return nil, fmt.Errorf("getting tree: %w", err)
		}
		tree = treeObj
		h = treeHash
	} else if obj.Type == protocol.ObjectTypeTree && obj.Hash.Is(h) {
		tree = obj
	} else {
		return nil, errors.New("not found")
	}

	// Convert PackfileTreeEntry to TreeEntry (direct children only)
	entries := make([]TreeEntry, len(tree.Tree))
	for i, entry := range tree.Tree {
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
		Hash:    h,
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
	if path == "" || path == "." {
		// Return the root tree
		return c.GetTree(ctx, rootHash)
	}

	// Split the path into parts
	parts := strings.Split(path, "/")
	currentHash := rootHash

	// Navigate through each part of the path
	for _, part := range parts {
		if part == "" {
			continue // Skip empty parts (e.g., from leading/trailing slashes)
		}

		// Get the current tree
		currentTree, err := c.GetTree(ctx, currentHash)
		if err != nil {
			return nil, fmt.Errorf("getting tree %s: %w", currentHash, err)
		}

		// Find the entry with the matching name
		found := false
		for _, entry := range currentTree.Entries {
			if entry.Name == part {
				if entry.Type != protocol.ObjectTypeTree {
					return nil, fmt.Errorf("path component '%s' is not a directory", part)
				}
				currentHash = entry.Hash
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("path component '%s' not found", part)
		}
	}

	// Get the final tree
	return c.GetTree(ctx, currentHash)
}
