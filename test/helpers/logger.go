package helpers

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestLogger implements the nanogit.Logger interface for testing purposes.
type TestLogger struct {
	mu      sync.Mutex
	t       *testing.T
	entries []struct {
		level string
		msg   string
		args  []any
	}
}

// NewTestLogger creates a new TestLogger instance.
func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{
		t: t,
		entries: make([]struct {
			level string
			msg   string
			args  []any
		}, 0),
	}
}

func (l *TestLogger) ForSubtest(t *testing.T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.t = t
}

// Debug implements nanogit.Logger.
func (l *TestLogger) Debug(msg string, keysAndValues ...any) {
	l.log("Debug", msg, keysAndValues)
}

// Info implements nanogit.Logger.
func (l *TestLogger) Info(msg string, keysAndValues ...any) {
	l.log("Info", msg, keysAndValues)
}

// Warn implements nanogit.Logger.
func (l *TestLogger) Warn(msg string, keysAndValues ...any) {
	l.log("Warn", msg, keysAndValues)
}

// Error implements nanogit.Logger.
func (l *TestLogger) Error(msg string, keysAndValues ...any) {
	l.log("Error", msg, keysAndValues)
}

// log is a helper method to log messages and store them in entries.
func (l *TestLogger) log(level, msg string, args []any) {
	// Format the message with key-value pairs
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

	// Log to test output with colors and emojis
	switch level {
	case "Debug":
		l.t.Logf("%s🔍 [DEBUG] %s%s", ColorBlue, formattedMsg, ColorReset)
	case "Info":
		l.t.Logf("%sℹ️  [INFO] %s%s", ColorGreen, formattedMsg, ColorReset)
	case "Warn":
		l.t.Logf("%s⚠️  [WARN] %s%s", ColorYellow, formattedMsg, ColorReset)
	case "Error":
		l.t.Logf("%s❌ [ERROR] %s%s", ColorRed, formattedMsg, ColorReset)
	}

	// Store in entries
	l.entries = append(l.entries, struct {
		level string
		msg   string
		args  []any
	}{level, msg, args})
}

// GetEntries returns all logged entries.
func (l *TestLogger) GetEntries() []struct {
	level string
	msg   string
	args  []any
} {
	return l.entries
}

// Clear clears all logged entries.
func (l *TestLogger) Clear() {
	l.entries = make([]struct {
		level string
		msg   string
		args  []any
	}, 0)
}
