package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/stretchr/testify/require"
)

// LocalGitRepo represents a local Git repository used for testing.
// It provides methods to initialize, modify, and manage a Git repository
// in a temporary directory.
type LocalGitRepo struct {
	logger *TestLogger
	Path   string // Path to the local Git repository
}

// NewLocalGitRepo creates a new LocalGitRepo instance with a temporary directory
// as its path. The temporary directory is automatically cleaned up when the test
// completes.
func NewLocalGitRepo(t *testing.T, logger *TestLogger) *LocalGitRepo {
	p := t.TempDir()
	t.Cleanup(func() {
		t.Logf("%s📦 [LOCAL] 🧹 Cleaning up local repository at %s%s", ColorYellow, p, ColorReset)
		require.NoError(t, os.RemoveAll(p))
	})

	logger.Logf("%s📦 [LOCAL] 📁 Creating new local repository at %s%s", ColorBlue, p, ColorReset)
	r := &LocalGitRepo{Path: p, logger: logger}
	r.Git(t, "init")
	logger.Logf("%s📦 [LOCAL] ✅ Local repository initialized successfully%s", ColorGreen, ColorReset)
	return r
}

// CreateDirPath creates a directory path in the repository.
// It creates all necessary parent directories if they don't exist.
func (r *LocalGitRepo) CreateDirPath(t *testing.T, dirpath string) {
	r.logger.Logf("%s📦 [LOCAL] 📁 Creating directory path '%s' in repository%s", ColorBlue, dirpath, ColorReset)
	err := os.MkdirAll(filepath.Join(r.Path, dirpath), 0755)
	require.NoError(t, err)
	r.logger.Logf("%s📦 [LOCAL] ✅ Directory path '%s' created successfully%s", ColorGreen, dirpath, ColorReset)
}

// CreateFile creates a new file in the repository with the specified filename
// and content. The file is created with read/write permissions for the owner only.
func (r *LocalGitRepo) CreateFile(t *testing.T, filename, content string) {
	r.logger.Logf("%s📦 [LOCAL] 📝 Creating file '%s' in repository%s", ColorBlue, filename, ColorReset)
	err := os.WriteFile(filepath.Join(r.Path, filename), []byte(content), 0600)
	require.NoError(t, err)
	r.logger.Logf("%s📦 [LOCAL] ✅ File '%s' created successfully%s", ColorGreen, filename, ColorReset)
}

// Git executes a Git command in the repository directory.
// It logs the command being executed and its output for debugging purposes.
// The command is executed with GIT_TERMINAL_PROMPT=0 to prevent interactive prompts.
func (r *LocalGitRepo) Git(t *testing.T, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_TRACE_PACKET=1")

	// Format the command for display
	cmdStr := strings.Join(args, " ")

	// Log the git command being executed with a special format
	r.logger.Logf("%s📦 [LOCAL] %s┌─────────────────────────────────────────────┐%s", ColorBlue, ColorPurple, ColorReset)
	r.logger.Logf("%s📦 [LOCAL] %s│ %sGit Command%s%s", ColorBlue, ColorPurple, ColorCyan, ColorPurple, ColorReset)
	r.logger.Logf("%s📦 [LOCAL] %s├─────────────────────────────────────────────┤%s", ColorBlue, ColorPurple, ColorReset)
	r.logger.Logf("%s📦 [LOCAL] %s│ %s$ git %s%s", ColorBlue, ColorPurple, ColorCyan, cmdStr, ColorReset)
	r.logger.Logf("%s📦 [LOCAL] %s│ %sPath: %s%s", ColorBlue, ColorPurple, ColorCyan, r.Path, ColorReset)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Add error information to the same box
		r.logger.Logf("%s📦 [LOCAL] %s├─────────────────────────────────────────────┤%s", ColorRed, ColorPurple, ColorReset)
		r.logger.Logf("%s📦 [LOCAL] %s│ %sError: %s%s", ColorRed, ColorPurple, ColorRed, err.Error(), ColorReset)
		if len(output) > 0 {
			r.logger.Logf("%s📦 [LOCAL] %s│ %sOutput:%s", ColorRed, ColorPurple, ColorRed, ColorReset)
			for _, line := range strings.Split(string(output), "\n") {
				if line != "" {
					r.logger.Logf("%s📦 [LOCAL] %s│ %s  %s%s", ColorRed, ColorPurple, ColorRed, line, ColorReset)
				}
			}
		}
		r.logger.Logf("%s📦 [LOCAL] %s└─────────────────────────────────────────────┘%s", ColorRed, ColorPurple, ColorReset)
		require.NoError(t, err, "git command failed: %s\nOutput: %s", cmdStr, output)
	} else if len(output) > 0 {
		// Add output to the same box
		r.logger.Logf("%s📦 [LOCAL] %s├─────────────────────────────────────────────┤%s", ColorCyan, ColorPurple, ColorReset)
		r.logger.Logf("%s📦 [LOCAL] %s│ %sOutput:%s", ColorCyan, ColorPurple, ColorCyan, ColorReset)
		for _, line := range strings.Split(string(output), "\n") {
			if line != "" {
				r.logger.Logf("%s📦 [LOCAL] %s│ %s%s%s", ColorCyan, ColorPurple, ColorCyan, line, ColorReset)
			}
		}
		r.logger.Logf("%s📦 [LOCAL] %s└─────────────────────────────────────────────┘%s", ColorCyan, ColorPurple, ColorReset)
	} else {
		// Close the box if there's no output
		r.logger.Logf("%s📦 [LOCAL] %s└─────────────────────────────────────────────┘%s", ColorBlue, ColorPurple, ColorReset)
	}
	return strings.TrimSpace(string(output))
}

func (r *LocalGitRepo) QuickInit(t *testing.T, user *User, remoteURL string) (client nanogit.Client, fileName string) {
	r.logger.Info("Setting up local repository")
	r.Git(t, "config", "user.name", user.Username)
	r.Git(t, "config", "user.email", user.Email)
	r.Git(t, "remote", "add", "origin", remoteURL)

	r.logger.Info("Creating and committing test file")
	testContent := []byte("test content")
	r.CreateFile(t, "test.txt", string(testContent))
	r.Git(t, "add", "test.txt")
	r.Git(t, "commit", "-m", "Initial commit")

	r.logger.Info("Setting up main branch and pushing changes")
	r.Git(t, "branch", "-M", "main")
	r.Git(t, "push", "origin", "main", "--force")

	r.logger.Info("Tracking current branch")
	r.Git(t, "branch", "--set-upstream-to=origin/main", "main")

	client, err := nanogit.NewClient(remoteURL, nanogit.WithBasicAuth(user.Username, user.Password))
	require.NoError(t, err)
	return client, "test.txt"
}

func (r *LocalGitRepo) LogRepoContents(t *testing.T) {
	r.logger.Info("Logging repository contents")
	var printDir func(path string, indent string)
	printDir = func(path string, indent string) {
		files, err := os.ReadDir(path)
		require.NoError(t, err)
		for _, file := range files {
			fullPath := filepath.Join(path, file.Name())
			if file.IsDir() {
				r.logger.Info("📁 Directory", "name", indent+file.Name()+"/")
				printDir(fullPath, indent+"  ")
			} else {
				r.logger.Info("📄 File", "name", indent+file.Name())
			}
		}
	}
	printDir(r.Path, "")
}
