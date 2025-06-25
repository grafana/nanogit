package performance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ClientConsistencyTest verifies that all Git clients produce identical results
type ClientConsistencyTest struct {
	suite         *BenchmarkSuite
	gitCLIWorkdir string
}

// NewClientConsistencyTest creates a new consistency test
func NewClientConsistencyTest(ctx context.Context) (*ClientConsistencyTest, error) {
	suite, err := NewBenchmarkSuite(ctx, 0) // No network latency for consistency tests
	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark suite: %w", err)
	}

	// Create temporary directory for Git CLI operations
	tempDir, err := os.MkdirTemp("", "nanogit_consistency_test_")
	if err != nil {
		suite.Cleanup(ctx)
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &ClientConsistencyTest{
		suite:         suite,
		gitCLIWorkdir: tempDir,
	}, nil
}

// Cleanup cleans up test resources
func (c *ClientConsistencyTest) Cleanup(ctx context.Context) error {
	var errs []error
	if c.suite != nil {
		if err := c.suite.Cleanup(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if c.gitCLIWorkdir != "" {
		if err := os.RemoveAll(c.gitCLIWorkdir); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

// cloneWithGitCLI clones a repository using Git CLI for verification
func (c *ClientConsistencyTest) cloneWithGitCLI(ctx context.Context, repoURL, localPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, localPath)
	cmd.Dir = c.gitCLIWorkdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}
	return nil
}

// getFileContentWithGitCLI gets file content using Git CLI
func (c *ClientConsistencyTest) getFileContentWithGitCLI(ctx context.Context, repoPath, filePath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("HEAD:%s", filePath))
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}

// listFilesWithGitCLI lists all files in the repository using Git CLI
func (c *ClientConsistencyTest) listFilesWithGitCLI(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "--name-only", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// getCommitCountWithGitCLI gets the number of commits using Git CLI
func (c *ClientConsistencyTest) getCommitCountWithGitCLI(ctx context.Context, repoPath string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("git rev-list failed: %w", err)
	}

	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	if err != nil {
		return 0, fmt.Errorf("failed to parse commit count: %w", err)
	}
	return count, nil
}

// getLatestCommitMessageWithGitCLI gets the latest commit message using Git CLI
func (c *ClientConsistencyTest) getLatestCommitMessageWithGitCLI(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%s")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// TestClientConsistency tests that all clients produce identical results
func TestClientConsistency(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	test, err := NewClientConsistencyTest(ctx)
	require.NoError(t, err)
	defer test.Cleanup(ctx)

	// Use small repository for consistency tests
	repo := test.suite.GetRepository("small")
	require.NotNil(t, repo, "Small repository not found")

	t.Run("CreateNewFileConsistency", func(t *testing.T) {
		test.testCreateNewFileConsistency(t, ctx, repo)
	})

	t.Run("UpdateExistingFileConsistency", func(t *testing.T) {
		test.testUpdateExistingFileConsistency(t, ctx, repo)
	})

	t.Run("BulkOperationsConsistency", func(t *testing.T) {
		test.testBulkOperationsConsistency(t, ctx, repo)
	})
}

// testCreateNewFileConsistency tests that all clients create files with identical repository state
func (c *ClientConsistencyTest) testCreateNewFileConsistency(t *testing.T, ctx context.Context, repo *Repository) {
	testFile := "consistency_test/new_file.txt"
	testContent := "This is a new file created for consistency testing"
	commitMessage := "Add new file for consistency test"

	results := make(map[string]struct {
		initialCommitCount int
		finalCommitCount   int
		fileContent        string
		commitMessage      string
		allFiles           []string
	})

	// Test each client
	for _, client := range c.suite.clients {
		clientName := client.Name()

		t.Run(clientName, func(t *testing.T) {
			// Clone repository with Git CLI for verification
			localPath := filepath.Join(c.gitCLIWorkdir, fmt.Sprintf("test_%s", clientName))
			err := c.cloneWithGitCLI(ctx, repo.AuthURL(), localPath)
			require.NoError(t, err, "Failed to clone repository for %s", clientName)

			// Get initial state
			initialCommitCount, err := c.getCommitCountWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get initial commit count for %s", clientName)

			initialFiles, err := c.listFilesWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to list initial files for %s", clientName)

			// Perform operation with the client
			err = client.CreateFile(ctx, repo.AuthURL(), testFile, testContent, commitMessage)
			require.NoError(t, err, "Failed to create file with %s", clientName)

			// Wait a bit for changes to propagate
			time.Sleep(2 * time.Second)

			// Pull latest changes with Git CLI
			cmd := exec.CommandContext(ctx, "git", "pull", "origin", "main")
			cmd.Dir = localPath
			_, err = cmd.CombinedOutput()
			require.NoError(t, err, "Failed to pull changes for %s", clientName)

			// Verify final state
			finalCommitCount, err := c.getCommitCountWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get final commit count for %s", clientName)

			fileContent, err := c.getFileContentWithGitCLI(ctx, localPath, testFile)
			require.NoError(t, err, "Failed to read file content for %s", clientName)

			latestCommitMessage, err := c.getLatestCommitMessageWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get latest commit message for %s", clientName)

			finalFiles, err := c.listFilesWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to list final files for %s", clientName)

			// Verify that only one new commit was added
			require.Equal(t, initialCommitCount+1, finalCommitCount, "Commit count mismatch for %s", clientName)

			// Verify file content matches
			require.Equal(t, testContent, fileContent, "File content mismatch for %s", clientName)

			// Verify commit message matches
			require.Equal(t, commitMessage, latestCommitMessage, "Commit message mismatch for %s", clientName)

			// Verify all original files are still present
			for _, originalFile := range initialFiles {
				require.Contains(t, finalFiles, originalFile, "Original file missing after operation by %s: %s", clientName, originalFile)
			}

			// Verify the new file was added
			require.Contains(t, finalFiles, testFile, "New file not found after operation by %s", clientName)

			// Store results for cross-client comparison
			results[clientName] = struct {
				initialCommitCount int
				finalCommitCount   int
				fileContent        string
				commitMessage      string
				allFiles           []string
			}{
				initialCommitCount: initialCommitCount,
				finalCommitCount:   finalCommitCount,
				fileContent:        fileContent,
				commitMessage:      latestCommitMessage,
				allFiles:           finalFiles,
			}
		})

		// Clean up for next client test (create new file with different name)
		testFile = fmt.Sprintf("consistency_test/new_file_%s.txt", clientName)
	}

	// Verify all clients produce consistent results
	var referenceResult struct {
		initialCommitCount int
		finalCommitCount   int
		fileContent        string
		commitMessage      string
		allFiles           []string
	}
	var referenceClient string

	for clientName, result := range results {
		if referenceClient == "" {
			referenceResult = result
			referenceClient = clientName
			continue
		}

		// All clients should have same commit structure
		require.Equal(t, referenceResult.finalCommitCount-referenceResult.initialCommitCount,
			result.finalCommitCount-result.initialCommitCount,
			"Commit increment mismatch between %s and %s", referenceClient, clientName)

		// All clients should create files with same content
		require.Equal(t, referenceResult.fileContent, result.fileContent,
			"File content mismatch between %s and %s", referenceClient, clientName)

		// All clients should use same commit message
		require.Equal(t, referenceResult.commitMessage, result.commitMessage,
			"Commit message mismatch between %s and %s", referenceClient, clientName)

		// All clients should preserve existing files, and since each creates their own file,
		// subsequent clients should see more files than earlier ones (at least the same number)
		// We can't guarantee exact file count equality since clients run sequentially and add files
		// Just verify that no original files were lost and the new file was added
		require.GreaterOrEqual(t, len(result.allFiles), len(referenceResult.allFiles),
			"File count should not decrease between %s (%d files) and %s (%d files)",
			referenceClient, len(referenceResult.allFiles), clientName, len(result.allFiles))
	}
}

// testUpdateExistingFileConsistency tests that all clients update files consistently
func (c *ClientConsistencyTest) testUpdateExistingFileConsistency(t *testing.T, ctx context.Context, repo *Repository) {
	// First, create a file that all clients can update
	baseFileName := "consistency_test/update_target.txt"
	baseContent := "Original content"
	baseCommitMessage := "Create file for update test"

	// Use first client to create the base file
	firstClient := c.suite.clients[0]
	err := firstClient.CreateFile(ctx, repo.AuthURL(), baseFileName, baseContent, baseCommitMessage)
	require.NoError(t, err, "Failed to create base file")

	// Wait for propagation
	time.Sleep(2 * time.Second)

	// Now test that all clients can update this file consistently
	updatedContent := "Updated content by consistency test"
	updateCommitMessage := "Update file for consistency test"

	results := make(map[string]struct {
		fileContent   string
		commitMessage string
	})

	for i, client := range c.suite.clients {
		clientName := client.Name()

		t.Run(fmt.Sprintf("%s_update", clientName), func(t *testing.T) {
			// Clone repository with Git CLI for verification
			localPath := filepath.Join(c.gitCLIWorkdir, fmt.Sprintf("update_test_%s", clientName))
			err := c.cloneWithGitCLI(ctx, repo.AuthURL(), localPath)
			require.NoError(t, err, "Failed to clone repository for %s", clientName)

			// Update the file with unique content per client
			clientSpecificContent := fmt.Sprintf("%s (by %s)", updatedContent, clientName)
			clientSpecificMessage := fmt.Sprintf("%s (by %s)", updateCommitMessage, clientName)
			err = client.UpdateFile(ctx, repo.AuthURL(), baseFileName, clientSpecificContent, clientSpecificMessage)
			require.NoError(t, err, "Failed to update file with %s", clientName)

			// Wait for propagation
			time.Sleep(2 * time.Second)

			// Pull latest changes with Git CLI
			cmd := exec.CommandContext(ctx, "git", "pull", "origin", "main")
			cmd.Dir = localPath
			_, err = cmd.CombinedOutput()
			require.NoError(t, err, "Failed to pull changes for %s", clientName)

			// Verify file was updated
			fileContent, err := c.getFileContentWithGitCLI(ctx, localPath, baseFileName)
			require.NoError(t, err, "Failed to read updated file content for %s", clientName)

			latestCommitMessage, err := c.getLatestCommitMessageWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get latest commit message for %s", clientName)

			// Store results
			results[clientName] = struct {
				fileContent   string
				commitMessage string
			}{
				fileContent:   fileContent,
				commitMessage: latestCommitMessage,
			}

			// Verify content was updated correctly
			require.Equal(t, clientSpecificContent, fileContent, "File content not updated correctly by %s", clientName)
			require.Equal(t, clientSpecificMessage, latestCommitMessage, "Commit message not set correctly by %s", clientName)
		})

		// For subsequent clients, wait and use the updated repository state
		if i < len(c.suite.clients)-1 {
			time.Sleep(5 * time.Second) // Allow time for repository state to stabilize
		}
	}
}

// testBulkOperationsConsistency tests that all clients handle bulk operations consistently
func (c *ClientConsistencyTest) testBulkOperationsConsistency(t *testing.T, ctx context.Context, repo *Repository) {
	// Define bulk operation test cases
	testFiles := []FileChange{
		{Path: "bulk_test/file1.txt", Content: "Content of file 1", Action: "create"},
		{Path: "bulk_test/file2.txt", Content: "Content of file 2", Action: "create"},
		{Path: "bulk_test/file3.txt", Content: "Content of file 3", Action: "create"},
	}
	commitMessage := "Bulk create test files"

	results := make(map[string]struct {
		commitCount  int
		allFiles     []string
		fileContents map[string]string
	})

	for _, client := range c.suite.clients {
		clientName := client.Name()

		t.Run(fmt.Sprintf("%s_bulk", clientName), func(t *testing.T) {
			// Clone repository with Git CLI for verification
			localPath := filepath.Join(c.gitCLIWorkdir, fmt.Sprintf("bulk_test_%s", clientName))
			err := c.cloneWithGitCLI(ctx, repo.AuthURL(), localPath)
			require.NoError(t, err, "Failed to clone repository for %s", clientName)

			initialCommitCount, err := c.getCommitCountWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get initial commit count for %s", clientName)

			// Modify files to be unique per client
			clientFiles := make([]FileChange, len(testFiles))
			for i, file := range testFiles {
				clientFiles[i] = FileChange{
					Path:    fmt.Sprintf("%s_%s", file.Path, clientName),
					Content: fmt.Sprintf("%s (created by %s)", file.Content, clientName),
					Action:  file.Action,
				}
			}

			// Perform bulk operation
			err = client.BulkCreateFiles(ctx, repo.AuthURL(), clientFiles, fmt.Sprintf("%s (by %s)", commitMessage, clientName))
			require.NoError(t, err, "Failed to perform bulk operation with %s", clientName)

			// Wait for propagation
			time.Sleep(3 * time.Second)

			// Pull latest changes with Git CLI
			cmd := exec.CommandContext(ctx, "git", "pull", "origin", "main")
			cmd.Dir = localPath
			_, err = cmd.CombinedOutput()
			require.NoError(t, err, "Failed to pull changes for %s", clientName)

			// Verify results
			finalCommitCount, err := c.getCommitCountWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get final commit count for %s", clientName)

			allFiles, err := c.listFilesWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to list files for %s", clientName)

			// Verify all bulk files were created
			fileContents := make(map[string]string)
			for _, file := range clientFiles {
				require.Contains(t, allFiles, file.Path, "Bulk file not found for %s: %s", clientName, file.Path)

				content, err := c.getFileContentWithGitCLI(ctx, localPath, file.Path)
				require.NoError(t, err, "Failed to read bulk file content for %s: %s", clientName, file.Path)
				require.Equal(t, file.Content, content, "Bulk file content mismatch for %s: %s", clientName, file.Path)

				fileContents[file.Path] = content
			}

			// Verify only one commit was added for bulk operation
			require.Equal(t, initialCommitCount+1, finalCommitCount, "Bulk operation should create exactly one commit for %s", clientName)

			// Store results
			results[clientName] = struct {
				commitCount  int
				allFiles     []string
				fileContents map[string]string
			}{
				commitCount:  finalCommitCount - initialCommitCount,
				allFiles:     allFiles,
				fileContents: fileContents,
			}
		})
	}

	// Verify consistency across clients
	for clientName, result := range results {
		// All clients should create exactly one commit for bulk operation
		require.Equal(t, 1, result.commitCount, "Bulk operation commit count inconsistent for %s", clientName)

		// All clients should have created the expected number of new files
		expectedNewFiles := len(testFiles)
		actualNewFiles := 0
		for _, file := range result.allFiles {
			if strings.Contains(file, "bulk_test/") && strings.Contains(file, clientName) {
				actualNewFiles++
			}
		}
		require.Equal(t, expectedNewFiles, actualNewFiles, "Bulk file count mismatch for %s", clientName)
	}
}
