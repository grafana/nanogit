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
	"time"

	"github.com/grafana/nanogit"
	//nolint:stylecheck // specifically ignore ST1001 (dot-imports)
	//lint:ignore ST1001 dot-imports
	. "github.com/onsi/ginkgo/v2"
	//nolint:stylecheck // specifically ignore ST1001 (dot-imports)
	//lint:ignore ST1001 dot-imports
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type containerLogger struct {
	*TestLogger
}

func (l *containerLogger) Accept(log testcontainers.Log) {
	content := strings.TrimSpace(string(log.Content))

	// Skip empty logs
	if content == "" {
		return
	}

	// Add emojis and colors based on log level/content
	switch {
	case strings.Contains(content, "401 Unauthorized"):
		l.Error("🖥️  [SERVER] 🔒 " + content)
	case strings.Contains(content, "403 Forbidden"):
		l.Error("🖥️  [SERVER] 🚫 " + content)
	case strings.Contains(content, "404 Not Found"):
		l.Info("🖥️  [SERVER] 🔍 " + content)
	case strings.Contains(content, "500 Internal Server Error"):
		l.Error("🖥️  [SERVER] 💥 " + content)
	case strings.Contains(content, "200 OK"):
		l.Success("🖥️ [SERVER] " + content)
	case strings.Contains(content, "201 Created"):
		l.Success("🖥️  [SERVER] ✨ " + content)
	case strings.Contains(content, "204 No Content"):
		l.Success("🖥️  [SERVER] ✨ " + content)
	case strings.Contains(strings.ToLower(content), "error"):
		l.Error("🖥️  [SERVER] " + content)
	case strings.Contains(strings.ToLower(content), "warn"):
		l.Warn("🖥️  [SERVER] " + content)
	case strings.Contains(strings.ToLower(content), "info"):
		l.Info("🖥️  [SERVER] " + content)
	default:
		l.Info("🖥️  [SERVER] " + content)
	}
}

// GitServer represents a Gitea server instance running in a container.
// It provides methods to manage users, repositories, and server operations
// for testing purposes.
type GitServer struct {
	Host      string                   // Host address of the Gitea server
	Port      string                   // Port number the server is running on
	container testcontainers.Container // The container running the Gitea server
	logger    *TestLogger
}

// NewGitServer creates and initializes a new Gitea server instance in a container.
// It configures the server with default settings and waits for it to be ready.
// The server is configured with:
// - SQLite database
// - Disabled registration
// - Pre-configured admin user
// - Disabled SSH and mailer
// Returns a GitServer instance ready for testing.
func NewGitServer(logger *TestLogger) *GitServer {
	ctx := context.Background()

	containerLogger := &containerLogger{logger}
	logger.Logf("%s🚀 Starting Gitea server container...%s", ColorGreen, ColorReset)

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
			Consumers: []testcontainers.LogConsumer{containerLogger},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		logger.Logf("%s🧹 Cleaning up Gitea server container...%s", ColorYellow, ColorReset)
		Expect(container.Terminate(ctx)).To(Succeed())
	})

	host, err := container.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	port, err := container.MappedPort(ctx, "3000")
	Expect(err).NotTo(HaveOccurred())

	logger.Logf("%s✅ Gitea server ready at http://%s:%s%s", ColorGreen, host, port.Port(), ColorReset)

	return &GitServer{
		Host:      host,
		Port:      port.Port(),
		container: container,
		logger:    logger,
	}
}

