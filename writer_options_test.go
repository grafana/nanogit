package nanogit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := defaultWriterOptions()
		assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
	})

	t.Run("apply nil options", func(t *testing.T) {
		opts, err := applyWriterOptions(nil)
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
	})

	t.Run("apply empty options", func(t *testing.T) {
		opts, err := applyWriterOptions([]WriterOption{})
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
	})

	t.Run("apply nil option in slice", func(t *testing.T) {
		opts, err := applyWriterOptions([]WriterOption{nil})
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
	})
}

func TestWithMemoryStorage(t *testing.T) {
	opts, err := applyWriterOptions([]WriterOption{WithMemoryStorage()})
	require.NoError(t, err)
	assert.Equal(t, PackfileStorageMemory, opts.StorageMode)
}

func TestWithDiskStorage(t *testing.T) {
	opts, err := applyWriterOptions([]WriterOption{WithDiskStorage()})
	require.NoError(t, err)
	assert.Equal(t, PackfileStorageDisk, opts.StorageMode)
}

func TestWithAutoStorage(t *testing.T) {
	opts, err := applyWriterOptions([]WriterOption{WithAutoStorage()})
	require.NoError(t, err)
	assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
}

func TestMultipleOptions(t *testing.T) {
	t.Run("last option wins", func(t *testing.T) {
		opts, err := applyWriterOptions([]WriterOption{
			WithMemoryStorage(),
			WithDiskStorage(),
			WithAutoStorage(),
		})
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
	})

	t.Run("mixed with nil", func(t *testing.T) {
		opts, err := applyWriterOptions([]WriterOption{
			WithMemoryStorage(),
			nil,
			WithDiskStorage(),
		})
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageDisk, opts.StorageMode)
	})
}

func TestPackfileStorageMode(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, PackfileStorageAuto, PackfileStorageMode(0))
		assert.Equal(t, PackfileStorageMemory, PackfileStorageMode(1))
		assert.Equal(t, PackfileStorageDisk, PackfileStorageMode(2))
	})
}

func TestWriterOptionFunction(t *testing.T) {
	t.Run("option function modifies options correctly", func(t *testing.T) {
		opts := &WriterOptions{StorageMode: PackfileStorageAuto}

		err := WithMemoryStorage()(opts)
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageMemory, opts.StorageMode)

		err = WithDiskStorage()(opts)
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageDisk, opts.StorageMode)

		err = WithAutoStorage()(opts)
		require.NoError(t, err)
		assert.Equal(t, PackfileStorageAuto, opts.StorageMode)
	})
}
