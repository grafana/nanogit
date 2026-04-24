package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputBlobJSON(t *testing.T) {
	blob := &nanogit.Blob{
		Hash:    hash.MustFromHex("1234567890123456789012345678901234567890"),
		Content: []byte("Hello, World!\n"),
	}
	path := "README.md"

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputBlobJSON(blob, path)
	require.NoError(t, err)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Verify JSON is valid
	var output blobJSON
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Equal(t, path, output.Path)
	assert.Equal(t, "1234567890123456789012345678901234567890", output.Hash)
	assert.Equal(t, 14, output.Size)
	assert.Equal(t, "Hello, World!\n", output.Content)
}

func TestOutputBlobRaw(t *testing.T) {
	content := []byte("This is test content\nWith multiple lines\n")
	blob := &nanogit.Blob{
		Hash:    hash.MustFromHex("1234567890123456789012345678901234567890"),
		Content: content,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputBlobRaw(blob)
	require.NoError(t, err)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Verify output matches original content
	assert.Equal(t, content, buf.Bytes())
}

func TestCatFileCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		envRepo     string
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
			name:        "two arguments returns error without env",
			args:        []string{"url", "ref"},
			expectError: true,
		},
		{
			name:        "too many arguments returns error",
			args:        []string{"url", "ref", "path", "extra"},
			expectError: true,
		},
		{
			name:        "valid arguments accepted",
			args:        []string{"https://github.com/grafana/nanogit.git", "main", "README.md"},
			expectError: false,
		},
		{
			name:        "two arguments accepted when NANOGIT_REPO is set",
			args:        []string{"main", "README.md"},
			envRepo:     "https://github.com/grafana/nanogit.git",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			globalJSON = false
			globalUsername = ""
			globalToken = ""

			t.Setenv("NANOGIT_REPO", tt.envRepo)

			// Test argument validation
			catFileCmd.SetArgs(tt.args)
			err := catFileCmd.Args(catFileCmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
