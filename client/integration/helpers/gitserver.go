package helpers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type containerLogger struct {
	t *testing.T
}

func NewContainerLogger(t *testing.T) *containerLogger {
	return &containerLogger{t: t}
}

func (l *containerLogger) Accept(log testcontainers.Log) {
	content := string(log.Content)
	// Add emojis based on log level/content
	switch {
	case strings.Contains(strings.ToLower(content), "error"):
		l.t.Logf("❌ %s", content)
	case strings.Contains(strings.ToLower(content), "warn"):
		l.t.Logf("⚠️ %s", content)
	case strings.Contains(strings.ToLower(content), "info"):
		l.t.Logf("ℹ️ %s", content)
	default:
		l.t.Logf("📝 %s", content)
	}
}

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

	t.Log("🚀 Starting Gitea server container...")

	logConsumer := NewContainerLogger(t)

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
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{logConsumer},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		t.Log("🧹 Cleaning up Gitea server container...")
		require.NoError(t, container.Terminate(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "3000")
	require.NoError(t, err)

	t.Logf("✅ Gitea server ready at http://%s:%s", host, port.Port())

	return &GitServer{
		Host:      host,
		Port:      port.Port(),
		container: container,
	}
}

// CreateUser creates a new user in the Gitea server with the specified credentials.
// The user is created with admin privileges and password change requirement disabled.
func (s *GitServer) CreateUser(t *testing.T) *User {
	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	require.NoError(t, err)
	suffix = suffix % 10000

	user := &User{
		Username: fmt.Sprintf("testuser-%d", suffix),
		Email:    fmt.Sprintf("test-%d@example.com", suffix),
		Password: fmt.Sprintf("testpass-%d", suffix),
	}
	t.Logf("👤 Creating test user '%s'...", user.Username)
	execResult, reader, err := s.container.Exec(context.Background(), []string{
		"su", "git", "-c", fmt.Sprintf("gitea admin user create --username %s --email %s --password %s --must-change-password=false --admin", user.Username, user.Email, user.Password),
	})

	require.NoError(t, err)
	execOutput, err := io.ReadAll(reader)
	require.NoError(t, err)
	t.Logf("📋 User creation output: %s", string(execOutput))
	require.Equal(t, 0, execResult)

	t.Logf("✅ Test user '%s' created successfully", user.Username)
	return user
}

// CreateRepo creates a new repository in the Gitea server for the specified user.
// It returns both the public repository URL and an authenticated repository URL
// that includes the user's credentials.
func (s *GitServer) CreateRepo(t *testing.T, repoName string, username, password string) *RemoteRepo {
	// FIXME: can I create one with CLI instead?
	t.Logf("📦 Creating repository '%s' for user '%s'...", repoName, username)
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

	t.Logf("✅ Repository '%s' created successfully", repoName)
	return NewRemoteRepo(t, repoName, username, password, s.Host, s.Port)
}
