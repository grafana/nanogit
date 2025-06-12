package nanogit

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestWithPackfileStorage(t *testing.T) {
	tests := []struct {
		name    string
		storage PackfileStorage
	}{
		{
			name:    "nil storage",
			storage: nil,
		},
		{
			name:    "valid storage",
			storage: storage.NewInMemoryStorage(context.Background()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &httpClient{}
			err := WithPackfileStorage(tt.storage)(cfg)

			require.NoError(t, err)
			require.Equal(t, tt.storage, cfg.packfileStorage)
		})
	}
}

func TestWithPackfileStorageFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		storage PackfileStorage
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
			ctx := WithPackfileStorageFromContext(tt.ctx, tt.storage)
			require.NotNil(t, ctx)

			require.Equal(t, tt.storage, getPackfileStorageFromContext(ctx))
		})
	}
}

func TestGetPackfileStorageFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		storage PackfileStorage
		want    PackfileStorage
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
				ctx = WithPackfileStorageFromContext(tt.ctx, tt.storage)
			} else {
				ctx = tt.ctx
			}

			got := getPackfileStorageFromContext(ctx)
			require.Equal(t, tt.want, got)
		})
	}
}
