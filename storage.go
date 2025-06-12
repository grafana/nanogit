package nanogit

import (
	"context"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

// WithPackfileStorage sets the packfile storage for the client.
func WithPackfileStorage(storage PackfileStorage) Option {
	return func(c *httpClient) error {
		c.packfileStorage = storage
		return nil
	}
}

// packfileStorageKey is the key for the packfile storage in the context.
type packfileStorageKey struct{}

// WithPackfileStorageFromContext sets the packfile storage for the client from the context.
func WithPackfileStorageFromContext(ctx context.Context, storage PackfileStorage) context.Context {
	return context.WithValue(ctx, packfileStorageKey{}, storage)
}

// getPackfileStorageFromContext gets the packfile storage from the context.
func getPackfileStorageFromContext(ctx context.Context) PackfileStorage {
	storage, ok := ctx.Value(packfileStorageKey{}).(PackfileStorage)
	if !ok {
		return nil
	}

	return storage
}

// getPackfileStorage gets the packfile storage from the context.
// If it's not set, it returns a no-op storage.
func (c *httpClient) getPackfileStorage(ctx context.Context) PackfileStorage {
	ctxStorage := getPackfileStorageFromContext(ctx)
	if ctxStorage != nil {
		return ctxStorage
	}

	return &noopPackfileStorage{}
}

// ensurePackfileStorage ensures that the packfile storage is set in the context.
// If it's not set, it creates a new in-memory storage and adds it to the context.
func (c *httpClient) ensurePackfileStorage(ctx context.Context) (context.Context, PackfileStorage) {
	ctxStorage := getPackfileStorageFromContext(ctx)
	if ctxStorage != nil {
		return ctx, ctxStorage
	}

	if c.packfileStorage != nil {
		return ctx, c.packfileStorage
	}

	storage := storage.NewInMemoryStorage(ctx)
	return WithPackfileStorageFromContext(ctx, storage), storage
}

// PackfileStorage is an interface for storing packfile objects.
type PackfileStorage interface {
	// Get retrieves an object by its hash.
	Get(key hash.Hash) (*protocol.PackfileObject, bool)
	// GetAllKeys returns all keys in the storage.
	GetAllKeys() []hash.Hash
	// Add adds objects to the storage.
	Add(objs ...*protocol.PackfileObject)
	// Delete deletes an object from the storage.
	Delete(key hash.Hash)
	// Len returns the number of objects in the storage.
	Len() int
}

// noopPackfileStorage is a no-op implementation of PackfileStorage.
type noopPackfileStorage struct{}

func (n *noopPackfileStorage) Get(key hash.Hash) (*protocol.PackfileObject, bool) {
	return nil, false
}

func (n *noopPackfileStorage) GetAllKeys() []hash.Hash {
	return nil
}

func (n *noopPackfileStorage) Add(objs ...*protocol.PackfileObject) {}

func (n *noopPackfileStorage) Delete(key hash.Hash) {}

func (n *noopPackfileStorage) Len() int {
	return 0
}
