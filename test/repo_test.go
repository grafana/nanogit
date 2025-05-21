//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_RepoExists(t *testing.T) {
	// set up remote repo
	gitServer := helpers.NewGitServer(t)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	// set up local repo
	local := helpers.NewLocalGitRepo(t)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	// Create and commit a test file
	local.CreateFile(t, "test.txt", "test content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")

	// Create and switch to main branch
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger := helpers.NewTestLogger(t)
	client, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test existing repository
	exists, err := client.RepoExists(ctx)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existent repository
	nonExistentClient, err := nanogit.NewClient(remote.URL()+"/nonexistent", nanogit.WithBasicAuth(user.Username, user.Password))
	require.NoError(t, err)

	exists, err = nonExistentClient.RepoExists(ctx)
	require.NoError(t, err)
	assert.False(t, exists)

	// Test unauthorized access
	unauthorizedClient, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
	require.NoError(t, err)

	exists, err = unauthorizedClient.RepoExists(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "401 Unauthorized")
	assert.False(t, exists)
}
