package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNoopRetrier(t *testing.T) {
	t.Parallel()

	retrier := &NoopRetrier{}

	t.Run("ShouldRetry always returns false", func(t *testing.T) {
		ctx := context.Background()
		require.False(t, retrier.ShouldRetry(ctx, errors.New("test error"), 1))
		require.False(t, retrier.ShouldRetry(ctx, nil, 1))
	})

	t.Run("Wait is a no-op", func(t *testing.T) {
		ctx := context.Background()
		err := retrier.Wait(ctx, 1)
		require.NoError(t, err)
	})

	t.Run("MaxAttempts returns 1", func(t *testing.T) {
		require.Equal(t, 1, retrier.MaxAttempts())
	})
}

func TestExponentialBackoffRetrier_ShouldRetry(t *testing.T) {
	t.Parallel()

	retrier := NewExponentialBackoffRetrier()

	t.Run("does not retry on network errors without timeout", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
		require.False(t, retrier.ShouldRetry(ctx, err, 2))
	})

	t.Run("retries on network timeout errors", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: &timeoutError{},
		}
		require.True(t, retrier.ShouldRetry(ctx, err, 1))
		require.True(t, retrier.ShouldRetry(ctx, err, 2))
	})

	t.Run("does not retry on temporary network errors without timeout", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: &temporaryError{},
		}
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on connection refused errors", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on context cancellation", func(t *testing.T) {
		ctx := context.Background()
		err := context.Canceled
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on context deadline exceeded", func(t *testing.T) {
		ctx := context.Background()
		err := context.DeadlineExceeded
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on 4xx client errors", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("got status code 404: Not Found")
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("does not retry on any errors", func(t *testing.T) {
		ctx := context.Background()
		retrier := NewExponentialBackoffRetrier().WithMaxAttempts(3)
		err := errors.New("some error")
		// ShouldRetry should return false for all errors
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
		require.False(t, retrier.ShouldRetry(ctx, err, 2))
		require.False(t, retrier.ShouldRetry(ctx, err, 3))

		// MaxAttempts should still return the correct value
		require.Equal(t, 3, retrier.MaxAttempts())
	})
}

func TestExponentialBackoffRetrier_Wait(t *testing.T) {
	t.Parallel()

	t.Run("waits with exponential backoff", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(10 * time.Millisecond).
			WithMaxDelay(100 * time.Millisecond).
			WithMultiplier(2.0).
			WithoutJitter()

		ctx := context.Background()

		start := time.Now()
		err := retrier.Wait(ctx, 1)
		duration := time.Since(start)
		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, 10*time.Millisecond)
		require.Less(t, duration, 50*time.Millisecond) // Some tolerance

		start = time.Now()
		err = retrier.Wait(ctx, 2)
		duration = time.Since(start)
		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, 20*time.Millisecond)
		require.Less(t, duration, 50*time.Millisecond)
	})

	t.Run("respects max delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(100 * time.Millisecond).
			WithMaxDelay(200 * time.Millisecond).
			WithMultiplier(10.0).
			WithoutJitter()

		ctx := context.Background()

		start := time.Now()
		err := retrier.Wait(ctx, 2) // Should be capped at max delay
		duration := time.Since(start)
		require.NoError(t, err)
		require.Less(t, duration, 300*time.Millisecond)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(1 * time.Second)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := retrier.Wait(ctx, 1)
		require.Error(t, err)
		require.True(t, errors.Is(err, context.Canceled))
	})

	t.Run("respects context deadline", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(1 * time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := retrier.Wait(ctx, 1)
		require.Error(t, err)
		require.True(t, errors.Is(err, context.DeadlineExceeded))
	})
}

func TestExponentialBackoffRetrier_MaxAttempts(t *testing.T) {
	t.Parallel()

	t.Run("default max attempts is 3", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		require.Equal(t, 3, retrier.MaxAttempts())
	})

	t.Run("custom max attempts", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().WithMaxAttempts(5)
		require.Equal(t, 5, retrier.MaxAttempts())
	})

	t.Run("zero max attempts uses default", func(t *testing.T) {
		retrier := &ExponentialBackoffRetrier{
			MaxAttemptsValue: 0,
		}
		require.Equal(t, 3, retrier.MaxAttempts())
	})
}

