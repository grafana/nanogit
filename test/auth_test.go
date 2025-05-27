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

	logger.Info("Creating and committing test file")
	local.CreateFile(t, "test.txt", "test content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("successful authorization", func(t *testing.T) {
		logger.ForSubtest(t)

		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
		require.NoError(t, err)
		auth, err := client.IsAuthorized(ctx)
		require.NoError(t, err)
		assert.True(t, auth)
	})

	t.Run("unauthorized access with wrong credentials", func(t *testing.T) {
		logger.ForSubtest(t)

		unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
		require.NoError(t, err)
		auth, err := unauthorizedClient.IsAuthorized(ctx)
		require.NoError(t, err)
		require.False(t, auth)
	})

	t.Run("successful authorization with access token", func(t *testing.T) {
		logger.ForSubtest(t)

		token := gitServer.GenerateUserToken(t, user.Username, user.Password)
		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithTokenAuth(token), nanogit.WithLogger(logger))
		require.NoError(t, err)
		auth, err := client.IsAuthorized(ctx)
		require.NoError(t, err)
		require.True(t, auth)
	})

	t.Run("unauthorized access with invalid token", func(t *testing.T) {
		logger.ForSubtest(t)

		invalidToken := "token invalid-token"
		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithTokenAuth(invalidToken), nanogit.WithLogger(logger))
		require.NoError(t, err)
		auth, err := client.IsAuthorized(ctx)
		require.NoError(t, err)
		require.False(t, auth)
	})
}
