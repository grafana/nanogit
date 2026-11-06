package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/retry"
	"github.com/stretchr/testify/require"
)

func TestHTTPRetrier_ShouldRetry(t *testing.T) {
	t.Parallel()

	t.Run("retries on server unavailable errors with GET operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodGet, http.StatusServiceUnavailable, errors.New("service unavailable"))

		// GET operations can be retried on 5xx
		// Max attempts are handled by retry.Do, not ShouldRetry
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 2))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 3))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 4))
	})

	t.Run("retries on server unavailable errors with DELETE operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodDelete, http.StatusInternalServerError, errors.New("internal server error"))

		// DELETE operations can be retried on 5xx (idempotent like GET)
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 2))
	})

	t.Run("does not retry on server unavailable errors with POST operation", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodPost, http.StatusServiceUnavailable, errors.New("service unavailable"))

		// POST operations cannot be retried on 5xx because request body is consumed
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on server unavailable errors with POST operation and 500 status", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodPost, http.StatusInternalServerError, errors.New("internal server error"))

		// POST operations cannot be retried on 5xx because request body is consumed
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on server unavailable errors without operation (conservative)", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError("", http.StatusServiceUnavailable, errors.New("service unavailable"))

		// Without operation info, be conservative and don't retry on 5xx
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on wrapped server unavailable errors with GET operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		underlying := errors.New("underlying error")
		err := fmt.Errorf("wrapped: %w", NewServerUnavailableError(http.MethodGet, http.StatusInternalServerError, underlying))

		// Should detect server unavailable error through error chain and retry GET operations
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on 429 Too Many Requests for GET operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodGet, http.StatusTooManyRequests, errors.New("too many requests"))

		// 429 can be retried for GET operations
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on 429 Too Many Requests for POST operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodPost, http.StatusTooManyRequests, errors.New("too many requests"))

		// 429 can be retried for POST operations (rate limiting doesn't consume body)
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on other 4xx status codes", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodGet, http.StatusNotFound, errors.New("not found"))

		// Other 4xx errors should not be retried
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on 4xx client errors", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := fmt.Errorf("got status code %d: Not Found", http.StatusNotFound)

		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on nil errors", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		require.False(t, tempRetrier.ShouldRetry(ctx, nil, 1))
	})

	t.Run("delegates context cancellation to wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := context.Canceled

		// Context cancellation is handled by wrapped retrier
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("delegates context deadline exceeded to wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := context.DeadlineExceeded

		// Context deadline exceeded is handled by wrapped retrier
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("delegates server unavailable error with context cancellation to wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		// Wrap context.Canceled in ServerUnavailableError
		err := NewServerUnavailableError(http.MethodGet, http.StatusInternalServerError, context.Canceled)

		// Server unavailable error with GET is retryable, but underlying context.Canceled
		// should be checked by wrapped retrier when we delegate
		// Since it's retryable, we return true directly (don't check underlying error)
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("delegates server unavailable error with context deadline exceeded to wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		// Wrap context.DeadlineExceeded in ServerUnavailableError
		err := NewServerUnavailableError(http.MethodGet, http.StatusInternalServerError, context.DeadlineExceeded)

		// Server unavailable error with GET is retryable, but underlying context.DeadlineExceeded
		// should be checked by wrapped retrier when we delegate
		// Since it's retryable, we return true directly (don't check underlying error)
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("ShouldRetry does not check max attempts", func(t *testing.T) {
		baseRetrier := newTestRetrier(2)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodGet, http.StatusInternalServerError, errors.New("internal server error"))

		// ShouldRetry should return true for all attempts - max attempts are handled by retry.Do
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 2))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 3))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 100))

		// MaxAttempts should still return the correct value
		require.Equal(t, 2, tempRetrier.MaxAttempts())
	})
}

func TestHTTPRetrier_Wait(t *testing.T) {
	t.Parallel()

	t.Run("delegates wait to wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier().
			WithInitialDelay(10 * time.Millisecond).
			WithoutJitter()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx := context.Background()

		start := time.Now()
		err := tempRetrier.Wait(ctx, 1)
		duration := time.Since(start)

		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, 10*time.Millisecond)
		require.Less(t, duration, 50*time.Millisecond)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier().
			WithInitialDelay(1 * time.Second)
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := tempRetrier.Wait(ctx, 1)
		require.Error(t, err)
		require.True(t, errors.Is(err, context.Canceled))
	})
}

