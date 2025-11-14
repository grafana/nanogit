package client

import (
	"context"
	"errors"

	"github.com/grafana/nanogit/retry"
)

// httpRetrier wraps a generic retrier and adds HTTP-specific retry logic.
// It retries on:
//   - Network errors (handled by the wrapped retrier)
//   - 5xx server errors (ServerUnavailableError)
type httpRetrier struct {
	retrier retry.Retrier
}

// NewHTTPRetrier creates a new HTTP retrier that wraps a generic retrier.
func NewHTTPRetrier(retrier retry.Retrier) retry.Retrier {
	return &httpRetrier{retrier: retrier}
}

// ShouldRetry determines if an error should be retried.
// It retries on network errors (via wrapped retrier) and 5xx server errors.
func (r *httpRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if err == nil {
		return false
	}

	// Retry on server unavailable errors (5xx)
	if errors.Is(err, ErrServerUnavailable) {
		return r.retrier.ShouldRetry(ctx, err, attempt)
	}

	// Delegate to wrapped retrier for network errors
	return r.retrier.ShouldRetry(ctx, err, attempt)
}

// Wait waits before the next retry attempt using the wrapped retrier.
func (r *httpRetrier) Wait(ctx context.Context, attempt int) error {
	return r.retrier.Wait(ctx, attempt)
}

// MaxAttempts returns the maximum number of attempts from the wrapped retrier.
func (r *httpRetrier) MaxAttempts() int {
	return r.retrier.MaxAttempts()
}


