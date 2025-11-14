// Package retry provides a pluggable retry mechanism for HTTP requests.
// It follows the same pattern as storage options, using context-based injection.
//
// The retry mechanism is designed to make HTTP operations more robust against
// transient network errors and server issues. By default, no retries are performed
// (backward compatible). Users can enable retries by injecting a retrier into the context.
//
// Example usage:
//
//	retrier := retry.NewExponentialBackoffRetrier().
//	    WithMaxAttempts(3).
//	    WithInitialDelay(100 * time.Millisecond)
//	ctx = retry.ToContext(ctx, retrier)
//	// All HTTP operations will now use retry logic
//
// Custom retriers can be implemented by implementing the Retrier interface.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/grafana/nanogit/protocol"
)

// Retrier defines the interface for retry behavior.
// Implementations determine when to retry and how long to wait between attempts.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ../mocks/retrier.go . Retrier
type Retrier interface {
	// ShouldRetry determines if an error should be retried.
	// attempt is the current attempt number (1-indexed).
	// Returns true if the error should be retried, false otherwise.
	ShouldRetry(err error, attempt int) bool

	// Wait waits before the next retry attempt.
	// attempt is the current attempt number (1-indexed).
	// Returns an error if the context was cancelled during the wait.
	Wait(ctx context.Context, attempt int) error

	// MaxAttempts returns the maximum number of attempts (including the initial attempt).
	// Returns 0 for unlimited attempts (not recommended).
	MaxAttempts() int
}

// NoopRetrier is a retrier that never retries.
// This is the default retrier used when none is provided in the context.
type NoopRetrier struct{}

// ShouldRetry always returns false for NoopRetrier.
func (r *NoopRetrier) ShouldRetry(err error, attempt int) bool {
	return false
}

// Wait is a no-op for NoopRetrier.
func (r *NoopRetrier) Wait(ctx context.Context, attempt int) error {
	return nil
}

// MaxAttempts returns 1 for NoopRetrier (no retries).
func (r *NoopRetrier) MaxAttempts() int {
	return 1
}

// ExponentialBackoffRetrier implements exponential backoff retry logic.
// It retries on network errors, timeouts, and 5xx status codes.
// It does not retry on 4xx client errors or context cancellation.
type ExponentialBackoffRetrier struct {
	// MaxAttempts is the maximum number of attempts (including the initial attempt).
	// Default is 3.
	MaxAttemptsValue int

	// InitialDelay is the initial delay before the first retry.
	// Default is 100ms.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Default is 5 seconds.
	MaxDelay time.Duration

	// Multiplier is the exponential backoff multiplier.
	// Default is 2.0.
	Multiplier float64

	// Jitter enables random jitter to prevent thundering herd.
	// Default is true.
	Jitter bool
}

// NewExponentialBackoffRetrier creates a new ExponentialBackoffRetrier with default values.
func NewExponentialBackoffRetrier() *ExponentialBackoffRetrier {
	return &ExponentialBackoffRetrier{
		MaxAttemptsValue: 3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		Multiplier:        2.0,
		Jitter:            true,
	}
}

// ShouldRetry determines if an error should be retried.
// Returns true for:
//   - Network errors (connection refused, timeouts, etc.)
//   - 5xx server errors (ServerUnavailableError)
//   - Temporary errors
//
// Returns false for:
//   - 4xx client errors
//   - Context cancellation errors
//   - Errors that should not be retried
func (r *ExponentialBackoffRetrier) ShouldRetry(err error, attempt int) bool {
	if err == nil {
		return false
	}

	// Don't retry if we've exceeded max attempts
	// With maxAttempts=3, we allow attempts 1, 2, and 3, so retry if attempt <= maxAttempts
	maxAttempts := r.MaxAttempts()
	if maxAttempts > 0 && attempt > maxAttempts {
		return false
	}

	// Don't retry on context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Retry on server unavailable errors (5xx)
	if errors.Is(err, protocol.ErrServerUnavailable) {
		return true
	}

	// Check for network errors
	var netErr interface {
		Error() string
		Timeout() bool
		Temporary() bool
	}
	if errors.As(err, &netErr) {
		// Retry on timeouts and temporary network errors
		if netErr.Timeout() || netErr.Temporary() {
			return true
		}
		// Retry on other network errors (connection refused, etc.)
		return true
	}

	// Don't retry on other errors (4xx, etc.)
	return false
}

// Wait waits before the next retry attempt using exponential backoff.
func (r *ExponentialBackoffRetrier) Wait(ctx context.Context, attempt int) error {
	// Calculate delay: initialDelay * (multiplier ^ (attempt - 1))
	delay := float64(r.InitialDelay) * math.Pow(r.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(r.MaxDelay) {
		delay = float64(r.MaxDelay)
	}

	// Add jitter if enabled (random value between 0 and delay)
	if r.Jitter {
		jitter := rand.Float64() * delay
		delay = delay*0.5 + jitter*0.5 // Add 50% jitter
	}

	// Convert to duration
	duration := time.Duration(delay)

	// Wait with context cancellation support
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// MaxAttempts returns the maximum number of attempts.
func (r *ExponentialBackoffRetrier) MaxAttempts() int {
	if r.MaxAttemptsValue <= 0 {
		return 3 // Default
	}
	return r.MaxAttemptsValue
}

// WithMaxAttempts sets the maximum number of attempts.
func (r *ExponentialBackoffRetrier) WithMaxAttempts(attempts int) *ExponentialBackoffRetrier {
	r.MaxAttemptsValue = attempts
	return r
}

// WithInitialDelay sets the initial delay before the first retry.
func (r *ExponentialBackoffRetrier) WithInitialDelay(delay time.Duration) *ExponentialBackoffRetrier {
	r.InitialDelay = delay
	return r
}

// WithMaxDelay sets the maximum delay between retries.
func (r *ExponentialBackoffRetrier) WithMaxDelay(delay time.Duration) *ExponentialBackoffRetrier {
	r.MaxDelay = delay
	return r
}

// WithMultiplier sets the exponential backoff multiplier.
func (r *ExponentialBackoffRetrier) WithMultiplier(multiplier float64) *ExponentialBackoffRetrier {
	r.Multiplier = multiplier
	return r
}

// WithJitter enables jitter to prevent thundering herd problems.
func (r *ExponentialBackoffRetrier) WithJitter() *ExponentialBackoffRetrier {
	r.Jitter = true
	return r
}

// WithoutJitter disables jitter.
func (r *ExponentialBackoffRetrier) WithoutJitter() *ExponentialBackoffRetrier {
	r.Jitter = false
	return r
}

