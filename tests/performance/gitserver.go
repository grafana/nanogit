package performance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	// Add network and capabilities if latency simulation is enabled
	if dockerNetwork != nil {
		req.Networks = []string{networkName}
		req.CapAdd = []string{"NET_ADMIN"}
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
		Name:  repoName,
		Owner: user.Username,
		Host:  s.Host,
		Port:  s.Port,
		User:  user,
	}, nil
}

// RepoSpec defines the characteristics of a test repository
type RepoSpec struct {
	Name        string
	FileCount   int
	CommitCount int
	MaxDepth    int
	FileSizes   []int // Various file sizes in bytes
	BinaryFiles int   // Number of binary files
	Branches    int   // Number of branches
}

// GetStandardSpecs returns predefined repository specifications
func GetStandardSpecs() []RepoSpec {
	return []RepoSpec{
		{Name: "small"},
		{Name: "medium"},
		{Name: "large"},
		{Name: "xlarge"},
	}
}

// ProvisionTestRepositories extracts and mounts pre-created repositories for performance testing
func (s *GitServer) ProvisionTestRepositories(ctx context.Context) ([]*Repository, error) {
	specs := GetStandardSpecs()
	repositories := make([]*Repository, len(specs))

	for i, spec := range specs {
		// Create user for this repository
		user, err := s.CreateUser(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create user for %s repository: %w", spec.Name, err)
		}

		// Extract and mount pre-created repository
		repo, err := s.extractAndMountRepository(ctx, spec.Name, user)
		if err != nil {
			return nil, fmt.Errorf("failed to extract and mount %s repository: %w", spec.Name, err)
		}

		repositories[i] = repo
	}

	return repositories, nil
}

// extractAndMountRepository extracts a pre-created repository archive and mounts it in Gitea
func (s *GitServer) extractAndMountRepository(ctx context.Context, repoName string, user *User) (*Repository, error) {
	// Path to the archive file
	archivePath := fmt.Sprintf("./testdata/%s-repo.tar.gz", repoName)

	// Check if archive exists
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("archive file not found: %s (run 'go run ./cmd/generate_repo' to create it)", archivePath)
	}

	// Create temporary directory to extract the repository
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("nanogit-%s-*", repoName))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	// Don't defer cleanup - we need the directory to remain for mounting

	// Extract the archive
	if err := extractArchive(archivePath, tempDir); err != nil {
		os.RemoveAll(tempDir) // Cleanup on error
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Create the repository record in Gitea via API first
	repoFullName := fmt.Sprintf("%s-repo", repoName)

	// Mount the extracted repository directory into the container
	gitDataPath := fmt.Sprintf("/data/git/repositories/%s/%s.git", user.Username, repoFullName)
	if err := s.mountRepositoryToContainer(ctx, tempDir, gitDataPath); err != nil {
		os.RemoveAll(tempDir) // Cleanup on error
		return nil, fmt.Errorf("failed to mount repository to container: %w", err)
	}

	if err := s.createRepositoryRecord(ctx, repoFullName, user); err != nil {
		return nil, fmt.Errorf("failed to create repository record: %w", err)
	}

	return &Repository{
		Name:    repoFullName,
		Owner:   user.Username,
		Host:    s.Host,
		Port:    s.Port,
		User:    user,
		tempDir: tempDir, // Store for cleanup later
	}, nil
}

// extractArchive extracts a tar.gz archive to the specified directory
func extractArchive(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		path := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", path, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to copy file content: %w", err)
			}
			outFile.Close()

			if err := os.Chmod(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	return nil
}

// mountRepositoryToContainer mounts the extracted repository directory into the container
func (s *GitServer) mountRepositoryToContainer(ctx context.Context, sourceDir, destPath string) error {
	// Create the parent directory structure in the container
	parentDir := filepath.Dir(destPath)
	execResult, reader, err := s.container.Exec(ctx, []string{"mkdir", "-p", parentDir})
	if err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read mkdir output: %w", err)
	}

	if execResult != 0 {
		return fmt.Errorf("mkdir failed with exit code %d: %s", execResult, string(output))
	}

	// Use testcontainers' CopyToContainer method which handles large files efficiently
	return s.copyRepositoryWithTestcontainers(ctx, sourceDir, destPath)
}

