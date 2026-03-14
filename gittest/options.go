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

// WithProtocolV1Only attempts to configure the Gitea server for Git protocol v1 only.
//
// IMPORTANT LIMITATION: Modern Git (2.18+) always advertises protocol v2 capabilities
// on the server side, regardless of configuration settings. This option sets
// ENABLE_AUTO_GIT_WIRE_PROTOCOL=false but cannot achieve true v1-only mode with
// modern Git versions.
//
// This option is kept for documentation purposes and for potential future use with
// custom Git builds. For testing v1 detection, use mock servers instead (see
// protocol_version_integration_test.go for examples using httptest).
func WithProtocolV1Only() ServerOption {
	return func(c *Config) {
		c.ProtocolV1Only = true
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
