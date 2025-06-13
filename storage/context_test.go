package storage_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/storage"
	"github.com/stretchr/testify/require"
)

func TestWithPackfileStorageFromContext(t *testing.T) {
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
			ctx := storage.WithPackfileStorageFromContext(tt.ctx, tt.storage)
			require.NotNil(t, ctx)

			require.Equal(t, tt.storage, storage.GetPackfileStorageFromContext(ctx))
		})
	}
}

func TestGetPackfileStorageFromContext(t *testing.T) {
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
				ctx = storage.WithPackfileStorageFromContext(tt.ctx, tt.storage)
			} else {
				ctx = tt.ctx
			}

			got := storage.GetPackfileStorageFromContext(ctx)
			require.Equal(t, tt.want, got)
		})
	}
}
