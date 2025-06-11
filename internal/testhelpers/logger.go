package testhelpers

import (
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2"
)

// TestLogger implements the nanogit.Logger interface for testing purposes.
// It uses Ginkgo's native logging capabilities for thread-safe output.
type TestLogger struct{}

// NewTestLogger creates a new TestLogger for Ginkgo tests.
func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

// Logf logs a message to the Ginkgo test output with colors and emojis.
func (l *TestLogger) Logf(format string, args ...any) {
	ginkgo.GinkgoWriter.Printf(format+"\n", args...)
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

// log is a helper method to log messages with proper formatting.
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

	// Log to Ginkgo output with colors and emojis
	switch level {
	case "Debug":
		ginkgo.GinkgoWriter.Printf("%s🔍 [DEBUG] %s%s\n", ColorGray, formattedMsg, ColorReset)
	case "Info":
		ginkgo.GinkgoWriter.Printf("%sℹ️  [INFO] %s%s\n", ColorBlue, formattedMsg, ColorReset)
	case "Warn":
		ginkgo.GinkgoWriter.Printf("%s⚠️  [WARN] %s%s\n", ColorYellow, formattedMsg, ColorReset)
	case "Error":
		ginkgo.GinkgoWriter.Printf("%s❌ [ERROR] %s%s\n", ColorRed, formattedMsg, ColorReset)
	case "Success":
		ginkgo.GinkgoWriter.Printf("%s✅ [SUCCESS] %s%s\n", ColorGreen, formattedMsg, ColorReset)
	}
}
