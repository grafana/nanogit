package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafana/nanogit/retry"
	"github.com/stretchr/testify/require"
)

func TestHTTPRetrier_ShouldRetry(t *testing.T) {
	t.Parallel()

	t.Run("retries on network errors with timeout error", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier().WithMaxAttempts(3)
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{},
		}

		// Should delegate to base retrier for timeout network errors
		// ShouldRetry should return true for all attempts - max attempts are handled by retry.Do
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 2))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 3))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 4))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 100))
	})

	t.Run("retries on network timeout errors", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: &timeoutError{},
		}

		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on timeout network errors", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: &timeoutError{},
		}

		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on network errors wrapped in url.Error", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &timeoutError{},
			},
		}

		// Should retry on timeout errors wrapped in url.Error
		require.True(t, tempRetrier.ShouldRetry(ctx, urlErr, 1))
	})

	t.Run("does not retry on url.Error with nil Err", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: nil,
		}

		require.False(t, tempRetrier.ShouldRetry(ctx, urlErr, 1))
	})

	t.Run("does not retry on url.Error with non-net.Error Err", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: errors.New("not a network error"),
		}

		require.False(t, tempRetrier.ShouldRetry(ctx, urlErr, 1))
	})

	t.Run("does not retry on url.Error with net.Error that is not a timeout", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: errors.New("connection refused"),
			},
		}

		require.False(t, tempRetrier.ShouldRetry(ctx, urlErr, 1))
	})

	t.Run("isTemporaryNetworkError returns false for url.Error with net.Error that is not a timeout", func(t *testing.T) {
		tempRetrier := newTemporaryErrorRetrier(retry.NewExponentialBackoffRetrier())

		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: errors.New("connection refused"),
			},
		}

		// net.OpError implements net.Error, but Timeout() returns false
		require.False(t, tempRetrier.isTemporaryNetworkError(urlErr))
	})
}

