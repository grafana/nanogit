package retry

import "context"

// Package retry provides context helpers for injecting and retrieving retriers.

// retrierKey is the key for the retrier in the context.
type retrierKey struct{}

// ToContext sets the retrier for the client from the context.
func ToContext(ctx context.Context, retrier Retrier) context.Context {
	return context.WithValue(ctx, retrierKey{}, retrier)
}

// FromContext gets the retrier from the context.
func FromContext(ctx context.Context) Retrier {
	retrier, ok := ctx.Value(retrierKey{}).(Retrier)
	if !ok {
		return nil
	}

	return retrier
}

// FromContextOrNoop returns the retrier from the context, or a NoopRetrier if none is set.
// This ensures that retry logic always has a retrier to work with.
func FromContextOrNoop(ctx context.Context) Retrier {
	retrier := FromContext(ctx)
	if retrier != nil {
		return retrier
	}

	return &NoopRetrier{}
}

