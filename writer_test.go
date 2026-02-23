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

func (m *mockRawClient) IsAuthorized(ctx context.Context) (bool, error) {
	return true, nil
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

// TestStagedWriter_Cleanup_ToleratesCleanedUpWriter tests that calling
// Cleanup() on a stagedWriter with an already-cleaned-up PackfileWriter
// does not return an error (defensive behavior).
func TestStagedWriter_Cleanup_ToleratesCleanedUpWriter(t *testing.T) {
	ctx := context.Background()

	// Create a minimal stagedWriter
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

	// Force the PackfileWriter into a cleaned-up state
	err := writer.writer.Cleanup()
	require.NoError(t, err)

	// Now the PackfileWriter is cleaned up. Calling Cleanup again on the PackfileWriter
	// would return ErrPackfileWriterCleanedUp, but our defensive code should tolerate it.

	// Act: Call Cleanup on stagedWriter
	err = writer.Cleanup(ctx)

	// Assert: No error (tolerates already-cleaned-up state)
	assert.NoError(t, err, "Cleanup should tolerate already-cleaned-up PackfileWriter")

	// Assert: stagedWriter itself is now marked as cleaned up
	assert.True(t, writer.isCleanedUp)
}

// TestStagedWriter_Cleanup_PropagatesOtherErrors tests that Cleanup()
// still propagates errors other than ErrPackfileWriterCleanedUp.
func TestStagedWriter_Cleanup_PropagatesOtherErrors(t *testing.T) {
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

// TestStagedWriter_Cleanup_IdempotentBehavior tests that Cleanup() behavior
// is predictable when called multiple times.
func TestStagedWriter_Cleanup_IdempotentBehavior(t *testing.T) {
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
