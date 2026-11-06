package client

import (
	"context"
	"errors"
	"net/http"

	"github.com/grafana/nanogit/retry"
)

// temporaryErrorRetrier wraps another retrier and handles server unavailable errors:
// - Server unavailable errors (ErrServerUnavailable) that are retryable based on HTTP method/status
//
// If the error is a retryable server unavailable error, it always retries. Otherwise, it delegates to the wrapped retrier.
// Network errors and context cancellation are handled by the wrapped retrier (e.g., ExponentialBackoffRetrier).
//
// This is an internal type used automatically by the rawClient.do method.
type temporaryErrorRetrier struct {
	// wrapped is the underlying retrier that provides the retry logic
	// (backoff timing, max attempts, etc.)
	wrapped retry.Retrier
	// replayableBody reports whether the request body can be sent again in
	// full, which gates retrying rate-limited requests that carry one.
	replayableBody bool
}

// newTemporaryErrorRetrier creates a new temporaryErrorRetrier that wraps the given retrier.
// replayableBody must report whether the request can be re-sent with its full body.
func newTemporaryErrorRetrier(wrapped retry.Retrier, replayableBody bool) *temporaryErrorRetrier {
	if wrapped == nil {
		wrapped = &retry.NoopRetrier{}
	}
	return &temporaryErrorRetrier{
		wrapped:        wrapped,
		replayableBody: replayableBody,
	}
}

// requestReplayable reports whether req can be sent again with its full body.
// A streamed body (io.Pipe, an arbitrary io.Reader) has no GetBody, so net/http
// cannot rewind it: a second attempt would send a truncated or empty payload.
func requestReplayable(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

// ShouldRetry determines if an error should be retried.
// If the error is a retryable server unavailable error, returns true (always retry).
// Otherwise, delegates to the wrapped retrier (which handles network errors and context cancellation).
//
// A server unavailable error is retryable if:
//   - POST operations are not retried on 5xx because request body is consumed
//   - GET and DELETE operations can be retried on 5xx (they are idempotent)
//   - a rate-limit rejection can be retried for any operation whose body can be
//     replayed, whatever status carried it
//
// Max attempts are handled by retry.Do, not by this method.
func (r *temporaryErrorRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if err == nil {
		return false
	}

	// Check for server unavailable errors that are retryable
	var serverErr *ServerUnavailableError
	if errors.As(err, &serverErr) {
		// Rate limiting is rejected before the body is consumed server-side, so
		// it is retryable for any method — but only when the body can be
		// replayed client-side. Keyed off the classification rather than the
		// status, so a detector-supplied signal on any status is honored.
		if serverErr.RateLimited {
			return r.replayableBody
		}
		if r.isRetryableOperation(serverErr.Operation, serverErr.StatusCode) {
			return true
		}
		// Server unavailable but not retryable, delegate to wrapped retrier.
		// The retrier is retrieved from the context. If no retrier is found, a NoopRetrier is used (no retries will be performed).
		return r.wrapped.ShouldRetry(ctx, err, attempt)
	}

	// Not a server unavailable error, delegate to wrapped retrier (handles network errors, context cancellation, etc.).
	// The retrier is retrieved from the context. If no retrier is found, a NoopRetrier is used (no retries will be performed).
	return r.wrapped.ShouldRetry(ctx, err, attempt)
}

// isRetryableOperation determines if a server error should be retried based on
// HTTP method and status code. Rate limiting is handled by ShouldRetry.
func (r *temporaryErrorRetrier) isRetryableOperation(operation string, statusCode int) bool {
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

// WaitAfter delegates to the wrapped retrier's error-aware wait when it has one,
// so a server's Retry-After is not lost to the wrapping.
func (r *temporaryErrorRetrier) WaitAfter(ctx context.Context, attempt int, err error) error {
	if waiter, ok := r.wrapped.(retry.ErrorAwareWaiter); ok {
		return waiter.WaitAfter(ctx, attempt, err)
	}
	return r.wrapped.Wait(ctx, attempt)
}

// MaxAttempts returns the maximum number of attempts by delegating to the wrapped retrier.
func (r *temporaryErrorRetrier) MaxAttempts() int {
	return r.wrapped.MaxAttempts()
}
