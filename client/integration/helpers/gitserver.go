// Package helpers provides utility functions and types for testing Git operations
// in both local and remote environments. This file specifically handles remote
// Git server operations using Gitea as a test server.
package helpers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// GitServer represents a Gitea server instance running in a container.
// It provides methods to manage users, repositories, and server operations
// for testing purposes.
type GitServer struct {
	Host      string                   // Host address of the Gitea server
	Port      string                   // Port number the server is running on
	container testcontainers.Container // The container running the Gitea server
}

// NewGitServer creates and initializes a new Gitea server instance in a container.
// It configures the server with default settings and waits for it to be ready.
// The server is configured with:
// - SQLite database
// - Disabled registration
// - Pre-configured admin user
// - Disabled SSH and mailer
// Returns a GitServer instance ready for testing.
func NewGitServer(t *testing.T) *GitServer {
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

	// Start following logs
	logs, err := container.Logs(ctx)
	require.NoError(t, err)
	go func() {
		_, err := io.Copy(os.Stdout, logs)
		if err != nil {
			t.Errorf("Error copying logs: %v", err)
		}
	}()

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "3000")
	require.NoError(t, err)

	return &GitServer{
		Host:      host,
		Port:      port.Port(),
		container: container,
	}
}

// CreateUser creates a new user in the Gitea server with the specified credentials.
// The user is created with admin privileges and password change requirement disabled.
func (s *GitServer) CreateUser(t *testing.T, username, email, password string) {
	t.Log("Creating test user...")
	execResult, reader, err := s.container.Exec(context.Background(), []string{
		"su", "git", "-c", "gitea admin user create --username testuser --email test@example.com --password testpass123 --must-change-password=false --admin",
	})

	require.NoError(t, err)
	execOutput, err := io.ReadAll(reader)
	require.NoError(t, err)
	t.Logf("User creation output: %s", string(execOutput))
	require.Equal(t, 0, execResult)
}

// CreateRepo creates a new repository in the Gitea server for the specified user.
// It returns both the public repository URL and an authenticated repository URL
// that includes the user's credentials.
func (s *GitServer) CreateRepo(t *testing.T, repoName string, username, password string) (repoURL string, authRepoURL string) {
	// FIXME: can I create one with CLI instead?
	t.Log("Creating repository...")
	httpClient := http.Client{}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", s.Host, s.Port)
	jsonData := []byte(fmt.Sprintf(`{"name":"%s"}`, repoName))
	reqCreate, err := http.NewRequestWithContext(context.Background(), "POST", createRepoURL, bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.SetBasicAuth(username, password)
	resp, reqErr := httpClient.Do(reqCreate)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, reqErr)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	return fmt.Sprintf("http://%s:%s/%s/%s.git", s.Host, s.Port, username, repoName),
		fmt.Sprintf("http://%s:%s@%s:%s/%s/%s.git", username, password, s.Host, s.Port, username, repoName)
}

// Cleanup terminates the Gitea server container and cleans up associated resources.
// This should be called when the server is no longer needed.
func (s *GitServer) Cleanup(t *testing.T) {
	if s.container != nil {
		err := s.container.Terminate(context.Background())
		require.NoError(t, err)
	}
}
