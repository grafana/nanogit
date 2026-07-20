package storage_test

import (
	"context"
	"time"

	"github.com/grafana/nanogit/storage"
)

// ExampleToContext shares an object cache across nanogit operations performed
// with the same context, so objects fetched by one operation can be reused by
// the next instead of being re-downloaded.
func ExampleToContext() {
	ctx := context.Background()

	cache := storage.NewInMemoryStorage(ctx, storage.WithTTL(5*time.Minute))
	ctx = storage.ToContext(ctx, cache)

	// Pass ctx to any nanogit operation: client.GetFlatTree(ctx, ...), etc.
	_ = ctx
}
