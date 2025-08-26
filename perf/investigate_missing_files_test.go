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
	"github.com/grafana/nanogit/protocol/hash"
)

// TestInvestigateMissingFiles creates a persistent clone to investigate the exact 10 missing files
func TestInvestigateMissingFiles(t *testing.T) {
	ctx := context.Background()
	// Use nil logger to suppress debug output for faster investigation
	// logger := &cloneTestLogger{}
	// ctx = nanolog.ToContext(ctx, logger)

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

	// Create temporary directory for clone - DO NOT CLEANUP
	tempDir, err := os.MkdirTemp("", "nanogit-investigation-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	// Explicitly NO defer cleanup - we want to examine this directory
	t.Logf("📁 INVESTIGATION Clone destination (will NOT be cleaned up): %s", tempDir)

	// Track failed files specifically
	tracker := &investigationProgressTracker{
		startTime:   time.Now(),
		failedFiles: make([]string, 0),
	}

	// Performance benchmark: Clone ALL files for 100% completeness test
	start := time.Now()
	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path: tempDir,
		Hash: commitHash,
		// NO IncludePaths - fetch everything
		// NO ExcludePaths - fetch everything for 100% completeness
		OnFileWritten: tracker.onFileWritten,
		OnFileFailed:  tracker.onFileFailed,
	})
	cloneDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Investigation clone failed: %v", err)
	}

	// Get final stats
	finalWritten := atomic.LoadInt64(&tracker.filesWritten)
	finalFailed := atomic.LoadInt64(&tracker.filesFailed)
	finalSize := atomic.LoadInt64(&tracker.totalSize)

	t.Logf("🔍 Investigation Results:")
	t.Logf("   • Total files in repository: %d", result.TotalFiles)
	t.Logf("   • Files matching filters: %d", result.FilteredFiles)
	t.Logf("   • Files successfully written: %d", finalWritten)
	t.Logf("   • Files failed: %d", finalFailed)
	t.Logf("   • Total data written: %.1f MB", float64(finalSize)/(1024*1024))
	t.Logf("   • Clone time: %v", cloneDuration)
	
	// List all failed files
	t.Logf("🚨 FAILED FILES (%d total):", len(tracker.failedFiles))
	for i, failedFile := range tracker.failedFiles {
		t.Logf("   %d. %s", i+1, failedFile)
	}

	// Look specifically for .gen.ts files in the cloned directory
	t.Logf("🔍 Examining .gen.ts files in grafana-schema directory...")
	genTsFiles := make([]string, 0)
	schemaDir := filepath.Join(tempDir, "packages", "grafana-schema", "src")
	if _, err := os.Stat(schemaDir); err == nil {
		err := filepath.Walk(schemaDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), ".gen.ts") {
				relPath, _ := filepath.Rel(tempDir, path)
				genTsFiles = append(genTsFiles, relPath)
			}
			return nil
		})
		if err != nil {
			t.Logf("Error walking schema directory: %v", err)
		}
	} else {
		t.Logf("Schema directory does not exist: %s", schemaDir)
	}

	t.Logf("📄 Found .gen.ts files in nanogit clone (%d total):", len(genTsFiles))
	for i, genFile := range genTsFiles {
		t.Logf("   %d. %s", i+1, genFile)
	}

	// Create a Git CLI clone for comparison in a separate directory
	gitTempDir, err := os.MkdirTemp("", "git-cli-comparison-*")
	if err != nil {
		t.Fatalf("Failed to create git temp dir: %v", err)
	}
	// Also don't cleanup - we want to examine both
	t.Logf("📁 Git CLI comparison clone (will NOT be cleaned up): %s", gitTempDir)

	// NOTE: We'll manually create the Git CLI clone after this test completes
	// to compare the results

	t.Logf("🎯 INVESTIGATION SUMMARY:")
	t.Logf("   • Nanogit clone directory: %s", tempDir)
	t.Logf("   • Git CLI comparison directory: %s", gitTempDir)
	t.Logf("   • Run this to create Git CLI comparison:")
	t.Logf("     cd %s && git clone --depth 1 https://github.com/grafana/grafana.git .", gitTempDir)
	t.Logf("   • Then run: find %s -name '*.gen.ts' | wc -l", gitTempDir)
	t.Logf("   • And: find %s -name '*.gen.ts' | wc -l", tempDir)
}

// investigationProgressTracker tracks files written/failed with specific file paths
type investigationProgressTracker struct {
	startTime    time.Time
	filesWritten int64
	filesFailed  int64
	totalSize    int64
	failedFiles  []string
}

func (t *investigationProgressTracker) onFileWritten(path string, size int64) {
	atomic.AddInt64(&t.filesWritten, 1)
	atomic.AddInt64(&t.totalSize, size)
}

func (t *investigationProgressTracker) onFileFailed(path string, err error) {
	atomic.AddInt64(&t.filesFailed, 1)
	t.failedFiles = append(t.failedFiles, fmt.Sprintf("%s (error: %v)", path, err))
}