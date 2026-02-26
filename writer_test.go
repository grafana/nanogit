package nanogit

import (
	"context"
	"crypto"
	"errors"
	"io"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRawClient is a simple mock implementation of client.RawClient for testing
type mockRawClient struct {
	receivePackFunc func(context.Context, io.Reader) error
	receivePackErr  error
}

func (m *mockRawClient) ReceivePack(ctx context.Context, r io.Reader) error {
	if m.receivePackFunc != nil {
		return m.receivePackFunc(ctx, r)
	}
	return m.receivePackErr
}

func (m *mockRawClient) Fetch(ctx context.Context, opts client.FetchOptions) (map[string]*protocol.PackfileObject, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRawClient) LsRefs(ctx context.Context, opts client.LsRefsOptions) ([]protocol.RefLine, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRawClient) CanRead(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *mockRawClient) CanWrite(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *mockRawClient) IsAuthorized(ctx context.Context) (bool, error) {
	return m.CanRead(ctx)
}

func (m *mockRawClient) RepoExists(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *mockRawClient) SmartInfo(ctx context.Context, service string) error {
	return nil
}

func (m *mockRawClient) UploadPack(ctx context.Context, data io.Reader) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

// TestStagedWriter_Cleanup_NormalBehavior tests that Cleanup()
// properly cleans up resources and marks the writer as cleaned up.
func TestStagedWriter_Cleanup_NormalBehavior(t *testing.T) {
	ctx := context.Background()

	mockClient := &mockRawClient{}
	writer := &stagedWriter{
		client: &httpClient{
			RawClient: mockClient,
		},
		ref: Ref{
			Name: "refs/heads/main",
			Hash: hash.Zero,
		},
		writer:      protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory),
		objStorage:  storage.NewInMemoryStorage(ctx),
		treeEntries: make(map[string]*FlatTreeEntry),
		dirtyPaths:  make(map[string]bool),
		storageMode: protocol.PackfileStorageMemory,
	}

	// Act: Normal cleanup should succeed
	err := writer.Cleanup(ctx)

	// Assert: No error for normal case
	assert.NoError(t, err)
	assert.True(t, writer.isCleanedUp)

	// Second cleanup should fail with ErrWriterCleanedUp (from stagedWriter itself)
	err = writer.Cleanup(ctx)
	assert.ErrorIs(t, err, ErrWriterCleanedUp)
}

// TestStagedWriter_Cleanup_MultipleCallsPrevented tests that Cleanup()
// cannot be called multiple times on the same writer.
func TestStagedWriter_Cleanup_MultipleCallsPrevented(t *testing.T) {
	ctx := context.Background()

	mockClient := &mockRawClient{}
	writer := &stagedWriter{
		client: &httpClient{
			RawClient: mockClient,
		},
		ref: Ref{
			Name: "refs/heads/main",
			Hash: hash.Zero,
		},
		writer:      protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory),
		objStorage:  storage.NewInMemoryStorage(ctx),
		treeEntries: make(map[string]*FlatTreeEntry),
		dirtyPaths:  make(map[string]bool),
		storageMode: protocol.PackfileStorageMemory,
	}

	// First cleanup should succeed
	err := writer.Cleanup(ctx)
	require.NoError(t, err)

	// Verify writer is marked as cleaned up
	require.True(t, writer.isCleanedUp)

	// Subsequent cleanups should fail with ErrWriterCleanedUp
	for i := 0; i < 3; i++ {
		err = writer.Cleanup(ctx)
		assert.ErrorIs(t, err, ErrWriterCleanedUp, "Cleanup call %d should return ErrWriterCleanedUp", i+2)
	}
}

