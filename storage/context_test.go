package storage_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/storage"
	"github.com/stretchr/testify/require"
)

func TestToContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		storage storage.PackfileStorage
	}{
		{
			name:    "nil storage",
			ctx:     context.Background(),
			storage: nil,
		},
		{
			name:    "valid storage",
			ctx:     context.Background(),
			storage: storage.NewInMemoryStorage(context.Background()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := storage.ToContext(tt.ctx, tt.storage)
			require.NotNil(t, ctx)

			require.Equal(t, tt.storage, storage.FromContext(ctx))
		})
	}
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		storage storage.PackfileStorage
		want    storage.PackfileStorage
	}{
		{
			name:    "no storage in context",
			ctx:     context.Background(),
			storage: nil,
			want:    nil,
		},
		{
			name:    "storage in context",
			ctx:     context.Background(),
			storage: storage.NewInMemoryStorage(context.Background()),
			want:    storage.NewInMemoryStorage(context.Background()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.ctx != nil && tt.storage != nil {
				ctx = storage.ToContext(tt.ctx, tt.storage)
			} else {
				ctx = tt.ctx
			}

			got := storage.FromContext(ctx)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFromContextOrInMemory(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		storage storage.PackfileStorage
		want    storage.PackfileStorage
	}{
		{
			name:    "no storage in context",
			ctx:     context.Background(),
			storage: nil,
			want:    nil, // Will be replaced with in-memory storage
		},
		{
			name:    "storage in context",
			ctx:     context.Background(),
			storage: storage.NewInMemoryStorage(context.Background()),
			want:    storage.NewInMemoryStorage(context.Background()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.ctx != nil && tt.storage != nil {
				ctx = storage.ToContext(tt.ctx, tt.storage)
			} else {
				ctx = tt.ctx
			}

			newCtx, got := storage.FromContextOrInMemory(ctx)
			require.NotNil(t, newCtx)
			require.NotNil(t, got)

			if tt.storage == nil {
				// Verify it's an in-memory storage
				_, ok := got.(*storage.InMemoryStorage)
				require.True(t, ok, "expected in-memory storage when no storage in context")
			} else {
				require.Equal(t, tt.want, got)
			}

			// Verify the storage is properly set in the new context
			storageFromCtx := storage.FromContext(newCtx)
			require.Equal(t, got, storageFromCtx)
		})
	}
}
