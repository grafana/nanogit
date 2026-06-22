package gittest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
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
//
// The server is automatically configured with sensible defaults for testing:
//   - SQLite database (no external database required)
//   - Disabled user registration (create users via CreateUser)
//   - Pre-configured admin user for internal operations
//   - Disabled SSH and email (HTTPS-only)
//   - Automatic port allocation (no conflicts)
//
// The function waits for the server to be fully ready before returning.
// Always call Cleanup() when done to stop and remove the container.
//
// Example:
//
//	server, err := gittest.NewServer(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer server.Cleanup()
//
// Options can be provided to customize the server:
//
//	server, err := gittest.NewServer(ctx,
//		gittest.WithLogger(gittest.NewTestLogger(t)),
//		gittest.WithTimeout(60*time.Second),
//	)
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

	// Build environment variables
	env := map[string]string{
		"GITEA__database__DB_TYPE":                "sqlite3",
		"GITEA__database__SQLITE_JOURNAL_MODE":    "WAL",
		"GITEA__database__SQLITE_TIMEOUT":         "10000",
		"GITEA__server__ROOT_URL":                 "http://localhost:3000/",
		"GITEA__server__HTTP_PORT":                "3000",
		"GITEA__service__DISABLE_REGISTRATION":    "true",
		"GITEA__security__INSTALL_LOCK":           "true",
		"GITEA__security__DEFAULT_ADMIN_NAME":     "giteaadmin",
		"GITEA__security__DEFAULT_ADMIN_PASSWORD": "admin123",
		"GITEA__security__SECRET_KEY":             "supersecretkey",
		"GITEA__security__INTERNAL_TOKEN":         "internal",
		"GITEA__security__DISABLE_GITEA_SSH":      "true",
		// Allow site admins to set custom git hooks via the API.
		// Gitea defaults this to true; we flip it so integration tests
		// can install per-repo pre-receive hooks to exercise the
		// receive-pack error path with realistic side-band channel-2
		// progress (e.g. hook stderr surfaced as remote: messages).
		"GITEA__security__DISABLE_GIT_HOOKS": "false",
		"GITEA__mailer__ENABLED":             "false",
	}

	if len(cfg.TrustedSSHKeys) > 0 {
		env["GITEA__repository_0X2E_signing__TRUSTED_SSH_KEYS"] = strings.Join(cfg.TrustedSSHKeys, ",")
	}

	// Start Gitea container
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"3000/tcp"},
		Env:          env,
		WaitingFor:   wait.ForHTTP("/api/v1/version").WithPort("3000").WithStartupTimeout(cfg.StartTimeout),
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

// CreateUser creates a new test user in the Gitea server.
//
// The user is automatically configured with:
//   - Auto-generated unique username (e.g., "user-1234567890ab")
//   - Auto-generated email address
//   - Auto-generated password
//   - Admin privileges for repository creation
//   - Pre-generated access token for authentication
//
// The username includes a timestamp and random suffix to prevent collisions
// in parallel test execution.
//
// Returns the created User with all credentials, or an error if creation fails.
//
// Example:
//
//	user, err := server.CreateUser(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// user.Username, user.Password, user.Token are now available
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

// CreateToken creates a new access token for the specified user in the Gitea server.
//
// The token is created with all permissions enabled and is prefixed with "token ".
// This is consistent with other Create* methods in the API.
//
// Returns the generated token string or an error if creation fails.
//
// Example:
//
//	token, err := server.CreateToken(ctx, user.Username)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// Use token for API authentication
func (s *Server) CreateToken(ctx context.Context, username string) (string, error) {
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

// CreateRepo creates a new Git repository in the Gitea server.
//
// The repository is created under the specified user's account with these settings:
//   - Private repository (not publicly accessible)
//   - No initial README or .gitignore
//   - Empty repository ready for pushes
//
// The returned Repo contains:
//   - URL: Public HTTPS URL (requires authentication to access)
//   - AuthURL: HTTPS URL with embedded credentials for easy cloning
//   - Name: Repository name
//   - Owner: Username of the repository owner
//
// Returns the created Repo with access URLs, or an error if creation fails.
//
// Example:
//
//	repo, err := server.CreateRepo(ctx, "myproject", user)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// Use repo.AuthURL for git operations that need authentication
func (s *Server) CreateRepo(ctx context.Context, name string, user *User) (*RemoteRepository, error) {
	s.logger.Logf("📦 Creating repository '%s' for user '%s'...", name, user.Username)

	httpClient := http.Client{}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", s.Host, s.Port)

	// Use json.Marshal to properly escape the repository name
	jsonData, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

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
		// Set to nil to make this method truly idempotent
		s.container = nil
	}

	// Cancel the context after cleanup is done
	if s.cancelContext != nil {
		s.cancelContext()
		s.cancelContext = nil
	}

	// Run any registered cleanup functions
	for _, cleanup := range s.cleanupFuncs {
		if err := cleanup(); err != nil {
			return err
		}
	}
	s.cleanupFuncs = nil

	return nil
}

// SetUserPrimaryEmail sets the primary email address for a user via the admin API.
// The user's Email field is updated to match on success.
func (s *Server) SetUserPrimaryEmail(ctx context.Context, user *User, email string) error {
	body, err := json.Marshal(map[string]any{
		"email":      email,
		"source_id":  0,
		"login_name": user.Username,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", s.URL()+"/api/v1/admin/users/"+user.Username, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user.Username, user.Password)

	if err := doOK(req, "set primary email"); err != nil {
		return err
	}
	user.Email = email
	return nil
}

// UploadGPGKey uploads an armored public GPG key to the authenticated user's account.
func (s *Server) UploadGPGKey(ctx context.Context, token string, armoredPublic []byte) error {
	body, err := json.Marshal(map[string]string{"armored_public_key": string(armoredPublic)})
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.URL()+"/api/v1/user/gpg_keys", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	return doOK(req, "upload gpg key")
}

// CommitVerification reports whether the given commit is verified by the server,
// along with the reason the server provides.
func (s *Server) CommitVerification(ctx context.Context, token, owner, repo, sha string) (bool, string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/git/commits/%s?verification=true", s.URL(), owner, repo, sha)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("commit lookup failed (status %d): %s", resp.StatusCode, body)
	}

	var got struct {
		Commit struct {
			Verification struct {
				Verified bool   `json:"verified"`
				Reason   string `json:"reason"`
			} `json:"verification"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		return false, "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return got.Commit.Verification.Verified, got.Commit.Verification.Reason, nil
}

func doOK(req *http.Request, what string) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s: %s", what, resp.Status, out)
	}
	return nil
}
