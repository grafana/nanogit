package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantEmail string
		wantErr   bool
	}{
		{"valid", "Jane Doe <jane@example.com>", "Jane Doe", "jane@example.com", false},
		{"extra whitespace", "  Jane Doe   <jane@example.com>  ", "Jane Doe", "jane@example.com", false},
		{"missing angle brackets", "Jane Doe jane@example.com", "", "", true},
		{"empty name", "<jane@example.com>", "", "", true},
		{"empty email", "Jane <>", "", "", true},
		{"empty string", "", "", "", true},
		{"reversed brackets", "Jane >jane@example.com<", "", "", true},
		{"trailing junk", "Jane <jane@example.com> junk", "", "", true},
		{"multiple open brackets", "Jane <x> <jane@example.com>", "", "", true},
		{"multiple close brackets", "Jane <jane@example.com>>", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, email, err := parseIdentity(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantEmail, email)
		})
	}
}

func TestResolveAuthorFromFlag(t *testing.T) {
	t.Setenv("NANOGIT_AUTHOR_NAME", "")
	t.Setenv("NANOGIT_AUTHOR_EMAIL", "")

	author, err := resolveAuthor("Jane Doe <jane@example.com>")
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", author.Name)
	assert.Equal(t, "jane@example.com", author.Email)
	assert.False(t, author.Time.IsZero())
}

func TestResolveAuthorFromEnv(t *testing.T) {
	t.Setenv("NANOGIT_AUTHOR_NAME", "Env User")
	t.Setenv("NANOGIT_AUTHOR_EMAIL", "env@example.com")

	author, err := resolveAuthor("")
	require.NoError(t, err)
	assert.Equal(t, "Env User", author.Name)
	assert.Equal(t, "env@example.com", author.Email)
}

func TestResolveAuthorMissingErrors(t *testing.T) {
	t.Setenv("NANOGIT_AUTHOR_NAME", "")
	t.Setenv("NANOGIT_AUTHOR_EMAIL", "")

	_, err := resolveAuthor("")
	assert.Error(t, err)
}

func TestResolveAuthorPartialEnvErrors(t *testing.T) {
	t.Setenv("NANOGIT_AUTHOR_NAME", "Only Name")
	t.Setenv("NANOGIT_AUTHOR_EMAIL", "")

	_, err := resolveAuthor("")
	assert.Error(t, err)
}

func TestResolveCommitterFallsBackToAuthor(t *testing.T) {
	t.Setenv("NANOGIT_COMMITTER_NAME", "")
	t.Setenv("NANOGIT_COMMITTER_EMAIL", "")

	author, err := resolveAuthor("Jane Doe <jane@example.com>")
	require.NoError(t, err)

	committer, err := resolveCommitter("", author)
	require.NoError(t, err)
	assert.Equal(t, author.Name, committer.Name)
	assert.Equal(t, author.Email, committer.Email)
}

func TestResolveCommitterFromFlagOverridesAuthor(t *testing.T) {
	t.Setenv("NANOGIT_COMMITTER_NAME", "")
	t.Setenv("NANOGIT_COMMITTER_EMAIL", "")

	author, err := resolveAuthor("Jane Doe <jane@example.com>")
	require.NoError(t, err)

	committer, err := resolveCommitter("Robot <robot@example.com>", author)
	require.NoError(t, err)
	assert.Equal(t, "Robot", committer.Name)
	assert.Equal(t, "robot@example.com", committer.Email)
}

func TestResolveCommitterFromEnv(t *testing.T) {
	t.Setenv("NANOGIT_COMMITTER_NAME", "Env Committer")
	t.Setenv("NANOGIT_COMMITTER_EMAIL", "envc@example.com")

	author, err := resolveAuthor("Jane Doe <jane@example.com>")
	require.NoError(t, err)

	committer, err := resolveCommitter("", author)
	require.NoError(t, err)
	assert.Equal(t, "Env Committer", committer.Name)
	assert.Equal(t, "envc@example.com", committer.Email)
}

func TestReadPutFileContentFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	want := []byte("hello from a file\n")
	require.NoError(t, os.WriteFile(path, want, 0o644))

	got, err := readPutFileContent(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestReadPutFileContentMissingFile(t *testing.T) {
	_, err := readPutFileContent(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Error(t, err)
}

func TestPutFileCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"no arguments", []string{}, true},
		{"two arguments", []string{"repo", "ref"}, true},
		{"three arguments", []string{"repo", "ref", "path"}, false},
		{"four arguments with stdin marker", []string{"repo", "ref", "path", "-"}, false},
		{"four arguments with invalid marker", []string{"repo", "ref", "path", "bad"}, true},
		{"five arguments", []string{"repo", "ref", "path", "-", "extra"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := putFileCmd.Args(putFileCmd, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
