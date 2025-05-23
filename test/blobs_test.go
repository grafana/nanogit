//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Blobs(t *testing.T) {
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
	testContent := []byte("test content")
	local.CreateFile(t, "test.txt", string(testContent))
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Getting blob hash", "file", "test.txt")
	blobHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD:test.txt"))
	require.NoError(t, err)

	logger.Info("Creating client")
	client, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetBlob with valid hash", func(t *testing.T) {
		logger.Info("Testing GetBlob with valid hash", "hash", blobHash)
		content, err := client.GetBlob(ctx, blobHash)
		require.NoError(t, err)
		assert.Equal(t, testContent, content)
	})

	t.Run("GetBlob with non-existent hash", func(t *testing.T) {
		logger.Info("Testing GetBlob with non-existent hash")
		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		require.NoError(t, err)
		_, err = client.GetBlob(ctx, nonExistentHash)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
	})
}
