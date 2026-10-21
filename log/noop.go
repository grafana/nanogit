package log

// NoopLogger implements Logger but does nothing. It is the fallback returned
// by FromContext when no logger is stored in the context.
type NoopLogger struct{}

// Debug discards the message and key/value pairs.
func (n *NoopLogger) Debug(msg string, keysAndValues ...any) {}

// Info discards the message and key/value pairs.
func (n *NoopLogger) Info(msg string, keysAndValues ...any) {}

// Error discards the message and key/value pairs.
func (n *NoopLogger) Error(msg string, keysAndValues ...any) {}

// Warn discards the message and key/value pairs.
func (n *NoopLogger) Warn(msg string, keysAndValues ...any) {}
