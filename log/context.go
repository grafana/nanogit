package log

import (
	"context"
)

// loggerCtxKey is the key used to store the logger in the context.
type loggerCtxKey struct{}

// ToContext returns a copy of ctx carrying the given logger. nanogit
// operations performed with the returned context log through it.
func ToContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// FromContext returns the Logger stored in ctx, or a NoopLogger if none is
// stored, so callers can log unconditionally.
func FromContext(ctx context.Context) Logger {
	logger, ok := ctx.Value(loggerCtxKey{}).(Logger)
	if !ok {
		return &NoopLogger{}
	}

	return logger
}
