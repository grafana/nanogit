package clients

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GitCLIClient implements the GitClient interface using git CLI commands
type GitCLIClient struct {
	workDir string // Base directory for git operations
}

// NewGitCLIClient creates a new git CLI client
func NewGitCLIClient() (*GitCLIClient, error) {
	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "git-perf-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	return &GitCLIClient{
		workDir: workDir,
	}, nil
}

// Cleanup removes the temporary work directory
func (c *GitCLIClient) Cleanup() error {
	return os.RemoveAll(c.workDir)
}

// Name returns the client name
func (c *GitCLIClient) Name() string {
	return "git-cli"
}

// cloneRepo clones a fresh copy of the repository for each operation
func (c *GitCLIClient) cloneRepo(ctx context.Context, repoURL string) (string, error) {
	// Create unique directory for this repo clone
	repoName := filepath.Base(repoURL)
	if strings.HasSuffix(repoName, ".git") {
		repoName = repoName[:len(repoName)-4]
	}

	// Add timestamp to make directory unique for each clone
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	localPath := filepath.Join(c.workDir, fmt.Sprintf("%s-%s", repoName, timestamp))

	// Clone repository
	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, localPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return localPath, nil
}

// runGitCommand runs a git command in the repository directory
func (c *GitCLIClient) runGitCommand(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	return cmd.Output()
}

// runGitCommandWithOutput runs a git command and returns both stdout and stderr for better error reporting
func (c *GitCLIClient) runGitCommandWithOutput(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("git %s failed: %s (output: %s)", strings.Join(args, " "), err.Error(), string(output))
	}
	return output, nil
}

