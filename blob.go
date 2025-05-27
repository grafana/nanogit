package nanogit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *httpClient) GetBlob(ctx context.Context, blobID hash.Hash) (*Blob, error) {
	obj, err := c.getObject(ctx, blobID)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	if obj.Type == protocol.ObjectTypeBlob && obj.Hash.Is(blobID) {
		return &Blob{
			Hash:    blobID,
			Content: obj.Data,
		}, nil
	}

	return nil, fmt.Errorf("blob not found: %s", blobID)
}

type Blob struct {
	Hash    hash.Hash
	Content []byte
}

// GetBlobByPath retrieves a file from the repository at the given path
func (c *httpClient) GetBlobByPath(ctx context.Context, rootHash hash.Hash, path string) (*Blob, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

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

	// Get the final tree containing the target file
	finalTree, err := c.GetTree(ctx, currentHash)
	if err != nil {
		return nil, fmt.Errorf("getting final tree %s: %w", currentHash, err)
	}

	// Find the target file (last part of path)
	fileName := parts[len(parts)-1]
	if fileName == "" {
		return nil, errors.New("invalid path: ends with slash")
	}

	for _, entry := range finalTree.Entries {
		if entry.Name == fileName {
			if entry.Type != protocol.ObjectTypeBlob {
				return nil, fmt.Errorf("'%s' is not a file", fileName)
			}

			return c.GetBlob(ctx, entry.Hash)
		}
	}

	return nil, errors.New("file not found")
}
