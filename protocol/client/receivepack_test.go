package client

import (
	"bytes"
	"context"
	"errors"
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
			responseBody:   "refs data",
			expectedError:  "",
			expectedResult: "refs data",
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
				require.Empty(t, response)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, string(response))
			}
		})
	}
}

func TestCheckReceivePackErrors(t *testing.T) {
	tests := []struct {
		name              string
		responseBody      string
		expectError       bool
		expectedErrorType string
		expectedMessage   string
		expectedRefName   string
		expectedReason    string
	}{
		{
			name:         "no error - successful response",
			responseBody: "0043000eunpack ok\n0017ok refs/heads/main\n0000",
			expectError:  false,
		},
		{
			name:         "no error - empty response",
			responseBody: "",
			expectError:  false,
		},
		{
			name:              "error - cannot lock ref with side-band",
			responseBody:      "0094error: cannot lock ref 'refs/heads/main': is at abc123 but expected def456\n0043000eunpack ok\n002cng refs/heads/main failed to update ref\n00000000",
			expectError:       true,
			expectedErrorType: "GitServerError",
			expectedMessage:   "cannot lock ref 'refs/heads/main': is at abc123 but expected def456",
		},
		{
			name:              "error - reference update failure ng packet",
			responseBody:      "0043000eunpack ok\n002cng refs/heads/main failed to update ref\n0000",
			expectError:       true,
			expectedErrorType: "GitReferenceUpdateError",
			expectedRefName:   "refs/heads/main",
			expectedReason:    "failed to update ref",
		},
		{
			name:              "error - reference update failure ng packet with simple reason",
			responseBody:      "0043000eunpack ok\n001eng refs/heads/main failed\n0000",
			expectError:       true,
			expectedErrorType: "GitReferenceUpdateError",
			expectedRefName:   "refs/heads/main",
			expectedReason:    "failed",
		},
		{
			name:              "error - reference update failure ng packet minimal",
			responseBody:      "0043000eunpack ok\n0015ng refs/heads/main\n0000",
			expectError:       true,
			expectedErrorType: "GitReferenceUpdateError",
			expectedRefName:   "refs/heads/main",
			expectedReason:    "update failed", // default reason
		},
		{
			name:              "error - multiple refs with ng failure",
			responseBody:      "0043000eunpack ok\n0017ok refs/heads/feature\n002ang refs/heads/main rejected\n0000",
			expectError:       true,
			expectedErrorType: "GitReferenceUpdateError",
			expectedRefName:   "refs/heads/main",
			expectedReason:    "rejected",
		},
		{
			name:              "error - cannot lock ref in different format",
			responseBody:      "0089error: cannot lock ref 'refs/heads/feature': is at 123abc but expected 456def\n0043000eunpack ok\n0000",
			expectError:       true,
			expectedErrorType: "GitServerError",
			expectedMessage:   "cannot lock ref 'refs/heads/feature': is at 123abc but expected 456def",
		},
		{
			name:              "error - cannot lock ref with complex hash",
			responseBody:      "00a2error: cannot lock ref 'refs/heads/main': is at e05cf8c9566dac359fcb7c095d5d121e984619c2 but expected 702a02239830a6c6cd45e8f6e5bbb1816f1f4cff\n0043000eunpack ok\n002cng refs/heads/main failed to update ref\n0000",
			expectError:       true,
			expectedErrorType: "GitServerError",
			expectedMessage:   "cannot lock ref 'refs/heads/main': is at e05cf8c9566dac359fcb7c095d5d121e984619c2 but expected 702a02239830a6c6cd45e8f6e5bbb1816f1f4cff",
		},
		{
			name:         "no error - unpack ok only",
			responseBody: "000eunpack ok\n0000",
			expectError:  false,
		},
		{
			name:         "no error - successful push with ok response",
			responseBody: "000eunpack ok\n0017ok refs/heads/main\n0000",
			expectError:  false,
		},
		{
			name:              "error - ng with tag reference",
			responseBody:      "000eunpack ok\n001fng refs/tags/v1.0.0 denied\n0000",
			expectError:       true,
			expectedErrorType: "GitReferenceUpdateError",
			expectedRefName:   "refs/tags/v1.0.0",
			expectedReason:    "denied",
		},
		{
			name:         "no error - contains error text but not error pattern",
			responseBody: "0023Some message with error word\n0000",
			expectError:  false,
		},
		{
			name:         "no error - contains ng but not ng refs pattern",
			responseBody: "001cSome ng message here\n0000",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkReceivePackErrors([]byte(tt.responseBody))

			if tt.expectError {
				require.Error(t, err, "Expected an error but got none")

				switch tt.expectedErrorType {
				case "GitServerError":
					var gitServerErr *protocol.GitServerError
					require.True(t, errors.As(err, &gitServerErr), "Expected GitServerError but got %T", err)
					require.True(t, protocol.IsGitServerError(err), "IsGitServerError should return true")
					if tt.expectedMessage != "" {
						require.Contains(t, gitServerErr.Message, tt.expectedMessage)
					}
					require.Equal(t, "error", gitServerErr.ErrorType)

				case "GitReferenceUpdateError":
					var gitRefErr *protocol.GitReferenceUpdateError
					require.True(t, errors.As(err, &gitRefErr), "Expected GitReferenceUpdateError but got %T", err)
					require.True(t, protocol.IsGitReferenceUpdateError(err), "IsGitReferenceUpdateError should return true")
					if tt.expectedRefName != "" {
						require.Equal(t, tt.expectedRefName, gitRefErr.RefName)
					}
					if tt.expectedReason != "" {
						require.Equal(t, tt.expectedReason, gitRefErr.Reason)
					}

				default:
					t.Errorf("Unknown expected error type: %s", tt.expectedErrorType)
				}
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}
