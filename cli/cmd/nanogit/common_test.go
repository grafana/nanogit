package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoArgsExplicit(t *testing.T) {
	t.Setenv("NANOGIT_REPO", "")

	validator := repoArgs(2)
	require.NoError(t, validator(nil, []string{"https://example.com/repo.git", "main"}))

	// Without env, missing the repo URL is an error and hints at the env var.
	err := validator(nil, []string{"main"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), repoEnv)
}

func TestRepoArgsWithEnv(t *testing.T) {
	t.Setenv("NANOGIT_REPO", "https://example.com/repo.git")

	validator := repoArgs(2)
	require.NoError(t, validator(nil, []string{"main"}))
	require.NoError(t, validator(nil, []string{"https://example.com/other.git", "main"}))

	require.Error(t, validator(nil, []string{}))
	require.Error(t, validator(nil, []string{"a", "b", "c"}))
}

func TestResolveRepoURL(t *testing.T) {
	t.Setenv("NANOGIT_REPO", "https://env.example.com/repo.git")

	// Explicit arg wins over env.
	repoURL, rest := resolveRepoURL([]string{"https://flag.example.com/repo.git", "main"}, 2)
	assert.Equal(t, "https://flag.example.com/repo.git", repoURL)
	assert.Equal(t, []string{"main"}, rest)

	// Env fallback when arg count is shorter.
	repoURL, rest = resolveRepoURL([]string{"main"}, 2)
	assert.Equal(t, "https://env.example.com/repo.git", repoURL)
	assert.Equal(t, []string{"main"}, rest)
}

func TestLooksLikeRepoURL(t *testing.T) {
	assert.True(t, looksLikeRepoURL("https://github.com/foo/bar.git"))
	assert.True(t, looksLikeRepoURL("http://example.com/x"))
	assert.False(t, looksLikeRepoURL("./my-repo"))
	assert.False(t, looksLikeRepoURL("my-repo"))
	assert.False(t, looksLikeRepoURL("/tmp/repo"))
}