// CreateUser creates a new user in the Gitea server with the specified credentials.
// The user is created with admin privileges and password change requirement disabled.
func (s *GitServer) CreateUser() *User {
	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	Expect(err).NotTo(HaveOccurred())
	suffix = suffix % 10000

	user := &User{
		Username: fmt.Sprintf("testuser-%d", suffix),
		Email:    fmt.Sprintf("test-%d@example.com", suffix),
		Password: fmt.Sprintf("testpass-%d", suffix),
	}
	s.logger.Logf("%s👤 Creating test user '%s'...%s", ColorBlue, user.Username, ColorReset)
	execResult, reader, err := s.container.Exec(context.Background(), []string{
		"su", "git", "-c", fmt.Sprintf("gitea admin user create --username %s --email %s --password %s --must-change-password=false --admin", user.Username, user.Email, user.Password),
	})

	Expect(err).NotTo(HaveOccurred())
	execOutput, err := io.ReadAll(reader)
	Expect(err).NotTo(HaveOccurred())
	s.logger.Logf("%s📋 User creation output: %s%s", ColorCyan, string(execOutput), ColorReset)
	Expect(execResult).To(Equal(0))

	s.logger.Logf("%s✅ Test user '%s' created successfully%s", ColorGreen, user.Username, ColorReset)
	return user
}

// GenerateUserToken creates a new access token for the specified user in the Gitea server.
// The token is created with all permissions enabled.
func (s *GitServer) GenerateUserToken(username, password string) string {
	s.logger.Logf("%s🔑 Generating access token for user '%s'...%s", ColorBlue, username, ColorReset)
	execResult, reader, err := s.container.Exec(context.Background(), []string{
		"su", "git", "-c", fmt.Sprintf("gitea admin user generate-access-token --username %s --scopes all", username),
	})
	Expect(err).NotTo(HaveOccurred())
	execOutput, err := io.ReadAll(reader)
	Expect(err).NotTo(HaveOccurred())
	s.logger.Logf("%s📋 Token generation output: %s%s", ColorCyan, string(execOutput), ColorReset)
	Expect(execResult).To(Equal(0))

	// Extract token from output - it's the last line
	lines := strings.Split(strings.TrimSpace(string(execOutput)), "\n")
	Expect(lines).NotTo(BeEmpty(), "expected token output")
	tokenLine := strings.TrimSpace(lines[len(lines)-1])
	Expect(tokenLine).NotTo(BeEmpty(), "expected non-empty token")

	chunks := strings.Split(tokenLine, " ")
	Expect(chunks).NotTo(BeEmpty(), "expected chunks")
	token := chunks[len(chunks)-1]
	Expect(token).NotTo(BeEmpty(), "expected non-empty token")
	token = "token " + token

	s.logger.Logf("%s✅ Access token generated successfully for user '%s'%s (%s)", ColorGreen, username, ColorReset, token)
	return token
}

// CreateRepo creates a new repository in the Gitea server for the specified user.
// It returns both the public repository URL and an authenticated repository URL
// that includes the user's credentials.
func (s *GitServer) CreateRepo(repoName string, user *User) *RemoteRepo {
	// FIXME: can I create one with CLI instead?
	s.logger.Logf("%s📦 Creating repository '%s' for user '%s'...%s", ColorBlue, repoName, user.Username, ColorReset)
	httpClient := http.Client{}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", s.Host, s.Port)
	jsonData := []byte(fmt.Sprintf(`{"name":"%s"}`, repoName))
	reqCreate, err := http.NewRequestWithContext(context.Background(), "POST", createRepoURL, bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.SetBasicAuth(user.Username, user.Password)
	resp, reqErr := httpClient.Do(reqCreate)
	Expect(resp.Body.Close()).To(Succeed())
	Expect(reqErr).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	s.logger.Logf("%s✅ Repository '%s' created successfully%s", ColorGreen, repoName, ColorReset)
	return NewRemoteRepo(s.logger, repoName, user, s.Host, s.Port)
}

func (s *GitServer) TestRepo() (nanogit.Client, *RemoteRepo, *LocalGitRepo) {
	user := s.CreateUser()

	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	Expect(err).NotTo(HaveOccurred())
	suffix = suffix % 10000

	remote := s.CreateRepo(fmt.Sprintf("testrepo-%d", suffix), user)
	local := NewLocalGitRepo(s.logger)
	client, _ := local.QuickInit(user, remote.AuthURL())

	return client, remote, local
}
