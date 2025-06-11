package storage

import (
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

type InMemoryStorage map[string]*protocol.PackfileObject

func NewInMemoryStorage() InMemoryStorage {
	return make(InMemoryStorage)
}

func (s InMemoryStorage) Get(key hash.Hash) (*protocol.PackfileObject, bool) {
	obj, ok := s[key.String()]
	return obj, ok
}

func (s InMemoryStorage) GetAllKeys() []hash.Hash {
	keys := make([]hash.Hash, 0, len(s))
	for key := range s {
		keys = append(keys, hash.MustFromHex(key))
	}

	return keys
}

func (s InMemoryStorage) Add(objs ...*protocol.PackfileObject) {
	for _, obj := range objs {
		s[obj.Hash.String()] = obj
	}
}

// TODO: This is a temporary function to add a map of objects to the storage.
func (s InMemoryStorage) AddMap(objs map[string]*protocol.PackfileObject) {
	for _, obj := range objs {
		s[obj.Hash.String()] = obj
	}
}

func (s InMemoryStorage) Delete(key hash.Hash) {
	delete(s, key.String())
}

func (s InMemoryStorage) Len() int {
	return len(s)
}
