package metrics

import "time"

// NoopRecorder implements Recorder but does nothing. It is the fallback
// returned by FromContext when no recorder is stored in the context.
type NoopRecorder struct{}

// HTTPRequest discards the reported request outcome.
func (n *NoopRecorder) HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int) {
}

// ObjectsFetched discards the reported fetch outcome.
func (n *NoopRecorder) ObjectsFetched(count int, bytes int64) {}

// CacheAccess discards the reported cache lookup outcome.
func (n *NoopRecorder) CacheAccess(hit bool) {}