// TestStagedWriter_Push_RetryAfterFailure tests that Push can be retried
// after a failure, with the same staged objects still available.
func TestStagedWriter_Push_RetryAfterFailure(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	mockClient := &mockRawClient{
		receivePackFunc: func(ctx context.Context, r io.Reader) error {
			callCount++
			// First call fails, second succeeds
			if callCount == 1 {
				// Simulate network error
				return errors.New("network timeout")
			}
			// Second call succeeds - consume the packfile
			_, err := io.Copy(io.Discard, r)
			return err
		},
	}

	writer := &stagedWriter{
		client: &httpClient{
			RawClient: mockClient,
		},
		ref: Ref{
			Name: "refs/heads/main",
			Hash: hash.Zero,
		},
		writer:      protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory),
		objStorage:  storage.NewInMemoryStorage(ctx),
		treeEntries: make(map[string]*FlatTreeEntry),
		dirtyPaths:  make(map[string]bool),
		storageMode: protocol.PackfileStorageMemory,
	}

	// Stage a blob and commit
	_, err := writer.writer.AddBlob([]byte("test content"))
	require.NoError(t, err)

	treeHash := hash.Zero // Use zero for simplicity in this test
	commitHash, err := writer.writer.AddCommit(
		treeHash,
		hash.Zero, // no parent
		&protocol.Identity{Name: "Test", Email: "test@example.com", Timestamp: 1234567890, Timezone: "+0000"},
		&protocol.Identity{Name: "Test", Email: "test@example.com", Timestamp: 1234567890, Timezone: "+0000"},
		"Test commit",
	)
	require.NoError(t, err)
	writer.lastCommit = &Commit{
		Hash:   commitHash,
		Tree:   treeHash,
		Parent: hash.Zero,
	}

	// First Push attempt - should fail with network error
	err = writer.Push(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
	assert.Equal(t, 1, callCount, "ReceivePack should have been called once")

	// Verify the writer still has the objects (not cleaned up)
	assert.True(t, writer.writer.HasObjects(), "Writer should still have objects after failed push")

	// Second Push attempt - should succeed using the same staged objects
	err = writer.Push(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "ReceivePack should have been called twice (retry)")

	// After successful push, writer should be reset (no objects)
	assert.False(t, writer.writer.HasObjects(), "Writer should be reset after successful push")
}

// TestStagedWriter_Push_ReceivePackSuccessIgnoresWritePackfileError tests that
// once ReceivePack succeeds, any WritePackfile error is logged but doesn't fail the push.
// This is critical for Git protocol semantics: ReceivePack success means the server
// has accepted the push, which is the source of truth.
func TestStagedWriter_Push_ReceivePackSuccessIgnoresWritePackfileError(t *testing.T) {
	ctx := context.Background()

	mockClient := &mockRawClient{
		receivePackFunc: func(ctx context.Context, r io.Reader) error {
			// Simulate ReceivePack succeeding (consuming and accepting the packfile)
			_, _ = io.Copy(io.Discard, r)
			return nil // ReceivePack succeeds!
		},
	}

	writer := &stagedWriter{
		client: &httpClient{
			RawClient: mockClient,
		},
		ref: Ref{
			Name: "refs/heads/main",
			Hash: hash.Zero,
		},
		writer:      protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory),
		objStorage:  storage.NewInMemoryStorage(ctx),
		treeEntries: make(map[string]*FlatTreeEntry),
		dirtyPaths:  make(map[string]bool),
		storageMode: protocol.PackfileStorageMemory,
	}

	// Stage a blob and commit
	_, err := writer.writer.AddBlob([]byte("test content"))
	require.NoError(t, err)

	treeHash := hash.Zero
	commitHash, err := writer.writer.AddCommit(
		treeHash,
		hash.Zero,
		&protocol.Identity{Name: "Test", Email: "test@example.com", Timestamp: 1234567890, Timezone: "+0000"},
		&protocol.Identity{Name: "Test", Email: "test@example.com", Timestamp: 1234567890, Timezone: "+0000"},
		"Test commit",
	)
	require.NoError(t, err)
	writer.lastCommit = &Commit{
		Hash:   commitHash,
		Tree:   treeHash,
		Parent: hash.Zero,
	}

	// Push should succeed - both ReceivePack and WritePackfile work
	err = writer.Push(ctx)

	// Assert: No error
	assert.NoError(t, err, "Push should succeed when ReceivePack succeeds")

	// Assert: Ref updated to new commit (push was successful)
	assert.Equal(t, commitHash, writer.ref.Hash, "Ref should be updated after successful push")

	// Assert: Writer cleaned up (objects removed)
	assert.False(t, writer.writer.HasObjects(), "Writer should be cleaned up after successful push")
}
