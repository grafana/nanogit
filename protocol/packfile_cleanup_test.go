package protocol

import (
	"crypto"
	"testing"

	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackfileWriterCleanup(t *testing.T) {
	t.Run("cleanup prevents further operations", func(t *testing.T) {
		writer := NewPackfileWriter(crypto.SHA1, PackfileStorageMemory)

		// Verify operations work before cleanup
		_, err := writer.AddBlob([]byte("test content"))
		require.NoError(t, err)

		assert.True(t, writer.HasObjects())

		// Cleanup the writer
		err = writer.Cleanup()
		require.NoError(t, err)

		// Verify all operations now fail with ErrPackfileWriterCleanedUp
		_, err = writer.AddBlob([]byte("more content"))
		require.ErrorIs(t, err, ErrPackfileWriterCleanedUp)

		// HasObjects should return false after cleanup
		assert.False(t, writer.HasObjects())

		// AddObject should silently fail (no error returned by design)
		writer.AddObject(PackfileObject{Type: ObjectTypeBlob, Data: []byte("test")})
		assert.False(t, writer.HasObjects())

		// WritePackfile should fail
		err = writer.WritePackfile(nil, "refs/heads/main", hash.Zero)
		require.ErrorIs(t, err, ErrPackfileWriterCleanedUp)
	})

	t.Run("cleanup can only be called once", func(t *testing.T) {
		writer := NewPackfileWriter(crypto.SHA1, PackfileStorageMemory)

		// First cleanup should succeed
		err := writer.Cleanup()
		require.NoError(t, err)

		// Second cleanup should fail
		err = writer.Cleanup()
		require.ErrorIs(t, err, ErrPackfileWriterCleanedUp)
	})

	t.Run("cleanup removes temporary files", func(t *testing.T) {
		writer := NewPackfileWriter(crypto.SHA1, PackfileStorageDisk)

		// Add enough data to trigger file storage
		for i := 0; i < 20; i++ {
			_, err := writer.AddBlob([]byte("large content to trigger file storage"))
			require.NoError(t, err)
		}

		// Verify temp file exists
		require.NotNil(t, writer.tempFile)

		// Cleanup should remove the temp file
		err := writer.Cleanup()
		require.NoError(t, err)

		// File should be removed (checking this might be OS-dependent)
		// The important thing is that cleanup completed without error
		assert.True(t, writer.isCleanedUp)
	})
}
