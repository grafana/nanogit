// Package metrics defines the Recorder interface nanogit uses to report
// protocol/network-level instrumentation and the context plumbing to inject
// an implementation. Attach a recorder with ToContext; nanogit retrieves it
// with FromContext and falls back to a NoopRecorder when none is set.
//
//	ctx := metrics.ToContext(context.Background(), myRecorder)
//	client, err := nanogit.NewHTTPClient(repo, opts...)
//	ref, err := client.GetRef(ctx, "refs/heads/main")
//
// nanogit does not depend on any particular metrics backend (Prometheus,
// OpenTelemetry, etc.). Callers implement Recorder and bridge its calls to
// whichever backend they already use, the same way they bridge log.Logger —
// see the ExampleToContext function in this package for a minimal adapter,
// and https://grafana.github.io/nanogit/architecture/metrics for a full
// guide with Prometheus and OpenTelemetry bridges.
package metrics

import (
	"context"
	"time"
)

// Recorder receives protocol/network-level instrumentation events emitted by
// nanogit's HTTP and packfile-fetch layers. Implementations decide how to
// aggregate and export these events (e.g. as Prometheus or OpenTelemetry
// metrics); nanogit only reports raw values.
//
// Each method takes a single event struct rather than positional arguments,
// so nanogit can add fields to an event in a future minor version without
// breaking existing Recorder implementations — the same reason
// log/slog.Handler takes a slog.Record instead of a parameter list.
// Unrecognized fields should be ignored by implementations, not treated as
// exhaustive.
//
// ctx is the context of the nanogit operation that triggered the event. It
// is provided so implementations can attach trace-correlated exemplars (as
// OpenTelemetry's metric API requires) or read request-scoped values; it is
// not a signal to cancel or delay work. Recorder methods are called inline
// on the request path and must return promptly.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header ../internal/tools/fake_header.txt -o ../mocks/recorder.go . Recorder
type Recorder interface {
	// HTTPRequest reports the outcome of a single HTTP request/response
	// round trip made to the Git server.
	HTTPRequest(ctx context.Context, event HTTPRequestEvent)

	// ObjectsFetched reports objects retrieved over the network by a
	// single Fetch call.
	ObjectsFetched(ctx context.Context, event ObjectsFetchedEvent)

	// CacheAccess reports a single packfile object cache lookup performed
	// before deciding whether to fetch that object over the network.
	CacheAccess(ctx context.Context, event CacheAccessEvent)
}

// HTTPRequestEvent describes the outcome of a single HTTP request/response
// round trip made to the Git server.
type HTTPRequestEvent struct {
	// Operation identifies the Git protocol operation; see the Operation*
	// constants for the exhaustive set of values.
	Operation Operation
	// StatusCode is the HTTP status code, or 0 if the request failed
	// before a response was received.
	StatusCode int
	// Duration is the wall-clock time of this single attempt (not the
	// total across retries).
	Duration time.Duration
	// Attempt is the 1-indexed attempt number, so callers can derive a
	// retry count from repeated calls with Attempt > 1.
	Attempt int
}

// ObjectsFetchedEvent describes objects retrieved over the network by a
// single Fetch call.
type ObjectsFetchedEvent struct {
	// Count is the number of packfile objects parsed from the response.
	Count int
	// Bytes is the number of response bytes read.
	Bytes int64
}

// CacheAccessEvent describes a single packfile object cache lookup
// performed before deciding whether to fetch that object over the network.
type CacheAccessEvent struct {
	// Hit is true if the object was found in the configured
	// storage.PackfileStorage.
	Hit bool
}
