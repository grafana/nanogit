package client

import (
	"context"
	"errors"

	"github.com/grafana/nanogit/protocol"
)

// fakeRetrier is a simple mock implementation of retry.Retrier for testing
// This is used instead of the generated mock from mocks package to avoid import cycles
// (mocks package imports protocol/client, creating a cycle)
type fakeRetrier struct {
	maxAttempts      int
	shouldRetryFunc  func(ctx context.Context, err error, attempt int) bool
	waitFunc         func(ctx context.Context, attempt int) error
	shouldRetryCalls int
	waitCalls        int
}

func newFakeRetrier(maxAttempts int) *fakeRetrier {
	return &fakeRetrier{
		maxAttempts: maxAttempts,
		shouldRetryFunc: func(ctx context.Context, err error, attempt int) bool {
			return errors.Is(err, protocol.ErrServerUnavailable)
		},
	}
}

func (f *fakeRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	f.shouldRetryCalls++
	if f.shouldRetryFunc != nil {
		return f.shouldRetryFunc(ctx, err, attempt)
	}
	return false
}

func (f *fakeRetrier) Wait(ctx context.Context, attempt int) error {
	f.waitCalls++
	if f.waitFunc != nil {
		return f.waitFunc(ctx, attempt)
	}
	return nil
}

func (f *fakeRetrier) MaxAttempts() int {
	return f.maxAttempts
}

func (f *fakeRetrier) ShouldRetryCallCount() int {
	return f.shouldRetryCalls
}

func (f *fakeRetrier) WaitCallCount() int {
	return f.waitCalls
}

