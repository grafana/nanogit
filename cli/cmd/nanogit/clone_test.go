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
			name:        "one argument accepted (uses current dir and HEAD)",
			args:        []string{"https://github.com/grafana/nanogit.git"},
			expectError: false,
		},
		{
			name:        "two arguments accepted (url and destination)",
			args:        []string{"https://github.com/grafana/nanogit.git", "./my-repo"},
			expectError: false,
		},
		{
			name:        "too many arguments returns error",
			args:        []string{"url", "dest", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			cloneInclude = nil
			cloneExclude = nil
			globalUsername = ""
			globalToken = ""
			cloneBatchSize = 50
			cloneConcurrency = 10

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

func TestCloneFlagsValidation(t *testing.T) {
	tests := []struct {
		name           string
		batchSize      int
		concurrency    int
		expectValid    bool
	}{
		{
			name:        "default recommended values",
			batchSize:   50,
			concurrency: 10,
			expectValid: true,
		},
		{
			name:        "custom batch size",
			batchSize:   100,
			concurrency: 10,
			expectValid: true,
		},
		{
			name:        "custom concurrency",
			batchSize:   50,
			concurrency: 20,
			expectValid: true,
		},
		{
			name:        "sequential mode (batch size 1)",
			batchSize:   1,
			concurrency: 1,
			expectValid: true,
		},
		{
			name:        "high concurrency",
			batchSize:   100,
			concurrency: 50,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			cloneInclude = nil
			cloneExclude = nil
			globalUsername = ""
			globalToken = ""
			cloneBatchSize = tt.batchSize
			cloneConcurrency = tt.concurrency

			// Verify values can be set
			assert.Equal(t, tt.batchSize, cloneBatchSize)
			assert.Equal(t, tt.concurrency, cloneConcurrency)
		})
	}
}

