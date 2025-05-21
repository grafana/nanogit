package nanogit

import "errors"

// WithLogger sets a custom logger for the client.
// If not provided, the default slog logger will be used.
func WithLogger(logger Logger) Option {
	return func(c *clientImpl) error {
		if logger == nil {
			return errors.New("logger cannot be nil")
		}
		c.logger = logger
		return nil
	}
}

// Logger is a minimal logging interface for nanogit clients.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
}

// noopLogger implements Logger but does nothing.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, keysAndValues ...any) {}
func (n *noopLogger) Info(msg string, keysAndValues ...any)  {}
