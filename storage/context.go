package storage

import "context"

// packfileStorageKey is the key for the packfile storage in the context.
type packfileStorageKey struct{}

// ToContext returns a copy of ctx carrying the given packfile storage.
// nanogit clients retrieve it with FromContext during Git operations.
func ToContext(ctx context.Context, storage PackfileStorage) context.Context {
	return context.WithValue(ctx, packfileStorageKey{}, storage)
}

// FromContext returns the packfile storage stored in ctx, or nil if none is set.
func FromContext(ctx context.Context) PackfileStorage {
	storage, ok := ctx.Value(packfileStorageKey{}).(PackfileStorage)
	if !ok {
		return nil
	}

	return storage
}

// FromContextOrInMemory returns the packfile storage stored in ctx, if any.
// Otherwise it creates a new in-memory storage and returns it along with a
// derived context that carries it.
func FromContextOrInMemory(ctx context.Context) (context.Context, PackfileStorage) {
	storage := FromContext(ctx)
	if storage != nil {
		return ctx, storage
	}

	inMemoryStorage := NewInMemoryStorage(ctx)
	return ToContext(ctx, inMemoryStorage), inMemoryStorage
}
