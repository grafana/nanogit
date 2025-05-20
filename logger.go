package nanogit

// Logger is a minimal logging interface for nanogit clients.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// noopLogger implements Logger but does nothing.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, keysAndValues ...any) {}
func (n *noopLogger) Info(msg string, keysAndValues ...any)  {}
func (n *noopLogger) Warn(msg string, keysAndValues ...any)  {}
func (n *noopLogger) Error(msg string, keysAndValues ...any) {}