func TestHTTPRetrier_MaxAttempts(t *testing.T) {
	t.Parallel()

	t.Run("delegates max attempts to wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier().WithMaxAttempts(5)
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		require.Equal(t, 5, tempRetrier.MaxAttempts())
	})

	t.Run("uses default when wrapped retrier has default", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		require.Equal(t, 3, tempRetrier.MaxAttempts())
	})

	t.Run("handles nil wrapped retrier", func(t *testing.T) {
		tempRetrier := newTemporaryErrorRetrier(nil, true)

		// Should use NoopRetrier which returns 1
		require.Equal(t, 1, tempRetrier.MaxAttempts())
	})
}

func TestNewHTTPRetrier(t *testing.T) {
	t.Parallel()

	t.Run("creates retrier with wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier, true)

		require.NotNil(t, tempRetrier)
		require.Equal(t, baseRetrier, tempRetrier.wrapped)
	})

	t.Run("handles nil wrapped retrier", func(t *testing.T) {
		tempRetrier := newTemporaryErrorRetrier(nil, true)

		require.NotNil(t, tempRetrier)
		require.NotNil(t, tempRetrier.wrapped)
		// Should use NoopRetrier
		require.Equal(t, 1, tempRetrier.MaxAttempts())
	})
}

func TestHTTPRetrier_isRetryableOperation(t *testing.T) {
	t.Parallel()

	tempRetrier := newTemporaryErrorRetrier(retry.NewExponentialBackoffRetrier(), true)

	t.Run("retries GET operations on 5xx", func(t *testing.T) {
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusInternalServerError))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusServiceUnavailable))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusBadGateway))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusGatewayTimeout))
	})

	t.Run("does not retry POST operations on 5xx", func(t *testing.T) {
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusInternalServerError))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusServiceUnavailable))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusBadGateway))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusGatewayTimeout))
	})

	t.Run("retries 429 for all operations", func(t *testing.T) {
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusTooManyRequests))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusTooManyRequests))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodPut, http.StatusTooManyRequests))
		require.True(t, tempRetrier.isRetryableOperation("", http.StatusTooManyRequests))
	})

	t.Run("retries rate-limited 403 for all operations", func(t *testing.T) {
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusForbidden))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusForbidden))
		require.True(t, tempRetrier.isRetryableOperation("", http.StatusForbidden))
	})

	t.Run("does not retry on other 4xx", func(t *testing.T) {
		require.False(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusNotFound))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusBadRequest))
	})

	t.Run("retries DELETE operations on 5xx", func(t *testing.T) {
		require.True(t, tempRetrier.isRetryableOperation(http.MethodDelete, http.StatusInternalServerError))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodDelete, http.StatusServiceUnavailable))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodDelete, http.StatusBadGateway))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodDelete, http.StatusGatewayTimeout))
	})

	t.Run("does not retry unknown operations on 5xx (conservative)", func(t *testing.T) {
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPut, http.StatusInternalServerError))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPatch, http.StatusInternalServerError))
		require.False(t, tempRetrier.isRetryableOperation("", http.StatusInternalServerError))
	})
}

// retryAfterCapturingRetrier records the Retry-After delay carried by the error
// that triggered each wait.
type retryAfterCapturingRetrier struct {
	*testRetrier
	delays []time.Duration
}

func (r *retryAfterCapturingRetrier) WaitAfter(ctx context.Context, attempt int, err error) error {
	delay, _ := retry.RetryAfterFrom(err)
	r.delays = append(r.delays, delay)
	return r.Wait(ctx, attempt)
}

