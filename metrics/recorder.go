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

// Recorder receives protocol/network-level instrumentation samples emitted
// by nanogit's HTTP and packfile-fetch layers. Implementations decide how to
// aggregate and export these samples (e.g. as Prometheus or OpenTelemetry
// metrics); nanogit only reports raw values.
//
// Each method takes a single sample struct rather than positional arguments,
// so nanogit can add fields to a sample in a future minor version without
// breaking existing Recorder implementations — the same reason
// log/slog.Handler takes a slog.Record instead of a parameter list.
// Unrecognized fields should be ignored by implementations, not treated as
// exhaustive.
//
// ctx is the context of the nanogit operation that triggered the sample. It
// is provided so implementations can attach trace-correlated exemplars (as
// OpenTelemetry's metric API requires) or read request-scoped values; it is
// not a signal to cancel or delay work. Recorder methods are called inline
// on the request path and must return promptly.
//
// A Client is safe for concurrent use by multiple goroutines, and the same
// Recorder can be shared across all of them via one context (or reused
// across multiple contexts); implementations must therefore be safe for
// concurrent calls.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header ../internal/tools/fake_header.txt -o ../mocks/recorder.go . Recorder
type Recorder interface {
	// HTTPRequest reports the outcome of a single HTTP request/response
	// round trip made to the Git server.
	HTTPRequest(ctx context.Context, sample HTTPRequestSample)

	// ObjectsFetched reports objects retrieved over the network by a
	// single Fetch call.
	ObjectsFetched(ctx context.Context, sample ObjectsFetchedSample)

	// CacheAccess reports a single packfile object cache lookup performed
	// before deciding whether to fetch that object over the network.
	CacheAccess(ctx context.Context, sample CacheAccessSample)
}

// HTTPRequestSample describes the outcome of a single HTTP request/response
// round trip made to the Git server.
type HTTPRequestSample struct {
	// Operation identifies the Git protocol operation; see the Operation*
	// constants for the exhaustive set of values.
	Operation Operation
	// StatusCode is the HTTP status code, or 0 if the request failed
	// before a response was received.
	StatusCode int
	// Duration is the wall-clock time this single attempt took (not the
	// total across retries). For a request that received a response,
	// Duration spans from sending the request to closing the response
	// body, so it includes reading the whole body — the packfile for
	// upload-pack, which is often the slowest part of the operation — not
	// just the time to the response status line and headers. For an
	// attempt that failed before any response (a network error) or whose
	// response was rejected as server-unavailable and retried, the body is
	// never delivered to nanogit's caller, so Duration covers only the
	// time up to that failure.
	Duration time.Duration
	// Attempt is the 1-indexed attempt number, so callers can derive a
	// retry count from repeated calls with Attempt > 1.
	Attempt int
}

// ObjectsFetchedSample describes objects retrieved over the network by a
// single Fetch call. It is reported once a response body is being read,
// even if the fetch ultimately fails (a malformed, truncated, oversized,
// or mid-stream-failing response still consumed network bytes and may
// have parsed some objects before failing) — Count and Bytes reflect
// however much was read before any such failure.
type ObjectsFetchedSample struct {
	// Count is the number of packfile objects parsed from the response
	// before the fetch completed or failed.
	Count int
	// Bytes is the number of response bytes read before the fetch
	// completed or failed.
	Bytes int64
}

// CacheAccessSample describes a single packfile object cache lookup
// performed before deciding whether to fetch that object over the network.
type CacheAccessSample struct {
	// Hit is true if the object was found in the configured
	// storage.PackfileStorage.
	Hit bool
}
