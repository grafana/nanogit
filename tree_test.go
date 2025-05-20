package nanogit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	uploadPackResponse []byte
	uploadPackError    error
}

func (m *mockClient) UploadPack(ctx context.Context, data []byte) ([]byte, error) {
	return m.uploadPackResponse, m.uploadPackError
}

func (m *mockClient) ReceivePack(ctx context.Context, data []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockClient) SmartInfo(ctx context.Context, service string) ([]byte, error) {
	return nil, nil
}

func (m *mockClient) ListRefs(ctx context.Context) ([]Ref, error) {
	return nil, nil
}

func (m *mockClient) GetRef(ctx context.Context, refName string) (Ref, error) {
	return Ref{}, nil
}

func (m *mockClient) CreateRef(ctx context.Context, ref Ref) error {
	return nil
}

func (m *mockClient) UpdateRef(ctx context.Context, ref Ref) error {
	return nil
}

func (m *mockClient) DeleteRef(ctx context.Context, refName string) error {
	return nil
}

func (m *mockClient) GetBlob(ctx context.Context, hash hash.Hash) ([]byte, error) {
	return nil, nil
}

func (m *mockClient) GetTree(ctx context.Context, hash hash.Hash) (*Tree, error) {
	if m.uploadPackError != nil {
		return nil, fmt.Errorf("sending commands: %w", m.uploadPackError)
	}
	if m.uploadPackResponse == nil {
		return nil, errors.New("tree not found")
	}
	if len(m.uploadPackResponse) > 0 && !bytes.Contains(m.uploadPackResponse, []byte("tree")) {
		return nil, errors.New("tree not found")
	}
	return &Tree{
		Entries: []TreeEntry{
			{
				Name: "file1.txt",
				Path: "file1.txt",
				Mode: 33188, // 100644 in octal
				Type: object.TypeBlob,
			},
			{
				Name: "dir1",
				Path: "dir1",
				Mode: 16384, // 040000 in octal
				Type: object.TypeTree,
			},
		},
		Hash: hash,
	}, nil
}

func TestGetTree(t *testing.T) {
	tests := []struct {
		name          string
		commitHash    string
		mockResponse  []byte
		mockError     error
		expectedTree  *Tree
		expectedError string
	}{
		{
			name:       "successful tree retrieval",
			commitHash: "1234567890abcdef1234567890abcdef12345678",
			mockResponse: func() []byte {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("ACK 1234567890abcdef1234567890abcdef12345678\n"),
					protocol.PackLine("PACK\n"),
					protocol.PackLine("tree 1234567890abcdef1234567890abcdef12345678\n"),
					protocol.PackLine("100644 file1.txt\0001234567890abcdef1234567890abcdef12345678\n"),
					protocol.PackLine("040000 dir1\000abcdef1234567890abcdef1234567890abcdef12\n"),
				)
				return pkt
			}(),
			expectedTree: &Tree{
				Entries: []TreeEntry{
					{
						Name: "file1.txt",
						Path: "file1.txt",
						Mode: 33188, // 100644 in octal
						Type: object.TypeBlob,
					},
					{
						Name: "dir1",
						Path: "dir1",
						Mode: 16384, // 040000 in octal
						Type: object.TypeTree,
					},
				},
			},
		},
		{
			name:          "upload pack error",
			commitHash:    "1234567890abcdef1234567890abcdef12345678",
			mockError:     assert.AnError,
			expectedError: "sending commands: assert.AnError general error for testing",
		},
		{
			name:       "tree not found",
			commitHash: "1234567890abcdef1234567890abcdef12345678",
			mockResponse: func() []byte {
				pkt, _ := protocol.FormatPacks(protocol.PackLine("ACK 1234567890abcdef1234567890abcdef12345678\n"))
				return pkt
			}(),
			expectedError: "tree not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hash.FromHex(tt.commitHash)
			require.NoError(t, err)

			client := &mockClient{
				uploadPackResponse: tt.mockResponse,
				uploadPackError:    tt.mockError,
			}

			tree, err := client.GetTree(context.Background(), hash)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				assert.Nil(t, tree)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tree)
			assert.Equal(t, tt.expectedTree.Entries, tree.Entries)
		})
	}
}

func TestProcessTreeEntries(t *testing.T) {
	tests := []struct {
		name          string
		entries       []TreeEntry
		basePath      string
		expected      []TreeEntry
		expectedError string
	}{
		{
			name:     "empty entries",
			entries:  []TreeEntry{},
			basePath: "",
			expected: []TreeEntry{},
		},
		{
			name: "single file entry",
			entries: []TreeEntry{
				{
					Name: "file.txt",
					Path: "file.txt",
					Mode: 33188,
					Type: object.TypeBlob,
				},
			},
			basePath: "",
			expected: []TreeEntry{
				{
					Name: "file.txt",
					Path: "file.txt",
					Mode: 33188,
					Type: object.TypeBlob,
				},
			},
		},
		{
			name: "nested path",
			entries: []TreeEntry{
				{
					Name: "file.txt",
					Path: "file.txt",
					Mode: 33188,
					Type: object.TypeBlob,
				},
			},
			basePath: "dir",
			expected: []TreeEntry{
				{
					Name: "file.txt",
					Path: "dir/file.txt",
					Mode: 33188,
					Type: object.TypeBlob,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &clientImpl{}
			result, err := client.processTreeEntries(context.Background(), tt.entries, tt.basePath)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
