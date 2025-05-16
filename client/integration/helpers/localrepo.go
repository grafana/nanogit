package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// LocalGitRepo represents a local Git repository used for testing.
// It provides methods to initialize, modify, and manage a Git repository
// in a temporary directory.
type LocalGitRepo struct {
	Path string // Path to the local Git repository
}

// NewLocalGitRepo creates a new LocalGitRepo instance with a temporary directory
// as its path. The temporary directory is automatically cleaned up when the test
// completes.
func NewLocalGitRepo(t *testing.T) *LocalGitRepo {
	p := t.TempDir()
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(p))
	})

	r := &LocalGitRepo{Path: p}
	r.Git(t, "init")
	return r
}

// CreateFile creates a new file in the repository with the specified filename
// and content. The file is created with read/write permissions for the owner only.
func (r *LocalGitRepo) CreateFile(t *testing.T, filename, content string) {
	err := os.WriteFile(filepath.Join(r.Path, filename), []byte(content), 0600)
	require.NoError(t, err)
}

// Git executes a Git command in the repository directory.
// It logs the command being executed and its output for debugging purposes.
// The command is executed with GIT_TERMINAL_PROMPT=0 to prevent interactive prompts.
func (r *LocalGitRepo) Git(t *testing.T, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	// Log the git command being executed
	t.Logf("Running git command: git %s in directory: %s", strings.Join(args, " "), r.Path)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Git command output:\n%s", string(output))
		require.NoError(t, err, "git command failed %s: %s", args, output)
	}

	// Log successful command output
	t.Logf("Git command output:\n%s", string(output))
}