func TestHTTPRetrier_RateLimitedForbidden(t *testing.T) {
	t.Parallel()

	t.Run("retries a 403 carrying Retry-After and passes the delay to the retrier", func(t *testing.T) {
		attempts := 0
		successResponse := formatTestResponse(t, protocol.PackLine("version 2\n"))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(successResponse))
		}))
		defer server.Close()

		base := newTestRetrier(3)
		base.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool { return false }
		capturing := &retryAfterCapturingRetrier{testRetrier: base}

		ctx := retry.ToContext(context.Background(), capturing)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		require.NoError(t, client.SmartInfo(ctx, "git-upload-pack"))
		require.Equal(t, 3, attempts)
		require.Equal(t, []time.Duration{time.Second, time.Second}, capturing.delays)
	})

	t.Run("does not retry a 403 without rate-limit signals", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		retrier := newTestRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.SmartInfo(ctx, "git-upload-pack")
		require.ErrorIs(t, err, ErrPermissionDenied)
		require.Equal(t, 1, attempts)
	})
}

// writeRawResponse writes a response with header names spelled exactly as given,
// bypassing the canonicalization http.Header.Set would apply on the way out.
func writeRawResponse(t *testing.T, w http.ResponseWriter, status, headers string) {
	t.Helper()
	conn, buf, err := w.(http.Hijacker).Hijack()
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	_, err = buf.WriteString("HTTP/1.1 " + status + "\r\n" + headers + "Content-Length: 0\r\nConnection: close\r\n\r\n")
	require.NoError(t, err)
	require.NoError(t, buf.Flush())
}

func TestHTTPRetrier_LowercaseRateLimitHeaders(t *testing.T) {
	t.Parallel()

	// GitLab spells these in lowercase on the wire; net/http canonicalizes
	// incoming header names, so the lookups find them either way.
	tests := []struct {
		name    string
		headers string
	}{
		{name: "lowercase retry-after", headers: "retry-after: 1\r\n"},
		{name: "lowercase x-ratelimit-remaining", headers: "x-ratelimit-remaining: 0\r\n"},
		{name: "lowercase ratelimit-remaining", headers: "ratelimit-remaining: 0\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			attempts := 0
			successResponse := formatTestResponse(t, protocol.PackLine("version 2\n"))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts < 2 {
					writeRawResponse(t, w, "403 Forbidden", tt.headers)
					return
				}
				_, _ = w.Write([]byte(successResponse))
			}))
			defer server.Close()

			retrier := newTestRetrier(3)
			retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool { return false }
			ctx := retry.ToContext(context.Background(), retrier)
			client, err := NewRawClient(server.URL + "/repo")
			require.NoError(t, err)

			require.NoError(t, client.SmartInfo(ctx, "git-upload-pack"))
			require.Equal(t, 2, attempts)
		})
	}
}

func TestHTTPRetrier_RateLimitedPOSTBodyReplay(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte("x"), 64)

	newRateLimitingServer := func(t *testing.T, received *[][]byte) *httptest.Server {
		attempts := 0
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			*received = append(*received, body)
			if attempts < 2 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
	}

	t.Run("replayable body is resent in full", func(t *testing.T) {
		var received [][]byte
		server := newRateLimitingServer(t, &received)
		defer server.Close()

		ctx := retry.ToContext(context.Background(), retry.NewExponentialBackoffRetrier().
			WithMaxAttempts(3).WithInitialDelay(time.Millisecond).WithoutJitter())
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		body, err := client.UploadPack(ctx, bytes.NewReader(payload))
		require.NoError(t, err)
		require.NoError(t, body.Close())

		require.Len(t, received, 2)
		require.Equal(t, payload, received[0])
		require.Equal(t, payload, received[1], "the retry must carry the full payload, not a truncated one")
	})

	t.Run("streamed body is not retried rather than resent empty", func(t *testing.T) {
		var received [][]byte
		server := newRateLimitingServer(t, &received)
		defer server.Close()

		ctx := retry.ToContext(context.Background(), retry.NewExponentialBackoffRetrier().
			WithMaxAttempts(3).WithInitialDelay(time.Millisecond).WithoutJitter())
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// An opaque ReadCloser gives net/http no GetBody to rewind.
		_, err = client.UploadPack(ctx, io.NopCloser(bytes.NewReader(payload)))
		require.ErrorIs(t, err, ErrServerUnavailable)
		require.Len(t, received, 1, "a body that cannot be rewound must not be retried")
		require.Equal(t, payload, received[0])
	})
}
