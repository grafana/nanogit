package nanogit

import (
	"context"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *httpClient) getTreeObjects(ctx context.Context, want hash.Hash) (map[string]*protocol.PackfileObject, error) {
	objects, err := c.fetch(ctx, fetchOptions{
		NoCache:      true,
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

	return objects, nil
}

func (c *httpClient) getTree(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	storage := c.getPackfileStorage(ctx)
	if storage != nil {
		if obj, ok := storage.Get(want); ok {
			return obj, nil
		}
	}

	objects, err := c.getTreeObjects(ctx, want)
	if err != nil {
		return nil, fmt.Errorf("getting tree objects: %w", err)
	}

	// Due to Git protocol limitations, when fetching a tree object, we receive all tree objects
	// in the path. We must filter the response to extract only the requested tree.
	if obj, ok := objects[want.String()]; ok {
		return obj, nil
	}

	return nil, NewObjectNotFoundError(want)
}

func (c *httpClient) getCommit(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
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
		Deepen:       1,
		Shallow:      true,
		Done:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching commit objects: %w", err)
	}

	if len(objects) == 0 {
		return nil, NewObjectNotFoundError(want)
	}

	var foundObj *protocol.PackfileObject
	for _, obj := range objects {
		// Skip tree objects that are included in the response despite the blob:none filter.
		// Most Git servers don't support tree:0 filter specification, so we may receive
		// recursive tree objects that we need to filter out.
		if obj.Type == protocol.ObjectTypeTree {
			continue
		}

		if obj.Type != protocol.ObjectTypeCommit {
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeCommit, obj.Type)
		}

		// we got more commits than expected
		if foundObj != nil {
			return nil, NewUnexpectedObjectCountError(1, []*protocol.PackfileObject{foundObj, obj})
		}

		if obj.Hash.Is(want) {
			foundObj = obj
		}
	}

	if foundObj != nil {
		return foundObj, nil
	}

	return nil, NewObjectNotFoundError(want)
}

func (c *httpClient) getBlob(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	storage := c.getPackfileStorage(ctx)
	if storage != nil {
		if obj, ok := storage.Get(want); ok {
			return obj, nil
		}
	}

	// TODO: do we want a fetch one?
	objects, err := c.fetch(ctx, fetchOptions{
		NoProgress: true,
		Want:       []hash.Hash{want},
		Done:       true,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching blob objects: %w", err)
	}

	var foundObj *protocol.PackfileObject
	for _, obj := range objects {
		if obj.Type != protocol.ObjectTypeBlob {
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeBlob, obj.Type)
		}

		// we got more blobs than expected
		if foundObj != nil {
			return nil, NewUnexpectedObjectCountError(1, []*protocol.PackfileObject{foundObj, obj})
		}

		if obj.Hash.Is(want) {
			foundObj = obj
		}
	}

	if foundObj != nil {
		return foundObj, nil
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
