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
	client, remote, _ := gitServer.TestRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("existing repository", func(t *testing.T) {
		logger.ForSubtest(t)

		exists, err := client.RepoExists(ctx)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent repository", func(t *testing.T) {
		logger.ForSubtest(t)

		nonExistentClient, err := nanogit.NewHTTPClient(remote.URL()+"/nonexistent", nanogit.WithBasicAuth(remote.User.Username, remote.User.Password))
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
