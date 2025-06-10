package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/onsi/ginkgo/v2"

	//nolint:stylecheck // specifically ignore ST1001 (dot-imports)
	//lint:ignore ST1001 dot-imports
	. "github.com/onsi/gomega"
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
func NewLocalGitRepo(logger *TestLogger) *LocalGitRepo {
	p := ginkgo.GinkgoT().TempDir()
	ginkgo.DeferCleanup(func() {
		logger.Info("📦 [LOCAL] 🧹 Cleaning up local repository", "path", p)
		Expect(os.RemoveAll(p)).NotTo(HaveOccurred())
	})

	logger.Info("📦 [LOCAL] 📁 Creating new local repository at %s", p)
	r := &LocalGitRepo{Path: p, logger: logger}
	r.Git("init")
	logger.Success("📦 [LOCAL] Local repository initialized successfully")
	return r
}

// CreateDirPath creates a directory path in the repository.
// It creates all necessary parent directories if they don't exist.
func (r *LocalGitRepo) CreateDirPath(dirpath string) {
	r.logger.Info("📦 [LOCAL] 📁 Creating directory path '%s' in repository", dirpath)
	err := os.MkdirAll(filepath.Join(r.Path, dirpath), 0755)
	Expect(err).NotTo(HaveOccurred())
	r.logger.Success("📦 [LOCAL] Directory path '%s' created successfully", dirpath)
}

// CreateFile creates a new file in the repository with the specified filename
// and content. The file is created with read/write permissions for the owner only.
func (r *LocalGitRepo) CreateFile(filename, content string) {
	r.logger.Info("📦 [LOCAL] 📝 Creating file '%s' in repository", filename)
	err := os.WriteFile(filepath.Join(r.Path, filename), []byte(content), 0600)
	Expect(err).NotTo(HaveOccurred())
	r.logger.Success("📦 [LOCAL] 📝 File '%s' created successfully", filename)
}

// UpdateFile updates an existing file in the repository with new content.
// The file must exist before calling this method.
func (r *LocalGitRepo) UpdateFile(filename, content string) {
	r.logger.Info("📦 [LOCAL] 📝 Updating file '%s' in repository", filename)
	err := os.WriteFile(filepath.Join(r.Path, filename), []byte(content), 0600)
	Expect(err).NotTo(HaveOccurred())
	r.logger.Success("📦 [LOCAL] 📝 File '%s' updated successfully", filename)
}

// Git executes a Git command in the repository directory.
// It logs the command being executed and its output for debugging purposes.
// The command is executed with GIT_TERMINAL_PROMPT=0 to prevent interactive prompts.
func (r *LocalGitRepo) Git(args ...string) string {
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
		Expect(err).NotTo(HaveOccurred(), "git command failed: %s\nOutput: %s", cmdStr, output)
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

func (r *LocalGitRepo) QuickInit(user *User, remoteURL string) (client nanogit.Client, fileName string) {
	r.logger.Info("📦 [LOCAL] Setting up local repository")
	r.Git("config", "user.name", user.Username)
	r.Git("config", "user.email", user.Email)
	r.Git("remote", "add", "origin", remoteURL)

	r.logger.Info("📦 [LOCAL] Creating and committing test file")
	testContent := []byte("test content")
	r.CreateFile("test.txt", string(testContent))
	r.Git("add", "test.txt")
	r.Git("commit", "-m", "Initial commit")

	r.logger.Info("📦 [LOCAL] Setting up main branch and pushing changes")
	r.Git("branch", "-M", "main")
	r.Git("push", "origin", "main", "--force")

	r.logger.Info("📦 [LOCAL] Tracking current branch")
	r.Git("branch", "--set-upstream-to=origin/main", "main")

	client, err := nanogit.NewHTTPClient(remoteURL, nanogit.WithBasicAuth(user.Username, user.Password))
	Expect(err).NotTo(HaveOccurred())
	return client, "test.txt"
}

func (r *LocalGitRepo) LogRepoContents() {
	r.logger.Info("📦 [LOCAL] Repository contents:")
	var printDir func(path string, indent string)
	printDir = func(path string, indent string) {
		files, err := os.ReadDir(path)
		Expect(err).NotTo(HaveOccurred())
		for _, file := range files {
			fullPath := filepath.Join(path, file.Name())
			if file.IsDir() {
				r.logger.Info(indent + "📁 " + file.Name() + "/")
				printDir(fullPath, indent+"  ")
			} else {
				r.logger.Info(indent + "📄 " + file.Name())
			}
		}
	}
	printDir(r.Path, "")
}
