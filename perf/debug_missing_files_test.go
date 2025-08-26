package performance

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafana/nanogit"
	nanolog "github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol/hash"
)

// simpleLogger for debugging missing files
type simpleLogger struct {
	failureCount int64
}

func (l *simpleLogger) Debug(msg string, keysAndValues ...any) {
	if strings.Contains(msg, "missing blob") || 
	   strings.Contains(msg, "Blob fetch summary") ||
	   strings.Contains(msg, "Missing blobs") ||
	   strings.Contains(msg, "Individual fallback") ||
	   strings.Contains(msg, "individually") ||
	   strings.Contains(msg, "batch retry") ||
	   strings.Contains(msg, "still_missing") ||
	   strings.Contains(msg, "GetBlob") ||
	   strings.Contains(msg, "Failed to fetch") ||
	   strings.Contains(msg, "Handling missing blobs") ||
	   strings.Contains(msg, "processBatchesInParallel") ||
	   strings.Contains(msg, "handleMissingBlobs") {
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

func (l *simpleLogger) Info(msg string, keysAndValues ...any) {
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		fmt.Printf("INFO: %s | %s\n", msg, strings.Join(args, " "))
	} else {
		fmt.Printf("INFO: %s\n", msg)
	}
}

func (l *simpleLogger) Warn(msg string, keysAndValues ...any) {
	atomic.AddInt64(&l.failureCount, 1)
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

func (l *simpleLogger) Error(msg string, keysAndValues ...any) {
	atomic.AddInt64(&l.failureCount, 1)
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

// TestDebugMissingFiles runs a smaller clone with debug logging
func TestDebugMissingFiles(t *testing.T) {
	ctx := context.Background()
	logger := &simpleLogger{}
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

	t.Logf("📌 Testing against fixed commit: %s", targetCommitHash)

	// Create temporary directory for clone
	tempDir, err := os.MkdirTemp("", "nanogit-debug-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Logf("📁 Debug clone destination: %s", tempDir)

	// Test with just a single problematic directory to debug
	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path: tempDir,
		Hash: commitHash,
		IncludePaths: []string{
			"pkg/aggregator/**",
		},
		OnFileWritten: func(path string, size int64) {
			t.Logf("✅ Written: %s (%d bytes)", path, size)
		},
		OnFileFailed: func(path string, err error) {
			t.Logf("❌ Failed: %s - %v", path, err)
		},
	})

	if err != nil {
		t.Fatalf("Debug clone failed: %v", err)
	}

	t.Logf("🔍 Debug Results:")
	t.Logf("   • Total files in repository: %d", result.TotalFiles)
	t.Logf("   • Files matching filters: %d", result.FilteredFiles)
	t.Logf("   • Logger failure count: %d", atomic.LoadInt64(&logger.failureCount))
}