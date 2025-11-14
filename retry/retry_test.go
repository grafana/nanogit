package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/nanogit/protocol"
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
			return "", protocol.NewServerUnavailableError(500, errors.New("server error"))
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
		return "", protocol.NewServerUnavailableError(500, errors.New("server error"))
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

	ctx, cancel := context.WithCancel(context.Background())
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(100 * time.Millisecond)
	ctx = ToContext(ctx, retrier)

	attempts := 0
	cancel() // Cancel immediately

	result, err := Do(ctx, func() (string, error) {
		attempts++
		return "", protocol.NewServerUnavailableError(500, errors.New("server error"))
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
	require.Equal(t, "", result)
	require.Equal(t, 1, attempts) // Should stop after first attempt
}

func TestDo_ContextCancellationDuringWait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	retrier := NewExponentialBackoffRetrier().
		WithMaxAttempts(3).
		WithInitialDelay(100 * time.Millisecond)
	ctx = ToContext(ctx, retrier)

	attempts := 0
	result, err := Do(ctx, func() (string, error) {
		attempts++
		if attempts == 1 {
			// Cancel during wait
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()
			return "", protocol.NewServerUnavailableError(500, errors.New("server error"))
		}
		return "success", nil
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "context cancelled")
	require.Equal(t, "", result)
	require.Equal(t, 1, attempts) // Should stop after cancellation
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
			return protocol.NewServerUnavailableError(500, errors.New("server error"))
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

	originalErr := protocol.NewServerUnavailableError(500, errors.New("server error"))
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
		return "", protocol.NewServerUnavailableError(500, errors.New("server error"))
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
		return "", protocol.NewServerUnavailableError(500, errors.New("server error"))
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "max retry attempts")
	require.Equal(t, "", result)
	require.LessOrEqual(t, attempts, 3) // Should use default of 3
}

