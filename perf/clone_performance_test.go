package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	nanolog "github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol/hash"
)

// testLogger implements simple logging for clone performance tests
type cloneTestLogger struct{}

func (l *cloneTestLogger) Debug(msg string, keysAndValues ...any) {
	if strings.Contains(msg, "missing") || strings.Contains(msg, "batch retry") ||
		strings.Contains(msg, "individual fallback") || strings.Contains(msg, "Fetch single missing tree object") {
		// Only log important debug messages during performance tests
	}
}

func (l *cloneTestLogger) Info(msg string, keysAndValues ...any)  {}
func (l *cloneTestLogger) Warn(msg string, keysAndValues ...any)  {}
func (l *cloneTestLogger) Error(msg string, keysAndValues ...any) {}

// cloneProgressTracker tracks clone progress for performance tests
type cloneProgressTracker struct {
	filesWritten int64
	filesFailed  int64
	totalSize    int64
	startTime    time.Time
}

func (p *cloneProgressTracker) onFileWritten(filePath string, size int64) {
	atomic.AddInt64(&p.filesWritten, 1)
	atomic.AddInt64(&p.totalSize, size)
}

func (p *cloneProgressTracker) onFileFailed(filePath string, err error) {
	atomic.AddInt64(&p.filesFailed, 1)
}

// TestClonePerformanceSmall tests clone performance with a small subset of grafana/grafana
func TestClonePerformanceSmall(t *testing.T) {
	ctx := context.Background()
	logger := &cloneTestLogger{}
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
	tempDir, err := os.MkdirTemp("", "nanogit-clone-perf-small-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Logf("📁 Clone destination: %s", tempDir)

	// Set up progress tracking
	tracker := &cloneProgressTracker{
		startTime: time.Now(),
	}

	// Performance benchmark: Clone filtered subset (similar to main_small.go)
	start := time.Now()
	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path: tempDir,
		Hash: commitHash,
		IncludePaths: []string{
			"go.mod",
			"go.sum",
			"package.json",
			"Makefile",
			"README.md",
			"LICENSE",
			"CHANGELOG.md",
			"pkg/api/**", // API package
		},
		ExcludePaths: []string{
			"node_modules/**",
			"vendor/**",
			"public/**",
			"docs/**",
			"devenv/**",
			"e2e/**",
			"scripts/**",
			"conf/**",
			"pkg/build/**",
			"pkg/services/**",
		},
		OnFileWritten: tracker.onFileWritten,
		OnFileFailed:  tracker.onFileFailed,
	})
	cloneDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Get final stats
	finalWritten := atomic.LoadInt64(&tracker.filesWritten)
	finalFailed := atomic.LoadInt64(&tracker.filesFailed)
	finalSize := atomic.LoadInt64(&tracker.totalSize)

	// Performance assertions (based on known commit ac641e07fe82669e01f7eeb84dc9256259ff1323)
	expectedTotalFiles := 22188
	expectedFilteredFiles := 164
	expectedWrittenFiles := 150
	maxDuration := 5 * time.Second

	if result.TotalFiles != expectedTotalFiles {
		t.Errorf("Expected exactly %d total files, got %d", expectedTotalFiles, result.TotalFiles)
	}
	if result.FilteredFiles != expectedFilteredFiles {
		t.Errorf("Expected exactly %d filtered files, got %d", expectedFilteredFiles, result.FilteredFiles)
	}
	if finalWritten != int64(expectedWrittenFiles) {
		t.Errorf("Expected exactly %d written files, got %d", expectedWrittenFiles, finalWritten)
	}
	if finalFailed > 0 {
		t.Errorf("Expected no failures, got %d failures", finalFailed)
	}
	if cloneDuration > maxDuration {
		t.Errorf("Clone took too long: %v (expected <= %v)", cloneDuration, maxDuration)
	}

	// Performance benchmarks
	throughputMBps := float64(finalSize) / (1024 * 1024) / cloneDuration.Seconds()
	successRate := float64(finalWritten) / float64(result.FilteredFiles) * 100

	t.Logf("🎉 Small Clone Performance Results:")
	t.Logf("   • Total files in repository: %d", result.TotalFiles)
	t.Logf("   • Files matching filters: %d", result.FilteredFiles)
	t.Logf("   • Files successfully written: %d", finalWritten)
	t.Logf("   • Files failed: %d", finalFailed)
	t.Logf("   • Success rate: %.1f%%", successRate)
	t.Logf("   • Total data written: %.1f MB", float64(finalSize)/(1024*1024))
	t.Logf("   • Clone time: %v", cloneDuration)
	t.Logf("   • Throughput: %.1f MB/s", throughputMBps)
	t.Logf("   • Commit: %s", result.Commit.Hash.String())

	// Additional validation (success rate should be 100% for exact expected counts)
	if successRate != 100.0 {
		t.Errorf("Success rate should be exactly 100%%, got %.1f%%", successRate)
	}

	// Print tree structure for debugging
	t.Logf("📂 Cloned tree structure:")
	printTreeStructure(t, tempDir, "", 0, 3) // Max depth of 3 levels

	// Verify some expected files exist
	expectedFiles := []string{
		"README.md", "go.mod", "go.sum", "package.json", "LICENSE", "Makefile",
	}

	for _, expectedFile := range expectedFiles {
		filePath := filepath.Join(tempDir, expectedFile)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Expected file %s should exist", expectedFile)
		}
	}
}

