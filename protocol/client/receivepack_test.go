package client

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestReceivePack(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    options.Option
	}{
		{
			name:           "successful response",
			statusCode:     http.StatusOK,
			responseBody:   "000dunpack ok0000", // Valid Git packet format: unpack ok + flush
			expectedError:  "",
			expectedResult: "000dunpack ok0000",
			setupClient:    nil,
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			responseBody:   "not found",
			expectedError:  "got status code 404: 404 Not Found",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "server error",
			expectedError:  "got status code 500: 500 Internal Server Error",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "timeout error",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "context deadline exceeded",
			expectedResult: "",
			setupClient: options.WithHTTPClient(&http.Client{
				Timeout: 1 * time.Nanosecond,
			}),
		},
		{
			name:           "connection refused",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "i/o timeout",
			expectedResult: "",
			setupClient: options.WithHTTPClient(&http.Client{
				Transport: &http.Transport{
					DialContext: (&net.Dialer{
						Timeout: 1 * time.Nanosecond,
					}).DialContext,
				},
			}),
		},
		{
			name:       "git server error response",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "error: cannot lock ref 'refs/heads/main': is at d346cc9cd80dd0bbda023bb29a7ff2d887c75b19 but expected b6ce559b8c2e4834e075696cac5522b379448c13"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError:  "git server error:",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:       "git reference update error",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "ng refs/heads/main failed to update ref"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError:  "reference update failed for refs/heads/main:",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:       "git unpack error",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "unpack index-pack failed"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError:  "pack unpack failed:",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:       "git fatal error with unpack keyword",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "fatal: unpack failed due to corrupt data"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError:  "pack unpack failed:",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:       "git ERR packet",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "ERR push declined due to email policy"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError:  "git server ERR:",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:       "multi-line error like user's first example",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError:  "git server error:",
			expectedResult: "",
			setupClient:    nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/git-receive-pack" {
						t.Errorf("expected path /git-receive-pack, got %s", r.URL.Path)
						return
					}
					if r.Method != http.MethodPost {
						t.Errorf("expected method POST, got %s", r.Method)
						return
					}

					// Check default headers
					if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
						t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
						return
					}
					if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
						t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
						return
					}

					w.WriteHeader(tt.statusCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
				}))
				defer server.Close()
			}

			url := "http://127.0.0.1:0"
			if server != nil {
				url = server.URL
			}

			var (
				client *rawClient
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewRawClient(url, tt.setupClient)
			} else {
				client, err = NewRawClient(url)
			}
			require.NoError(t, err)

			response, err := client.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				
				// For Git protocol errors, we should still get the response body
				// even when there's an error, since it contains the error details
				if tt.statusCode == http.StatusOK && tt.responseBody != "" {
					require.NotEmpty(t, response, "should have response body even with Git protocol errors")
					require.Equal(t, tt.responseBody, string(response))
				} else {
					// For transport errors (non-200 status), response should be empty
					require.Empty(t, response)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, string(response))
			}
		})
	}
}
