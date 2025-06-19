package performance

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// determineRequiredRepositories analyzes command-line arguments to determine which repositories are needed
func determineRequiredRepositories() []string {
	// Check if explicitly set via environment variable first
	if envRepos := os.Getenv("PERF_TEST_REPOS"); envRepos != "" {
		repos := strings.Split(envRepos, ",")
		var result []string
		for _, repo := range repos {
			repo = strings.TrimSpace(repo)
			if repo != "" {
				result = append(result, repo)
			}
		}
		if len(result) > 0 {
			fmt.Printf("Using repositories from PERF_TEST_REPOS: %v\n", result)
			return result
		}
	}
	
	// Get test run arguments
	args := os.Args
	
	// Look for -run flag patterns that indicate specific repository sizes
	for i, arg := range args {
		if arg == "-run" && i+1 < len(args) {
			pattern := args[i+1]
			result := extractRepoSizesFromPattern(pattern)
			if len(result) < 4 { // Only log if we're doing selective provisioning
				fmt.Printf("Detected test pattern '%s', provisioning repositories: %v\n", pattern, result)
			}
			return result
		}
	}
	
	// Default: provision all repositories if no specific pattern found
	return []string{"small", "medium", "large", "xlarge"}
}

// extractRepoSizesFromPattern extracts repository sizes from test name patterns
func extractRepoSizesFromPattern(pattern string) []string {
	repoSizes := make(map[string]bool)
	
	// Check for size-specific patterns - be more thorough
	pattern = strings.ToLower(pattern)
	
	// Look for explicit size mentions in various contexts:
	// - ".*small" patterns 
	// - "small_repo" patterns
	// - "TestFunctionName.*small" patterns
	if strings.Contains(pattern, "small") {
		repoSizes["small"] = true
	}
	if strings.Contains(pattern, "medium") {
		repoSizes["medium"] = true
	}
	// Check for "large" but not "xlarge"
	if strings.Contains(pattern, "large") {
		if strings.Contains(pattern, "xlarge") {
			repoSizes["xlarge"] = true
		} else {
			repoSizes["large"] = true
		}
	}
	
	// Convert map to slice
	result := make([]string, 0, len(repoSizes))
	// Maintain consistent order
	for _, size := range []string{"small", "medium", "large", "xlarge"} {
		if repoSizes[size] {
			result = append(result, size)
		}
	}
	
	// If no specific sizes found, check if it's a known size-specific make target pattern
	if len(result) == 0 {
		// Check for patterns that match our make targets
		makeTargetPatterns := map[string][]string{
			"testfileoperationsperformance.*small|testcomparecommitsperformance.*small|testgetflattreeperformance.*small|testbulkoperationsperformance.*small": {"small"},
			"testfileoperationsperformance.*medium|testcomparecommitsperformance.*medium|testgetflattreeperformance.*medium|testbulkoperationsperformance.*medium": {"medium"},
			"testfileoperationsperformance.*large|testcomparecommitsperformance.*large|testgetflattreeperformance.*large|testbulkoperationsperformance.*large": {"large"},
			"testfileoperationsperformance.*xlarge|testcomparecommitsperformance.*xlarge|testgetflattreeperformance.*xlarge|testbulkoperationsperformance.*xlarge": {"xlarge"},
		}
		
		for makePattern, sizes := range makeTargetPatterns {
			if strings.Contains(pattern, makePattern) {
				return sizes
			}
		}
		
		// Default: provision all repositories if no specific pattern found
		return []string{"small", "medium", "large", "xlarge"}
	}
	
	return result
}

// Global suite instance to be shared across all test functions
var (
	globalSuite     *BenchmarkSuite
	globalSuiteOnce sync.Once
	globalSuiteMux  sync.RWMutex
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

	// Provision test repositories (extract and mount pre-created archives)
	repositories, err := gitServer.ProvisionTestRepositories(ctx)
	if err != nil {
		gitServer.Cleanup(ctx)
		return nil, fmt.Errorf("failed to provision test repositories: %w", err)
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
			Timeout: 10 * time.Minute,
		},
	}, nil
}

