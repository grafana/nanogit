package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no arguments returns error",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "too many arguments returns error",
			args:        []string{"url1", "url2"},
			expectError: true,
		},
		{
			name:        "valid arguments accepted",
			args:        []string{"https://github.com/grafana/nanogit.git"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			globalJSON = false
			globalUsername = ""
			globalToken = ""

			// Test argument validation
			checkCmd.SetArgs(tt.args)
			err := checkCmd.Args(checkCmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOutputCheckJSON(t *testing.T) {
	tests := []struct {
		name       string
		repoURL    string
		compatible bool
	}{
		{
			name:       "compatible server",
			repoURL:    "https://github.com/grafana/nanogit.git",
			compatible: true,
		},
		{
			name:       "incompatible server",
			repoURL:    "https://dev.azure.com/org/project/_git/repo",
			compatible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputCheckJSON(tt.repoURL, tt.compatible)
			require.NoError(t, err)

			// Restore stdout
			require.NoError(t, w.Close())
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			require.NoError(t, err)

			// Verify JSON is valid
			var output checkResultJSON
			err = json.Unmarshal(buf.Bytes(), &output)
			require.NoError(t, err)

			assert.Equal(t, tt.repoURL, output.Repository)
			assert.Equal(t, tt.compatible, output.Compatible)

			if tt.compatible {
				assert.Equal(t, "v2", output.Protocol)
				assert.Contains(t, output.Message, "compatible")
			} else {
				assert.Equal(t, "v1", output.Protocol)
				assert.Contains(t, output.Message, "v1")
			}
		})
	}
}

func TestOutputCheckHuman(t *testing.T) {
	tests := []struct {
		name       string
		repoURL    string
		compatible bool
		contains   []string
	}{
		{
			name:       "compatible output",
			repoURL:    "https://github.com/grafana/nanogit.git",
			compatible: true,
			contains:   []string{"✅", "Compatible", "protocol v2", "nanogit ls-remote"},
		},
		{
			name:       "incompatible output",
			repoURL:    "https://dev.azure.com/org/project/_git/repo",
			compatible: false,
			contains:   []string{"❌", "Not Compatible", "protocol v1", "Azure DevOps", "git CLI"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputCheckHuman(tt.repoURL, tt.compatible)
			require.NoError(t, err)

			// Restore stdout
			require.NoError(t, w.Close())
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			require.NoError(t, err)

			output := buf.String()

			// Verify expected content
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}

			assert.Contains(t, output, tt.repoURL)
		})
	}
}
