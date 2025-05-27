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
	obj, err := c.getSingleObject(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	var tree *protocol.PackfileObject
	if obj.Type == protocol.ObjectTypeCommit && obj.Hash.Is(h) {
		// Find the commit and tree in the packfile
		// TODO: should we make it work for commit object type?
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

	return c.processTree(ctx, h, tree)
}

// processTree converts a Git tree object into a flattened tree structure.
// This method handles the initial processing of a tree object, converting
// the raw Git tree entries into FlatTreeEntry objects and initiating
// recursive processing of any subdirectories.
//
// Parameters:
//   - ctx: Context for the operation
//   - treeHash: Hash of the tree being processed
//   - tree: Raw tree object from the Git repository
//
// Returns:
//   - *FlatTree: Processed flat tree structure
//   - error: Error if processing fails
func (c *httpClient) processTree(ctx context.Context, treeHash hash.Hash, tree *protocol.PackfileObject) (*FlatTree, error) {
	// Convert PackfileTreeEntry to TreeEntry
	entries := make([]FlatTreeEntry, len(tree.Tree))
	for i, entry := range tree.Tree {
		hash, err := hash.FromHex(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("parsing hash: %w", err)
		}

		// Determine the type based on the mode
		entryType := protocol.ObjectTypeBlob
		if entry.FileMode&0o40000 != 0 {
			entryType = protocol.ObjectTypeTree
		}

		entries[i] = FlatTreeEntry{
			Name: entry.FileName,
			Path: entry.FileName,
			Mode: uint32(entry.FileMode),
			Hash: hash,
			Type: entryType,
		}
	}

	// Process all entries recursively
	result, err := c.processTreeEntries(ctx, entries, "")
	if err != nil {
		return nil, fmt.Errorf("processing tree entries: %w", err)
	}

	if len(result) == 0 {
		return nil, errors.New("tree not found")
	}

	return &FlatTree{
		Entries: result,
		Hash:    treeHash,
	}, nil
}

// processTreeEntries recursively processes tree entries to build a complete flat list.
// This method traverses the entire tree structure, following directory entries
// to build a comprehensive list of all files and directories with their full paths.
//
// For each directory encountered, it fetches the directory's contents and recursively
// processes them, building up the complete path for each entry.
//
// Parameters:
//   - ctx: Context for the operation
//   - entries: Current level tree entries to process
//   - basePath: Current path prefix (empty for root level)
//
// Returns:
//   - []FlatTreeEntry: Complete list of all entries found recursively
//   - error: Error if any directory cannot be processed
func (c *httpClient) processTreeEntries(ctx context.Context, entries []FlatTreeEntry, basePath string) ([]FlatTreeEntry, error) {
	result := make([]FlatTreeEntry, 0, len(entries))
	for _, entry := range entries {
		// Build the full path for the entry
		entryPath := entry.Name
		if basePath != "" {
			entryPath = basePath + "/" + entry.Name
		}

		// Update the path for this entry
		entry.Path = entryPath

		// Always add the entry itself
		result = append(result, entry)

		// If this is a tree, recursively process its entries
		if entry.Type == protocol.ObjectTypeTree {
			// Fetch the nested tree
			// TODO: is there a way to avoid fetching the tree again?
			nestedTree, err := c.GetFlatTree(ctx, entry.Hash)
			if err != nil {
				return nil, fmt.Errorf("fetching nested tree %s: %w", entry.Hash, err)
			}

			// Process nested entries with the updated base path
			nestedEntries, err := c.processTreeEntries(ctx, nestedTree.Entries, entryPath)
			if err != nil {
				return nil, fmt.Errorf("processing nested tree entries: %w", err)
			}
			result = append(result, nestedEntries...)
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
