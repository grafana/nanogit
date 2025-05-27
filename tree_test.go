package nanogit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFlatTree(t *testing.T) {
	// FIXME: we should control the response from server in this unit and not use fixtures
	mustFromHex := func(hs string) hash.Hash {
		h, err := hash.FromHex(hs)
		if err != nil {
			t.Fatalf("failed to parse hash %s: %v", hs, err)
		}
		return h
	}

	tests := []struct {
		name          string
		commitHash    string
		mockResponse  []byte
		statusCode    int
		expectedTree  *FlatTree
		expectedError string
	}{
		{
			name:       "successful tree retrieval",
			commitHash: "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			statusCode: http.StatusOK,
			expectedTree: &FlatTree{
				Entries: []FlatTreeEntry{
					{
						Name: "root.txt",
						Path: "root.txt",
						Mode: 33188, // 100644 in octal
						Type: protocol.ObjectTypeBlob,
						Hash: mustFromHex("6eec10ba6e8a5379cae2c49d01d214fd41fb713f"),
					},
					{
						Name: "dir1",
						Path: "dir1",
						Mode: 16384, // 040000 in octal
						Type: protocol.ObjectTypeTree,
						Hash: mustFromHex("1ae8c212049c2661d606c787235163365d440dcc"),
					},
					{
						Name: "file1.txt",
						Path: "dir1/file1.txt",
						Mode: 33188, // 100644 in octal
						Type: protocol.ObjectTypeBlob,
						Hash: mustFromHex("dd954e7a4e1a62ff90c5a0709dce5928716535c1"),
					},
					{
						Name: "file2.txt",
						Path: "dir1/file2.txt",
						Mode: 33188, // 100644 in octal
						Type: protocol.ObjectTypeBlob,
						Hash: mustFromHex("db00fd65b218578127ea51f3dffac701f12f486a"),
					},
					{
						Name: "dir2",
						Path: "dir2",
						Mode: 16384, // 040000 in octal
						Type: protocol.ObjectTypeTree,
						Hash: mustFromHex("fb90cfcb8044471fec2bb75a67cca6b16e7de4bc"),
					},
					{
						Name: "file3.txt",
						Path: "dir2/file3.txt",
						Mode: 33188, // 100644 in octal
						Type: protocol.ObjectTypeBlob,
						Hash: mustFromHex("a2b32293aab475bf50798c7642f0fe0593c167f6"),
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
				return []byte("0049ERR upload-pack: not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
			}(),
			statusCode:    http.StatusOK,
			expectedError: "not our ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.mockResponse != nil {
					w.WriteHeader(tt.statusCode)
					if _, err := w.Write(tt.mockResponse); err != nil {
						t.Errorf("failed to write response: %v", err)
					}

					return
				}

				if r.URL.Path != "/git-upload-pack" {
					t.Errorf("expected path /git-upload-pack, got %s", r.URL.Path)
					return
				}
				if r.Method != http.MethodPost {
					t.Errorf("expected method POST, got %s", r.Method)
					return
				}

				// Read the request body to extract the commit hash
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read request body: %v", err)
					return
				}

				// Parse the request to find the commit hash
				lines, _, err := protocol.ParsePack(body)
				if err != nil {
					t.Errorf("failed to parse request: %v", err)
					return
				}

				// Extract commit hash from the want line
				var commitHash string
				for _, line := range lines {
					lineStr := string(line)
					if strings.HasPrefix(lineStr, "want ") {
						commitHash = strings.TrimSpace(strings.TrimPrefix(lineStr, "want "))
						break
					}
				}

				if commitHash == "" {
					t.Error("no commit hash found in request")
					return
				}

				// Load test data based on the extracted commit hash
				testData, err := os.ReadFile(fmt.Sprintf("testdata/%s_gettree", commitHash))
				if err != nil {
					t.Errorf("failed to read test data: %v", err)
					return
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write(testData); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewHTTPClient(server.URL)
			require.NoError(t, err)

			hash, err := hash.FromHex(tt.commitHash)
			require.NoError(t, err)

			tree, err := client.GetFlatTree(context.Background(), hash)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				assert.Nil(t, tree)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tree)
			require.Equal(t, len(tt.expectedTree.Entries), len(tree.Entries))
		})
	}
}

