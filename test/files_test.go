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

func TestClient_Files(t *testing.T) {
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
	testContent := []byte("test content")
	local.CreateFile(t, "test.txt", string(testContent))
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

	// Get the commit hash
	commitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	t.Run("GetFile with existing file", func(t *testing.T) {
		file, err := client.GetFile(ctx, commitHash, "test.txt")
		if err != nil {
			t.Logf("Failed to get file with hash %s and path %s: %v", commitHash, "test.txt", err)
		}
		require.NoError(t, err)
		assert.Equal(t, testContent, file.Content)
		assert.Equal(t, uint32(33188), file.Mode)

		fileHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD:test.txt"))
		require.NoError(t, err)
		assert.Equal(t, fileHash, file.Hash)
		assert.Equal(t, "test.txt", file.Path)
		assert.Equal(t, uint32(33188), file.Mode)
	})

	t.Run("GetFile with non-existent file", func(t *testing.T) {
		_, err := client.GetFile(ctx, commitHash, "nonexistent.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetFile with non-existent hash", func(t *testing.T) {
		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		require.NoError(t, err)
		_, err = client.GetFile(ctx, nonExistentHash, "test.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not our ref")
	})
}