// copyRepositoryWithTestcontainers uses testcontainers' built-in copy functionality
func (s *GitServer) copyRepositoryWithTestcontainers(ctx context.Context, sourceDir, destPath string) error {
	// Create tar archive file on disk (testcontainers will handle the transfer)
	tempTarFile, err := os.CreateTemp("", "repo-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temp tar file: %w", err)
	}
	defer os.Remove(tempTarFile.Name())
	defer tempTarFile.Close()

	// Create tar archive
	tarWriter := tar.NewWriter(tempTarFile)

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	tarWriter.Close()
	tempTarFile.Close()

	// Copy tar file to container and extract
	if err := s.container.CopyFileToContainer(ctx, tempTarFile.Name(), "/tmp/repo.tar", 0644); err != nil {
		return fmt.Errorf("failed to copy tar file to container: %w", err)
	}

	// Extract tar file in container to the correct destination
	execResult, reader, err := s.container.Exec(ctx, []string{
		"sh", "-c", fmt.Sprintf("mkdir -p %s && cd %s && echo 'Before extraction:' && ls -la && tar -xvf /tmp/repo.tar && echo 'After extraction:' && ls -la && rm /tmp/repo.tar", destPath, destPath),
	})
	if err != nil {
		return fmt.Errorf("failed to extract tar in container: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read tar extraction output: %w", err)
	}

	fmt.Printf("DEBUG - Tar extraction output:\n%s\n", string(output))

	if execResult != 0 {
		return fmt.Errorf("tar extraction failed with exit code %d: %s", execResult, string(output))
	}

	return nil
}

