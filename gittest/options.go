package gittest

import "time"

// ServerOption configures a Server instance.
type ServerOption func(*Config)

// WithLogger sets the logger for server operations.
func WithLogger(logger Logger) ServerOption {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithTimeout sets the startup timeout for the server container.
func WithTimeout(duration time.Duration) ServerOption {
	return func(c *Config) {
		c.StartTimeout = duration
	}
}

// WithGiteaImage sets the Docker image for the Gitea server.
func WithGiteaImage(image string) ServerOption {
	return func(c *Config) {
		c.GiteaImage = image
	}
}

// WithGiteaVersion sets the version tag for the Gitea Docker image.
func WithGiteaVersion(version string) ServerOption {
	return func(c *Config) {
		c.GiteaVersion = version
	}
}

// WithTrustedSSHKeys configures SSH public keys whose signatures Gitea trusts
// for commit verification, regardless of the committer.
func WithTrustedSSHKeys(keys ...string) ServerOption {
	return func(c *Config) {
		c.TrustedSSHKeys = keys
	}
}

// userConfig holds configuration for creating a test user.
type userConfig struct {
	username *string
}

// UserOption configures a test user created by Server.CreateUser.
type UserOption func(*userConfig)

// WithUsername sets the exact username for a test user. This is useful for
// tests that need stable repository URLs or server-side ownership paths.
//
// When this option is not provided, CreateUser generates a unique username.
func WithUsername(username string) UserOption {
	return func(c *userConfig) {
		c.username = &username
	}
}

// repoConfig holds configuration for LocalRepo.
type repoConfig struct {
	logger   Logger
	tempDir  string
	gitTrace bool
}

// RepoOption configures a LocalRepo instance.
type RepoOption func(*repoConfig)

// WithRepoLogger sets the logger for repository operations.
func WithRepoLogger(logger Logger) RepoOption {
	return func(c *repoConfig) {
		c.logger = logger
	}
}

// WithTempDir sets the parent directory for temporary repository creation.
// If not specified, os.MkdirTemp will use the system default.
func WithTempDir(dir string) RepoOption {
	return func(c *repoConfig) {
		c.tempDir = dir
	}
}

// WithGitTrace enables Git protocol tracing (GIT_TRACE_PACKET=1).
// This is useful for debugging Git protocol issues but should be used with caution
// as it generates verbose output and may expose sensitive information in logs.
func WithGitTrace() RepoOption {
	return func(c *repoConfig) {
		c.gitTrace = true
	}
}
