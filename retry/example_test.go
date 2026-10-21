package retry_test

import (
	"context"
	"time"

	"github.com/grafana/nanogit/retry"
)

// ExampleToContext enables retries with exponential backoff for every nanogit
// operation performed with the returned context. Without a retrier in the
// context, operations are attempted exactly once.
func ExampleToContext() {
	retrier := retry.NewExponentialBackoffRetrier().
		WithMaxAttempts(5).
		WithInitialDelay(200 * time.Millisecond).
		WithMaxDelay(10 * time.Second).
		WithJitter()

	ctx := retry.ToContext(context.Background(), retrier)

	// Pass ctx to any nanogit operation: client.GetRef(ctx, ...), etc.
	_ = ctx
}