func TestExponentialBackoffRetrier_Configuration(t *testing.T) {
	t.Parallel()

	t.Run("with max attempts", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().WithMaxAttempts(5)
		require.Equal(t, 5, retrier.MaxAttemptsValue)
	})

	t.Run("with initial delay", func(t *testing.T) {
		delay := 200 * time.Millisecond
		retrier := NewExponentialBackoffRetrier().WithInitialDelay(delay)
		require.Equal(t, delay, retrier.InitialDelay)
	})

	t.Run("with max delay", func(t *testing.T) {
		delay := 10 * time.Second
		retrier := NewExponentialBackoffRetrier().WithMaxDelay(delay)
		require.Equal(t, delay, retrier.MaxDelay)
	})

	t.Run("with multiplier", func(t *testing.T) {
		multiplier := 3.0
		retrier := NewExponentialBackoffRetrier().WithMultiplier(multiplier)
		require.Equal(t, multiplier, retrier.Multiplier)
	})

	t.Run("with jitter", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().WithoutJitter()
		require.False(t, retrier.Jitter)

		retrier = retrier.WithJitter()
		require.True(t, retrier.Jitter)
	})

	t.Run("ignores zero initial delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		originalDelay := retrier.InitialDelay
		retrier.WithInitialDelay(0)
		require.Equal(t, originalDelay, retrier.InitialDelay)
	})

	t.Run("ignores negative initial delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		originalDelay := retrier.InitialDelay
		retrier.WithInitialDelay(-1 * time.Millisecond)
		require.Equal(t, originalDelay, retrier.InitialDelay)
	})

	t.Run("ignores zero max delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		originalMaxDelay := retrier.MaxDelay
		retrier.WithMaxDelay(0)
		require.Equal(t, originalMaxDelay, retrier.MaxDelay)
	})

	t.Run("ignores negative max delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		originalMaxDelay := retrier.MaxDelay
		retrier.WithMaxDelay(-1 * time.Second)
		require.Equal(t, originalMaxDelay, retrier.MaxDelay)
	})

	t.Run("ignores zero multiplier", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		originalMultiplier := retrier.Multiplier
		retrier.WithMultiplier(0)
		require.Equal(t, originalMultiplier, retrier.Multiplier)
	})

	t.Run("ignores negative multiplier", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier()
		originalMultiplier := retrier.Multiplier
		retrier.WithMultiplier(-1.0)
		require.Equal(t, originalMultiplier, retrier.Multiplier)
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

type retryAfterError struct {
	delay time.Duration
}

func (e *retryAfterError) Error() string { return "retry after " + e.delay.String() }

func (e *retryAfterError) RetryAfterDelay() (time.Duration, bool) {
	return e.delay, e.delay > 0
}

func TestExponentialBackoffRetrier_WaitAfter(t *testing.T) {
	t.Parallel()

	t.Run("honors a server-requested delay instead of the backoff schedule", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(time.Millisecond).
			WithMaxDelay(time.Second).
			WithoutJitter()

		start := time.Now()
		err := retrier.WaitAfter(context.Background(), 1, &retryAfterError{delay: 50 * time.Millisecond})
		duration := time.Since(start)
		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, 50*time.Millisecond)
		require.Less(t, duration, 250*time.Millisecond)
	})

	t.Run("caps a server-requested delay at max delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(time.Millisecond).
			WithMaxDelay(20 * time.Millisecond).
			WithoutJitter()

		start := time.Now()
		err := retrier.WaitAfter(context.Background(), 1, &retryAfterError{delay: time.Hour})
		duration := time.Since(start)
		require.NoError(t, err)
		require.Less(t, duration, 500*time.Millisecond)
	})

	t.Run("adds jitter on top of a server-requested delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithMaxDelay(time.Second).
			WithJitter()

		start := time.Now()
		err := retrier.WaitAfter(context.Background(), 1, &retryAfterError{delay: 50 * time.Millisecond})
		duration := time.Since(start)
		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, 50*time.Millisecond)
	})

	t.Run("falls back to backoff when the error carries no delay", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().
			WithInitialDelay(30 * time.Millisecond).
			WithMaxDelay(time.Second).
			WithoutJitter()

		start := time.Now()
		err := retrier.WaitAfter(context.Background(), 1, errors.New("boom"))
		duration := time.Since(start)
		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, 30*time.Millisecond)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		retrier := NewExponentialBackoffRetrier().WithMaxDelay(time.Minute)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := retrier.WaitAfter(ctx, 1, &retryAfterError{delay: 30 * time.Second})
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestRetryAfterFrom(t *testing.T) {
	t.Parallel()

	t.Run("reads the delay from a wrapped error", func(t *testing.T) {
		err := fmt.Errorf("fetching refs: %w", &retryAfterError{delay: 5 * time.Second})

		delay, ok := RetryAfterFrom(err)
		require.True(t, ok)
		require.Equal(t, 5*time.Second, delay)
	})

	t.Run("reports no delay for unrelated errors", func(t *testing.T) {
		delay, ok := RetryAfterFrom(errors.New("boom"))
		require.False(t, ok)
		require.Zero(t, delay)
	})

	t.Run("reports no delay for a nil error", func(t *testing.T) {
		delay, ok := RetryAfterFrom(nil)
		require.False(t, ok)
		require.Zero(t, delay)
	})
}

func TestExponentialBackoffRetrier_WaitAfterOverflow(t *testing.T) {
	t.Parallel()

	// A Retry-After near the Duration limit overflows negative if jitter is
	// added before the cap, which skips the wait instead of applying MaxDelay.
	for _, delay := range []time.Duration{math.MaxInt64, math.MaxInt64 - 1000, 9223372036 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			t.Parallel()
			retrier := NewExponentialBackoffRetrier().
				WithInitialDelay(time.Hour).
				WithMaxDelay(40 * time.Millisecond).
				WithJitter()

			start := time.Now()
			err := retrier.WaitAfter(context.Background(), 1, &retryAfterError{delay: delay})
			elapsed := time.Since(start)
			require.NoError(t, err)
			require.GreaterOrEqual(t, elapsed, 40*time.Millisecond, "must still wait MaxDelay, not return immediately")
			require.Less(t, elapsed, time.Second)
		})
	}
}