// createRepositoryRecord creates a repository record in Gitea database using unadopted repositories
func (s *GitServer) createRepositoryRecord(ctx context.Context, repoName string, user *User) error {
	// Path should match what was used in mountRepositoryToContainer
	gitDataPath := fmt.Sprintf("/data/git/repositories/%s/%s.git", user.Username, repoName)

	// Verify the repository was mounted successfully
	execResult, reader, err := s.container.Exec(ctx, []string{
		"test", "-d", gitDataPath,
	})
	if err != nil {
		return fmt.Errorf("failed to check if repository path exists: %w", err)
	}

	if execResult != 0 {
		return fmt.Errorf("repository path does not exist: %s", gitDataPath)
	}

	// Change ownership to git user (critical for Gitea to recognize the repository)
	execResult, reader, err = s.container.Exec(ctx, []string{
		"chown", "-R", "git:git", gitDataPath,
	})
	if err != nil {
		return fmt.Errorf("failed to change ownership: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read chown output: %w", err)
	}

	if execResult != 0 {
		return fmt.Errorf("chown failed with exit code %d: %s", execResult, string(output))
	}

	// Debug: Check repository content before adoption
	execResult, reader, err = s.container.Exec(ctx, []string{
		"sh", "-c", fmt.Sprintf("echo 'Repository content check:'; ls -la %s/; echo 'Objects check:'; ls -la %s/objects/; echo 'Pack files:'; ls -la %s/objects/pack/ 2>/dev/null || echo 'No pack directory'", gitDataPath, gitDataPath, gitDataPath),
	})
	if err == nil {
		debugOutput, _ := io.ReadAll(reader)
		fmt.Printf("DEBUG - Repository content before adoption:\n%s\n", string(debugOutput))
	}

	// Check if repository has commits (use proper bare repo syntax)
	execResult, reader, err = s.container.Exec(ctx, []string{
		"sh", "-c", fmt.Sprintf("cd %s && git --git-dir=. log --oneline 2>/dev/null | wc -l || echo '0'", gitDataPath),
	})
	if err == nil {
		commitCountOutput, _ := io.ReadAll(reader)
		fmt.Printf("DEBUG - Commit count: %s\n", strings.TrimSpace(string(commitCountOutput)))
	}

	// Check if repository already exists in Gitea
	httpClient := &http.Client{}
	checkRepoURL := fmt.Sprintf("http://%s:%s/api/v1/repos/%s/%s", s.Host, s.Port, user.Username, repoName)

	checkReq, err := http.NewRequestWithContext(ctx, "GET", checkRepoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create check request: %w", err)
	}
	checkReq.SetBasicAuth(user.Username, user.Password)

	checkResp, err := httpClient.Do(checkReq)
	if err != nil {
		return fmt.Errorf("failed to check repository existence: %w", err)
	}
	defer checkResp.Body.Close()

	if checkResp.StatusCode == http.StatusOK {
		// Repository already exists in Gitea, we're good
		fmt.Printf("DEBUG - Repository already exists in Gitea\n")
		return nil
	}

	// Repository doesn't exist in Gitea's database, but files exist on filesystem
	// This is the "unadopted repository" scenario
	// We need to create the repository record but Gitea will refuse due to existing files

	// Option 1: Try to create empty repo first, then replace with our content
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", s.Host, s.Port)

	// Ensure parent directory has proper permissions for git user
	parentDir := filepath.Dir(gitDataPath)
	execResult, reader, err = s.container.Exec(ctx, []string{
		"chown", "-R", "git:git", parentDir,
	})
	if err != nil {
		return fmt.Errorf("failed to set parent directory ownership: %w", err)
	}
	if execResult != 0 {
		output, _ := io.ReadAll(reader)
		return fmt.Errorf("failed to set parent directory ownership, exit code %d: %s", execResult, string(output))
	}

	// Set proper permissions on parent directory
	execResult, reader, err = s.container.Exec(ctx, []string{
		"chmod", "755", parentDir,
	})
	if err != nil {
		return fmt.Errorf("failed to set parent directory permissions: %w", err)
	}
	if execResult != 0 {
		output, _ := io.ReadAll(reader)
		return fmt.Errorf("failed to set parent directory permissions, exit code %d: %s", execResult, string(output))
	}

	// First, temporarily rename our repository to avoid conflict
	tempPath := gitDataPath + ".temp"
	execResult, reader, err = s.container.Exec(ctx, []string{
		"mv", gitDataPath, tempPath,
	})
	if err != nil {
		return fmt.Errorf("failed to temporarily move repository: %w", err)
	}
	if execResult != 0 {
		output, _ := io.ReadAll(reader)
		return fmt.Errorf("failed to move repository, exit code %d: %s", execResult, string(output))
	}

	// Create repository via API (this will create empty repo)
	jsonData := []byte(fmt.Sprintf(`{
		"name": "%s",
		"auto_init": false,
		"default_branch": "main"
	}`, repoName))

	req, err := http.NewRequestWithContext(ctx, "POST", createRepoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create API request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user.Username, user.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create repository via API: %w", err)
	}
	defer resp.Body.Close()

	apiOutput, _ := io.ReadAll(resp.Body)
	fmt.Printf("DEBUG - API create response (status %d): %s\n", resp.StatusCode, string(apiOutput))

	if resp.StatusCode != http.StatusCreated {
		// Restore our repository
		s.container.Exec(ctx, []string{"mv", tempPath, gitDataPath})
		return fmt.Errorf("failed to create repository via API, status: %d, body: %s", resp.StatusCode, string(apiOutput))
	}

	// Remove the empty repository that was created and restore our content
	execResult, reader, err = s.container.Exec(ctx, []string{
		"rm", "-rf", gitDataPath,
	})
	if err != nil {
		return fmt.Errorf("failed to remove empty repository: %w", err)
	}
	if execResult != 0 {
		output, _ := io.ReadAll(reader)
		return fmt.Errorf("failed to remove empty repository, exit code %d: %s", execResult, string(output))
	}

	// Restore our repository content
	execResult, reader, err = s.container.Exec(ctx, []string{
		"mv", tempPath, gitDataPath,
	})
	if err != nil {
		return fmt.Errorf("failed to restore repository content: %w", err)
	}
	if execResult != 0 {
		output, _ := io.ReadAll(reader)
		return fmt.Errorf("failed to restore repository content, exit code %d: %s", execResult, string(output))
	}

	// Ensure proper ownership after restoration
	execResult, reader, err = s.container.Exec(ctx, []string{
		"chown", "-R", "git:git", gitDataPath,
	})
	if err != nil {
		return fmt.Errorf("failed to restore ownership: %w", err)
	}
	if execResult != 0 {
		output, _ := io.ReadAll(reader)
		return fmt.Errorf("failed to restore ownership, exit code %d: %s", execResult, string(output))
	}

	// Verify our repository content is still there after API creation
	execResult, reader, err = s.container.Exec(ctx, []string{
		"sh", "-c", fmt.Sprintf("echo 'Post-API Repository check:'; ls -la %s/; echo 'Post-API Commit count:'; cd %s && git log --oneline 2>/dev/null | wc -l || echo 'No commits'", gitDataPath, gitDataPath),
	})
	if err == nil {
		postAPIOutput, _ := io.ReadAll(reader)
		fmt.Printf("DEBUG - Post-API repository state:\n%s\n", string(postAPIOutput))
	}

	return nil
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

// CleanupRepository cleans up temporary directories for a repository
func (r *Repository) Cleanup() error {
	if r.tempDir != "" {
		return os.RemoveAll(r.tempDir)
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
	Name    string
	Owner   string
	Host    string
	Port    string
	User    *User
	tempDir string // For cleanup
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
