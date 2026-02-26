package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Server represents a Gitea server instance running in a container.
// It provides methods to manage users, repositories, and server operations
// for testing purposes.
type Server struct {
	Host string
	Port string

	container     testcontainers.Container
	logger        Logger
	cleanupFuncs  []func() error
	ctx           context.Context
	cancelContext context.CancelFunc
}

// containerLogger adapts our Logger interface to testcontainers.LogConsumer.
type containerLogger struct {
	logger Logger
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
		l.logger.Logf("🖥️  [SERVER] 🔒 %s", content)
	case strings.Contains(content, "403 Forbidden"):
		l.logger.Logf("🖥️  [SERVER] 🚫 %s", content)
	case strings.Contains(content, "404 Not Found"):
		l.logger.Logf("🖥️  [SERVER] 🔍 %s", content)
	case strings.Contains(content, "500 Internal Server Error"):
		l.logger.Logf("🖥️  [SERVER] 💥 %s", content)
	case strings.Contains(content, "200 OK"):
		l.logger.Logf("🖥️ [SERVER] %s", content)
	case strings.Contains(content, "201 Created"):
		l.logger.Logf("🖥️  [SERVER] ✨ %s", content)
	case strings.Contains(content, "204 No Content"):
		l.logger.Logf("🖥️  [SERVER] ✨ %s", content)
	case strings.Contains(strings.ToLower(content), "error"):
		l.logger.Logf("🖥️  [SERVER] [ERROR] %s", content)
	case strings.Contains(strings.ToLower(content), "warn"):
		l.logger.Logf("🖥️  [SERVER] [WARN] %s", content)
	case strings.Contains(strings.ToLower(content), "info"):
		l.logger.Logf("🖥️  [SERVER] [INFO] %s", content)
	default:
		l.logger.Logf("🖥️  [SERVER] %s", content)
	}
}

