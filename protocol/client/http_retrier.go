package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/retry"
)

// HTTPRetrier wraps another retrier and only retries on HTTP-specific errors:
// - Network errors (net.Error)
// - Server unavailable errors (ErrServerUnavailable)
//
// All other errors are not retried. This allows HTTP-specific retry logic
// to be layered on top of a base retrier (e.g., ExponentialBackoffRetrier).
//
// Example usage:
//
//	baseRetrier := retry.NewExponentialBackoffRetrier().
//	    WithMaxAttempts(3).
//	    WithInitialDelay(100 * time.Millisecond)
//	httpRetrier := NewHTTPRetrier(baseRetrier)
//	ctx = retry.ToContext(ctx, httpRetrier)
type HTTPRetrier struct {
	// wrapped is the underlying retrier that provides the retry logic
	// (backoff timing, max attempts, etc.)
	wrapped retry.Retrier
}

// NewHTTPRetrier creates a new HTTPRetrier that wraps the given retrier.
func NewHTTPRetrier(wrapped retry.Retrier) *HTTPRetrier {
	if wrapped == nil {
		wrapped = &retry.NoopRetrier{}
	}
	return &HTTPRetrier{
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
func (r *HTTPRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if err == nil {
		return false
	}

	// Check for network errors - only retry if Timeout() or Temporary()
	if r.isTemporaryNetworkError(err) {
		// Delegate to wrapped retrier for temporary network errors
		return r.wrapped.ShouldRetry(ctx, err, attempt)
	}

	// Check for server unavailable errors
	if errors.Is(err, ErrServerUnavailable) {
		// Don't retry on context cancellation
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}

		// Extract operation and status code from error
		operation := r.extractOperation(err)
		statusCode := r.extractStatusCode(err)

		// Check if operation and status code indicate retryability
		if !r.isRetryableOperation(operation, statusCode) {
			return false
		}

		return true
	}

	// Don't retry on other errors (4xx client errors, etc.)
	return false
}

// isTemporaryNetworkError checks if an error is a temporary network error that should be retried.
// It checks for net.Error with Timeout(), including errors wrapped in url.Error.
// Temporary() is deprecated and most temporary errors are timeouts, so we only check Timeout().
func (r *HTTPRetrier) isTemporaryNetworkError(err error) bool {
	// Check for net.Error directly
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Retry on timeouts (most temporary errors are timeouts)
		if netErr.Timeout() {
			return true
		}
	}

	// Check for url.Error and unwrap to check underlying net.Error
	// Most errors from http.Client are wrapped in *url.Error
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Err != nil {
			var underlyingNetErr net.Error
			if errors.As(urlErr.Err, &underlyingNetErr) {
				// Retry on timeouts (most temporary errors are timeouts)
				if underlyingNetErr.Timeout() {
					return true
				}
			}
		}
	}

	return false
}

// extractOperation extracts the HTTP method from the error chain.
// It checks ServerUnavailableError first, then tries to extract from url.Error.
func (r *HTTPRetrier) extractOperation(err error) string {
	// Check for ServerUnavailableError with operation
	var serverErr *ServerUnavailableError
	if errors.As(err, &serverErr) && serverErr.Operation != "" {
		return serverErr.Operation
	}

	// Try to extract from url.Error
	// url.Error.Op contains the HTTP method (GET, POST, etc.)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Op != "" {
			return urlErr.Op
		}
	}

	return ""
}

// extractStatusCode extracts the status code from the error chain.
func (r *HTTPRetrier) extractStatusCode(err error) int {
	var serverErr *ServerUnavailableError
	if errors.As(err, &serverErr) {
		return serverErr.StatusCode
	}
	return 0
}

// isRetryableOperation determines if an operation should be retried based on HTTP method and status code.
// POST operations cannot be retried on 5xx because request body is consumed.
// GET and DELETE operations can be retried on 5xx (they are idempotent).
// HTTP 429 (Too Many Requests) can be retried for all operations.
// Network errors (statusCode == 0) are always retryable.
func (r *HTTPRetrier) isRetryableOperation(operation string, statusCode int) bool {
	// Network errors (no status code) are always retryable
	if statusCode == 0 {
		return true
	}

	// HTTP 429 (Too Many Requests) can be retried for all operations
	// Rate limiting is temporary and doesn't consume request body
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
		if operation == http.MethodPost {
			return false
		}

		// GET and DELETE operations can be retried on 5xx (they are idempotent)
		if operation == http.MethodGet || operation == http.MethodDelete {
			return true
		}

		// For unknown operations, be conservative and don't retry on 5xx
		// (they might be POST operations)
		return false
	default:
		// Don't retry on other status codes
		return false
	}
}

// Wait waits before the next retry attempt by delegating to the wrapped retrier.
func (r *HTTPRetrier) Wait(ctx context.Context, attempt int) error {
	return r.wrapped.Wait(ctx, attempt)
}

// MaxAttempts returns the maximum number of attempts by delegating to the wrapped retrier.
func (r *HTTPRetrier) MaxAttempts() int {
	return r.wrapped.MaxAttempts()
}
