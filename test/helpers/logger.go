package helpers

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestLogger implements the nanogit.Logger interface for testing purposes.
type TestLogger struct {
	getCurrentT func() *testing.T // Function to get current test context
	mu          sync.Mutex        // Protects concurrent access to testing.T
}

// NewSuiteLogger creates a new TestLogger that automatically uses the current test context from a suite.
func NewSuiteLogger(getCurrentT func() *testing.T) *TestLogger {
	return &TestLogger{
		getCurrentT: getCurrentT,
	}
}

// getCurrentTestContext returns the current test context, either from the function or the stored t
func (l *TestLogger) getCurrentTestContext() *testing.T {
	return l.getCurrentT()
}

// Logf logs a message to the test output with colors and emojis.
func (l *TestLogger) Logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.getCurrentTestContext().Logf(format, args...)
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

// Success implements nanogit.Logger.
func (l *TestLogger) Success(msg string, keysAndValues ...any) {
	l.log("Success", msg, keysAndValues)
}

// log is a helper method to log messages and store them in entries.
func (l *TestLogger) log(level, msg string, args []any) {
	// Protect concurrent access to testing.T
	l.mu.Lock()
	defer l.mu.Unlock()

	currentT := l.getCurrentTestContext()

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
		currentT.Logf("%s🔍 [DEBUG] %s%s", ColorGray, formattedMsg, ColorReset)
	case "Info":
		currentT.Logf("%sℹ️  [INFO] %s%s", ColorBlue, formattedMsg, ColorReset)
	case "Warn":
		currentT.Logf("%s⚠️  [WARN] %s%s", ColorYellow, formattedMsg, ColorReset)
	case "Error":
		currentT.Logf("%s❌ [ERROR] %s%s", ColorRed, formattedMsg, ColorReset)
	case "Success":
		currentT.Logf("%s✅ [SUCCESS] %s%s", ColorGreen, formattedMsg, ColorReset)
	}
}
