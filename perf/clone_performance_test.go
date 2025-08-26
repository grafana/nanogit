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
	// Log all debug messages for investigation
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		println("DEBUG: " + msg + " | " + strings.Join(args, " "))
	} else {
		println("DEBUG: " + msg)
	}
}

func (l *cloneTestLogger) Info(msg string, keysAndValues ...any) {
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		println("INFO: " + msg + " | " + strings.Join(args, " "))
	} else {
		println("INFO: " + msg)
	}
}

func (l *cloneTestLogger) Warn(msg string, keysAndValues ...any) {
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		println("WARN: " + msg + " | " + strings.Join(args, " "))
	} else {
		println("WARN: " + msg)
	}
}

func (l *cloneTestLogger) Error(msg string, keysAndValues ...any) {
	args := make([]string, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = fmt.Sprint(v)
	}
	if len(args) > 0 {
		println("ERROR: " + msg + " | " + strings.Join(args, " "))
	} else {
		println("ERROR: " + msg)
	}
}

// cloneProgressTracker tracks clone progress for performance tests
type cloneProgressTracker struct {
	filesWritten int64
	filesFailed  int64
	totalSize    int64
	startTime    time.Time
	writtenList  []string
	failedList   []string
}

func (p *cloneProgressTracker) onFileWritten(filePath string, size int64) {
	atomic.AddInt64(&p.filesWritten, 1)
	atomic.AddInt64(&p.totalSize, size)
	p.writtenList = append(p.writtenList, filePath)
}

func (p *cloneProgressTracker) onFileFailed(filePath string, err error) {
	atomic.AddInt64(&p.filesFailed, 1)
	p.failedList = append(p.failedList, fmt.Sprintf("%s: %v", filePath, err))
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
	expectedTotalFiles := 18347  // Only files, not directories
	expectedFilteredFiles := 150 // Only files that pass filtering (directories now correctly excluded)
	expectedWrittenFiles := 150  // All filtered files should be successfully written
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
	// Now FilteredFiles only counts files (not directories), so success rate should be 100%
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

	// Debug: Check if there are any directories in the filtered tree
	dirCount := 0
	fileCount := 0
	for _, entry := range result.FlatTree.Entries {
		if entry.Mode&0o40000 != 0 {
			dirCount++
		} else {
			fileCount++
		}
	}
	t.Logf("🔍 FlatTree composition:")
	t.Logf("   • Files in FlatTree: %d", fileCount)
	t.Logf("   • Directories in FlatTree: %d", dirCount)
	t.Logf("   • Total FlatTree entries: %d", len(result.FlatTree.Entries))

	// Success rate validation - should be 100% if all filtered files can be written
	// If < 100%, it means some files passed filtering but couldn't be written (missing blobs)
	if finalFailed == 0 && successRate != 100.0 {
		t.Logf("⚠️  Note: %d files passed filtering but weren't written (FilteredFiles=%d, Written=%d)",
			result.FilteredFiles-int(finalWritten), result.FilteredFiles, finalWritten)
		t.Logf("This could indicate files that exist in the tree but have missing blob data")

		// Let's identify which files are missing by examining the clone destination
		t.Logf("🔍 Investigating missing files...")
		missingCount := 0
		missingFiles := []string{}
		writtenFiles := []string{}

		// First, get list of written files from filesystem
		err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(tempDir, path)
				if err == nil {
					writtenFiles = append(writtenFiles, filepath.ToSlash(relPath))
				}
			}
			return nil
		})
		if err != nil {
			t.Logf("Error walking directory: %v", err)
		}

		t.Logf("📁 Files written to disk: %d", len(writtenFiles))
		if len(writtenFiles) <= 20 {
			for i, file := range writtenFiles {
				t.Logf("   [%d] %s", i+1, file)
			}
		}

		// Now compare against filtered tree entries
		writtenSet := make(map[string]bool)
		for _, file := range writtenFiles {
			writtenSet[file] = true
		}

		t.Logf("🌳 Files in filtered tree: %d", len(result.FlatTree.Entries))
		for i, entry := range result.FlatTree.Entries {
			if i < 20 { // Show first 20 filtered files
				t.Logf("   [%d] %s (type=%s, mode=0o%o)", i+1, entry.Path, entry.Type, entry.Mode)
			}

			if !writtenSet[entry.Path] {
				missingCount++
				missingFiles = append(missingFiles, entry.Path)
				if len(missingFiles) <= 15 { // Show first 15 missing files with details
					t.Logf("   MISSING: %s (type=%s, mode=0o%o, hash=%s)",
						entry.Path, entry.Type, entry.Mode, entry.Hash.String())
				}
			}
		}

		if len(missingFiles) > 15 {
			t.Logf("   ... and %d more missing files", len(missingFiles)-15)
		}

		t.Logf("📊 Analysis:")
		t.Logf("   • Files in FlatTree: %d", len(result.FlatTree.Entries))
		t.Logf("   • Files written to disk: %d", len(writtenFiles))
		t.Logf("   • Files missing from disk: %d", missingCount)
		t.Logf("   • Callback reported written: %d", finalWritten)
		t.Logf("   • Callback reported failed: %d", finalFailed)

		// Show what the callbacks actually reported
		t.Logf("🔄 Callback Details:")
		if len(tracker.writtenList) <= 20 {
			t.Logf("   Files reported as written by callback:")
			for i, file := range tracker.writtenList {
				t.Logf("     [%d] %s", i+1, file)
			}
		} else {
			t.Logf("   First 20 files reported as written by callback:")
			for i, file := range tracker.writtenList[:20] {
				t.Logf("     [%d] %s", i+1, file)
			}
			t.Logf("     ... and %d more", len(tracker.writtenList)-20)
		}

		if len(tracker.failedList) > 0 {
			t.Logf("   Files reported as failed by callback:")
			for i, file := range tracker.failedList {
				t.Logf("     [%d] %s", i+1, file)
			}
		}

		// This proves the discrepancy: we expect FilteredFiles == finalWritten, but we found missing files
		if missingCount != (result.FilteredFiles - int(finalWritten)) {
			t.Logf("🚨 INCONSISTENCY DETECTED!")
			t.Logf("   Expected missing files: %d (FilteredFiles - Written = %d - %d)",
				result.FilteredFiles-int(finalWritten), result.FilteredFiles, finalWritten)
			t.Logf("   Actual missing files found: %d", missingCount)
		}

		// Check if callback reported count matches filesystem count
		if len(writtenFiles) != int(finalWritten) {
			t.Logf("🚨 CALLBACK-FILESYSTEM MISMATCH!")
			t.Logf("   Callback reported: %d written files", finalWritten)
			t.Logf("   Filesystem shows: %d written files", len(writtenFiles))
		}

		// Check if callback reported count matches tracker list length
		if len(tracker.writtenList) != int(finalWritten) {
			t.Logf("🚨 CALLBACK INTERNAL INCONSISTENCY!")
			t.Logf("   Atomic counter: %d", finalWritten)
			t.Logf("   Written list: %d", len(tracker.writtenList))
		}
	}

	// Tree structure printing removed for cleaner output

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
			// Generated TypeScript files (these often have missing blobs in shallow clones)
			"**/*.gen.ts",
			"**/*_gen.ts",
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
	expectedTotalFiles := 18347  // Only files, not directories (same as small test)
	expectedFilteredFiles := 524 // After excluding generated .gen.ts files with enhanced pattern matching
	expectedWrittenFiles := 524  // All filtered files should be written (perfect success rate)
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
	// Now FilteredFiles only counts files (not directories), so success rate should be 100%
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
}
