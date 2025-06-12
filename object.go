package nanogit

import (
	"context"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *httpClient) getTree(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	storage := c.getPackfileStorage(ctx)
	if storage != nil {
		if obj, ok := storage.Get(want); ok {
			return obj, nil
		}
	}

	objects, err := c.fetch(ctx, fetchOptions{
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

type getCommitTreeOptions struct {
	deepen  int
	shallow bool
}

// getRootTree fetches the root tree of the repository.
func (c *httpClient) getCommitTree(ctx context.Context, commitHash hash.Hash, opts getCommitTreeOptions) (map[string]*protocol.PackfileObject, error) {
	return c.fetch(ctx, fetchOptions{
		NoProgress:   true,
		NoBlobFilter: true,
		Want:         []hash.Hash{commitHash},
		Deepen:       opts.deepen,
		Shallow:      opts.shallow,
		Done:         true,
	})
}
