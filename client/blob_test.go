package client

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			blobID:       "1234567890123456789012345678901234567890",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			uploadPackResp: func(t *testing.T) []byte {
				content := []byte("test blob content")
				blobObj := []byte(fmt.Sprintf("blob %d\x00%s", len(content), content))
				var compressed bytes.Buffer
				zw := zlib.NewWriter(&compressed)
				if _, err := zw.Write(blobObj); err != nil {
					t.Fatalf("failed to compress blob: %v", err)
				}
				if err := zw.Close(); err != nil {
					t.Fatalf("failed to close zlib writer: %v", err)
				}
				packfile := make([]byte, 0, 1024)
				packfile = append(packfile, []byte("PACK")...)
				packfile = append(packfile, 0, 0, 0, 2)
				packfile = append(packfile, 0, 0, 0, 1)
				size := uint64(len(blobObj))
				header := make([]byte, 0, 8)
				header = append(header, 0x30)
				for size > 0 {
					header = append(header, byte(size&0x7f))
					size >>= 7
				}
				packfile = append(packfile, header...)
				packfile = append(packfile, compressed.Bytes()...)
				var response bytes.Buffer
				response.Write([]byte("0000"))      // flush
				response.Write([]byte("0008NAK\n")) // NAK pkt-line
				response.Write(packfile)            // raw packfile
				// NO flush after packfile!
				return response.Bytes()
			},
			expectedData:  []byte("test blob content"),
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
			expectedError: "blob not found",
			statusCode:    http.StatusOK,
		},
		{
			name:         "invalid hash format",
			blobID:       "invalid-hash",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			uploadPackResp: func(t *testing.T) []byte {
				return []byte("0000")
			},
			expectedData:  nil,
			expectedError: "invalid hash format",
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

			client, err := New(server.URL)
			require.NoError(t, err)

			data, err := client.GetBlob(context.Background(), tt.blobID)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedData, data)
			}
		})
	}
}
