package log

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ../mocks/logger.go . Logger

// Logger is a minimal logging interface for nanogit clients.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
}
