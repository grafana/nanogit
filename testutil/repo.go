package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
)

// LocalRepo represents a local Git repository used for testing.
// It provides methods to initialize, modify, and manage a Git repository
// in a temporary directory.
type LocalRepo struct {
	Path string

	logger       Logger
	cleanupFunc  func() error
	coloredGit   bool
	gitEnv       []string
}

// NewLocalRepo creates a new LocalRepo instance with a temporary directory
// as its path. The temporary directory is automatically cleaned up when
// Cleanup() is called.
//
// The repository is automatically initialized with `git init`.
func NewLocalRepo(ctx context.Context, opts ...RepoOption) (*LocalRepo, error) {
	cfg := &repoConfig{
		logger:  NoopLogger(),
		tempDir: "",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp(cfg.tempDir, "nanogit-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cfg.logger.Logf("📦 [LOCAL] 📁 Creating new local repository at %s", tempDir)

	repo := &LocalRepo{
		Path:   tempDir,
		logger: cfg.logger,
		cleanupFunc: func() error {
			cfg.logger.Logf("📦 [LOCAL] 🧹 Cleaning up local repository at %s", tempDir)
			return os.RemoveAll(tempDir)
		},
		coloredGit: true,
		gitEnv:     []string{"GIT_TERMINAL_PROMPT=0", "GIT_TRACE_PACKET=1"},
	}

	// Initialize the repository
	if _, err := repo.Git("init"); err != nil {
		_ = repo.Cleanup()
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	cfg.logger.Logf("📦 [LOCAL] ✅ Local repository initialized successfully")
	return repo, nil
}

// Init initializes the repository with `git init`.
// This is automatically called by NewLocalRepo, but can be used
// to reinitialize if needed.
func (r *LocalRepo) Init() error {
	_, err := r.Git("init")
	return err
}

// CreateDirPath creates a directory path in the repository.
// It creates all necessary parent directories if they don't exist.
func (r *LocalRepo) CreateDirPath(dirpath string) error {
	r.logger.Logf("📦 [LOCAL] 📁 Creating directory path '%s' in repository", dirpath)
	err := os.MkdirAll(filepath.Join(r.Path, dirpath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	r.logger.Logf("📦 [LOCAL] ✅ Directory path '%s' created successfully", dirpath)
	return nil
}

// CreateFile creates a new file in the repository with the specified filename
// and content. The file is created with read/write permissions for the owner only.
// Creates parent directories if they don't exist.
func (r *LocalRepo) CreateFile(path, content string) error {
	r.logger.Logf("📦 [LOCAL] 📝 Creating file '%s' in repository", path)
	fullPath := filepath.Join(r.Path, path)

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	r.logger.Logf("📦 [LOCAL] ✅ File '%s' created successfully", path)
	return nil
}

// UpdateFile updates an existing file in the repository with new content.
// The file must exist before calling this method.
func (r *LocalRepo) UpdateFile(path, content string) error {
	r.logger.Logf("📦 [LOCAL] 📝 Updating file '%s' in repository", path)
	fullPath := filepath.Join(r.Path, path)

	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	r.logger.Logf("📦 [LOCAL] ✅ File '%s' updated successfully", path)
	return nil
}

// DeleteFile removes a file from the repository.
func (r *LocalRepo) DeleteFile(path string) error {
	r.logger.Logf("📦 [LOCAL] 🗑️  Deleting file '%s' from repository", path)
	fullPath := filepath.Join(r.Path, path)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	r.logger.Logf("📦 [LOCAL] ✅ File '%s' deleted successfully", path)
	return nil
}

// Git executes a Git command in the repository directory.
// It logs the command being executed and its output for debugging purposes.
// The command is executed with GIT_TERMINAL_PROMPT=0 to prevent interactive prompts.
//
// Returns the command output and any error encountered.
func (r *LocalRepo) Git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path
	cmd.Env = append(os.Environ(), r.gitEnv...)

	// Format the command for display
	cmdStr := strings.Join(args, " ")

	if r.coloredGit {
		// Log the git command being executed with a special format
		r.logger.Logf("📦 [LOCAL] ┌─────────────────────────────────────────────┐")
		r.logger.Logf("📦 [LOCAL] │ Git Command")
		r.logger.Logf("📦 [LOCAL] ├─────────────────────────────────────────────┤")
		r.logger.Logf("📦 [LOCAL] │ $ git %s", cmdStr)
		r.logger.Logf("📦 [LOCAL] │ Path: %s", r.Path)
	} else {
		r.logger.Logf("📦 [LOCAL] Executing: git %s", cmdStr)
	}

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if r.coloredGit {
			r.logger.Logf("📦 [LOCAL] ├─────────────────────────────────────────────┤")
			r.logger.Logf("📦 [LOCAL] │ [ERROR] %s", err.Error())
			if len(outputStr) > 0 {
				r.logger.Logf("📦 [LOCAL] │ Output:")
				for _, line := range strings.Split(outputStr, "\n") {
					if line != "" {
						r.logger.Logf("📦 [LOCAL] │   %s", line)
					}
				}
			}
			r.logger.Logf("📦 [LOCAL] └─────────────────────────────────────────────┘")
		} else {
			r.logger.Logf("📦 [LOCAL] Error: %v\nOutput: %s", err, outputStr)
		}
		return outputStr, fmt.Errorf("git command failed: %w", err)
	}

	if len(outputStr) > 0 {
		if r.coloredGit {
			r.logger.Logf("📦 [LOCAL] ├─────────────────────────────────────────────┤")
			r.logger.Logf("📦 [LOCAL] │ Output:")
			for _, line := range strings.Split(outputStr, "\n") {
				if line != "" {
					r.logger.Logf("📦 [LOCAL] │ %s", line)
				}
			}
			r.logger.Logf("📦 [LOCAL] └─────────────────────────────────────────────┘")
		} else {
			r.logger.Logf("📦 [LOCAL] Output: %s", outputStr)
		}
	} else if r.coloredGit {
		r.logger.Logf("📦 [LOCAL] └─────────────────────────────────────────────┘")
	}

	return outputStr, nil
}

// GitWithError is an alias for Git that returns both output and error.
// Provided for compatibility with test code that expects this signature.
func (r *LocalRepo) GitWithError(args ...string) (string, error) {
	return r.Git(args...)
}

// QuickInit is a convenience method that sets up a repository with:
// - Git user configuration
// - Remote URL configuration
// - Initial commit with a test file
// - Force push to main branch
// - Branch tracking setup
//
// Returns a configured nanogit.Client and the name of the test file created.
func (r *LocalRepo) QuickInit(user *User, remoteURL string) (nanogit.Client, string, error) {
	r.logger.Logf("📦 [LOCAL] Setting up local repository")

	if _, err := r.Git("config", "user.name", user.Username); err != nil {
		return nil, "", err
	}
	if _, err := r.Git("config", "user.email", user.Email); err != nil {
		return nil, "", err
	}
	if _, err := r.Git("remote", "add", "origin", remoteURL); err != nil {
		return nil, "", err
	}

	r.logger.Logf("📦 [LOCAL] Creating and committing test file")
	testContent := "test content"
	fileName := "test.txt"
	if err := r.CreateFile(fileName, testContent); err != nil {
		return nil, "", err
	}
	if _, err := r.Git("add", fileName); err != nil {
		return nil, "", err
	}
	if _, err := r.Git("commit", "-m", "Initial commit"); err != nil {
		return nil, "", err
	}

	r.logger.Logf("📦 [LOCAL] Setting up main branch and pushing changes")
	if _, err := r.Git("branch", "-M", "main"); err != nil {
		return nil, "", err
	}
	if _, err := r.Git("push", "origin", "main", "--force"); err != nil {
		return nil, "", err
	}

	r.logger.Logf("📦 [LOCAL] Tracking current branch")
	if _, err := r.Git("branch", "--set-upstream-to=origin/main", "main"); err != nil {
		return nil, "", err
	}

	client, err := nanogit.NewHTTPClient(remoteURL, options.WithBasicAuth(user.Username, user.Password))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create nanogit client: %w", err)
	}

	return client, fileName, nil
}

// LogContents logs the contents of the repository directory tree.
// Useful for debugging test failures.
func (r *LocalRepo) LogContents() {
	r.logger.Logf("📦 [LOCAL] Repository contents:")
	var printDir func(path string, indent string)
	printDir = func(path string, indent string) {
		files, err := os.ReadDir(path)
		if err != nil {
			r.logger.Logf("%s[ERROR reading directory: %v]", indent, err)
			return
		}
		for _, file := range files {
			fullPath := filepath.Join(path, file.Name())
			if file.IsDir() {
				r.logger.Logf("%s📁 %s/", indent, file.Name())
				printDir(fullPath, indent+"  ")
			} else {
				r.logger.Logf("%s📄 %s", indent, file.Name())
			}
		}
	}
	printDir(r.Path, "")
}

// Cleanup removes the temporary directory and all its contents.
// This should be called when the repository is no longer needed.
// This method is safe to call multiple times.
func (r *LocalRepo) Cleanup() error {
	if r.cleanupFunc != nil {
		err := r.cleanupFunc()
		r.cleanupFunc = nil
		return err
	}
	return nil
}
