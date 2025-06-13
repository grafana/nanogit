package nanogit

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/storage"
	"github.com/stretchr/testify/require"
)

func TestWithPackfileStorage(t *testing.T) {
	tests := []struct {
		name    string
		storage storage.PackfileStorage
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
			cfg := &rawClient{}
			err := WithPackfileStorage(tt.storage)(cfg)

			require.NoError(t, err)
			require.Equal(t, tt.storage, cfg.packfileStorage)
		})
	}
}
