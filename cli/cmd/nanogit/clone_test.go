package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "This is a commit message",
			expected: "This is a commit message",
		},
		{
			name:     "multi-line message",
			input:    "First line\nSecond line\nThird line",
			expected: "First line",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only newline",
			input:    "\n",
			expected: "",
		},
		{
			name:     "message with trailing newline",
			input:    "Commit message\n",
			expected: "Commit message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstLine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCloneCommand(t *testing.T) {
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
			name:        "one argument returns error",
			args:        []string{"url"},
			expectError: true,
		},
		{
			name:        "two arguments returns error",
			args:        []string{"url", "ref"},
			expectError: true,
		},
		{
			name:        "too many arguments returns error",
			args:        []string{"url", "ref", "dest", "extra"},
			expectError: true,
		},
		{
			name:        "valid arguments accepted",
			args:        []string{"https://github.com/grafana/nanogit.git", "main", "./my-repo"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			cloneInclude = nil
			cloneExclude = nil
			cloneUsername = ""
			cloneToken = ""

			// Test argument validation
			cloneCmd.SetArgs(tt.args)
			err := cloneCmd.Args(cloneCmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
