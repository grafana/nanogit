package client

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPRetrier(t *testing.T) {
	wrapped := newTestRetrier(3)
	retrier := NewHTTPRetrier(wrapped)

	require.NotNil(t, retrier)
	require.Equal(t, 3, retrier.MaxAttempts())
}

func TestHTTPRetrier_ShouldRetry(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		attempt           int
		wrappedShouldRetry bool
		expectedResult    bool
		expectedCalls     int
	}{
		{
			name:              "nil error returns false",
			err:               nil,
			attempt:           1,
			wrappedShouldRetry: true,
			expectedResult:    false,
			expectedCalls:     0, // Should not call wrapped retrier for nil errors
		},
		{
			name:              "ServerUnavailableError delegates to wrapped retrier",
			err:               NewServerUnavailableError(500, errors.New("server error")),
			attempt:           1,
			wrappedShouldRetry: true,
			expectedResult:    true,
			expectedCalls:     1,
		},
		{
			name:              "ServerUnavailableError respects wrapped retrier decision",
			err:               NewServerUnavailableError(500, errors.New("server error")),
			attempt:           1,
			wrappedShouldRetry: false,
			expectedResult:    false,
			expectedCalls:     1,
		},
		{
			name:              "network error delegates to wrapped retrier",
			err:               &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection refused")},
			attempt:           1,
			wrappedShouldRetry: true,
			expectedResult:    true,
			expectedCalls:     1,
		},
		{
			name:              "network error respects wrapped retrier decision",
			err:               &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection refused")},
			attempt:           1,
			wrappedShouldRetry: false,
			expectedResult:    false,
			expectedCalls:     1,
		},
		{
			name:              "other error delegates to wrapped retrier",
			err:               errors.New("some other error"),
			attempt:           1,
			wrappedShouldRetry: true,
			expectedResult:    true,
			expectedCalls:     1,
		},
		{
			name:              "other error respects wrapped retrier decision",
			err:               errors.New("some other error"),
			attempt:           1,
			wrappedShouldRetry: false,
			expectedResult:    false,
			expectedCalls:     1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			wrapped := newTestRetrier(3)
			wrapped.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
				return tt.wrappedShouldRetry
			}

			retrier := NewHTTPRetrier(wrapped)
			ctx := context.Background()

			result := retrier.ShouldRetry(ctx, tt.err, tt.attempt)

			require.Equal(t, tt.expectedResult, result)
			require.Equal(t, tt.expectedCalls, wrapped.ShouldRetryCallCount())
		})
	}
}

func TestHTTPRetrier_ShouldRetry_WithWrappedError(t *testing.T) {
	// Test that wrapped ServerUnavailableError is detected
	wrappedErr := NewServerUnavailableError(500, errors.New("server error"))
	wrapped := newTestRetrier(3)
	wrapped.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
		return errors.Is(err, ErrServerUnavailable)
	}

	retrier := NewHTTPRetrier(wrapped)
	ctx := context.Background()

	result := retrier.ShouldRetry(ctx, wrappedErr, 1)
	require.True(t, result)
	require.Equal(t, 1, wrapped.ShouldRetryCallCount())
}

func TestHTTPRetrier_Wait(t *testing.T) {
	wrapped := newTestRetrier(3)
	wrapped.waitFunc = func(ctx context.Context, attempt int) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	retrier := NewHTTPRetrier(wrapped)
	ctx := context.Background()

	err := retrier.Wait(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, wrapped.WaitCallCount())
}

func TestHTTPRetrier_Wait_DelegatesError(t *testing.T) {
	expectedErr := errors.New("wait error")
	wrapped := newTestRetrier(3)
	wrapped.waitFunc = func(ctx context.Context, attempt int) error {
		return expectedErr
	}

	retrier := NewHTTPRetrier(wrapped)
	ctx := context.Background()

	err := retrier.Wait(ctx, 1)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 1, wrapped.WaitCallCount())
}

func TestHTTPRetrier_MaxAttempts(t *testing.T) {
	tests := []struct {
		name         string
		maxAttempts  int
		expected     int
	}{
		{
			name:        "returns wrapped retrier max attempts",
			maxAttempts: 3,
			expected:    3,
		},
		{
			name:        "returns wrapped retrier max attempts for different value",
			maxAttempts: 5,
			expected:    5,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			wrapped := newTestRetrier(tt.maxAttempts)
			retrier := NewHTTPRetrier(wrapped)

			result := retrier.MaxAttempts()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHTTPRetrier_Integration(t *testing.T) {
	// Integration test to verify the retrier works end-to-end
	wrapped := newTestRetrier(3)
	wrapped.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
		// Retry on ServerUnavailableError or network errors
		if errors.Is(err, ErrServerUnavailable) {
			return attempt < 3
		}
		var netErr *net.OpError
		if errors.As(err, &netErr) {
			return attempt < 3
		}
		return false
	}

	retrier := NewHTTPRetrier(wrapped)
	ctx := context.Background()

	// Test retry flow with ServerUnavailableError
	err := NewServerUnavailableError(500, errors.New("server error"))
	attempt := 1
	for attempt <= 3 {
		shouldRetry := retrier.ShouldRetry(ctx, err, attempt)
		if attempt < 3 {
			require.True(t, shouldRetry, "should retry on attempt %d", attempt)
			waitErr := retrier.Wait(ctx, attempt)
			require.NoError(t, waitErr)
		} else {
			require.False(t, shouldRetry, "should not retry on attempt %d", attempt)
		}
		attempt++
	}

	require.Equal(t, 3, wrapped.ShouldRetryCallCount())
	require.Equal(t, 2, wrapped.WaitCallCount())
	require.Equal(t, 3, retrier.MaxAttempts())
}