// NewBenchmarkSuiteWithSelectedRepos creates a new benchmark suite with only selected repositories
func NewBenchmarkSuiteWithSelectedRepos(ctx context.Context, networkLatency time.Duration, repoSizes []string) (*BenchmarkSuite, error) {
	// Create Gitea server with optional network latency
	gitServer, err := NewGitServer(ctx, networkLatency)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git server: %w", err)
	}

	// Provision only selected repositories
	repositories, err := gitServer.ProvisionSelectedRepositories(ctx, repoSizes)
	if err != nil {
		gitServer.Cleanup(ctx)
		return nil, fmt.Errorf("failed to provision selected repositories: %w", err)
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
			Timeout: 10 * time.Minute,
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

// getGlobalSuite returns the shared benchmark suite (should be created by TestMain)
func getGlobalSuite() *BenchmarkSuite {
	globalSuiteMux.RLock()
	defer globalSuiteMux.RUnlock()
	return globalSuite
}

// cleanupGlobalSuite cleans up the global suite (should be called in TestMain)
func cleanupGlobalSuite(ctx context.Context) error {
	globalSuiteMux.Lock()
	defer globalSuiteMux.Unlock()

	if globalSuite != nil {
		err := globalSuite.Cleanup(ctx)
		globalSuite = nil
		return err
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

	// Use the global suite (repository setup already done)
	suite := getGlobalSuite()
	require.NotNil(t, suite, "Global suite not initialized - TestMain should have set this up")

	allTestCases := []struct {
		name      string
		repoSize  string
		fileCount int
	}{
		{"small_repo", "small", 50},
		{"medium_repo", "medium", 500},
		{"large_repo", "large", 2000},
		{"xlarge_repo", "xlarge", 10000},
	}

	// Only run test cases for repositories that are actually available
	var testCases []struct {
		name      string
		repoSize  string
		fileCount int
	}
	for _, tc := range allTestCases {
		if suite.GetRepository(tc.repoSize) != nil {
			testCases = append(testCases, tc)
		}
	}

	if len(testCases) == 0 {
		t.Skip("No repositories available for testing")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := suite.GetRepository(tc.repoSize)
			require.NotNil(t, repo, "Repository not found for size: %s", tc.repoSize)

			ctx, cancel := context.WithTimeout(ctx, suite.config.Timeout)
			defer cancel()

			for _, client := range suite.clients {
				t.Run(client.Name(), func(t *testing.T) {
					// Run each operation 3 times for better statistical data
					for i := 0; i < 3; i++ {
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
	err := suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// TestCompareCommitsPerformance tests commit comparison operations
func TestCompareCommitsPerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	// Use the global suite (repository setup already done)
	suite := getGlobalSuite()
	require.NotNil(t, suite, "Global suite not initialized - TestMain should have set this up")

	allTestCases := []struct {
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

	// Only run test cases for repositories that are actually available
	var testCases []struct {
		name         string
		repoSize     string
		baseCommit   string
		headCommit   string
		expectedDiff int
	}
	for _, tc := range allTestCases {
		if suite.GetRepository(tc.repoSize) != nil {
			testCases = append(testCases, tc)
		}
	}

	if len(testCases) == 0 {
		t.Skip("No repositories available for testing")
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
	err := suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// TestGetFlatTreePerformance tests tree listing operations
func TestGetFlatTreePerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	// Use the global suite (repository setup already done)
	suite := getGlobalSuite()
	require.NotNil(t, suite, "Global suite not initialized - TestMain should have set this up")

	allTestCases := []struct {
		name      string
		repoSize  string
		ref       string
		fileCount int
	}{
		{"small_tree", "small", "HEAD", 50},
		{"medium_tree", "medium", "HEAD", 500},
		{"large_tree", "large", "HEAD", 2000},
	}

	// Only run test cases for repositories that are actually available
	var testCases []struct {
		name      string
		repoSize  string
		ref       string
		fileCount int
	}
	for _, tc := range allTestCases {
		if suite.GetRepository(tc.repoSize) != nil {
			testCases = append(testCases, tc)
		}
	}

	if len(testCases) == 0 {
		t.Skip("No repositories available for testing")
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
	err := suite.collector.SaveReport("./reports")
	require.NoError(t, err)
}

// TestBulkOperationsPerformance tests bulk file operations
func TestBulkOperationsPerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	// Use the global suite (repository setup already done)
	suite := getGlobalSuite()
	require.NotNil(t, suite, "Global suite not initialized - TestMain should have set this up")

	allTestCases := []struct {
		name      string
		repoSize  string
		fileCount int
	}{
		{"bulk_10_files", "small", 10},
		{"bulk_100_files", "medium", 100},
		{"bulk_1000_files", "large", 1000},
	}

	// Only run test cases for repositories that are actually available
	var testCases []struct {
		name      string
		repoSize  string
		fileCount int
	}
	for _, tc := range allTestCases {
		if suite.GetRepository(tc.repoSize) != nil {
			testCases = append(testCases, tc)
		}
	}

	if len(testCases) == 0 {
		t.Skip("No repositories available for testing")
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
	err := suite.collector.SaveReport("./reports")
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

// TestMain handles global setup and cleanup for all performance tests
func TestMain(m *testing.M) {
	// Skip if not explicitly enabled
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		os.Exit(m.Run())
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

	// Determine which repositories are needed based on test patterns
	requiredRepos := determineRequiredRepositories()
	fmt.Printf("Setting up global benchmark suite with repositories: %v (this may take 1-2 minutes)...\n", requiredRepos)
	
	// Create global suite with only required repositories
	globalSuiteMux.Lock()
	var err error
	globalSuite, err = NewBenchmarkSuiteWithSelectedRepos(ctx, networkLatency, requiredRepos)
	globalSuiteMux.Unlock()
	
	if err != nil {
		fmt.Printf("Failed to create global benchmark suite: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Global benchmark suite ready with %d repositories\n", len(globalSuite.repositories))

	// Run tests
	exitCode := m.Run()

	// Cleanup
	fmt.Printf("Cleaning up global benchmark suite...\n")
	if err := cleanupGlobalSuite(ctx); err != nil {
		fmt.Printf("Warning: failed to cleanup global suite: %v\n", err)
	}

	os.Exit(exitCode)
}

// BenchmarkFileOperations provides Go benchmark functions for more detailed performance analysis
func BenchmarkFileOperations(b *testing.B) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		b.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	// Use the global suite (repository setup already done)
	suite := getGlobalSuite()
	if suite == nil {
		b.Fatal("Global suite not initialized - TestMain should have set this up")
	}

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
