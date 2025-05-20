package nanogit

import (
	"context"
	"errors"

	"github.com/grafana/nanogit/protocol/hash"
)

type File struct {
	Name    string
	Mode    uint32
	Hash    hash.Hash
	Path    string
	Content []byte
}

// GetFile retrieves a file from the repository at the given path
func (c *clientImpl) GetFile(ctx context.Context, hash hash.Hash, path string) (*File, error) {
	tree, err := c.GetTree(ctx, hash)
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

			return &File{
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
