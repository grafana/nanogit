package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/retry"
)

// httpRetrier wraps another retrier and only retries on HTTP-specific errors:
// - Network errors (net.Error)
// - Server unavailable errors (ErrServerUnavailable)
//
// All other errors are not retried. This allows HTTP-specific retry logic
// to be layered on top of a base retrier (e.g., ExponentialBackoffRetrier).
//
// This is an internal type used automatically by the rawClient.do method.
type httpRetrier struct {
	// wrapped is the underlying retrier that provides the retry logic
	// (backoff timing, max attempts, etc.)
	wrapped retry.Retrier
}

// newHTTPRetrier creates a new httpRetrier that wraps the given retrier.
func newHTTPRetrier(wrapped retry.Retrier) *httpRetrier {
	if wrapped == nil {
		wrapped = &retry.NoopRetrier{}
	}
	return &httpRetrier{
		wrapped: wrapped,
	}
}

// ShouldRetry determines if an error should be retried.
// Returns true only if:
//   - The error is a net.Error with Timeout() (network errors, timeouts, etc.)
//   - The error is a server unavailable error (ErrServerUnavailable) with retryable operation/status code
//
// For network errors, checks Timeout() before delegating to the wrapped retrier.
// For server unavailable errors, checks HTTP method and status code:
//   - POST operations are not retried on 5xx because request body is consumed
//   - GET and DELETE operations can be retried on 5xx (they are idempotent)
//   - HTTP 429 (Too Many Requests) can be retried for all operations
//
// Max attempts are handled by retry.Do, not by this method.
// All other errors are not retried.
func (r *httpRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if err == nil {
		return false
	}

	if r.isTemporaryNetworkError(err) {
		return r.wrapped.ShouldRetry(ctx, err, attempt)
	}

	if !errors.Is(err, ErrServerUnavailable) {
		return false
	}

	// Don't retry on context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return r.isRetryableOperation(r.extractOperation(err), r.extractStatusCode(err))
}

// isTemporaryNetworkError checks if an error is a temporary network error that should be retried.
// It checks for net.Error with Timeout(), including errors wrapped in url.Error.
func (r *httpRetrier) isTemporaryNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	// Check if wrapped in url.Error
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil && errors.As(urlErr.Err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

// extractOperation extracts the HTTP method from the error chain.
func (r *httpRetrier) extractOperation(err error) string {
	var serverErr *ServerUnavailableError
	if errors.As(err, &serverErr) && serverErr.Operation != "" {
		return serverErr.Operation
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Op != "" {
		return urlErr.Op
	}
	return ""
}

// extractStatusCode extracts the status code from the error chain.
func (r *httpRetrier) extractStatusCode(err error) int {
	var serverErr *ServerUnavailableError
	if errors.As(err, &serverErr) {
		return serverErr.StatusCode
	}
	return 0
}

// isRetryableOperation determines if an operation should be retried based on HTTP method and status code.
func (r *httpRetrier) isRetryableOperation(operation string, statusCode int) bool {
	// Network errors (no status code) are always retryable
	if statusCode == 0 {
		return true
	}
	// HTTP 429 (Too Many Requests) can be retried for all operations
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	// Check for specific 5xx status codes
	switch statusCode {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		// POST operations cannot be retried on 5xx because request body is consumed
		// GET and DELETE operations can be retried on 5xx (they are idempotent)
		return operation == http.MethodGet || operation == http.MethodDelete
	default:
		return false
	}
}

// Wait waits before the next retry attempt by delegating to the wrapped retrier.
func (r *httpRetrier) Wait(ctx context.Context, attempt int) error {
	return r.wrapped.Wait(ctx, attempt)
}

// MaxAttempts returns the maximum number of attempts by delegating to the wrapped retrier.
func (r *httpRetrier) MaxAttempts() int {
	return r.wrapped.MaxAttempts()
}
