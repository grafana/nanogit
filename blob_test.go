package nanogit

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/require"
)

func TestGetBlob(t *testing.T) {
	tests := []struct {
		name           string
		blobID         string
		infoRefsResp   string
		uploadPackResp func(t *testing.T) []byte
		expectedData   []byte
		expectedError  string
		statusCode     int
	}{
		{
			name:         "successful blob retrieval",
			blobID:       "08cf6101416f0ce0dda3c80e627f333854c4085c",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			uploadPackResp: func(t *testing.T) []byte {
				// Read test data from file
				data, err := os.ReadFile("testdata/upload-pack-get-blob")
				if err != nil {
					t.Fatalf("failed to read test data: %v", err)
				}
				return data
			},
			expectedData:  []byte("test content"),
			expectedError: "",
			statusCode:    http.StatusOK,
		},
		{
			name:         "blob not found",
			blobID:       "1234567890123456789012345678901234567890",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			uploadPackResp: func(t *testing.T) []byte {
				var response bytes.Buffer
				response.Write([]byte("0000"))                                                           // flush
				response.Write([]byte("0008NAK\n"))                                                      // NAK pkt-line
				response.Write([]byte("0045ERR not our ref 1234567890123456789012345678901234567890\n")) // error packet
				response.Write([]byte("0000"))                                                           // flush
				return response.Bytes()
			},
			expectedData:  nil,
			expectedError: "not our ref",
			statusCode:    http.StatusOK,
		},
		{
			name:         "server error",
			blobID:       "1234567890123456789012345678901234567890",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			uploadPackResp: func(t *testing.T) []byte {
				return []byte("Internal Server Error")
			},
			expectedData:  nil,
			expectedError: "got status code 500",
			statusCode:    http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/info/refs") {
					if _, err := w.Write([]byte(tt.infoRefsResp)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
					return
				}
				if r.URL.Path == "/git-upload-pack" {
					w.WriteHeader(tt.statusCode)
					if _, err := w.Write(tt.uploadPackResp(t)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
					return
				}
				t.Errorf("unexpected request path: %s", r.URL.Path)
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			h, err := hash.FromHex(tt.blobID)
			require.NoError(t, err)

			blob, err := client.GetBlob(context.Background(), h)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Nil(t, blob)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedData, blob.Content)
				require.Equal(t, h, blob.Hash)
			}
		})
	}
}
