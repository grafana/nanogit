package storage_test

import (
	"context"
	"testing"
	"time"

	. "github.com/grafana/nanogit/internal/storage"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorage(t *testing.T) {
	t.Run("NewInMemoryStorage", func(t *testing.T) {
		storage := NewInMemoryStorage(context.Background())
		require.NotNil(t, storage)
		require.Equal(t, 0, storage.Len())
	})

	t.Run("Add and Get", func(t *testing.T) {
		storage := NewInMemoryStorage(context.Background())
		obj := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}

		storage.Add(obj)
		got, ok := storage.Get(obj.Hash)
		require.True(t, ok)
		require.Equal(t, obj, got)
		require.Equal(t, 1, storage.Len())
	})

	t.Run("Get non-existent", func(t *testing.T) {
		storage := NewInMemoryStorage(context.Background())
		hash := hash.MustFromHex("0123456789abcdef")
		got, ok := storage.Get(hash)
		require.False(t, ok)
		require.Nil(t, got)
	})

	t.Run("GetAllKeys", func(t *testing.T) {
		storage := NewInMemoryStorage(context.Background())
		obj1 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}
		obj2 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("fedcba9876543210"),
			Type: protocol.ObjectTypeTree,
		}

		storage.Add(obj1, obj2)
		keys := storage.GetAllKeys()
		require.Len(t, keys, 2)
		require.Equal(t, 2, storage.Len())
		require.Contains(t, keys, obj1.Hash)
		require.Contains(t, keys, obj2.Hash)
	})

	t.Run("Delete", func(t *testing.T) {
		storage := NewInMemoryStorage(context.Background())
		obj := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}

		storage.Add(obj)
		storage.Delete(obj.Hash)
		got, ok := storage.Get(obj.Hash)
		require.False(t, ok)
		require.Nil(t, got)
		require.Equal(t, 0, storage.Len())
	})

	t.Run("Add multiple objects", func(t *testing.T) {
		storage := NewInMemoryStorage(context.Background())
		obj1 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}
		obj2 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("fedcba9876543210"),
			Type: protocol.ObjectTypeTree,
		}

		storage.Add(obj1, obj2)
		require.Equal(t, 2, storage.Len())

		got1, ok1 := storage.Get(obj1.Hash)
		require.True(t, ok1)
		require.Equal(t, obj1, got1)

		got2, ok2 := storage.Get(obj2.Hash)
		require.True(t, ok2)
		require.Equal(t, obj2, got2)
	})

	t.Run("TTL", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create storage with 100ms TTL
		storage := NewInMemoryStorage(ctx, WithTTL(100*time.Millisecond))

		obj1 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}
		obj2 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("fedcba9876543210"),
			Type: protocol.ObjectTypeTree,
		}
		obj3 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("1111111111111111"),
			Type: protocol.ObjectTypeBlob,
		}

		// Test Add with TTL
		storage.Add(obj1, obj2)
		require.Equal(t, 2, storage.Len())

		// Wait for TTL to expire
		time.Sleep(50 * time.Millisecond)

		// Access obj1 to refresh its TTL
		got1, ok1 := storage.Get(obj1.Hash)
		require.True(t, ok1)
		require.Equal(t, obj1, got1)

		// Wait for TTL to expire
		time.Sleep(100 * time.Millisecond)

		// obj1 should still exist (was accessed)
		got1, ok1 = storage.Get(obj1.Hash)
		require.True(t, ok1)
		require.Equal(t, obj1, got1)

		// obj2 and obj3 should be gone (not accessed)
		got2, ok2 := storage.Get(obj2.Hash)
		require.False(t, ok2)
		require.Nil(t, got2)

		got3, ok3 := storage.Get(obj3.Hash)
		require.False(t, ok3)
		require.Nil(t, got3)

		// Verify length
		require.Equal(t, 1, storage.Len())
	})
}
