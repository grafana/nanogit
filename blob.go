package nanogit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// GetBlob retrieves a blob (file content) from the repository by its hash.
// This method fetches the raw content of a file stored in the Git object database.
//
// Parameters:
//   - ctx: Context for the operation
//   - blobID: SHA-1 hash of the blob object to retrieve
//
// Returns:
//   - *Blob: The blob object containing hash and file content
//   - error: Error if the blob is not found or cannot be retrieved
//
// Example:
//
//	blob, err := client.GetBlob(ctx, blobHash)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("File content: %s\n", string(blob.Content))
func (c *httpClient) GetBlob(ctx context.Context, blobID hash.Hash) (*Blob, error) {
	obj, err := c.getBlob(ctx, blobID)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	// FIXME: where to check for the type? here or in getBlob?
	if obj.Type != protocol.ObjectTypeBlob {
		return nil, NewUnexpectedObjectTypeError(blobID, protocol.ObjectTypeBlob, obj.Type)
	}

	if obj.Hash.Is(blobID) {
		return &Blob{
			Hash:    blobID,
			Content: obj.Data,
		}, nil
	}

	return nil, NewObjectNotFoundError(blobID)
}

// Blob represents a Git blob object, which stores the content of a file.
// In Git, all file content is stored as blob objects in the object database.
type Blob struct {
	// Hash is the SHA-1 hash of the blob object
	Hash hash.Hash
	// Content is the raw file content as bytes
	Content []byte
}

// GetBlobByPath retrieves a file from the repository by navigating through
// the directory structure to the specified path. This method efficiently
// traverses the tree hierarchy to locate and fetch the file content.
//
// The path should use forward slashes ("/") as separators, similar to Unix paths.
// The method navigates through directory trees to find the target file.
//
// Parameters:
//   - ctx: Context for the operation
//   - rootHash: Hash of the root tree to start navigation from
//   - path: File path to retrieve (e.g., "src/main.go" or "docs/readme.md")
//
// Returns:
//   - *Blob: The blob object containing the file content
//   - error: Error if path doesn't exist, contains non-files, or retrieval fails
//
// Example:
//
//	// Get the content of a specific file
//	blob, err := client.GetBlobByPath(ctx, rootTreeHash, "src/main.go")
//	if err != nil {
//	    return fmt.Errorf("file not found: %w", err)
//	}
//	fmt.Printf("File content: %s\n", string(blob.Content))
func (c *httpClient) GetBlobByPath(ctx context.Context, rootHash hash.Hash, path string) (*Blob, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	// Add in-memory storage as it's a complex operation with multiple calls
	// and we may get more objects in the same request than expected in some responses
	ctx, _ = c.ensurePackfileStorage(ctx)

	// Split the path into parts
	parts := strings.Split(path, "/")
	currentHash := rootHash

	// Navigate through all but the last part (directories)
	for _, part := range parts[:len(parts)-1] {
		if part == "" {
			continue // Skip empty parts (e.g., from leading/trailing slashes)
		}

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
					return nil, NewUnexpectedObjectTypeError(entry.Hash, protocol.ObjectTypeTree, entry.Type)
				}

				currentHash = entry.Hash
				found = true
				break
			}
		}

		if !found {
			return nil, NewPathNotFoundError(path)
		}
	}

	// Get the final tree containing the target file
	finalTree, err := c.GetTree(ctx, currentHash)
	if err != nil {
		return nil, fmt.Errorf("get final tree %s: %w", currentHash, err)
	}

	// Find the target file (last part of path)
	fileName := parts[len(parts)-1]
	if fileName == "" {
		return nil, errors.New("invalid path: ends with slash")
	}

	for _, entry := range finalTree.Entries {
		if entry.Name == fileName {
			if entry.Type != protocol.ObjectTypeBlob {
				return nil, NewUnexpectedObjectTypeError(entry.Hash, protocol.ObjectTypeBlob, entry.Type)
			}

			return c.GetBlob(ctx, entry.Hash)
		}
	}

	return nil, NewPathNotFoundError(path)
}
