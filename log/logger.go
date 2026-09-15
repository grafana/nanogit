// Package log defines the Logger interface nanogit uses for diagnostics and
// the context plumbing to inject an implementation. Attach a logger with
// ToContext; nanogit retrieves it with FromContext and falls back to a
// NoopLogger when none is set.
package log

// Logger is a minimal logging interface for nanogit clients.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header ../internal/tools/fake_header.txt -o ../mocks/logger.go . Logger
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header ../internal/tools/fake_header.txt -o mocks/logger.go . Logger
type Logger interface {
	// Debug logs a debug-level message with alternating key/value pairs.
	Debug(msg string, keysAndValues ...any)
	// Info logs an info-level message with alternating key/value pairs.
	Info(msg string, keysAndValues ...any)
	// Error logs an error-level message with alternating key/value pairs.
	Error(msg string, keysAndValues ...any)
	// Warn logs a warning-level message with alternating key/value pairs.
	Warn(msg string, keysAndValues ...any)
}
