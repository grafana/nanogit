package testutil

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// Logger is a minimal logging interface for test output.
//
// This simple interface allows testutil to work with any testing framework
// by accepting different logging implementations. Built-in implementations:
//   - NoopLogger: Discards all output (default, zero overhead)
//   - NewTestLogger: Logs to testing.T (standard Go tests)
//   - NewWriterLogger: Logs to io.Writer (e.g., Ginkgo's GinkgoWriter)
//   - NewColoredLogger: Logs with ANSI colors and emojis
//
// Custom implementations can be provided for other frameworks or use cases.
type Logger interface {
	Logf(format string, args ...any)
}

// ANSI color codes for terminal output (optional, used by ColoredLogger)
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// noopLogger is a logger that discards all output.
type noopLogger struct{}

func (noopLogger) Logf(format string, args ...any) {}

// NoopLogger returns a logger that discards all output.
//
// This is the default logger if none is specified. It has zero overhead
// and is suitable for CI environments or when you don't need log output.
//
// Example:
//
//	server, err := testutil.NewServer(ctx,
//		testutil.WithLogger(testutil.NoopLogger()),
//	)
func NoopLogger() Logger {
	return noopLogger{}
}

// testLogger wraps a testing.TB for standard Go test output.
type testLogger struct {
	t testing.TB
}

func (l *testLogger) Logf(format string, args ...any) {
	l.t.Logf(format, args...)
}

// NewTestLogger creates a logger that outputs to testing.TB.
//
// This is the standard logger for Go tests using the testing package.
// Output appears in test logs and is captured by `go test -v`.
//
// Example:
//
//	func TestSomething(t *testing.T) {
//		server, err := testutil.NewServer(ctx,
//			testutil.WithLogger(testutil.NewTestLogger(t)),
//		)
//		// Server logs will appear in test output
//	}
func NewTestLogger(t testing.TB) Logger {
	return &testLogger{t: t}
}

// writerLogger writes to an io.Writer (useful for Ginkgo's GinkgoWriter).
type writerLogger struct {
	w io.Writer
}

func (l *writerLogger) Logf(format string, args ...any) {
	fmt.Fprintf(l.w, format+"\n", args...)
}

// NewWriterLogger creates a logger that writes to an io.Writer.
// Useful for Ginkgo tests with GinkgoWriter.
func NewWriterLogger(w io.Writer) Logger {
	return &writerLogger{w: w}
}

// coloredLogger is a writer logger that adds colors and emojis.
type coloredLogger struct {
	w io.Writer
}

func (l *coloredLogger) Logf(format string, args ...any) {
	// Check if this is a structured log by looking for level markers
	msg := fmt.Sprintf(format, args...)

	// Add color based on content hints
	switch {
	case strings.Contains(msg, "[ERROR]") || strings.Contains(strings.ToLower(msg), "error"):
		fmt.Fprintf(l.w, "%s%s%s\n", ColorRed, msg, ColorReset)
	case strings.Contains(msg, "[WARN]") || strings.Contains(strings.ToLower(msg), "warn"):
		fmt.Fprintf(l.w, "%s%s%s\n", ColorYellow, msg, ColorReset)
	case strings.Contains(msg, "[SUCCESS]") || strings.Contains(msg, "✅") || strings.Contains(msg, "✨"):
		fmt.Fprintf(l.w, "%s%s%s\n", ColorGreen, msg, ColorReset)
	case strings.Contains(msg, "[DEBUG]"):
		fmt.Fprintf(l.w, "%s%s%s\n", ColorGray, msg, ColorReset)
	case strings.Contains(msg, "[INFO]"):
		fmt.Fprintf(l.w, "%s%s%s\n", ColorBlue, msg, ColorReset)
	default:
		fmt.Fprintf(l.w, "%s\n", msg)
	}
}

// NewColoredLogger creates a logger that adds colors and emojis to output.
// Best used with terminal-supporting writers.
func NewColoredLogger(w io.Writer) Logger {
	return &coloredLogger{w: w}
}

// structuredLogger provides nanogit.Logger-compatible methods.
// Useful for internal logging compatibility.
type structuredLogger struct {
	logger Logger
}

func (l *structuredLogger) Logf(format string, args ...any) {
	l.logger.Logf(format, args...)
}

func (l *structuredLogger) Debug(msg string, keysAndValues ...any) {
	l.log("DEBUG", msg, keysAndValues)
}

func (l *structuredLogger) Info(msg string, keysAndValues ...any) {
	l.log("INFO", msg, keysAndValues)
}

func (l *structuredLogger) Warn(msg string, keysAndValues ...any) {
	l.log("WARN", msg, keysAndValues)
}

func (l *structuredLogger) Error(msg string, keysAndValues ...any) {
	l.log("ERROR", msg, keysAndValues)
}

func (l *structuredLogger) Success(msg string, keysAndValues ...any) {
	l.log("SUCCESS", msg, keysAndValues)
}

func (l *structuredLogger) log(level, msg string, args []any) {
	formattedMsg := msg
	if len(args) > 0 {
		var pairs []string
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				pairs = append(pairs, fmt.Sprintf("%s=%v", args[i], args[i+1]))
			}
		}
		formattedMsg = fmt.Sprintf("%s (%s)", msg, strings.Join(pairs, ", "))
	}

	var emoji string
	switch level {
	case "DEBUG":
		emoji = "🔍"
	case "INFO":
		emoji = "ℹ️ "
	case "WARN":
		emoji = "⚠️ "
	case "ERROR":
		emoji = "❌"
	case "SUCCESS":
		emoji = "✅"
	}

	l.logger.Logf("%s [%s] %s", emoji, level, formattedMsg)
}

// NewStructuredLogger wraps a Logger to provide structured logging methods.
// Returns a logger compatible with nanogit.Logger interface.
func NewStructuredLogger(logger Logger) interface {
	Logger
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Success(msg string, keysAndValues ...any)
} {
	return &structuredLogger{logger: logger}
}
