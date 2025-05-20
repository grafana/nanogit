package nanogit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTree(t *testing.T) {
	tests := []struct {
		name          string
		commitHash    string
		mockResponse  []byte
		statusCode    int
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
			statusCode: http.StatusOK,
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
			statusCode:    http.StatusInternalServerError,
			mockResponse:  []byte("Internal Server Error"),
			expectedError: "got status code 500",
		},
		{
			name:       "tree not found",
			commitHash: "1234567890abcdef1234567890abcdef12345678",
			mockResponse: func() []byte {
				pkt, _ := protocol.FormatPacks(protocol.PackLine("ACK 1234567890abcdef1234567890abcdef12345678\n"))
				return pkt
			}(),
			statusCode:    http.StatusOK,
			expectedError: "tree not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/git-upload-pack" {
					t.Errorf("expected path /git-upload-pack, got %s", r.URL.Path)
					return
				}
				if r.Method != http.MethodPost {
					t.Errorf("expected method POST, got %s", r.Method)
					return
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write(tt.mockResponse); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			hash, err := hash.FromHex(tt.commitHash)
			require.NoError(t, err)

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
