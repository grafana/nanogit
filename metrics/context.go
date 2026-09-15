package metrics

import "context"

// recorderCtxKey is the key used to store the recorder in the context.
type recorderCtxKey struct{}

// ToContext returns a copy of ctx carrying the given recorder. nanogit
// operations performed with the returned context report metrics through it.
func ToContext(ctx context.Context, recorder Recorder) context.Context {
	return context.WithValue(ctx, recorderCtxKey{}, recorder)
}

// FromContext returns the Recorder stored in ctx, or a NoopRecorder if none
// is stored, so callers can report metrics unconditionally.
func FromContext(ctx context.Context) Recorder {
	recorder, ok := ctx.Value(recorderCtxKey{}).(Recorder)
	if !ok {
		return &NoopRecorder{}
	}

	return recorder
}
