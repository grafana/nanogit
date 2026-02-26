package gittest

import (
	"fmt"
	"net/url"
	"time"
)

// User represents a test user account in the Git server.
//
// All fields are automatically generated when created via Server.CreateUser():
//   - Username: Unique identifier with timestamp suffix (e.g., "user-1234567890ab")
//   - Email: Auto-generated email address
//   - Password: Auto-generated password for HTTPS authentication
//   - Token: Pre-generated access token for API operations
//
// Example:
//
//	user, err := server.CreateUser(ctx)
//	// user.Username, user.Password, user.Token are ready to use
type User struct {
	Username string // Unique username
	Email    string // Email address
	Password string // Password for HTTPS authentication
	Token    string // Access token for API operations
}

// Repo represents a remote Git repository for testing.
//
// Created via Server.CreateRepo(), this type provides access URLs:
//   - URL: Public HTTPS URL without credentials
//   - AuthURL: HTTPS URL with embedded username:password for easy cloning
//   - Name: Repository name
//   - Owner: Username of the repository owner
//   - User: Reference to the User who owns this repository
//
// The AuthURL is particularly useful for git operations:
//
//	repo, err := server.CreateRepo(ctx, "myrepo", user)
//	// Use repo.AuthURL for git clone, fetch, push, etc.
type Repo struct {
	Name    string // Repository name
	Owner   string // Owner username
	URL     string // Public URL (requires authentication)
	AuthURL string // URL with embedded credentials
	User    *User  // Repository owner

	host string
	port string
}

// CloneURL returns the authenticated clone URL for the repository.
func (r *Repo) CloneURL() string {
	return r.AuthURL
}

// PublicURL returns the public URL for the repository (without auth).
func (r *Repo) PublicURL() string {
	return r.URL
}

// newRepo creates a new Repo instance with the specified configuration.
func newRepo(repoName string, user *User, host, port string) *Repo {
	// Build AuthURL with properly URL-encoded credentials
	authURL := &url.URL{
		Scheme: "http",
		User:   url.UserPassword(user.Username, user.Password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   fmt.Sprintf("/%s/%s.git", user.Username, repoName),
	}

	return &Repo{
		Name:    repoName,
		Owner:   user.Username,
		URL:     fmt.Sprintf("http://%s:%s/%s/%s.git", host, port, user.Username, repoName),
		AuthURL: authURL.String(),
		User:    user,
		host:    host,
		port:    port,
	}
}

// Config holds configuration for test utilities.
type Config struct {
	Logger        Logger
	StartTimeout  time.Duration
	GiteaImage    string
	GiteaVersion  string
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		Logger:       NoopLogger(),
		StartTimeout: 30 * time.Second,
		GiteaImage:   "gitea/gitea",
		GiteaVersion: "latest",
	}
}
