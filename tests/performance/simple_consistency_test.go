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

// TestSimpleClientConsistency tests basic functionality of all clients
func TestSimpleClientConsistency(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()

	suite, err := NewBenchmarkSuite(ctx, 0) // No network latency for consistency tests
	require.NoError(t, err)
	defer suite.Cleanup(ctx)

	// Use small repository for consistency tests
	repo := suite.GetRepository("small")
	require.NotNil(t, repo, "Small repository not found")

	// Create temporary directory for Git CLI operations
	tempDir, err := os.MkdirTemp("", "simple_consistency_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("CreateFileConsistency", func(t *testing.T) {
		testCreateFileConsistency(t, ctx, suite.clients, repo, tempDir)
	})

	t.Run("UpdateFileConsistency", func(t *testing.T) {
		testUpdateFileConsistency(t, ctx, suite.clients, repo, tempDir)
	})

	t.Run("BulkOperationsConsistency", func(t *testing.T) {
		testBulkOperationsConsistency(t, ctx, suite.clients, repo, tempDir)
	})
}

// testCreateFileConsistency tests that all clients can create files successfully
func testCreateFileConsistency(t *testing.T, ctx context.Context, clients []GitClient, repo *Repository, tempDir string) {
	// Test each client creates a file successfully
	for i, client := range clients {
		clientName := client.Name()
		testFile := fmt.Sprintf("test_create_%s_%d.txt", clientName, i)
		testContent := fmt.Sprintf("Content created by %s", clientName)
		commitMessage := fmt.Sprintf("Create file by %s", clientName)

		t.Run(clientName, func(t *testing.T) {
			// Create the file
			err := client.CreateFile(ctx, repo.AuthURL(), testFile, testContent, commitMessage)
			require.NoError(t, err, "Failed to create file with %s", clientName)

			// Wait for propagation
			time.Sleep(2 * time.Second)

			// Verify the file exists using Git CLI
			localPath := filepath.Join(tempDir, fmt.Sprintf("verify_%s_%d", clientName, i))
			err = cloneWithGitCLI(ctx, repo.AuthURL(), localPath)
			require.NoError(t, err, "Failed to clone for verification")

			// Check file content
			content, err := getFileContentWithGitCLI(ctx, localPath, testFile)
			require.NoError(t, err, "Failed to read file content")
			require.Equal(t, testContent, content, "File content mismatch")

			// Check latest commit message
			message, err := getLatestCommitMessageWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get commit message")
			require.Equal(t, commitMessage, message, "Commit message mismatch")
		})
	}
}

// testUpdateFileConsistency tests that all clients can update files successfully
func testUpdateFileConsistency(t *testing.T, ctx context.Context, clients []GitClient, repo *Repository, tempDir string) {
	// First, create a base file with the first client
	baseFile := "test_update_target.txt"
	baseContent := "Original content for update test"
	baseMessage := "Create base file for update test"

	firstClient := clients[0]
	err := firstClient.CreateFile(ctx, repo.AuthURL(), baseFile, baseContent, baseMessage)
	require.NoError(t, err, "Failed to create base file")

	time.Sleep(2 * time.Second)

	// Now test that each client can update this file
	for i, client := range clients {
		clientName := client.Name()
		updatedContent := fmt.Sprintf("Updated content by %s at %d", clientName, time.Now().Unix())
		updateMessage := fmt.Sprintf("Update file by %s", clientName)

		t.Run(clientName, func(t *testing.T) {
			// Update the file
			err := client.UpdateFile(ctx, repo.AuthURL(), baseFile, updatedContent, updateMessage)
			require.NoError(t, err, "Failed to update file with %s", clientName)

			// Wait for propagation
			time.Sleep(2 * time.Second)

			// Verify the file was updated using Git CLI
			localPath := filepath.Join(tempDir, fmt.Sprintf("update_verify_%s_%d", clientName, i))
			err = cloneWithGitCLI(ctx, repo.AuthURL(), localPath)
			require.NoError(t, err, "Failed to clone for update verification")

			// Check file content
			content, err := getFileContentWithGitCLI(ctx, localPath, baseFile)
			require.NoError(t, err, "Failed to read updated file content")
			require.Equal(t, updatedContent, content, "Updated file content mismatch")

			// Check latest commit message
			message, err := getLatestCommitMessageWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get update commit message")
			require.Equal(t, updateMessage, message, "Update commit message mismatch")
		})
	}
}

// testBulkOperationsConsistency tests that all clients can perform bulk operations successfully
func testBulkOperationsConsistency(t *testing.T, ctx context.Context, clients []GitClient, repo *Repository, tempDir string) {
	// Test each client performs bulk operations successfully
	for i, client := range clients {
		clientName := client.Name()
		
		// Create bulk test files specific to this client
		files := []FileChange{
			{Path: fmt.Sprintf("bulk_%s/file1.txt", clientName), Content: fmt.Sprintf("Bulk file 1 by %s", clientName), Action: "create"},
			{Path: fmt.Sprintf("bulk_%s/file2.txt", clientName), Content: fmt.Sprintf("Bulk file 2 by %s", clientName), Action: "create"},
			{Path: fmt.Sprintf("bulk_%s/file3.txt", clientName), Content: fmt.Sprintf("Bulk file 3 by %s", clientName), Action: "create"},
		}
		commitMessage := fmt.Sprintf("Bulk create by %s", clientName)

		t.Run(clientName, func(t *testing.T) {
			// Perform bulk operation
			err := client.BulkCreateFiles(ctx, repo.AuthURL(), files, commitMessage)
			require.NoError(t, err, "Failed to perform bulk operation with %s", clientName)

			// Wait for propagation
			time.Sleep(3 * time.Second)

			// Verify all files were created using Git CLI
			localPath := filepath.Join(tempDir, fmt.Sprintf("bulk_verify_%s_%d", clientName, i))
			err = cloneWithGitCLI(ctx, repo.AuthURL(), localPath)
			require.NoError(t, err, "Failed to clone for bulk verification")

			// Check each file was created
			for _, file := range files {
				content, err := getFileContentWithGitCLI(ctx, localPath, file.Path)
				require.NoError(t, err, "Failed to read bulk file %s", file.Path)
				require.Equal(t, file.Content, content, "Bulk file content mismatch for %s", file.Path)
			}

			// Check latest commit message
			message, err := getLatestCommitMessageWithGitCLI(ctx, localPath)
			require.NoError(t, err, "Failed to get bulk commit message")
			require.Equal(t, commitMessage, message, "Bulk commit message mismatch")
		})
	}
}

// Helper functions for Git CLI operations

func cloneWithGitCLI(ctx context.Context, repoURL, localPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}
	return nil
}

func getFileContentWithGitCLI(ctx context.Context, repoPath, filePath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("HEAD:%s", filePath))
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}

func getLatestCommitMessageWithGitCLI(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%s")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}