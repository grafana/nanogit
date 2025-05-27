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
	logger := helpers.NewTestLogger(t)
	logger.Info("Setting up remote repository")
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	logger.Info("Setting up local repository")
	local := helpers.NewLocalGitRepo(t, logger)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	logger.Info("Creating and committing a test file")
	local.CreateFile(t, "test.txt", "test content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")

	logger.Info("Creating and switching to main branch")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("existing repository", func(t *testing.T) {
		logger.ForSubtest(t)

		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
		require.NoError(t, err)

		exists, err := client.RepoExists(ctx)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent repository", func(t *testing.T) {
		logger.ForSubtest(t)

		nonExistentClient, err := nanogit.NewHTTPClient(remote.URL()+"/nonexistent", nanogit.WithBasicAuth(user.Username, user.Password))
		require.NoError(t, err)

		exists, err := nonExistentClient.RepoExists(ctx)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		logger.ForSubtest(t)

		unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
		require.NoError(t, err)

		exists, err := unauthorizedClient.RepoExists(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "401 Unauthorized")
		assert.False(t, exists)
	})
}
