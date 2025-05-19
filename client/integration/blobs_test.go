//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/nanogit/client"
	"github.com/grafana/nanogit/client/integration/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Blobs(t *testing.T) {
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
	local.Git(t, "push", "origin", "main", "--force")

	// Get the blob hash
	blobHash := local.Git(t, "rev-parse", "HEAD:test.txt")

	gitClient, err := client.New(remote.URL(), client.WithBasicAuth(user.Username, user.Password))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GetBlob
	content, err := gitClient.GetBlob(ctx, blobHash)
	require.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Test GetBlob with non-existent hash
	_, err = gitClient.GetBlob(ctx, "b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
}
