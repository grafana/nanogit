package client

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/retry"
	"github.com/stretchr/testify/require"
)

func TestReceivePack(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
		setupClient   options.Option
	}{
		{
			name:          "successful response",
			statusCode:    http.StatusOK,
			responseBody:  "000dunpack ok0000", // Valid Git packet format: unpack ok + flush
			expectedError: "",
			setupClient:   nil,
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  "not found",
			expectedError: "got status code 404: 404 Not Found",
			setupClient:   nil,
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "server error",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "bad gateway",
			statusCode:    http.StatusBadGateway,
			responseBody:  "bad gateway",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "service unavailable",
			statusCode:    http.StatusServiceUnavailable,
			responseBody:  "service unavailable",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "gateway timeout",
			statusCode:    http.StatusGatewayTimeout,
			responseBody:  "gateway timeout",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "timeout error",
			statusCode:    0,
			responseBody:  "",
			expectedError: "context deadline exceeded",
			setupClient: options.WithHTTPClient(&http.Client{
				Timeout: 1 * time.Nanosecond,
			}),
		},
		{
			name:          "connection refused",
			statusCode:    0,
			responseBody:  "",
			expectedError: "i/o timeout",
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
			expectedError: "git server error:",
			setupClient:   nil,
		},
		{
			name:       "git reference update error",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "ng refs/heads/main failed to update ref"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "reference update failed for refs/heads/main:",
			setupClient:   nil,
		},
		{
			name:       "git unpack error",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "unpack index-pack failed"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "index-pack failed",
			setupClient:   nil,
		},
		{
			name:       "git fatal error with unpack keyword",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "fatal: unpack failed due to corrupt data"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "unpack failed due to corrupt data",
			setupClient:   nil,
		},
		{
			name:       "git ERR packet",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "ERR push declined due to email policy"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "push declined due to email policy",
			setupClient:   nil,
		},
		{
			name:       "multi-line error like user's first example",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted",
			setupClient:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/repo.git/git-receive-pack" {
						t.Errorf("expected path /repo.git/git-receive-pack, got %s", r.URL.Path)
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

			url := "http://127.0.0.1:0/repo"
			if server != nil {
				url = server.URL + "/repo"
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

			err = client.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				// Verify ServerUnavailableError for 5xx status codes
				if tt.statusCode >= 500 && tt.statusCode < 600 {
					require.True(t, errors.Is(err, protocol.ErrServerUnavailable), "error should be ErrServerUnavailable")
					var serverErr *protocol.ServerUnavailableError
					require.ErrorAs(t, err, &serverErr, "error should be ServerUnavailableError type")
					require.Equal(t, tt.statusCode, serverErr.StatusCode, "status code should match")
					require.NotNil(t, serverErr.Underlying, "underlying error should not be nil")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReceivePack_Retry(t *testing.T) {
	t.Parallel()

	t.Run("retries on network errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				// Simulate network error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000dunpack ok0000"))
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// Note: This test verifies retries are attempted
		_ = client.ReceivePack(ctx, strings.NewReader("test data"))

		// Verify retrier was called
		shouldRetryCalls := retrier.getShouldRetryCalls()
		require.GreaterOrEqual(t, len(shouldRetryCalls), 1, "ShouldRetry should be called for network errors")
	})

	t.Run("does not retry on 5xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.ReceivePack(ctx, strings.NewReader("test data"))
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "Should not retry POST requests on 5xx errors")

		// Verify retrier Wait was not called (no retries for 5xx POST errors)
		waitCalls := retrier.getWaitCalls()
		require.Equal(t, 0, len(waitCalls), "Wait should not be called for 5xx POST errors")
	})

	t.Run("works without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000dunpack ok0000"))
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should work but not retry
		err = client.ReceivePack(context.Background(), strings.NewReader("test"))
		require.NoError(t, err)
		require.Equal(t, 1, attemptCount, "should make single attempt without retrier")
	})
}
