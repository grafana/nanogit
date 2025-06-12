package nanogit

import (
	"context"
	"errors"
)

// WithLogger configures a custom logger for the Git client.
// This allows integration with existing logging infrastructure and debugging.
// If not provided, a no-op logger will be used by default.
//
// Parameters:
//   - logger: Custom logger implementation
//
// Returns:
//   - Option: Configuration function for the client
//   - error: Error if the provided logger is nil
func WithLogger(logger Logger) Option {
	return func(c *httpClient) error {
		if logger == nil {
			return errors.New("logger cannot be nil")
		}
		c.logger = logger
		return nil
	}
}

// loggerCtxKey is the key used to store the logger in the context.
type loggerCtxKey struct{}

// WithContextLogger adds a logger to the context that can be retrieved later.
// The logger will be used for operations performed with this context.
// If no logger is provided in the context, a no-op logger will be used.
//
// Parameters:
//   - ctx: The context to add the logger to
//   - logger: The logger to store in the context
//
// Returns:
//   - context.Context: A new context with the logger stored
func WithContextLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// getContextLogger retrieves the logger from the context.
// If no logger is stored in the context, nil will be returned.
//
// Parameters:
//   - ctx: The context to retrieve the logger from
//
// Returns:
//   - Logger: The logger stored in the context, or nil if none is found
func getContextLogger(ctx context.Context) Logger {
	logger, ok := ctx.Value(loggerCtxKey{}).(Logger)
	if !ok {
		return nil
	}

	return logger
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o mocks/logger.go . Logger

// Logger is a minimal logging interface for nanogit clients.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
}

// noopLogger implements Logger but does nothing.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, keysAndValues ...any) {}
func (n *noopLogger) Info(msg string, keysAndValues ...any)  {}
func (n *noopLogger) Error(msg string, keysAndValues ...any) {}
func (n *noopLogger) Warn(msg string, keysAndValues ...any)  {}
