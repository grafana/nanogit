package nanogit

import "errors"

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

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o mocks/logger.go . Logger

// Logger is a minimal logging interface for nanogit clients.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
}

// noopLogger implements Logger but does nothing.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, keysAndValues ...any) {}
func (n *noopLogger) Info(msg string, keysAndValues ...any)  {}
