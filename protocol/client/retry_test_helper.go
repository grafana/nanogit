package client

import (
	"context"
	"errors"
	"sync"

	"github.com/grafana/nanogit/protocol"
)

// trackingRetrier tracks calls to ShouldRetry and Wait for testing
type trackingRetrier struct {
	shouldRetryCalls []shouldRetryCall
	waitCalls        []waitCall
	maxAttempts      int
	shouldRetryFunc  func(ctx context.Context, err error, attempt int) bool
	waitFunc         func(ctx context.Context, attempt int) error
	mu               sync.Mutex
}

type shouldRetryCall struct {
	ctx     context.Context
	err     error
	attempt int
	result  bool
}

type waitCall struct {
	ctx     context.Context
	attempt int
	err     error
}

func newTrackingRetrier(maxAttempts int) *trackingRetrier {
	return &trackingRetrier{
		maxAttempts: maxAttempts,
		shouldRetryCalls: make([]shouldRetryCall, 0),
		waitCalls:        make([]waitCall, 0),
	}
}

func (r *trackingRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result bool
	if r.shouldRetryFunc != nil {
		result = r.shouldRetryFunc(ctx, err, attempt)
	} else {
		// Default: retry on server unavailable errors
		result = errors.Is(err, protocol.ErrServerUnavailable)
	}

	r.shouldRetryCalls = append(r.shouldRetryCalls, shouldRetryCall{
		ctx:     ctx,
		err:     err,
		attempt: attempt,
		result:  result,
	})
	return result
}

func (r *trackingRetrier) Wait(ctx context.Context, attempt int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	if r.waitFunc != nil {
		err = r.waitFunc(ctx, attempt)
	}

	r.waitCalls = append(r.waitCalls, waitCall{
		ctx:     ctx,
		attempt: attempt,
		err:     err,
	})
	return err
}

func (r *trackingRetrier) MaxAttempts() int {
	return r.maxAttempts
}

func (r *trackingRetrier) getShouldRetryCalls() []shouldRetryCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]shouldRetryCall{}, r.shouldRetryCalls...)
}

func (r *trackingRetrier) getWaitCalls() []waitCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]waitCall{}, r.waitCalls...)
}

