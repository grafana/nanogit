package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/stretchr/testify/require"
)

// TestCloneGrafanaGrafana tests cloning performance for the grafana/grafana repository
// with subpath filtering. This is a real-world performance test against a large public repository.
func TestCloneGrafanaGrafana(t *testing.T) {
	if os.Getenv("RUN_CLONE_PERF_TESTS") != "true" {
		t.Skip("Clone performance tests disabled. Set RUN_CLONE_PERF_TESTS=true to run.")
	}

	ctx := context.Background()

	// Repository configuration
	repoURL := "https://github.com/grafana/grafana.git"

	// Test scenarios with different subpaths
	// Expected file counts are for commit fd14d4a5ed3ad8dfd5948a9f2d7b9074e15ea655
	// These may need updating as the grafana/grafana repository evolves
	scenarios := []struct {
		name              string
		includePaths      []string
		description       string
		expectedFileCount int // Expected number of files for this filter at the pinned commit
	}{
		{
			name:              "pkg_api_subpath",
			includePaths:      []string{"pkg/api/**"},
			description:       "Clone only pkg/api directory",
			expectedFileCount: 159,
		},
		{
			name:              "docs_sources",
			includePaths:      []string{"docs/sources/**/*.md"},
			description:       "Clone only markdown files in docs/sources",
			expectedFileCount: 69,
		},
		{
			name:              "go_files_root",
			includePaths:      []string{"*.go", "go.mod", "go.sum"},
			description:       "Clone only Go files in root directory",
			expectedFileCount: 4,
		},
		{
			name:              "scripts_and_makefile",
			includePaths:      []string{"scripts/**", "Makefile", "*.mk"},
			description:       "Clone scripts directory and makefiles",
			expectedFileCount: 142,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create temporary directory for clone
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("nanogit_clone_perf_%s_", scenario.name))
			require.NoError(t, err, "Failed to create temp directory")
			defer os.RemoveAll(tempDir)

			clonePath := filepath.Join(tempDir, "clone")

			// Create nanogit client
			client, err := nanogit.NewHTTPClient(repoURL)
			require.NoError(t, err, "Failed to create nanogit client")

			// Get main branch reference
			ref, err := client.GetRef(ctx, "refs/heads/main")
			require.NoError(t, err, "Failed to get main branch reference")

			t.Logf("Cloning grafana/grafana with scenario: %s", scenario.description)
			t.Logf("Include paths: %v", scenario.includePaths)
			t.Logf("Target commit: %s", ref.Hash.String()[:8])

			// Measure clone time
			startTime := time.Now()

			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         clonePath,
				Hash:         ref.Hash,
				IncludePaths: scenario.includePaths,
			})

			duration := time.Since(startTime)

			require.NoError(t, err, "Clone operation failed")
			require.NotNil(t, result, "Clone result should not be nil")

			// Log results
			t.Logf("Clone completed successfully")
			t.Logf("Duration: %v", duration)
			t.Logf("Total files in repository: %d", result.TotalFiles)
			t.Logf("Filtered files cloned: %d", result.FilteredFiles)
			t.Logf("Clone path: %s", result.Path)

			// Verify files were written to disk
			require.Greater(t, result.FilteredFiles, 0, "Should have cloned at least one file")

			// Check that the clone directory exists and contains files
			entries, err := os.ReadDir(clonePath)
			require.NoError(t, err, "Should be able to read clone directory")
			require.Greater(t, len(entries), 0, "Clone directory should contain files")

			// Assert exact file count if expected count is set (non-zero)
			if scenario.expectedFileCount > 0 {
				require.Equal(t, scenario.expectedFileCount, result.FilteredFiles,
					"Expected exactly %d files for scenario %s, but got %d. "+
						"If the grafana/grafana repository has changed, update the expectedFileCount in the test.",
					scenario.expectedFileCount, scenario.name, result.FilteredFiles)
			} else {
				t.Logf("WARNING: expectedFileCount not set for scenario %s. Current count: %d",
					scenario.name, result.FilteredFiles)
				t.Logf("Please update the test with: expectedFileCount: %d", result.FilteredFiles)
			}

			// Performance assertions - these are guidelines and can be adjusted
			// based on network conditions and expected performance
			t.Logf("Performance metrics:")
			t.Logf("  - Files/second: %.2f", float64(result.FilteredFiles)/duration.Seconds())

			// Warn if clone takes too long (these are soft warnings, not hard failures)
			if duration > 2*time.Minute {
				t.Logf("WARNING: Clone took longer than 2 minutes (%v)", duration)
			}

			// Verify filtering worked as expected
			ratio := float64(result.FilteredFiles) / float64(result.TotalFiles)
			t.Logf("  - Filter ratio: %.2f%% of files cloned", ratio*100)
			require.Less(t, result.FilteredFiles, result.TotalFiles,
				"Filtered files should be less than total files (subpath filtering should work)")
		})
	}
}

// TestCloneGrafanaGrafanaFullRepository tests cloning the entire grafana/grafana repository
// without filtering. This is useful for benchmarking full clone performance.
func TestCloneGrafanaGrafanaFullRepository(t *testing.T) {
	if os.Getenv("RUN_CLONE_PERF_TESTS") != "true" {
		t.Skip("Clone performance tests disabled. Set RUN_CLONE_PERF_TESTS=true to run.")
	}

	// Additional guard for full clone - requires explicit opt-in
	if os.Getenv("RUN_FULL_CLONE_TEST") != "true" {
		t.Skip("Full repository clone test disabled. Set RUN_FULL_CLONE_TEST=true to run.")
	}

	ctx := context.Background()

	// Repository configuration
	repoURL := "https://github.com/grafana/grafana.git"

	// Create temporary directory for clone
	tempDir, err := os.MkdirTemp("", "nanogit_clone_perf_full_")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	clonePath := filepath.Join(tempDir, "clone")

	// Create nanogit client
	client, err := nanogit.NewHTTPClient(repoURL)
	require.NoError(t, err, "Failed to create nanogit client")

	// Get main branch reference
	ref, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err, "Failed to get main branch reference")

	t.Logf("Cloning full grafana/grafana repository")
	t.Logf("Target commit: %s", ref.Hash.String()[:8])
	t.Logf("Clone path: %s", clonePath)

	// Measure clone time
	startTime := time.Now()

	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path: clonePath,
		Hash: ref.Hash,
		// No include/exclude paths - clone everything
	})

	duration := time.Since(startTime)

	require.NoError(t, err, "Clone operation failed")
	require.NotNil(t, result, "Clone result should not be nil")

	// Log results
	t.Logf("Clone completed successfully")
	t.Logf("Duration: %v", duration)
	t.Logf("Total files cloned: %d", result.TotalFiles)
	t.Logf("Clone path: %s", result.Path)

	// Verify files were written to disk
	require.Greater(t, result.TotalFiles, 0, "Should have cloned at least one file")

	// Check that the clone directory exists and contains files
	entries, err := os.ReadDir(clonePath)
	require.NoError(t, err, "Should be able to read clone directory")
	require.Greater(t, len(entries), 0, "Clone directory should contain files")

	// Performance metrics
	t.Logf("Performance metrics:")
	t.Logf("  - Files/second: %.2f", float64(result.TotalFiles)/duration.Seconds())

	// Warn if clone takes too long
	if duration > 5*time.Minute {
		t.Logf("WARNING: Full clone took longer than 5 minutes (%v)", duration)
	}
}
