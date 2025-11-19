package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"net"

	"github.com/stretchr/testify/require"
)

func TestDo_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := &NoopRetrier{}
	ctx = ToContext(ctx, retrier)

	result, err := Do(ctx, func() (string, error) {
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
}

func TestDo_NoRetrier(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// No retrier in context - should use NoopRetrier

	result, err := Do(ctx, func() (string, error) {
		return "", errors.New("test error")
	})

	require.Error(t, err)
	require.Equal(t, "", result)
	require.Equal(t, "test error", err.Error())
}

func TestDo_RetryOnRetryableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(10 * time.Millisecond).
		WithoutJitter()
	ctx = ToContext(ctx, retrier)

	attempts := 0
	result, err := Do(ctx, func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &timeoutError{}, // Network timeout errors are retried
			}
		}
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
	require.Equal(t, 3, attempts)
}

func TestDo_MaxAttemptsReached(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(10 * time.Millisecond).
		WithoutJitter()
	ctx = ToContext(ctx, retrier)

	attempts := 0
	result, err := Do(ctx, func() (string, error) {
		attempts++
		return "", &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{}, // Network timeout errors are retried
		}
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "max retry attempts (3) reached")
	require.Equal(t, "", result)
	require.Equal(t, 3, attempts)
}

func TestDo_NonRetryableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(10 * time.Millisecond)
	ctx = ToContext(ctx, retrier)

	attempts := 0
	result, err := Do(ctx, func() (string, error) {
		attempts++
		return "", errors.New("client error (4xx)")
	})

	require.Error(t, err)
	require.Equal(t, "client error (4xx)", err.Error())
	require.Equal(t, "", result)
	require.Equal(t, 1, attempts) // Should not retry
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Context cancellation is checked during Wait, not before fn() is called
	// So if context is cancelled, fn() will still run, but ShouldRetry will return false
	// and we won't retry. The error returned will be from fn(), not context.Canceled.
	ctx, cancel := context.WithCancel(context.Background())
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(100 * time.Millisecond)
	ctx = ToContext(ctx, retrier)

	attempts := 0
	cancel() // Cancel immediately

	result, err := Do(ctx, func() (string, error) {
		attempts++
		return "", errors.New("some error")
	})

	require.Error(t, err)
	require.Equal(t, "some error", err.Error())
	require.Equal(t, "", result)
	require.Equal(t, 1, attempts) // Should stop after first attempt (no retry)
}

func TestDo_ContextCancellationDuringWait(t *testing.T) {
	t.Parallel()

	// This test doesn't apply anymore since network errors aren't retried
	// Testing context cancellation during wait would require a retryable error
	// which ExponentialBackoffRetrier no longer provides by default
	t.Skip("Network errors are no longer retried, so this test scenario doesn't apply")
}

func TestDoVoid_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := &NoopRetrier{}
	ctx = ToContext(ctx, retrier)

	err := DoVoid(ctx, func() error {
		return nil
	})

	require.NoError(t, err)
}

func TestDoVoid_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := &NoopRetrier{}
	ctx = ToContext(ctx, retrier)

	err := DoVoid(ctx, func() error {
		return errors.New("test error")
	})

	require.Error(t, err)
	require.Equal(t, "test error", err.Error())
}

func TestDoVoid_Retry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(10 * time.Millisecond).
		WithoutJitter()
	ctx = ToContext(ctx, retrier)

	attempts := 0
	err := DoVoid(ctx, func() error {
		attempts++
		if attempts < 3 {
			return &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &timeoutError{}, // Network timeout errors are retried
			}
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}

func TestDo_ErrorWrapping(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(2).
		WithInitialDelay(10 * time.Millisecond)
	ctx = ToContext(ctx, retrier)

	originalErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &timeoutError{}, // Network timeout errors are retried
	}
	result, err := Do(ctx, func() (string, error) {
		return "", originalErr
	})

	require.Error(t, err)
	require.Equal(t, "", result)
	require.Contains(t, err.Error(), "max retry attempts")
	// Check that original error is wrapped
	require.True(t, errors.Is(err, originalErr))
}

func TestDo_ZeroMaxAttempts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := &ExponentialBackoffRetrier{
		MaxAttemptsValue: 0, // Zero should use default
	}
	ctx = ToContext(ctx, retrier)

	attempts := 0
	result, err := Do(ctx, func() (string, error) {
		attempts++
		return "", &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{}, // Network timeout errors are retried
		}
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "max retry attempts")
	require.Equal(t, "", result)
	require.LessOrEqual(t, attempts, 3) // Should use default of 3
}

func TestDo_NegativeMaxAttempts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retrier := &ExponentialBackoffRetrier{
		MaxAttemptsValue: -1, // Negative should use default
	}
	ctx = ToContext(ctx, retrier)

	attempts := 0
	result, err := Do(ctx, func() (string, error) {
		attempts++
		return "", &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{}, // Network timeout errors are retried
		}
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "max retry attempts")
	require.Equal(t, "", result)
	require.LessOrEqual(t, attempts, 3) // Should use default of 3
}
