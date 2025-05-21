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

func TestClient_IsAuthorized(t *testing.T) {
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

	// Test successful authorization
	auth, err := client.IsAuthorized(ctx)
	require.NoError(t, err)
	assert.True(t, auth)

	// Test unauthorized access with wrong credentials
	unauthorizedClient, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
	require.NoError(t, err)

	auth, err = unauthorizedClient.IsAuthorized(ctx)
	require.NoError(t, err)
	assert.False(t, auth)
}
