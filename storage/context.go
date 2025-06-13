package storage

import "context"

// packfileStorageKey is the key for the packfile storage in the context.
type packfileStorageKey struct{}

// ToContext sets the packfile storage for the client from the context.
func ToContext(ctx context.Context, storage PackfileStorage) context.Context {
	return context.WithValue(ctx, packfileStorageKey{}, storage)
}

// FromContext gets the packfile storage from the context.
func FromContext(ctx context.Context) PackfileStorage {
	storage, ok := ctx.Value(packfileStorageKey{}).(PackfileStorage)
	if !ok {
		return nil
	}

	return storage
}
