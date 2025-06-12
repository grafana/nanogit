package nanogit

import (
	"context"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
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

// GetPackfileStorageFromContext gets the packfile storage from the context.
func GetPackfileStorageFromContext(ctx context.Context) PackfileStorage {
	storage, ok := ctx.Value(packfileStorageKey{}).(PackfileStorage)
	if !ok {
		return nil
	}

	return storage
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
