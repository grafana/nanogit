package protocol

import (
	"crypto"
	"os"
	"testing"

	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackfileStorageMode(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, PackfileStorageMode(0), PackfileStorageAuto)
		assert.Equal(t, PackfileStorageMode(1), PackfileStorageMemory)
		assert.Equal(t, PackfileStorageMode(2), PackfileStorageDisk)
	})
}

func TestNewPackfileWriter(t *testing.T) {
	tests := []struct {
		name        string
		storageMode PackfileStorageMode
	}{
		{"auto mode", PackfileStorageAuto},
		{"memory mode", PackfileStorageMemory},
		{"disk mode", PackfileStorageDisk},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewPackfileWriter(crypto.SHA1, tt.storageMode)
			assert.NotNil(t, writer)
			assert.Equal(t, crypto.SHA1, writer.algo)
			assert.Equal(t, tt.storageMode, writer.storageMode)
			assert.NotNil(t, writer.objectHashes)
			assert.NotNil(t, writer.memoryObjects)
			assert.Nil(t, writer.tempFile)
		})
	}
}

func TestPackfileWriter_StorageMode_Memory(t *testing.T) {
	writer := NewPackfileWriter(crypto.SHA1, PackfileStorageMemory)
	defer writer.Cleanup()

	// Add multiple objects - should all go to memory
	for i := 0; i < 20; i++ {
		obj := PackfileObject{
			Type: ObjectTypeBlob,
			Data: []byte("test data"),
			Hash: hash.Hash{}, // Hash will be computed in real usage
		}
		err := writer.addObject(obj)
		require.NoError(t, err)
	}

	// Should have objects in memory, no temp file
	assert.Len(t, writer.memoryObjects, 20)
	assert.Nil(t, writer.tempFile)
}

func TestPackfileWriter_StorageMode_Disk(t *testing.T) {
	writer := NewPackfileWriter(crypto.SHA1, PackfileStorageDisk)
	defer writer.Cleanup()

	// Add single object - should go to disk
	obj := PackfileObject{
		Type: ObjectTypeBlob,
		Data: []byte("test data"),
		Hash: hash.Hash{}, // Hash will be computed in real usage
	}
	err := writer.addObject(obj)
	require.NoError(t, err)

	// Should have temp file, no objects in memory
	assert.Empty(t, writer.memoryObjects)
	assert.NotNil(t, writer.tempFile)

	// Check temp file exists
	_, err = os.Stat(writer.tempFile.Name())
	assert.NoError(t, err)
}

func TestPackfileWriter_StorageMode_Auto(t *testing.T) {
	writer := NewPackfileWriter(crypto.SHA1, PackfileStorageAuto)
	defer writer.Cleanup()

	// Add objects below threshold - should use memory
	// We test the behavior as if objects were added through normal flow
	for i := 0; i < MemoryThreshold-1; i++ {
		obj := PackfileObject{
			Type: ObjectTypeBlob,
			Data: []byte("test data"),
			Hash: hash.Hash{}, // Hash will be computed in real usage
		}
		hashStr := string(rune('a' + i)) // Create unique hash strings
		
		// Add to hash map to simulate normal object addition
		writer.objectHashes[hashStr] = true
		
		// Call addObject - at this point len(objectHashes) should be < MemoryThreshold
		err := writer.addObject(obj)
		require.NoError(t, err)
		
		// Should be in memory
		assert.Len(t, writer.memoryObjects, i+1, "Object %d should be in memory", i)
		assert.Nil(t, writer.tempFile, "No temp file should exist yet for object %d", i)
	}

	// Now add one more object to reach the threshold
	obj := PackfileObject{
		Type: ObjectTypeBlob,
		Data: []byte("test data"),
		Hash: hash.Hash{}, // Hash will be computed in real usage
	}
	writer.objectHashes["final"] = true // Now len(objectHashes) == MemoryThreshold
	
	err := writer.addObject(obj)
	require.NoError(t, err)

	// At this point, the condition len(objectHashes) < MemoryThreshold should be false
	// So it should trigger migration to file
	assert.NotNil(t, writer.tempFile, "Temp file should exist after reaching threshold")
	assert.Empty(t, writer.memoryObjects, "Objects should have been migrated to file")

	// Check temp file exists
	_, err = os.Stat(writer.tempFile.Name())
	assert.NoError(t, err)
}

func TestPackfileWriter_StorageMode_UnknownMode(t *testing.T) {
	writer := NewPackfileWriter(crypto.SHA1, PackfileStorageMode(999))
	defer writer.Cleanup()

	obj := PackfileObject{
		Type: ObjectTypeBlob,
		Data: []byte("test data"),
		Hash: hash.Hash{}, // Hash will be computed in real usage
	}
	
	err := writer.addObject(obj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown storage mode")
}

func TestPackfileWriter_Cleanup(t *testing.T) {
	t.Run("cleanup with temp file", func(t *testing.T) {
		writer := NewPackfileWriter(crypto.SHA1, PackfileStorageDisk)
		
		// Add object to create temp file
		obj := PackfileObject{
			Type: ObjectTypeBlob,
			Data: []byte("test data"),
			Hash: hash.Hash{}, // Hash will be computed in real usage
		}
		err := writer.addObject(obj)
		require.NoError(t, err)
		require.NotNil(t, writer.tempFile)
		
		tempFileName := writer.tempFile.Name()
		
		// File should exist
		_, err = os.Stat(tempFileName)
		require.NoError(t, err)
		
		// Cleanup should remove file
		err = writer.Cleanup()
		assert.NoError(t, err)
		
		// File should be gone
		_, err = os.Stat(tempFileName)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("cleanup without temp file", func(t *testing.T) {
		writer := NewPackfileWriter(crypto.SHA1, PackfileStorageMemory)
		
		// Should not error when no temp file exists
		err := writer.Cleanup()
		assert.NoError(t, err)
	})
}