func TestHTTPRetrier_isTemporaryNetworkError(t *testing.T) {
	t.Parallel()

	tempRetrier := newTemporaryErrorRetrier(retry.NewExponentialBackoffRetrier())

	t.Run("returns true for net.Error with Timeout", func(t *testing.T) {
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{},
		}
		require.True(t, tempRetrier.isTemporaryNetworkError(err))
	})

	t.Run("returns false for net.Error without Timeout", func(t *testing.T) {
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.False(t, tempRetrier.isTemporaryNetworkError(err))
	})

	t.Run("returns true for url.Error wrapping net.Error with Timeout", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &timeoutError{},
			},
		}
		require.True(t, tempRetrier.isTemporaryNetworkError(urlErr))
	})

	t.Run("returns false for url.Error wrapping net.Error without Timeout", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: errors.New("connection refused"),
			},
		}
		require.False(t, tempRetrier.isTemporaryNetworkError(urlErr))
	})

	t.Run("returns false for url.Error with nil Err", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: nil,
		}
		require.False(t, tempRetrier.isTemporaryNetworkError(urlErr))
	})

	t.Run("returns false for url.Error with non-net.Error Err", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/repo.git/info/refs",
			Err: errors.New("not a network error"),
		}
		require.False(t, tempRetrier.isTemporaryNetworkError(urlErr))
	})

	t.Run("returns false for non-network error", func(t *testing.T) {
		err := errors.New("not a network error")
		require.False(t, tempRetrier.isTemporaryNetworkError(err))
	})

	t.Run("does not retry on network errors without Timeout", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		// net.OpError with a plain error that doesn't implement Timeout()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}

		// Should not retry if error doesn't have Timeout()
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on network errors with only Temporary (deprecated)", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		// temporaryError has Temporary() = true but Timeout() = false
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &temporaryError{},
		}

		// Should not retry since we only check Timeout(), not Temporary()
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on server unavailable errors with GET operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

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
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodDelete, http.StatusInternalServerError, errors.New("internal server error"))

		// DELETE operations can be retried on 5xx (idempotent like GET)
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 2))
	})

	t.Run("does not retry on server unavailable errors with POST operation", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodPost, http.StatusServiceUnavailable, errors.New("service unavailable"))

		// POST operations cannot be retried on 5xx because request body is consumed
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on server unavailable errors with POST operation and 500 status", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodPost, http.StatusInternalServerError, errors.New("internal server error"))

		// POST operations cannot be retried on 5xx because request body is consumed
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on server unavailable errors without operation (conservative)", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := NewServerUnavailableError("", http.StatusServiceUnavailable, errors.New("service unavailable"))

		// Without operation info, be conservative and don't retry on 5xx
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on server unavailable errors with network error (statusCode == 0)", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		// Network error wrapped in ServerUnavailableError (statusCode 0 means network error)
		err := NewServerUnavailableError("", 0, errors.New("connection refused"))

		// Network errors are always retryable
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on wrapped server unavailable errors with GET operation", func(t *testing.T) {
		// Use a test retrier that retries on all errors
		baseRetrier := newTestRetrier(3)
		baseRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

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
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

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
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodPost, http.StatusTooManyRequests, errors.New("too many requests"))

		// 429 can be retried for POST operations (rate limiting doesn't consume body)
		require.True(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on other 4xx status codes", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := NewServerUnavailableError(http.MethodGet, http.StatusNotFound, errors.New("not found"))

		// Other 4xx errors should not be retried
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on 4xx client errors", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := fmt.Errorf("got status code %d: Not Found", http.StatusNotFound)

		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on nil errors", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		require.False(t, tempRetrier.ShouldRetry(ctx, nil, 1))
	})

	t.Run("does not retry on context cancellation", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := context.Canceled

		// Even though it's not a network error, the base retrier will check
		// and return false for context cancellation
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on context deadline exceeded", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := context.DeadlineExceeded

		// Even though it's not a network error, the base retrier will check
		// and return false for context deadline exceeded
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on server unavailable error with context cancellation", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		// Wrap context.Canceled in ServerUnavailableError
		err := NewServerUnavailableError(http.MethodGet, http.StatusInternalServerError, context.Canceled)

		// Should not retry even if it's a server unavailable error with GET
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on server unavailable error with context deadline exceeded", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		// Wrap context.DeadlineExceeded in ServerUnavailableError
		err := NewServerUnavailableError(http.MethodGet, http.StatusInternalServerError, context.DeadlineExceeded)

		// Should not retry even if it's a server unavailable error with GET
		require.False(t, tempRetrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("ShouldRetry does not check max attempts", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier().WithMaxAttempts(2)
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		ctx := context.Background()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{},
		}

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
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

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
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

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
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		require.Equal(t, 5, tempRetrier.MaxAttempts())
	})

	t.Run("uses default when wrapped retrier has default", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		require.Equal(t, 3, tempRetrier.MaxAttempts())
	})

	t.Run("handles nil wrapped retrier", func(t *testing.T) {
		tempRetrier := newTemporaryErrorRetrier(nil)

		// Should use NoopRetrier which returns 1
		require.Equal(t, 1, tempRetrier.MaxAttempts())
	})
}

func TestNewHTTPRetrier(t *testing.T) {
	t.Parallel()

	t.Run("creates retrier with wrapped retrier", func(t *testing.T) {
		baseRetrier := retry.NewExponentialBackoffRetrier()
		tempRetrier := newTemporaryErrorRetrier(baseRetrier)

		require.NotNil(t, tempRetrier)
		require.Equal(t, baseRetrier, tempRetrier.wrapped)
	})

	t.Run("handles nil wrapped retrier", func(t *testing.T) {
		tempRetrier := newTemporaryErrorRetrier(nil)

		require.NotNil(t, tempRetrier)
		require.NotNil(t, tempRetrier.wrapped)
		// Should use NoopRetrier
		require.Equal(t, 1, tempRetrier.MaxAttempts())
	})
}

func TestHTTPRetrier_isRetryableOperation(t *testing.T) {
	t.Parallel()

	tempRetrier := newTemporaryErrorRetrier(retry.NewExponentialBackoffRetrier())

	t.Run("retries network errors (statusCode == 0)", func(t *testing.T) {
		require.True(t, tempRetrier.isRetryableOperation(http.MethodPost, 0))
		require.True(t, tempRetrier.isRetryableOperation(http.MethodGet, 0))
		require.True(t, tempRetrier.isRetryableOperation("", 0))
	})

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

	t.Run("does not retry on other 4xx", func(t *testing.T) {
		require.False(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusNotFound))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodPost, http.StatusBadRequest))
		require.False(t, tempRetrier.isRetryableOperation(http.MethodGet, http.StatusForbidden))
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

// Helper types for testing

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

type temporaryError struct{}

func (e *temporaryError) Error() string   { return "temporary" }
func (e *temporaryError) Timeout() bool   { return false }
func (e *temporaryError) Temporary() bool { return true }
