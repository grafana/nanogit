package metrics_test

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/grafana/nanogit/metrics"
)

// counterRecorder is a minimal metrics.Recorder that tallies request and
// object counts in memory. Real implementations typically forward these
// calls to a metrics backend (Prometheus, OpenTelemetry, etc.) instead of
// aggregating locally; see the package documentation for links to fuller
// examples.
type counterRecorder struct {
	requests atomic.Int64
	objects  atomic.Int64
}

func (c *counterRecorder) HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int) {
	c.requests.Add(1)
}

func (c *counterRecorder) ObjectsFetched(count int, bytes int64) {
	c.objects.Add(int64(count))
}

func (c *counterRecorder) CacheAccess(hit bool) {}

// ExampleToContext wires a recorder into the context so nanogit operations
// performed with that context report HTTP request and object-fetch metrics.
// Without it, nanogit reports nothing (NoopRecorder).
func ExampleToContext() {
	recorder := &counterRecorder{}

	ctx := metrics.ToContext(context.Background(), recorder)

	// Pass ctx to any nanogit operation: client.GetRef(ctx, ...), etc.
	_ = ctx
}
