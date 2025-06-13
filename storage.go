package nanogit

import (
	"context"

	"github.com/grafana/nanogit/storage"
)

// WithPackfileStorage sets the packfile storage for the client.
func WithPackfileStorage(storage storage.PackfileStorage) Option {
	return func(c *rawClient) error {
		c.packfileStorage = storage
		return nil
	}
}

// FIXME:  refactor this to have storage outsite raw client and not inject deal with storage in 2 layers
// getPackfileStorage gets the packfile storage from the context.
// If it's not set, it returns a no-op storage.
func (c *rawClient) getPackfileStorage(ctx context.Context) storage.PackfileStorage {
	ctxStorage := storage.GetPackfileStorageFromContext(ctx)
	if ctxStorage != nil {
		return ctxStorage
	}

	return &storage.NoopPackfileStorage{}
}

// getPackfileStorage gets the packfile storage from the context.
// If it's not set, it returns a no-op storage.
func (c *httpClient) getPackfileStorage(ctx context.Context) storage.PackfileStorage {
	ctxStorage := storage.GetPackfileStorageFromContext(ctx)
	if ctxStorage != nil {
		return ctxStorage
	}

	return &storage.NoopPackfileStorage{}
}

// ensurePackfileStorage ensures that the packfile storage is set in the context.
// If it's not set, it creates a new in-memory storage and adds it to the context.
func (c *httpClient) ensurePackfileStorage(ctx context.Context) (context.Context, storage.PackfileStorage) {
	ctxStorage := storage.GetPackfileStorageFromContext(ctx)
	if ctxStorage != nil {
		return ctx, ctxStorage
	}

	if c.packfileStorage != nil {
		return ctx, c.packfileStorage
	}

	inMemoryStorage := storage.NewInMemoryStorage(ctx)

	return storage.WithPackfileStorageFromContext(ctx, inMemoryStorage), inMemoryStorage
}
