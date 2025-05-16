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

	// commit something
	local.CreateFile(t, "test.txt", "test content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")
	hash := local.Git(t, "rev-parse", "HEAD")

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

	wantRefs := []client.Ref{
		{Name: "HEAD", Hash: hash},
		{Name: "refs/heads/main", Hash: hash},
		{Name: "refs/heads/test-branch", Hash: hash},
		{Name: "refs/tags/v1.0.0", Hash: hash},
	}

	assert.ElementsMatch(t, wantRefs, refs)
}
