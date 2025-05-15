package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type LocalGitRepo struct {
	Path string
}

func NewLocalGitRepo(t *testing.T) *LocalGitRepo {
	return &LocalGitRepo{Path: t.TempDir()}
}

func (r *LocalGitRepo) Init(t *testing.T) {
	r.Run(t, "init")
}

func (r *LocalGitRepo) CreateFile(t *testing.T, filename, content string) {
	err := os.WriteFile(filepath.Join(r.Path, filename), []byte(content), 0600)
	require.NoError(t, err)
}

func (r *LocalGitRepo) Cleanup(t *testing.T) {
	if r.Path != "" {
		require.NoError(t, os.RemoveAll(r.Path))
	}
}

func (r *LocalGitRepo) Run(t *testing.T, args ...string) {
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
