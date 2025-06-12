package nanogit

import (
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

// PackfileStorage is an interface for storing packfile objects.
type PackfileStorage interface {
	// Get retrieves an object by its hash.
	Get(key hash.Hash) (*protocol.PackfileObject, bool)
	// GetAllKeys returns all keys in the storage.
	GetAllKeys() []hash.Hash
	// Add adds objects to the storage.
	Add(objs ...*protocol.PackfileObject)
	// AddMap adds objects to the storage.
	AddMap(objs map[string]*protocol.PackfileObject)
	// Delete deletes an object from the storage.
	Delete(key hash.Hash)
	// Len returns the number of objects in the storage.
	Len() int
}
