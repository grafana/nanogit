package performance

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// GitServer represents a Gitea server instance running in a container
// with network latency simulation for performance testing
type GitServer struct {
	Host      string
	Port      string
	container testcontainers.Container
	network   *testcontainers.DockerNetwork
	users     map[string]*User // Cache created users
}

// User represents a Gitea user for testing
type User struct {
	Username string
	Email    string
	Password string
	Token    string
}

// NewGitServer creates a new Gitea server with optional network latency simulation
func NewGitServer(ctx context.Context, latency time.Duration) (*GitServer, error) {
	// Create a network for latency simulation if specified
	var dockerNetwork *testcontainers.DockerNetwork
	var networkName string
	
	if latency > 0 {
		net, err := network.New(ctx, network.WithDriver("bridge"))
		if err != nil {
			return nil, fmt.Errorf("failed to create network: %w", err)
		}
		dockerNetwork = net
		networkName = net.Name
	}

	// Configure container request
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
			"GITEA__security__DISABLE_GIT_SSH":        "true",
			"GITEA__mailer__ENABLED":                  "false",
		},
		WaitingFor: wait.ForHTTP("/api/v1/version").WithPort("3000").WithStartupTimeout(60 * time.Second),
	}

	// Add network if latency simulation is enabled
	if dockerNetwork != nil {
		req.Networks = []string{networkName}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	port, err := container.MappedPort(ctx, "3000")
	if err != nil {
		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	server := &GitServer{
		Host:      host,
		Port:      port.Port(),
		container: container,
		network:   dockerNetwork,
		users:     make(map[string]*User),
	}

	// Add network latency if specified
	if latency > 0 {
		if err := server.configureNetworkLatency(ctx, latency); err != nil {
			return nil, fmt.Errorf("failed to configure network latency: %w", err)
		}
	}

	return server, nil
}

// configureNetworkLatency adds network latency simulation using tc (traffic control)
func (s *GitServer) configureNetworkLatency(ctx context.Context, latency time.Duration) error {
	// Install traffic control tools and add latency
	commands := [][]string{
		// Install iproute2 package for tc command
		{"apk", "add", "iproute2"},
		// Add latency to the network interface
		{"tc", "qdisc", "add", "dev", "eth0", "root", "netem", "delay", fmt.Sprintf("%dms", latency.Milliseconds())},
	}

	for _, cmd := range commands {
		execResult, reader, err := s.container.Exec(ctx, cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command %v: %w", cmd, err)
		}
		
		output, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read command output: %w", err)
		}
		
		if execResult != 0 {
			return fmt.Errorf("command %v failed with exit code %d: %s", cmd, execResult, string(output))
		}
	}

	return nil
}

