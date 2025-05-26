package nanogit

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *clientImpl) GetBlob(ctx context.Context, blobID hash.Hash) ([]byte, error) {
	obj, err := c.getObject(ctx, blobID)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	if obj.Type == protocol.ObjectTypeBlob && obj.Hash.Is(blobID) {
		return obj.Data, nil
	}

	return nil, fmt.Errorf("blob not found: %s", blobID)
}

type Blob struct {
	Name    string
	Mode    uint32
	Hash    hash.Hash
	Path    string
	Content []byte
}

// GetBlobByPath retrieves a file from the repository at the given path
func (c *clientImpl) GetBlobByPath(ctx context.Context, rootHash hash.Hash, path string) (*Blob, error) {
	tree, err := c.GetFlatTree(ctx, rootHash)
	if err != nil {
		return nil, err
	}

	// TODO: Is there a way to do this without iterating over all entries?
	for _, entry := range tree.Entries {
		if entry.Path == path {
			content, err := c.GetBlob(ctx, entry.Hash)
			if err != nil {
				return nil, err
			}

			return &Blob{
				Name:    entry.Name,
				Mode:    entry.Mode,
				Hash:    entry.Hash,
				Path:    entry.Path,
				Content: content,
			}, nil
		}
	}

	return nil, errors.New("file not found")
}
