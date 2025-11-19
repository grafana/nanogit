package client

import (
	"context"
	"errors"
	"time"
)

// testRetrier is a simple retrier implementation for testing
type testRetrier struct {
	maxAttempts      int
	shouldRetryFunc  func(ctx context.Context, err error, attempt int) bool
	waitFunc         func(ctx context.Context, attempt int) error
	shouldRetryCalls int
	waitCalls        int
}

func newTestRetrier(maxAttempts int) *testRetrier {
	return &testRetrier{
		maxAttempts: maxAttempts,
		shouldRetryFunc: func(ctx context.Context, err error, attempt int) bool {
			// Default: retry only on network errors (matches real retrier behavior)
			var netErr interface {
				Error() string
				Timeout() bool
				Temporary() bool
			}
			return errors.As(err, &netErr)
		},
	}
}

func (t *testRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	t.shouldRetryCalls++
	if t.shouldRetryFunc != nil {
		return t.shouldRetryFunc(ctx, err, attempt)
	}
	return false
}

func (t *testRetrier) Wait(ctx context.Context, attempt int) error {
	t.waitCalls++
	if t.waitFunc != nil {
		return t.waitFunc(ctx, attempt)
	}
	// Default: fast wait for testing
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (t *testRetrier) MaxAttempts() int {
	return t.maxAttempts
}

func (t *testRetrier) ShouldRetryCallCount() int {
	return t.shouldRetryCalls
}

func (t *testRetrier) WaitCallCount() int {
	return t.waitCalls
}