// CreateFile creates a new file in the repository
func (c *GitCLIClient) CreateFile(ctx context.Context, repoURL, path, content, message string) error {
	repoPath, err := c.cloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	// Create file
	filePath := filepath.Join(repoPath, path)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Add file
	if _, err := c.runGitCommand(ctx, repoPath, "add", path); err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Commit
	if _, err := c.runGitCommand(ctx, repoPath, "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push changes to remote
	if _, err := c.runGitCommandWithOutput(ctx, repoPath, "push", "origin", "main"); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// UpdateFile updates an existing file in the repository
func (c *GitCLIClient) UpdateFile(ctx context.Context, repoURL, path, content, message string) error {
	repoPath, err := c.cloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	// Update file
	filePath := filepath.Join(repoPath, path)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Add and commit
	if _, err := c.runGitCommand(ctx, repoPath, "add", path); err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	if _, err := c.runGitCommand(ctx, repoPath, "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push changes to remote
	if _, err := c.runGitCommandWithOutput(ctx, repoPath, "push", "origin", "main"); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from the repository
func (c *GitCLIClient) DeleteFile(ctx context.Context, repoURL, path, message string) error {
	repoPath, err := c.cloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	// Remove file
	if _, err := c.runGitCommand(ctx, repoPath, "rm", path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Commit
	if _, err := c.runGitCommand(ctx, repoPath, "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push changes to remote
	if _, err := c.runGitCommandWithOutput(ctx, repoPath, "push", "origin", "main"); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// CompareCommits compares two commits and returns the differences
func (c *GitCLIClient) CompareCommits(ctx context.Context, repoURL, base, head string) (*CommitComparison, error) {
	repoPath, err := c.cloneRepo(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	// Get diff statistics
	output, err := c.runGitCommand(ctx, repoPath, "diff", "--numstat", base+"..."+head)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff stats: %w", err)
	}

	comparison := &CommitComparison{
		Files: make([]FileChangeSummary, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		additions, _ := strconv.Atoi(parts[0])
		deletions, _ := strconv.Atoi(parts[1])
		path := parts[2]

		status := "modified"
		if parts[0] == "-" {
			// Binary file or other special case
			additions = 0
		}
		if parts[1] == "-" {
			deletions = 0
		}

		comparison.Files = append(comparison.Files, FileChangeSummary{
			Path:      path,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
		})

		comparison.Additions += additions
		comparison.Deletions += deletions
	}

	comparison.FilesChanged = len(comparison.Files)
	return comparison, nil
}

// GetFlatTree returns a flat listing of all files in the repository at a given ref
func (c *GitCLIClient) GetFlatTree(ctx context.Context, repoURL, ref string) (*TreeResult, error) {
	repoPath, err := c.cloneRepo(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	// List files at the given ref
	output, err := c.runGitCommand(ctx, repoPath, "ls-tree", "-r", "--long", ref)
	if err != nil {
		return nil, fmt.Errorf("failed to list tree: %w", err)
	}

	result := &TreeResult{
		Files: make([]TreeFile, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse ls-tree output: mode type hash size path
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}

		size, _ := strconv.ParseInt(parts[3], 10, 64)
		path := strings.Join(parts[4:], " ") // Handle paths with spaces

		result.Files = append(result.Files, TreeFile{
			Path: path,
			Size: size,
			Type: "blob",
		})
	}

	result.Count = len(result.Files)
	return result, nil
}

// BulkCreateFiles creates multiple files in a single commit
func (c *GitCLIClient) BulkCreateFiles(ctx context.Context, repoURL string, files []FileChange, message string) error {
	repoPath, err := c.cloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	// Check if this is the first commit by checking if HEAD exists
	isFirstCommit := false
	if _, err := c.runGitCommand(ctx, repoPath, "rev-parse", "HEAD"); err != nil {
		isFirstCommit = true
	}

	// Process all files
	for _, fileChange := range files {
		switch strings.ToLower(fileChange.Action) {
		case "create", "update":
			filePath := filepath.Join(repoPath, fileChange.Path)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", fileChange.Path, err)
			}

			if err := os.WriteFile(filePath, []byte(fileChange.Content), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", fileChange.Path, err)
			}

			if _, err := c.runGitCommand(ctx, repoPath, "add", fileChange.Path); err != nil {
				return fmt.Errorf("failed to add file %s: %w", fileChange.Path, err)
			}

		case "delete":
			// Check if file is tracked by git before trying to delete it
			output, err := c.runGitCommand(ctx, repoPath, "ls-files", fileChange.Path)
			if err == nil && len(strings.TrimSpace(string(output))) > 0 {
				// File is tracked, remove it with git rm
				if _, err := c.runGitCommand(ctx, repoPath, "rm", fileChange.Path); err != nil {
					return fmt.Errorf("failed to remove file %s: %w", fileChange.Path, err)
				}
			}
		}
	}

	// Check if there are any changes to commit
	statusOutput, err := c.runGitCommand(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// Only commit if there are changes
	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		if _, err := c.runGitCommand(ctx, repoPath, "commit", "-m", message); err != nil {
			return fmt.Errorf("failed to commit bulk changes: %w", err)
		}

		// If this is the first commit, set up the main branch and push to establish the repository
		if isFirstCommit {
			// Set up main branch
			if _, err := c.runGitCommand(ctx, repoPath, "branch", "-M", "main"); err != nil {
				return fmt.Errorf("failed to rename branch to main: %w", err)
			}

			// Push to origin to establish the repository
			if _, err := c.runGitCommandWithOutput(ctx, repoPath, "push", "origin", "main", "--force"); err != nil {
				return fmt.Errorf("failed to push initial commit: %w", err)
			}

			// Set up tracking
			if _, err := c.runGitCommand(ctx, repoPath, "branch", "--set-upstream-to=origin/main", "main"); err != nil {
				return fmt.Errorf("failed to set upstream: %w", err)
			}
		} else {
			// For subsequent commits, just push to the existing main branch
			if _, err := c.runGitCommandWithOutput(ctx, repoPath, "push", "origin", "main"); err != nil {
				return fmt.Errorf("failed to push bulk changes: %w", err)
			}
		}
	}

	return nil
}
