package performance

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	nanolog "github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol/hash"
)

// detailedLogger captures specific debug messages for full clone analysis
type detailedLogger struct{}

func (l *detailedLogger) Debug(msg string, keysAndValues ...any) {
	// Only log individual fallback messages to see if it's working
	if strings.Contains(msg, "Individual fallback completed") ||
		strings.Contains(msg, "Some blobs missing from batch") ||
		strings.Contains(msg, "Attempting individual blob fetch") ||
		strings.Contains(msg, "Individual blob fetch succeeded") ||
		strings.Contains(msg, "Individual blob fetch failed") {
		args := make([]string, len(keysAndValues))
		for i, v := range keysAndValues {
			args[i] = fmt.Sprint(v)
		}
		if len(args) > 0 {
			fmt.Printf("DEBUG: %s | %s\n", msg, strings.Join(args, " "))
		} else {
			fmt.Printf("DEBUG: %s\n", msg)
		}
	}
}

func (l *detailedLogger) Info(msg string, keysAndValues ...any) {}
func (l *detailedLogger) Warn(msg string, keysAndValues ...any) {
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		fmt.Printf("WARN: %s | %s\n", msg, strings.Join(args, " "))
	} else {
		fmt.Printf("WARN: %s\n", msg)
	}
}

func (l *detailedLogger) Error(msg string, keysAndValues ...any) {
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		fmt.Printf("ERROR: %s | %s\n", msg, strings.Join(args, " "))
	} else {
		fmt.Printf("ERROR: %s\n", msg)
	}
}

// TestDebugFullClone runs a limited full clone to see individual fallback behavior
func TestDebugFullClone(t *testing.T) {
	ctx := context.Background()
	logger := &detailedLogger{}
	ctx = nanolog.ToContext(ctx, logger)

	// Create HTTP client for public repository
	client, err := nanogit.NewHTTPClient("https://github.com/grafana/grafana.git")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Use fixed commit hash for consistent testing
	targetCommitHash := "ac641e07fe82669e01f7eeb84dc9256259ff1323"
	commitHash, err := hash.FromHex(targetCommitHash)
	if err != nil {
		t.Fatalf("Failed to parse target commit hash: %v", err)
	}

	t.Logf("🔍 Testing full clone with individual fallback debug logging")

	// Create temporary directory for clone
	tempDir, err := os.MkdirTemp("", "nanogit-debug-full-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone a larger subset to trigger parallel batching (>100 files) but not the full repo
	start := time.Now()
	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path: tempDir,
		Hash: commitHash,
		IncludePaths: []string{
			"pkg/aggregator/**",
			"pkg/api/**", // Add more files to trigger parallel batching
		},
	})
	duration := time.Since(start)

	if err != nil {
		t.Logf("⚠️ Clone failed (expected if many blobs missing): %v", err)
	}

	t.Logf("🔍 Full Clone Debug Results:")
	t.Logf("   • Total files: %d", result.TotalFiles)
	t.Logf("   • Filtered files: %d", result.TotalFilteredFiles)
	t.Logf("   • Duration: %v", duration)
}

