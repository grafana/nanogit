package metrics

import "context"

// NoopRecorder implements Recorder but does nothing. It is the fallback
// returned by FromContext when no recorder is stored in the context.
type NoopRecorder struct{}

// HTTPRequest discards the reported request outcome.
func (n *NoopRecorder) HTTPRequest(ctx context.Context, event HTTPRequestEvent) {}

// ObjectsFetched discards the reported fetch outcome.
func (n *NoopRecorder) ObjectsFetched(ctx context.Context, event ObjectsFetchedEvent) {}

// CacheAccess discards the reported cache lookup outcome.
func (n *NoopRecorder) CacheAccess(ctx context.Context, event CacheAccessEvent) {}