// TestClonePerformanceLarge tests clone performance with a larger subset
func TestClonePerformanceLarge(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()
	logger := &cloneTestLogger{}
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
	tempDir, err := os.MkdirTemp("", "nanogit-clone-perf-large-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Logf("📁 Clone destination: %s", tempDir)

	// Set up progress tracking
	tracker := &cloneProgressTracker{
		startTime: time.Now(),
	}

	// Performance benchmark: Clone larger subset (similar to main.go)
	start := time.Now()
	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path: tempDir,
		Hash: commitHash,
		IncludePaths: []string{
			// Core files
			"go.mod",
			"go.sum",
			"package.json",
			"Makefile",
			"README.md",
			"LICENSE",
			"CHANGELOG.md",
			// Essential directories
			"pkg/api/**",
			"pkg/infra/**",
			"pkg/models/**",
			"pkg/util/**",
			"pkg/setting/**",
			"pkg/middleware/**",
			"pkg/web/**",
			// Schema files
			"packages/grafana-schema/src/**",
			// Configuration
			"conf/**",
		},
		ExcludePaths: []string{
			// Heavy directories
			"node_modules/**",
			"vendor/**",
			"public/**",
			"docs/**",
			"devenv/**",
			"e2e/**",
			"scripts/**",
			"pkg/build/**",
			// Service layer (very large)
			"pkg/services/**",
			// Tests and mocks in large dirs
			"**/*_test.go",
			"**/mocks/**",
			"**/testdata/**",
		},
		OnFileWritten: tracker.onFileWritten,
		OnFileFailed:  tracker.onFileFailed,
	})
	cloneDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Large clone failed: %v", err)
	}

	// Get final stats
	finalWritten := atomic.LoadInt64(&tracker.filesWritten)
	finalFailed := atomic.LoadInt64(&tracker.filesFailed)
	finalSize := atomic.LoadInt64(&tracker.totalSize)

	// Performance assertions (based on known commit ac641e07fe82669e01f7eeb84dc9256259ff1323)
	expectedTotalFiles := 22188
	expectedFilteredFiles := 781 // Larger set includes more directories
	expectedWrittenFiles := 570  // Some files filtered out at clone time
	maxDuration := 10 * time.Second

	if result.TotalFiles != expectedTotalFiles {
		t.Errorf("Expected exactly %d total files, got %d", expectedTotalFiles, result.TotalFiles)
	}
	if result.FilteredFiles != expectedFilteredFiles {
		t.Errorf("Expected exactly %d filtered files, got %d", expectedFilteredFiles, result.FilteredFiles)
	}
	if finalWritten != int64(expectedWrittenFiles) {
		t.Errorf("Expected exactly %d written files, got %d", expectedWrittenFiles, finalWritten)
	}
	if finalFailed > 10 {
		t.Errorf("Too many failures for large clone: %d (expected <= 10)", finalFailed)
	}
	if cloneDuration > maxDuration {
		t.Errorf("Clone took too long: %v (expected <= %v)", cloneDuration, maxDuration)
	}

	// Performance metrics
	throughputMBps := float64(finalSize) / (1024 * 1024) / cloneDuration.Seconds()
	successRate := float64(finalWritten) / float64(result.FilteredFiles) * 100

	t.Logf("🎉 Large Clone Performance Results:")
	t.Logf("   • Total files in repository: %d", result.TotalFiles)
	t.Logf("   • Files matching filters: %d", result.FilteredFiles)
	t.Logf("   • Files successfully written: %d", finalWritten)
	t.Logf("   • Files failed: %d", finalFailed)
	t.Logf("   • Success rate: %.1f%%", successRate)
	t.Logf("   • Total data written: %.1f MB", float64(finalSize)/(1024*1024))
	t.Logf("   • Clone time: %v", cloneDuration)
	t.Logf("   • Throughput: %.1f MB/s", throughputMBps)
	t.Logf("   • Commit: %s", result.Commit.Hash.String())

	// Print tree structure for debugging
	t.Logf("📂 Cloned tree structure (large clone):")
	printTreeStructure(t, tempDir, "", 0, 2) // Max depth of 2 levels for large clone
}

