package retry

import (
	"context"
	"errors"
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
		// Network errors should not be retried by NoopRetrier
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.False(t, retrier.ShouldRetry(ctx, err, 1))
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

	t.Run("retries on network errors", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.True(t, retrier.ShouldRetry(ctx, err, 1))
		require.True(t, retrier.ShouldRetry(ctx, err, 2))
	})

	t.Run("retries on network timeout errors", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: &timeoutError{},
		}
		require.True(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on temporary network errors", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: &temporaryError{},
		}
		require.True(t, retrier.ShouldRetry(ctx, err, 1))
	})

	t.Run("retries on connection refused errors", func(t *testing.T) {
		ctx := context.Background()
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.True(t, retrier.ShouldRetry(ctx, err, 1))
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

	t.Run("ShouldRetry does not check max attempts", func(t *testing.T) {
		ctx := context.Background()
		retrier := NewExponentialBackoffRetrier().WithMaxAttempts(3)
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		// ShouldRetry should return true for all attempts - max attempts are handled by retry.Do
		require.True(t, retrier.ShouldRetry(ctx, err, 1))
		require.True(t, retrier.ShouldRetry(ctx, err, 2))
		require.True(t, retrier.ShouldRetry(ctx, err, 3))
		require.True(t, retrier.ShouldRetry(ctx, err, 4))
		require.True(t, retrier.ShouldRetry(ctx, err, 100))
		
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