func TestProcessFlatTreeEntries(t *testing.T) {
	tests := []struct {
		name          string
		entries       []FlatTreeEntry
		basePath      string
		expected      []FlatTreeEntry
		expectedError string
	}{
		{
			name:     "empty entries",
			entries:  []FlatTreeEntry{},
			basePath: "",
			expected: []FlatTreeEntry{},
		},
		{
			name: "single file entry",
			entries: []FlatTreeEntry{
				{
					Name: "file.txt",
					Path: "file.txt",
					Mode: 33188,
					Type: protocol.ObjectTypeBlob,
				},
			},
			basePath: "",
			expected: []FlatTreeEntry{
				{
					Name: "file.txt",
					Path: "file.txt",
					Mode: 33188,
					Type: protocol.ObjectTypeBlob,
				},
			},
		},
		{
			name: "nested path",
			entries: []FlatTreeEntry{
				{
					Name: "file.txt",
					Path: "file.txt",
					Mode: 33188,
					Type: protocol.ObjectTypeBlob,
				},
			},
			basePath: "dir",
			expected: []FlatTreeEntry{
				{
					Name: "file.txt",
					Path: "dir/file.txt",
					Mode: 33188,
					Type: protocol.ObjectTypeBlob,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &httpClient{}
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

func TestGetTree(t *testing.T) {
	mustFromHex := func(hs string) hash.Hash {
		h, err := hash.FromHex(hs)
		if err != nil {
			t.Fatalf("failed to parse hash %s: %v", hs, err)
		}
		return h
	}

	tests := []struct {
		name          string
		treeHash      string
		mockResponse  []byte
		statusCode    int
		expectedTree  *Tree
		expectedError string
	}{
		{
			name:       "successful root tree retrieval",
			treeHash:   "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			statusCode: http.StatusOK,
			expectedTree: &Tree{
				Entries: []TreeEntry{
					{
						Name: "root.txt",
						Mode: 33188, // 100644 in octal
						Type: protocol.ObjectTypeBlob,
						Hash: mustFromHex("6eec10ba6e8a5379cae2c49d01d214fd41fb713f"),
					},
					{
						Name: "dir1",
						Mode: 16384, // 040000 in octal
						Type: protocol.ObjectTypeTree,
						Hash: mustFromHex("1ae8c212049c2661d606c787235163365d440dcc"),
					},
					{
						Name: "dir2",
						Mode: 16384, // 040000 in octal
						Type: protocol.ObjectTypeTree,
						Hash: mustFromHex("fb90cfcb8044471fec2bb75a67cca6b16e7de4bc"),
					},
				},
			},
		},
		{
			name:          "tree not found",
			treeHash:      "1234567890abcdef1234567890abcdef12345678",
			mockResponse:  []byte("0049ERR upload-pack: not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0"),
			statusCode:    http.StatusOK,
			expectedError: "not our ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.mockResponse != nil {
					w.WriteHeader(tt.statusCode)
					if _, err := w.Write(tt.mockResponse); err != nil {
						t.Errorf("failed to write response: %v", err)
					}
					return
				}

				// Load test data based on the tree hash
				testData, err := os.ReadFile(fmt.Sprintf("testdata/%s_gettree", tt.treeHash))
				if err != nil {
					t.Errorf("failed to read test data: %v", err)
					return
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write(testData); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewHTTPClient(server.URL)
			require.NoError(t, err)

			treeHash, err := hash.FromHex(tt.treeHash)
			require.NoError(t, err)

			tree, err := client.GetTree(context.Background(), treeHash)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				assert.Nil(t, tree)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tree)
			assert.Equal(t, len(tt.expectedTree.Entries), len(tree.Entries))

			// Verify each entry (order doesn't matter)
			for _, expectedEntry := range tt.expectedTree.Entries {
				found := false
				for _, actualEntry := range tree.Entries {
					if actualEntry.Name == expectedEntry.Name {
						assert.Equal(t, expectedEntry.Mode, actualEntry.Mode)
						assert.Equal(t, expectedEntry.Type, actualEntry.Type)
						assert.Equal(t, expectedEntry.Hash, actualEntry.Hash)
						found = true
						break
					}
				}
				assert.True(t, found, "Expected entry %s not found", expectedEntry.Name)
			}
		})
	}
}

func TestGetTreeByPath(t *testing.T) {
	tests := []struct {
		name          string
		rootHash      string
		path          string
		expectedError string
		expectedName  string // Name of the directory we expect to get
	}{
		{
			name:         "get root tree with empty path",
			rootHash:     "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			path:         "",
			expectedName: "", // Root doesn't have a name
		},
		{
			name:         "get root tree with dot path",
			rootHash:     "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			path:         ".",
			expectedName: "",
		},
		{
			name:         "get subdirectory tree",
			rootHash:     "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			path:         "dir1",
			expectedName: "dir1",
		},
		{
			name:          "nonexistent path",
			rootHash:      "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			path:          "nonexistent",
			expectedError: "path component 'nonexistent' not found",
		},
		{
			name:          "path to file instead of directory",
			rootHash:      "dc3245b0d6b48a874ae6fc599a26ce990ea05ff2",
			path:          "root.txt",
			expectedError: "path component 'root.txt' is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Load test data - using the same test data as GetFlatTree
				testData, err := os.ReadFile(fmt.Sprintf("testdata/%s_gettree", tt.rootHash))
				if err != nil {
					t.Errorf("failed to read test data: %v", err)
					return
				}

				w.WriteHeader(http.StatusOK)
				if _, err := w.Write(testData); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewHTTPClient(server.URL)
			require.NoError(t, err)

			rootHash, err := hash.FromHex(tt.rootHash)
			require.NoError(t, err)

			tree, err := client.GetTreeByPath(context.Background(), rootHash, tt.path)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				assert.Nil(t, tree)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tree)
			assert.NotEmpty(t, tree.Entries, "Tree should have entries")
		})
	}
}
