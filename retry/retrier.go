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
	"math/rand/v2"
	"net"
	"sync"
	"time"
)

// Retrier defines the interface for retry behavior.
// Implementations determine when to retry and how long to wait between attempts.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header ../internal/tools/fake_header.txt -o ../mocks/retrier.go . Retrier
type Retrier interface {
	// ShouldRetry determines if an error should be retried.
	// ctx is the context for the operation (may be used for context-aware decisions).
	// attempt is the current attempt number (1-indexed).
	// Returns true if the error should be retried, false otherwise.
	ShouldRetry(ctx context.Context, err error, attempt int) bool

	// Wait waits before the next retry attempt.
	// attempt is the current attempt number (1-indexed).
	// Returns an error if the context was cancelled or its deadline was exceeded during the wait.
	// Specifically, may return context.Canceled or context.DeadlineExceeded.
	Wait(ctx context.Context, attempt int) error

	// MaxAttempts returns the maximum number of attempts (including the initial attempt).
	// Returns 0 for unlimited attempts (not recommended).
	MaxAttempts() int
}

// RetryAfter is implemented by errors that carry a server-requested delay before
// the next attempt, such as an HTTP Retry-After header.
type RetryAfter interface {
	// RetryAfterDelay returns the requested delay and whether one was provided.
	RetryAfterDelay() (time.Duration, bool)
}

// ErrorAwareWaiter is an optional Retrier extension. When a retrier implements
// it, Do calls WaitAfter instead of Wait so the wait can depend on the error
// that triggered the retry.
type ErrorAwareWaiter interface {
	WaitAfter(ctx context.Context, attempt int, err error) error
}

// RetryAfterFrom returns the server-requested delay carried by err, if any.
func RetryAfterFrom(err error) (time.Duration, bool) {
	var retryAfter RetryAfter
	if errors.As(err, &retryAfter) {
		return retryAfter.RetryAfterDelay()
	}
	return 0, false
}

// NoopRetrier is a retrier that never retries.
// This is the default retrier used when none is provided in the context.
type NoopRetrier struct{}

// ShouldRetry always returns false for NoopRetrier.
func (r *NoopRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
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

// retryAfterJitterFraction is how much jitter is added on top of a
// server-requested Retry-After delay to desynchronize clients.
const retryAfterJitterFraction = 0.1

// ExponentialBackoffRetrier implements exponential backoff retry logic.
// It retries on temporary network errors (timeouts, connection failures, etc.).
// It does not retry on context cancellation or other errors.
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

	// rng is a random number generator for jitter calculation.
	// Protected by rngMu for concurrent access.
	rng   *rand.Rand
	rngMu sync.Mutex
}

// NewExponentialBackoffRetrier creates a new ExponentialBackoffRetrier with default values.
func NewExponentialBackoffRetrier() *ExponentialBackoffRetrier {
	return &ExponentialBackoffRetrier{
		MaxAttemptsValue: 3,
		InitialDelay:     100 * time.Millisecond,
		MaxDelay:         5 * time.Second,
		Multiplier:       2.0,
		Jitter:           true,
		rng:              rand.New(rand.NewPCG(0, 0)), // Concurrency-safe per-instance generator
	}
}

// ShouldRetry determines if an error should be retried.
// Returns true for temporary network errors (timeouts, connection failures, etc.).
// Returns false for all other errors.
//
// Returns true for:
//   - Network errors with Timeout() (net.Error)
//
// Returns false for:
//   - Context cancellation errors
//   - Non-network errors
//   - Network errors without Timeout()
//
// Max attempts are handled by retry.Do, not by this method.
func (r *ExponentialBackoffRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if err == nil {
		return false
	}

	// Don't retry on context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for temporary network errors (timeouts)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// Don't retry on other errors
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
	// Using a per-instance random number generator protected by mutex for concurrency safety.
	// Predictability is acceptable here as jitter is only used to prevent
	// thundering herd problems, not for security.
	if r.Jitter {
		delay = delay*0.5 + r.randomFraction()*delay*0.5 // Add 50% jitter
	}

	return r.sleep(ctx, time.Duration(delay))
}

// WaitAfter waits before the next retry attempt, honoring a server-requested
// Retry-After delay when err carries one and falling back to exponential
// backoff otherwise. The delay is capped at MaxDelay so a large or hostile
// value cannot stall the caller past its configured ceiling.
func (r *ExponentialBackoffRetrier) WaitAfter(ctx context.Context, attempt int, err error) error {
	delay, ok := RetryAfterFrom(err)
	if !ok || delay <= 0 {
		return r.Wait(ctx, attempt)
	}

	// Cap before adding jitter: a Retry-After near the Duration limit would
	// otherwise overflow negative and skip the wait altogether.
	if delay > r.MaxDelay {
		delay = r.MaxDelay
	}

	// Retry-After is a floor, so jitter goes on top of it rather than around
	// it, bounded by whatever headroom MaxDelay leaves.
	if r.Jitter {
		jitter := time.Duration(r.randomFraction() * retryAfterJitterFraction * float64(delay))
		delay += min(jitter, r.MaxDelay-delay)
	}

	return r.sleep(ctx, delay)
}

// randomFraction returns a value in [0, 1) from the per-instance generator.
func (r *ExponentialBackoffRetrier) randomFraction() float64 {
	r.rngMu.Lock()
	defer r.rngMu.Unlock()
	return r.rng.Float64()
}

// sleep waits for the given duration or until the context is done.
func (r *ExponentialBackoffRetrier) sleep(ctx context.Context, duration time.Duration) error {
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
// If delay is <= 0, the value is ignored and the current delay remains unchanged.
func (r *ExponentialBackoffRetrier) WithInitialDelay(delay time.Duration) *ExponentialBackoffRetrier {
	if delay > 0 {
		r.InitialDelay = delay
	}
	return r
}

// WithMaxDelay sets the maximum delay between retries.
// If delay is <= 0, the value is ignored and the current max delay remains unchanged.
func (r *ExponentialBackoffRetrier) WithMaxDelay(delay time.Duration) *ExponentialBackoffRetrier {
	if delay > 0 {
		r.MaxDelay = delay
	}
	return r
}

// WithMultiplier sets the exponential backoff multiplier.
// If multiplier is <= 0, the value is ignored and the current multiplier remains unchanged.
func (r *ExponentialBackoffRetrier) WithMultiplier(multiplier float64) *ExponentialBackoffRetrier {
	if multiplier > 0 {
		r.Multiplier = multiplier
	}
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
