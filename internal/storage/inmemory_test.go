package storage_test

import (
	"testing"

	. "github.com/grafana/nanogit/internal/storage"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorage(t *testing.T) {
	t.Run("NewInMemoryStorage", func(t *testing.T) {
		storage := NewInMemoryStorage()
		require.NotNil(t, storage)
		require.Empty(t, storage)
	})

	t.Run("Add and Get", func(t *testing.T) {
		storage := NewInMemoryStorage()
		obj := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}

		storage.Add(obj)
		got, ok := storage.Get(obj.Hash)
		require.True(t, ok)
		require.Equal(t, obj, got)
	})

	t.Run("Get non-existent", func(t *testing.T) {
		storage := NewInMemoryStorage()
		hash := hash.MustFromHex("0123456789abcdef")
		got, ok := storage.Get(hash)
		require.False(t, ok)
		require.Nil(t, got)
	})

	t.Run("GetAllKeys", func(t *testing.T) {
		storage := NewInMemoryStorage()
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
		require.Contains(t, keys, obj1.Hash)
		require.Contains(t, keys, obj2.Hash)
	})

	t.Run("Delete", func(t *testing.T) {
		storage := NewInMemoryStorage()
		obj := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}

		storage.Add(obj)
		storage.Delete(obj.Hash)
		got, ok := storage.Get(obj.Hash)
		require.False(t, ok)
		require.Nil(t, got)
	})

	t.Run("Add multiple objects", func(t *testing.T) {
		storage := NewInMemoryStorage()
		obj1 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("0123456789abcdef"),
			Type: protocol.ObjectTypeBlob,
		}
		obj2 := &protocol.PackfileObject{
			Hash: hash.MustFromHex("fedcba9876543210"),
			Type: protocol.ObjectTypeTree,
		}

		storage.Add(obj1, obj2)
		require.Len(t, storage, 2)

		got1, ok1 := storage.Get(obj1.Hash)
		require.True(t, ok1)
		require.Equal(t, obj1, got1)

		got2, ok2 := storage.Get(obj2.Hash)
		require.True(t, ok2)
		require.Equal(t, obj2, got2)
	})
}
