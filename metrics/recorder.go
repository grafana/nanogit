// Package metrics defines the Recorder interface nanogit uses to report
// protocol/network-level instrumentation and the context plumbing to inject
// an implementation. Attach a recorder with ToContext; nanogit retrieves it
// with FromContext and falls back to a NoopRecorder when none is set.
//
// nanogit does not depend on any particular metrics backend (Prometheus,
// OpenTelemetry, etc.). Callers implement Recorder and bridge its calls to
// whichever backend they already use, the same way they bridge log.Logger.
package metrics

import "time"

// Recorder receives protocol/network-level instrumentation events emitted by
// nanogit's HTTP and packfile-fetch layers. Implementations decide how to
// aggregate and export these events (e.g. as Prometheus or OpenTelemetry
// metrics); nanogit only reports raw values.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header ../internal/tools/fake_header.txt -o ../mocks/recorder.go . Recorder
type Recorder interface {
	// HTTPRequest reports the outcome of a single HTTP request/response
	// round trip made to the Git server. operation identifies the Git
	// protocol operation ("smart-info", "upload-pack", "receive-pack",
	// "receive-pack-capabilities", "compatibility"). statusCode is the
	// HTTP status code, or 0 if the request failed before a response was
	// received. attempt is the 1-indexed attempt number, so callers can
	// derive a retry count from repeated calls with attempt > 1.
	HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int)

	// ObjectsFetched reports objects retrieved over the network by a
	// single Fetch call. count is the number of packfile objects parsed
	// from the response; bytes is the number of response bytes read.
	ObjectsFetched(count int, bytes int64)

	// CacheAccess reports a single packfile object cache lookup performed
	// before deciding whether to fetch that object over the network.
	CacheAccess(hit bool)
}
