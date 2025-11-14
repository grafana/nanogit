package client

import (
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

func TestSmartInfo(t *testing.T) {
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
			responseBody:  "000eversion 2\n0000", // Valid Git protocol response
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
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if !strings.HasPrefix(r.URL.Path, "/repo.git/info/refs") {
						t.Errorf("expected path starting with /repo.git/info/refs, got %s", r.URL.Path)
						return
					}
					if r.URL.Query().Get("service") != "custom-service" {
						t.Errorf("expected service=custom-service, got %s", r.URL.Query().Get("service"))
						return
					}
					if r.Method != http.MethodGet {
						t.Errorf("expected method GET, got %s", r.Method)
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

			err = client.SmartInfo(context.Background(), "custom-service")
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

func TestSmartInfo_Retry(t *testing.T) {
	t.Parallel()

	t.Run("retries on 5xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000eversion 2\n0000"))
		}))
		defer server.Close()

		retrier := newFakeRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return errors.Is(err, protocol.ErrServerUnavailable)
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			// Fast wait for testing
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.SmartInfo(ctx, "git-upload-pack")
		require.NoError(t, err)
		require.Equal(t, 3, attemptCount)

		// Verify retrier was called
		require.GreaterOrEqual(t, retrier.ShouldRetryCallCount(), 2, "ShouldRetry should be called at least twice")
		require.GreaterOrEqual(t, retrier.WaitCallCount(), 2, "Wait should be called at least twice")
	})

	t.Run("does not retry on 4xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		retrier := newFakeRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.SmartInfo(ctx, "git-upload-pack")
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "Should not retry on 4xx errors")

		// Verify retrier was not called for 4xx errors
		// 4xx errors are checked outside the retry wrapper, so ShouldRetry is never invoked
		require.Equal(t, 0, retrier.ShouldRetryCallCount(), "ShouldRetry should not be called for 4xx errors")
		require.Equal(t, 0, retrier.WaitCallCount(), "Wait should not be called for 4xx errors")
	})

	t.Run("retries on network errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				// Simulate network error by closing connection
				hj, ok := w.(http.Hijacker)
				if !ok {
					return
				}
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000eversion 2\n0000"))
		}))
		defer server.Close()

		retrier := newFakeRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			// Retry on any error for this test
			return err != nil
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// This might fail, but we're testing that retries are attempted
		_ = client.SmartInfo(ctx, "git-upload-pack")

		// Verify retrier was called
		require.GreaterOrEqual(t, retrier.ShouldRetryCallCount(), 1, "ShouldRetry should be called")
	})

	t.Run("works without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000eversion 2\n0000"))
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should work but not retry
		err = client.SmartInfo(context.Background(), "git-upload-pack")
		require.NoError(t, err)
		require.Equal(t, 1, attemptCount, "should make single attempt without retrier")
	})

	t.Run("does not retry on 5xx without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should fail immediately
		err = client.SmartInfo(context.Background(), "git-upload-pack")
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "should not retry without retrier")
	})
}
