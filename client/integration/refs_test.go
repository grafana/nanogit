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

func TestClient_ListRefs(t *testing.T) {

	// set up remote repo
	gitServer := helpers.NewGitServer(t)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	// set up local repo
	local := helpers.NewLocalGitRepo(t)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	// Easy way to add remote with username and password without modifying the host configuration
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	// test commit
	local.CreateFile(t, "test.txt", "test content")
	local.Git(t, "add", "test.txt")
	// TODO: Get the commit hash and use it in the message
	local.Git(t, "commit", "-m", "Initial commit")

	// main branch
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "-u", "origin", "main", "--force")

	// test-branch
	local.Git(t, "branch", "test-branch")
	local.Git(t, "push", "origin", "test-branch", "--force")

	// v1.0.0 tag
	local.Git(t, "tag", "v1.0.0")
	local.Git(t, "push", "origin", "v1.0.0", "--force")

	gitClient, err := client.New(remote.URL(), client.WithBasicAuth(user.Username, user.Password))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refs, err := gitClient.ListRefs(ctx)
	require.NoError(t, err, "ListRefs failed: %v", err)
	require.Len(t, refs, 4, "should have 4 references")

	onlyRefNames := make([]string, 0, len(refs))
	for _, ref := range refs {
		onlyRefNames = append(onlyRefNames, ref.Name)
		require.Len(t, ref.Hash, 40, "hash should be 40 characters")
	}

	wantRefs := []string{
		"HEAD",
		"refs/heads/main",
		"refs/heads/test-branch",
		"refs/tags/v1.0.0",
	}

	assert.ElementsMatch(t, wantRefs, onlyRefNames)
}
