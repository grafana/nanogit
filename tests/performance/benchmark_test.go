package performance

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BenchmarkSuite manages the performance benchmark suite with containerized Git server
type BenchmarkSuite struct {
	clients      []GitClient
	collector    *MetricsCollector
	config       BenchmarkConfig
	gitServer    *GitServer
	repositories []*Repository
}

// NewBenchmarkSuite creates a new benchmark suite with containerized Gitea server
func NewBenchmarkSuite(ctx context.Context, networkLatency time.Duration) (*BenchmarkSuite, error) {
	// Create Gitea server with optional network latency
	gitServer, err := NewGitServer(ctx, networkLatency)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git server: %w", err)
	}

	// Provision test repositories
	repositories, err := gitServer.ProvisionTestRepositories(ctx)
	if err != nil {
		gitServer.Cleanup(ctx)
		return nil, fmt.Errorf("failed to provision test repositories: %w", err)
	}

	// Generate test data for repositories using git CLI client
	gitCLI, err := NewGitCLIClientWrapper()
	if err != nil {
		gitServer.Cleanup(ctx)
		return nil, fmt.Errorf("failed to create git CLI client for test data: %w", err)
	}

	for i, repo := range repositories {
		spec := GetStandardSpecs()[i]
		generator := NewTestDataGenerator(repo.AuthURL(), gitCLI)
		if err := generator.GenerateRepository(ctx, spec); err != nil {
			gitServer.Cleanup(ctx)
			return nil, fmt.Errorf("failed to generate test data for %s repository: %w", spec.Name, err)
		}
	}

	// Initialize clients
	var allClients []GitClient
	allClients = append(allClients, NewNanogitClientWrapper())
	allClients = append(allClients, NewGoGitClientWrapper())

	gitCLIWrapper, err := NewGitCLIClientWrapper()
	if err != nil {
		gitServer.Cleanup(ctx)
		return nil, fmt.Errorf("failed to create git CLI client: %w", err)
	}
	allClients = append(allClients, gitCLIWrapper)

	return &BenchmarkSuite{
		clients:      allClients,
		collector:    NewMetricsCollector(),
		gitServer:    gitServer,
		repositories: repositories,
		config: BenchmarkConfig{
			Iterations: 3,
			Timeout:    5 * time.Minute,
		},
	}, nil
}

// Cleanup stops the Git server and cleans up resources
func (s *BenchmarkSuite) Cleanup(ctx context.Context) error {
	if s.gitServer != nil {
		return s.gitServer.Cleanup(ctx)
	}
	return nil
}

// GetRepository returns a repository by size specification
func (s *BenchmarkSuite) GetRepository(size string) *Repository {
	for _, repo := range s.repositories {
		if strings.Contains(repo.Name, size) {
			return repo
		}
	}
	return nil
}

// TestFileOperationsPerformance tests file create/update/delete operations
func TestFileOperationsPerformance(t *testing.T) {
	// Skip if not explicitly enabled (these tests require Docker and are slow)
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	// Get network latency from environment (default 0ms)
	latencyMs := 0
	if envLatency := os.Getenv("PERF_TEST_LATENCY_MS"); envLatency != "" {
		if parsed, err := strconv.Atoi(envLatency); err == nil {
			latencyMs = parsed
		}
	}
	networkLatency := time.Duration(latencyMs) * time.Millisecond

	suite, err := NewBenchmarkSuite(ctx, networkLatency)
	require.NoError(t, err)
	defer suite.Cleanup(ctx)

	testCases := []struct {
		name      string
		repoSize  string
		fileCount int
	}{
		{"small_repo", "small", 50},
		{"medium_repo", "medium", 500},
		// {"large_repo", "large", 2000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := suite.GetRepository(tc.repoSize)
			require.NotNil(t, repo, "Repository not found for size: %s", tc.repoSize)

			ctx, cancel := context.WithTimeout(ctx, suite.config.Timeout)
			defer cancel()

			for _, client := range suite.clients {
				t.Run(client.Name(), func(t *testing.T) {
					// Run each operation 5 times for better statistical data
					for i := 0; i < 5; i++ {
						// Test file creation
						suite.collector.RecordOperation(
							client.Name(), "CreateFile", tc.name, tc.repoSize, tc.fileCount,
							func() error {
								filename := fmt.Sprintf("test/new_file_%d.txt", i)
								return client.CreateFile(ctx, repo.AuthURL(), filename, "test content", "Add test file")
							},
						)

						// Test file update
						suite.collector.RecordOperation(
							client.Name(), "UpdateFile", tc.name, tc.repoSize, tc.fileCount,
							func() error {
								filename := fmt.Sprintf("test/new_file_%d.txt", i)
								return client.UpdateFile(ctx, repo.AuthURL(), filename, "updated content", "Update test file")
							},
						)

						// Test file deletion
						suite.collector.RecordOperation(
							client.Name(), "DeleteFile", tc.name, tc.repoSize, tc.fileCount,
							func() error {
								filename := fmt.Sprintf("test/new_file_%d.txt", i)
								return client.DeleteFile(ctx, repo.AuthURL(), filename, "Delete test file")
							},
						)
					}
				})
			}
		})
	}

	// Save results
	err = suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// TestCompareCommitsPerformance tests commit comparison operations
func TestCompareCommitsPerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()
	latencyMs := 0
	if envLatency := os.Getenv("PERF_TEST_LATENCY_MS"); envLatency != "" {
		if parsed, err := strconv.Atoi(envLatency); err == nil {
			latencyMs = parsed
		}
	}
	networkLatency := time.Duration(latencyMs) * time.Millisecond

	suite, err := NewBenchmarkSuite(ctx, networkLatency)
	require.NoError(t, err)
	defer suite.Cleanup(ctx)

	testCases := []struct {
		name         string
		repoSize     string
		baseCommit   string
		headCommit   string
		expectedDiff int
	}{
		{"adjacent_commits", "medium", "HEAD~1", "HEAD", 1},
		{"distant_commits", "medium", "HEAD~10", "HEAD", 10},
		{"large_diff", "large", "HEAD~20", "HEAD", 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := suite.GetRepository(tc.repoSize)
			require.NotNil(t, repo, "Repository not found for size: %s", tc.repoSize)

			ctx, cancel := context.WithTimeout(ctx, suite.config.Timeout)
			defer cancel()

			for _, client := range suite.clients {
				t.Run(client.Name(), func(t *testing.T) {
					suite.collector.RecordOperation(
						client.Name(), "CompareCommits", tc.name, tc.repoSize, tc.expectedDiff,
						func() error {
							_, err := client.CompareCommits(ctx, repo.AuthURL(), tc.baseCommit, tc.headCommit)
							return err
						},
					)
				})
			}
		})
	}

	// Save results
	err = suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// TestGetFlatTreePerformance tests tree listing operations
func TestGetFlatTreePerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()
	latencyMs := 0
	if envLatency := os.Getenv("PERF_TEST_LATENCY_MS"); envLatency != "" {
		if parsed, err := strconv.Atoi(envLatency); err == nil {
			latencyMs = parsed
		}
	}
	networkLatency := time.Duration(latencyMs) * time.Millisecond

	suite, err := NewBenchmarkSuite(ctx, networkLatency)
	require.NoError(t, err)
	defer suite.Cleanup(ctx)

	testCases := []struct {
		name      string
		repoSize  string
		ref       string
		fileCount int
	}{
		{"small_tree", "small", "HEAD", 50},
		{"medium_tree", "medium", "HEAD", 500},
		{"large_tree", "large", "HEAD", 2000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := suite.GetRepository(tc.repoSize)
			require.NotNil(t, repo, "Repository not found for size: %s", tc.repoSize)

			ctx, cancel := context.WithTimeout(ctx, suite.config.Timeout)
			defer cancel()

			for _, client := range suite.clients {
				t.Run(client.Name(), func(t *testing.T) {
					suite.collector.RecordOperation(
						client.Name(), "GetFlatTree", tc.name, tc.repoSize, tc.fileCount,
						func() error {
							_, err := client.GetFlatTree(ctx, repo.AuthURL(), tc.ref)
							return err
						},
					)
				})
			}
		})
	}

	// Save results
	err = suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// TestBulkOperationsPerformance tests bulk file operations
func TestBulkOperationsPerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()
	latencyMs := 0
	if envLatency := os.Getenv("PERF_TEST_LATENCY_MS"); envLatency != "" {
		if parsed, err := strconv.Atoi(envLatency); err == nil {
			latencyMs = parsed
		}
	}
	networkLatency := time.Duration(latencyMs) * time.Millisecond

	suite, err := NewBenchmarkSuite(ctx, networkLatency)
	require.NoError(t, err)
	defer suite.Cleanup(ctx)

	testCases := []struct {
		name      string
		repoSize  string
		fileCount int
	}{
		{"bulk_10_files", "small", 10},
		{"bulk_100_files", "medium", 100},
		{"bulk_1000_files", "large", 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := suite.GetRepository(tc.repoSize)
			require.NotNil(t, repo, "Repository not found for size: %s", tc.repoSize)

			ctx, cancel := context.WithTimeout(ctx, suite.config.Timeout)
			defer cancel()

			// Generate test files
			files := generateTestFiles(tc.fileCount)

			for _, client := range suite.clients {
				t.Run(client.Name(), func(t *testing.T) {
					suite.collector.RecordOperation(
						client.Name(), "BulkCreateFiles", tc.name, tc.repoSize, tc.fileCount,
						func() error {
							return client.BulkCreateFiles(ctx, repo.AuthURL(), files, fmt.Sprintf("Bulk create %d files", tc.fileCount))
						},
					)
				})
			}
		})
	}

	// Save results
	err = suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// Helper functions

// generateTestFiles creates test file data for bulk operations
func generateTestFiles(count int) []FileChange {
	files := make([]FileChange, count)

	for i := 0; i < count; i++ {
		files[i] = FileChange{
			Path:    fmt.Sprintf("bulk/file_%04d.txt", i),
			Content: fmt.Sprintf("This is test file number %d\nGenerated for bulk operation testing\n", i),
			Action:  "create",
		}
	}

	return files
}

// BenchmarkFileOperations provides Go benchmark functions for more detailed performance analysis
func BenchmarkFileOperations(b *testing.B) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		b.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	// No latency for benchmarks (they need to be fast)
	suite, err := NewBenchmarkSuite(ctx, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer suite.Cleanup(ctx)

	repo := suite.GetRepository("small")
	if repo == nil {
		b.Fatal("Small repository not found")
	}

	for _, client := range suite.clients {
		b.Run(fmt.Sprintf("%s_CreateFile", client.Name()), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				path := fmt.Sprintf("bench/file_%d.txt", i)
				err := client.CreateFile(ctx, repo.AuthURL(), path, "benchmark content", "Benchmark create")
				if err != nil {
					b.Error(err)
				}
			}
		})

		b.Run(fmt.Sprintf("%s_GetFlatTree", client.Name()), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := client.GetFlatTree(ctx, repo.AuthURL(), "HEAD")
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

