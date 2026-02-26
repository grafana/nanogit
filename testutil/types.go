package testutil

import (
	"fmt"
	"time"
)

// User represents a Git user for testing purposes.
type User struct {
	Username string
	Email    string
	Password string
	Token    string // Generated access token (if applicable)
}

// Repo represents a remote Git repository for testing.
type Repo struct {
	Name     string
	Owner    string
	URL      string // Public URL (no auth)
	AuthURL  string // Authenticated URL (with credentials)
	User     *User
	host     string
	port     string
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
	return &Repo{
		Name:    repoName,
		Owner:   user.Username,
		URL:     fmt.Sprintf("http://%s:%s/%s/%s.git", host, port, user.Username, repoName),
		AuthURL: fmt.Sprintf("http://%s:%s@%s:%s/%s/%s.git", user.Username, user.Password, host, port, user.Username, repoName),
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
