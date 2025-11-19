package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/retry"
)

// temporaryErrorRetrier wraps another retrier and only retries on temporary errors:
// - Network errors with Timeout() (net.Error)
// - Server unavailable errors (ErrServerUnavailable) that are retryable based on HTTP method/status
//
// If the error is not temporary, it won't retry at all. Otherwise, it delegates to the base retrier.
// This allows temporary error filtering to be layered on top of a base retrier (e.g., ExponentialBackoffRetrier).
//
// This is an internal type used automatically by the rawClient.do method.
type temporaryErrorRetrier struct {
	// wrapped is the underlying retrier that provides the retry logic
	// (backoff timing, max attempts, etc.)
	wrapped retry.Retrier
}

// newTemporaryErrorRetrier creates a new temporaryErrorRetrier that wraps the given retrier.
func newTemporaryErrorRetrier(wrapped retry.Retrier) *temporaryErrorRetrier {
	if wrapped == nil {
		wrapped = &retry.NoopRetrier{}
	}
	return &temporaryErrorRetrier{
		wrapped: wrapped,
	}
}

// ShouldRetry determines if an error should be retried.
// Returns false if the error is not temporary. Otherwise, delegates to the wrapped retrier.
//
// An error is considered temporary if:
//   - It is a net.Error with Timeout() (network errors, timeouts, etc.)
//   - It is a server unavailable error (ErrServerUnavailable) with retryable operation/status code:
//   - POST operations are not retried on 5xx because request body is consumed
//   - GET and DELETE operations can be retried on 5xx (they are idempotent)
//   - HTTP 429 (Too Many Requests) can be retried for all operations
//
// Max attempts are handled by retry.Do, not by this method.
func (r *temporaryErrorRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if err == nil {
		return false
	}

	// Check if error is temporary
	if !r.isTemporaryError(err) {
		return false
	}

	// Error is temporary, delegate to wrapped retrier
	return r.wrapped.ShouldRetry(ctx, err, attempt)
}

// isTemporaryError checks if an error is temporary and should be considered for retry.
func (r *temporaryErrorRetrier) isTemporaryError(err error) bool {
	// Don't retry on context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for temporary network errors
	if r.isTemporaryNetworkError(err) {
		return true
	}

	// Check for server unavailable errors that are retryable
	if errors.Is(err, ErrServerUnavailable) {
		return r.isRetryableOperation(r.extractOperation(err), r.extractStatusCode(err))
	}

	return false
}

// isTemporaryNetworkError checks if an error is a temporary network error that should be retried.
// It checks for net.Error with Timeout(), including errors wrapped in url.Error.
func (r *temporaryErrorRetrier) isTemporaryNetworkError(err error) bool {
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
func (r *temporaryErrorRetrier) extractOperation(err error) string {
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
func (r *temporaryErrorRetrier) extractStatusCode(err error) int {
	var serverErr *ServerUnavailableError
	if errors.As(err, &serverErr) {
		return serverErr.StatusCode
	}
	return 0
}

// isRetryableOperation determines if an operation should be retried based on HTTP method and status code.
func (r *temporaryErrorRetrier) isRetryableOperation(operation string, statusCode int) bool {
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
func (r *temporaryErrorRetrier) Wait(ctx context.Context, attempt int) error {
	return r.wrapped.Wait(ctx, attempt)
}

// MaxAttempts returns the maximum number of attempts by delegating to the wrapped retrier.
func (r *temporaryErrorRetrier) MaxAttempts() int {
	return r.wrapped.MaxAttempts()
}
