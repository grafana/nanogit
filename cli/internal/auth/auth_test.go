package auth

import (
	"os"
	"testing"

	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name: "NANOGIT_TOKEN has highest priority",
			envVars: map[string]string{
				"NANOGIT_TOKEN": "nanogit-token",
				"GITHUB_TOKEN":  "github-token",
				"GITLAB_TOKEN":  "gitlab-token",
			},
			expected: &Config{
				Token: "nanogit-token",
			},
		},
		{
			name: "GITHUB_TOKEN is second priority",
			envVars: map[string]string{
				"GITHUB_TOKEN": "github-token",
				"GITLAB_TOKEN": "gitlab-token",
			},
			expected: &Config{
				Token: "github-token",
			},
		},
		{
			name: "GITLAB_TOKEN is third priority",
			envVars: map[string]string{
				"GITLAB_TOKEN": "gitlab-token",
			},
			expected: &Config{
				Token: "gitlab-token",
			},
		},
		{
			name: "basic auth from environment",
			envVars: map[string]string{
				"NANOGIT_USERNAME": "user",
				"NANOGIT_PASSWORD": "pass",
			},
			expected: &Config{
				Username: "user",
				Password: "pass",
			},
		},
		{
			name:    "empty config when no env vars",
			envVars: map[string]string{},
			expected: &Config{
				Token:    "",
				Username: "",
				Password: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars
			os.Unsetenv("NANOGIT_TOKEN")
			os.Unsetenv("GITHUB_TOKEN")
			os.Unsetenv("GITLAB_TOKEN")
			os.Unsetenv("NANOGIT_USERNAME")
			os.Unsetenv("NANOGIT_PASSWORD")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			config := FromEnvironment()
			assert.Equal(t, tt.expected.Token, config.Token)
			assert.Equal(t, tt.expected.Username, config.Username)
			assert.Equal(t, tt.expected.Password, config.Password)
		})
	}
}

func TestConfigMerge(t *testing.T) {
	tests := []struct {
		name             string
		initialConfig    *Config
		token            string
		username         string
		password         string
		expectedToken    string
		expectedUsername string
		expectedPassword string
	}{
		{
			name: "flags override environment token",
			initialConfig: &Config{
				Token: "env-token",
			},
			token:         "flag-token",
			expectedToken: "flag-token",
		},
		{
			name: "flags override environment basic auth",
			initialConfig: &Config{
				Username: "env-user",
				Password: "env-pass",
			},
			username:         "flag-user",
			password:         "flag-pass",
			expectedUsername: "flag-user",
			expectedPassword: "flag-pass",
		},
		{
			name: "empty flags don't override environment",
			initialConfig: &Config{
				Token:    "env-token",
				Username: "env-user",
				Password: "env-pass",
			},
			token:            "",
			username:         "",
			password:         "",
			expectedToken:    "env-token",
			expectedUsername: "env-user",
			expectedPassword: "env-pass",
		},
		{
			name: "partial flag override",
			initialConfig: &Config{
				Token:    "env-token",
				Username: "env-user",
				Password: "env-pass",
			},
			token:            "flag-token",
			expectedToken:    "flag-token",
			expectedUsername: "env-user",
			expectedPassword: "env-pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.initialConfig
			config.Merge(tt.token, tt.username, tt.password)

			assert.Equal(t, tt.expectedToken, config.Token)
			assert.Equal(t, tt.expectedUsername, config.Username)
			assert.Equal(t, tt.expectedPassword, config.Password)
		})
	}
}

func TestConfigToOptions(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectToken bool
		expectBasic bool
	}{
		{
			name: "token auth",
			config: &Config{
				Token: "test-token",
			},
			expectToken: true,
			expectBasic: false,
		},
		{
			name: "basic auth",
			config: &Config{
				Username: "user",
				Password: "pass",
			},
			expectToken: false,
			expectBasic: true,
		},
		{
			name: "token takes precedence over basic auth",
			config: &Config{
				Token:    "test-token",
				Username: "user",
				Password: "pass",
			},
			expectToken: true,
			expectBasic: false,
		},
		{
			name:        "no auth returns empty options",
			config:      &Config{},
			expectToken: false,
			expectBasic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.config.ToOptions()

			if tt.expectToken {
				require.NotEmpty(t, opts, "expected auth options")
				// Apply options to a test client to verify they work
				// We can't easily test the actual auth without creating a client,
				// so we just verify options were returned
			} else if tt.expectBasic {
				require.NotEmpty(t, opts, "expected auth options")
			} else {
				require.Empty(t, opts, "expected no auth options")
			}
		})
	}
}

func TestConfigToOptionsApplies(t *testing.T) {
	// Test that options can be applied (basic smoke test)
	config := &Config{
		Token: "test-token",
	}

	opts := config.ToOptions()
	require.Len(t, opts, 1, "expected one option")

	// Verify the option is an options.Option
	var _ options.Option = opts[0]
}