// CreateUser creates a new user in the Gitea server
func (s *GitServer) CreateUser(ctx context.Context) (*User, error) {
	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}
	suffix = suffix % 10000

	user := &User{
		Username: fmt.Sprintf("testuser-%d", suffix),
		Email:    fmt.Sprintf("test-%d@example.com", suffix),
		Password: fmt.Sprintf("testpass-%d", suffix),
	}

	// Create user using Gitea CLI
	execResult, reader, err := s.container.Exec(ctx, []string{
		"su", "git", "-c", 
		fmt.Sprintf("gitea admin user create --username %s --email %s --password %s --must-change-password=false --admin", 
			user.Username, user.Email, user.Password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read user creation output: %w", err)
	}

	if execResult != 0 {
		return nil, fmt.Errorf("user creation failed with exit code %d: %s", execResult, string(output))
	}

	// Generate access token
	token, err := s.generateUserToken(ctx, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	user.Token = token

	s.users[user.Username] = user
	return user, nil
}

// generateUserToken creates an access token for the user
func (s *GitServer) generateUserToken(ctx context.Context, username string) (string, error) {
	execResult, reader, err := s.container.Exec(ctx, []string{
		"su", "git", "-c", 
		fmt.Sprintf("gitea admin user generate-access-token --username %s --scopes all", username),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read token output: %w", err)
	}

	if execResult != 0 {
		return "", fmt.Errorf("token generation failed with exit code %d: %s", execResult, string(output))
	}

	// Extract token from output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no token output received")
	}

	tokenLine := strings.TrimSpace(lines[len(lines)-1])
	if tokenLine == "" {
		return "", fmt.Errorf("empty token received")
	}

	chunks := strings.Split(tokenLine, " ")
	if len(chunks) == 0 {
		return "", fmt.Errorf("malformed token output")
	}

	token := chunks[len(chunks)-1]
	if token == "" {
		return "", fmt.Errorf("empty token extracted")
	}

	return "token " + token, nil
}

// CreateRepo creates a new repository for the user
func (s *GitServer) CreateRepo(ctx context.Context, repoName string, user *User) (*Repository, error) {
	httpClient := &http.Client{}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", s.Host, s.Port)
	
	jsonData := []byte(fmt.Sprintf(`{"name":"%s"}`, repoName))
	req, err := http.NewRequestWithContext(ctx, "POST", createRepoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user.Username, user.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create repository, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return &Repository{
		Name:    repoName,
		Owner:   user.Username,
		Host:    s.Host,
		Port:    s.Port,
		User:    user,
	}, nil
}

// ProvisionTestRepositories creates repositories with different characteristics for performance testing
func (s *GitServer) ProvisionTestRepositories(ctx context.Context) ([]*Repository, error) {
	specs := GetStandardSpecs()
	repositories := make([]*Repository, len(specs))

	for i, spec := range specs {
		// Create user for this repository
		user, err := s.CreateUser(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create user for %s repository: %w", spec.Name, err)
		}

		// Create repository
		repo, err := s.CreateRepo(ctx, fmt.Sprintf("%s-repo", spec.Name), user)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s repository: %w", spec.Name, err)
		}

		// Create initial commit so repository is not empty
		if err := s.createInitialCommit(ctx, repo); err != nil {
			return nil, fmt.Errorf("failed to create initial commit for %s repository: %w", spec.Name, err)
		}

		repositories[i] = repo
	}

	return repositories, nil
}

// Cleanup stops the container and cleans up resources
func (s *GitServer) Cleanup(ctx context.Context) error {
	if s.container != nil {
		if err := s.container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}

	if s.network != nil {
		if err := s.network.Remove(ctx); err != nil {
			return fmt.Errorf("failed to remove network: %w", err)
		}
	}

	return nil
}

// createInitialCommit creates an initial commit in the repository to make it non-empty
func (s *GitServer) createInitialCommit(ctx context.Context, repo *Repository) error {
	httpClient := &http.Client{}
	
	// Create initial README file via API
	createFileURL := fmt.Sprintf("http://%s:%s/api/v1/repos/%s/%s/contents/README.md", 
		s.Host, s.Port, repo.Owner, repo.Name)
	
	readmeContent := fmt.Sprintf("# %s\n\nTest repository for performance benchmarking.", repo.Name)
	
	// Base64 encode the content
	encodedContent := base64.StdEncoding.EncodeToString([]byte(readmeContent))
	
	jsonData := []byte(fmt.Sprintf(`{
		"message": "Initial commit",
		"content": "%s",
		"author": {
			"name": "Performance Test",
			"email": "test@example.com"
		},
		"committer": {
			"name": "Performance Test", 
			"email": "test@example.com"
		}
	}`, encodedContent))
	
	req, err := http.NewRequestWithContext(ctx, "POST", createFileURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(repo.User.Username, repo.User.Password)
	
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create initial file: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create initial file, status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Repository represents a test repository
type Repository struct {
	Name  string
	Owner string
	Host  string
	Port  string
	User  *User
}

// HTTPURL returns the HTTP URL for the repository
func (r *Repository) HTTPURL() string {
	return fmt.Sprintf("http://%s:%s/%s/%s.git", r.Host, r.Port, r.Owner, r.Name)
}

// AuthURL returns the authenticated HTTP URL for the repository
func (r *Repository) AuthURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%s/%s/%s.git", 
		r.User.Username, r.User.Password, r.Host, r.Port, r.Owner, r.Name)
}