// TestCloneConsistency tests that clone operations are consistent across multiple runs
func TestCloneConsistency(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	const numAttempts = 3

	ctx := context.Background()
	logger := &cloneTestLogger{}
	ctx = nanolog.ToContext(ctx, logger)

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

	var results []struct {
		filesWritten int64
		filesFailed  int64
		duration     time.Duration
	}

	for i := 0; i < numAttempts; i++ {
		t.Logf("=== Consistency Test Attempt %d ===", i+1)

		tempDir, err := os.MkdirTemp("", "nanogit-clone-consistency-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		tracker := &cloneProgressTracker{
			startTime: time.Now(),
		}

		start := time.Now()
		result, err := client.Clone(ctx, nanogit.CloneOptions{
			Path: tempDir,
			Hash: commitHash,
			IncludePaths: []string{
				"go.mod", "go.sum", "package.json", "README.md", "LICENSE",
				"pkg/api/admin.go", "pkg/api/api.go", "pkg/api/user.go",
			},
			OnFileWritten: tracker.onFileWritten,
			OnFileFailed:  tracker.onFileFailed,
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Consistency test attempt %d failed: %v", i+1, err)
		}

		// Basic validation of result
		if result.FilteredFiles < 5 {
			t.Errorf("Unexpected filtered files count: %d", result.FilteredFiles)
		}

		finalWritten := atomic.LoadInt64(&tracker.filesWritten)
		finalFailed := atomic.LoadInt64(&tracker.filesFailed)

		results = append(results, struct {
			filesWritten int64
			filesFailed  int64
			duration     time.Duration
		}{finalWritten, finalFailed, duration})

		t.Logf("✅ Attempt %d: %d files written, %d failed, %v duration",
			i+1, finalWritten, finalFailed, duration)

		// Should be consistent per attempt
		if finalWritten < 7 {
			t.Errorf("Attempt %d: too few files written: %d (expected >= 7)", i+1, finalWritten)
		}
		if finalFailed > 0 {
			t.Errorf("Attempt %d: unexpected failures: %d", i+1, finalFailed)
		}
	}

	// Check consistency across attempts
	baseWritten := results[0].filesWritten
	for i := 1; i < len(results); i++ {
		if results[i].filesWritten != baseWritten {
			t.Errorf("Inconsistent results: attempt 1 wrote %d files, attempt %d wrote %d files",
				baseWritten, i+1, results[i].filesWritten)
		}
	}

	t.Logf("🎉 All %d attempts succeeded consistently!", numAttempts)
}

// printTreeStructure recursively prints the directory structure for debugging
func printTreeStructure(t *testing.T, path string, prefix string, depth int, maxDepth int) {
	if depth > maxDepth {
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		t.Logf("%s❌ Error reading %s: %v", prefix, path, err)
		return
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1
		var currentPrefix, nextPrefix string

		if isLast {
			currentPrefix = prefix + "└── "
			nextPrefix = prefix + "    "
		} else {
			currentPrefix = prefix + "├── "
			nextPrefix = prefix + "│   "
		}

		if entry.IsDir() {
			t.Logf("%s📁 %s/", currentPrefix, entry.Name())
			if depth < maxDepth {
				printTreeStructure(t, filepath.Join(path, entry.Name()), nextPrefix, depth+1, maxDepth)
			}
		} else {
			// Get file size
			info, err := entry.Info()
			var sizeStr string
			if err == nil {
				if info.Size() < 1024 {
					sizeStr = fmt.Sprintf(" (%d B)", info.Size())
				} else if info.Size() < 1024*1024 {
					sizeStr = fmt.Sprintf(" (%.1f KB)", float64(info.Size())/1024)
				} else {
					sizeStr = fmt.Sprintf(" (%.1f MB)", float64(info.Size())/(1024*1024))
				}
			}
			t.Logf("%s📄 %s%s", currentPrefix, entry.Name(), sizeStr)
		}
	}
}