// NewServer creates and initializes a new Gitea server instance in a container.
// It configures the server with default settings and waits for it to be ready.
// The server is configured with:
// - SQLite database
// - Disabled registration
// - Pre-configured admin user
// - Disabled SSH and mailer
//
// Returns a Server instance ready for testing. Use Cleanup() to stop and remove
// the container, or use the cleanup function returned by QuickServer.
func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Create a cancellable context for the container lifecycle
	containerCtx, cancel := context.WithCancel(ctx)

	cfg.Logger.Logf("🚀 Starting Gitea server container...")

	containerLog := &containerLogger{logger: cfg.Logger}

	// Determine the full image name
	image := cfg.GiteaImage
	if cfg.GiteaVersion != "" {
		image = fmt.Sprintf("%s:%s", cfg.GiteaImage, cfg.GiteaVersion)
	}

	// Start Gitea container
	req := testcontainers.ContainerRequest{
		Image:        image,
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
		WaitingFor: wait.ForHTTP("/api/v1/version").WithPort("3000").WithStartupTimeout(cfg.StartTimeout),
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{containerLog},
		},
	}

	container, err := testcontainers.GenericContainer(containerCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(containerCtx)
	if err != nil {
		cancel()
		_ = container.Terminate(containerCtx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(containerCtx, "3000")
	if err != nil {
		cancel()
		_ = container.Terminate(containerCtx)
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	cfg.Logger.Logf("✅ Gitea server ready at http://%s:%s", host, port.Port())

	server := &Server{
		Host:          host,
		Port:          port.Port(),
		container:     container,
		logger:        cfg.Logger,
		ctx:           containerCtx,
		cancelContext: cancel,
	}

	return server, nil
}

// CreateUser creates a new user in the Gitea server with the specified credentials.
// The user is created with admin privileges and password change requirement disabled.
// Uses a unique suffix based on timestamp and random data to avoid collisions in parallel tests.
//
// Returns the created User or an error if creation fails.
func (s *Server) CreateUser(ctx context.Context) (*User, error) {
	// Generate a unique suffix using nanosecond timestamp + random bytes
	// This ensures uniqueness even when tests run in parallel
	now := time.Now().UnixNano()
	var randomBytes [4]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Combine timestamp and random data for a unique suffix
	suffix := fmt.Sprintf("%d%x", now, randomBytes)

	user := &User{
		Username: fmt.Sprintf("testuser-%s", suffix),
		Email:    fmt.Sprintf("test-%s@example.com", suffix),
		Password: fmt.Sprintf("testpass-%s", suffix),
	}
	s.logger.Logf("👤 Creating test user '%s'...", user.Username)

	execResult, reader, err := s.container.Exec(ctx, []string{
		"su", "git", "-c",
		fmt.Sprintf("gitea admin user create --username %s --email %s --password %s --must-change-password=false --admin",
			user.Username, user.Email, user.Password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute user creation command: %w", err)
	}

	execOutput, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read command output: %w", err)
	}

	s.logger.Logf("📋 User creation output: %s", string(execOutput))

	if execResult != 0 {
		return nil, fmt.Errorf("failed to create user (exit code %d): %s", execResult, string(execOutput))
	}

	s.logger.Logf("✅ Test user '%s' created successfully", user.Username)
	return user, nil
}

// GenerateUserToken creates a new access token for the specified user in the Gitea server.
// The token is created with all permissions enabled.
//
// Returns the generated token string (prefixed with "token ") or an error if generation fails.
func (s *Server) GenerateUserToken(ctx context.Context, username string) (string, error) {
	s.logger.Logf("🔑 Generating access token for user '%s'...", username)

	execResult, reader, err := s.container.Exec(ctx, []string{
		"su", "git", "-c",
		fmt.Sprintf("gitea admin user generate-access-token --username %s --scopes all", username),
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute token generation command: %w", err)
	}

	execOutput, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read command output: %w", err)
	}

	s.logger.Logf("📋 Token generation output: %s", string(execOutput))

	if execResult != 0 {
		return "", fmt.Errorf("failed to generate token (exit code %d): %s", execResult, string(execOutput))
	}

	// Extract token from output - it's the last line
	lines := strings.Split(strings.TrimSpace(string(execOutput)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("unexpected empty token output")
	}

	tokenLine := strings.TrimSpace(lines[len(lines)-1])
	if tokenLine == "" {
		return "", fmt.Errorf("unexpected empty token line")
	}

	chunks := strings.Split(tokenLine, " ")
	if len(chunks) == 0 {
		return "", fmt.Errorf("unexpected token format")
	}

	token := chunks[len(chunks)-1]
	if token == "" {
		return "", fmt.Errorf("unexpected empty token")
	}

	token = "token " + token

	s.logger.Logf("✅ Access token generated successfully for user '%s'", username)
	return token, nil
}

// CreateRepo creates a new repository in the Gitea server for the specified user.
// It returns both the public repository URL and an authenticated repository URL
// that includes the user's credentials.
//
// Returns the created Repo or an error if creation fails.
func (s *Server) CreateRepo(ctx context.Context, name string, user *User) (*Repo, error) {
	s.logger.Logf("📦 Creating repository '%s' for user '%s'...", name, user.Username)

	httpClient := http.Client{}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", s.Host, s.Port)
	jsonData := []byte(fmt.Sprintf(`{"name":"%s"}`, name))

	req, err := http.NewRequestWithContext(ctx, "POST", createRepoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user.Username, user.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	s.logger.Logf("✅ Repository '%s' created successfully", name)
	return newRepo(name, user, s.Host, s.Port), nil
}

// URL returns the base URL of the Gitea server.
func (s *Server) URL() string {
	return fmt.Sprintf("http://%s:%s", s.Host, s.Port)
}

// Cleanup stops and removes the Gitea server container.
// It should be called when the server is no longer needed.
// This method is safe to call multiple times.
func (s *Server) Cleanup() error {
	if s.container != nil {
		s.logger.Logf("🧹 Cleaning up Gitea server container...")
		// Use a fresh context for cleanup to avoid "context canceled" errors
		cleanupCtx := context.Background()
		if err := s.container.Terminate(cleanupCtx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}

	// Cancel the context after cleanup is done
	if s.cancelContext != nil {
		s.cancelContext()
	}

	// Run any registered cleanup functions
	for _, cleanup := range s.cleanupFuncs {
		if err := cleanup(); err != nil {
			return err
		}
	}

	return nil
}
