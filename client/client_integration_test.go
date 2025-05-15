//go:build integration
// +build integration

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupGiteaServer(t *testing.T) (repoURL string, cleanup func()) {
	ctx := context.Background()

	// Start Gitea container
	req := testcontainers.ContainerRequest{
		Image:        "gitea/gitea:latest",
		ExposedPorts: []string{"3000/tcp"},
		Env: map[string]string{
			"GITEA__database__DB_TYPE":                "sqlite3",
			"GITEA__server__ROOT_URL":                 "http://localhost:3000/",
			"GITEA__server__HTTP_PORT":                "3000",
			"GITEA__service__DISABLE_REGISTRATION":    "true",
			"GITEA__security__INSTALL_LOCK":           "true",
			"GITEA__security__DEFAULT_ADMIN_NAME":     "giteaadmin",
			"GITEA__security__DEFAULT_ADMIN_PASSWORD": "admin123",
			"GITEA__security__SECRET_KEY":             "supersecretkey",
			"GITEA__security__INTERNAL_TOKEN":         "internal",
			"GITEA__security__DISABLE_GITEA_SSH":      "true",
			"GITEA__mailer__ENABLED":                  "false",
		},
		WaitingFor: wait.ForHTTP("/api/v1/version").WithPort("3000").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "3000")
	require.NoError(t, err)

	fmt.Println("Creating test user")
	// Create test user using Gitea CLI
	execResult, reader, err := container.Exec(ctx, []string{
		"su", "git", "-c", "gitea admin user create --username testuser --email test@example.com --password testpass123 --must-change-password=false --admin",
	})
	require.NoError(t, err)
	_, err = io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, 0, execResult)

	// Create a repository without authentication
	repoName := "testrepo"
	repoURL = fmt.Sprintf("http://%s:%s/%s/%s.git", host, port.Port(), "testuser", repoName)
	authRepoURL := fmt.Sprintf("http://testuser:testpass123@%s:%s/%s/%s.git", host, port.Port(), "testuser", repoName)

	// Create the repository using the Gitea API
	httpClient := http.Client{}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", host, port.Port())
	jsonData := []byte(fmt.Sprintf(`{"name":"%s"}`, repoName))
	reqCreate, _ := http.NewRequest("POST", createRepoURL, bytes.NewBuffer(jsonData))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.SetBasicAuth("testuser", "testpass123")
	resp, reqErr := httpClient.Do(reqCreate)
	require.NoError(t, reqErr)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Initialize a new git repository from scratch
	tmpDir, err := os.MkdirTemp("", "gitea-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	runGitCmd(t, tmpDir, "init")
	runGitCmd(t, tmpDir, "config", "user.name", "testuser")
	runGitCmd(t, tmpDir, "config", "user.email", "test@example.com")
	createTestFile(t, tmpDir, "test.txt", "test content")
	runGitCmd(t, tmpDir, "add", "test.txt")
	runGitCmd(t, tmpDir, "commit", "-m", "Initial commit")
	runGitCmd(t, tmpDir, "branch", "-M", "main")
	runGitCmd(t, tmpDir, "branch", "test-branch")
	runGitCmd(t, tmpDir, "tag", "v1.0.0")

	// Push the repository to Gitea
	runGitCmd(t, tmpDir, "remote", "add", "origin", authRepoURL)
	runGitCmd(t, tmpDir, "push", "-u", "origin", "main", "--force")
	runGitCmd(t, tmpDir, "push", "origin", "test-branch", "--force")
	runGitCmd(t, tmpDir, "push", "origin", "v1.0.0", "--force")

	cleanup = func() {
		err := container.Terminate(ctx)
		require.NoError(t, err)
	}

	return repoURL, cleanup
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	// Log the git command being executed
	t.Logf("Running git command: git %s in directory: %s", strings.Join(args, " "), dir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Git command output:\n%s", string(output))
		require.NoError(t, err, "git command failed %s: %s", args, output)
	}

	// Log successful command output
	t.Logf("Git command output:\n%s", string(output))
}

func createTestFile(t *testing.T, dir, name, content string) {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

func TestClient_ListRefs(t *testing.T) {
	repoURL, cleanup := setupGiteaServer(t)
	defer cleanup()

	client, err := New(repoURL, WithBasicAuth("testuser", "testpass123"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refs, err := client.ListRefs(ctx)
	require.NoError(t, err, "ListRefs failed: %v", err)

	assert.NotEmpty(t, refs, "should have at least one reference")

	var (
		masterRef     *Ref
		testBranchRef *Ref
		tagRef        *Ref
	)

	for _, ref := range refs {
		switch ref.Name {
		case "refs/heads/main":
			masterRef = &ref
		case "refs/heads/test-branch":
			testBranchRef = &ref
		case "refs/tags/v1.0.0":
			tagRef = &ref
		}
	}

	require.NotNil(t, masterRef, "should have master branch")
	require.NotNil(t, testBranchRef, "should have test-branch")
	require.NotNil(t, tagRef, "should have v1.0.0 tag")

	for _, ref := range []*Ref{masterRef, testBranchRef, tagRef} {
		require.Len(t, ref.Hash, 40, "hash should be 40 characters")
	}
}
