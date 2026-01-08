package auth

import (
	"os"

	"github.com/grafana/nanogit/options"
)

// Config holds authentication configuration
type Config struct {
	Token    string
	Username string
	Password string
}

// FromEnvironment reads authentication from environment variables.
// Priority: NANOGIT_TOKEN > GITHUB_TOKEN > GITLAB_TOKEN
func FromEnvironment() *Config {
	// Try nanogit-specific token first
	token := os.Getenv("NANOGIT_TOKEN")
	if token == "" {
		// Fallback to GitHub token
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		// Fallback to GitLab token
		token = os.Getenv("GITLAB_TOKEN")
	}

	return &Config{
		Token:    token,
		Username: os.Getenv("NANOGIT_USERNAME"),
		Password: os.Getenv("NANOGIT_PASSWORD"),
	}
}

// Merge combines environment auth with command-line flags.
// Command-line flags take precedence over environment variables.
func (c *Config) Merge(flagToken, flagUsername, flagPassword string) {
	if flagToken != "" {
		c.Token = flagToken
	}
	if flagUsername != "" {
		c.Username = flagUsername
	}
	if flagPassword != "" {
		c.Password = flagPassword
	}
}

// ToOptions converts authentication config to nanogit options.
func (c *Config) ToOptions() []options.Option {
	var opts []options.Option

	if c.Token != "" {
		opts = append(opts, options.WithTokenAuth(c.Token))
	} else if c.Username != "" && c.Password != "" {
		opts = append(opts, options.WithBasicAuth(c.Username, c.Password))
	}

	return opts
}

// HasAuth returns true if any authentication is configured
func (c *Config) HasAuth() bool {
	return c.Token != "" || (c.Username != "" && c.Password != "")
}
