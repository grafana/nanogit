package storage

import (
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// NoopPackfileStorage is a no-op implementation of PackfileStorage.
// It stores nothing, so every operation runs uncached.
type NoopPackfileStorage struct{}

// Get always reports a miss.
func (n *NoopPackfileStorage) Get(key hash.Hash) (*protocol.PackfileObject, bool) {
	return nil, false
}

// GetAllKeys returns nil.
func (n *NoopPackfileStorage) GetAllKeys() []hash.Hash {
	return nil
}

// Add discards the given objects.
func (n *NoopPackfileStorage) Add(objs ...*protocol.PackfileObject) {}

// Delete does nothing.
func (n *NoopPackfileStorage) Delete(key hash.Hash) {}

// Len returns 0.
func (n *NoopPackfileStorage) Len() int {
	return 0
}
