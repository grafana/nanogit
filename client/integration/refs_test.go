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
	gitServer := helpers.NewGitServer(t)
	localRepo := helpers.NewLocalGitRepo(t)

	// TODO: Simplify This
	gitServer.CreateUser(t, "testuser", "test@example.com", "testpass123")
	remoteRepo := gitServer.CreateRepo(t, "testrepo", "testuser", "testpass123")

	// set up local repo
	localRepo.Git(t, "config", "user.name", "testuser")
	localRepo.Git(t, "config", "user.email", "test@example.com")
	// FIXME: Gitea doesn't seem to work if the remote URL doesn't contain username and password like this
	localRepo.Git(t, "remote", "add", "origin", remoteRepo.AuthURL())

	// test commit
	localRepo.CreateFile(t, "test.txt", "test content")
	localRepo.Git(t, "add", "test.txt")
	localRepo.Git(t, "commit", "-m", "Initial commit")

	// main branch
	localRepo.Git(t, "branch", "-M", "main")
	localRepo.Git(t, "push", "-u", "origin", "main", "--force")

	// test-branch
	localRepo.Git(t, "branch", "test-branch")
	localRepo.Git(t, "push", "origin", "test-branch", "--force")

	// v1.0.0 tag
	localRepo.Git(t, "tag", "v1.0.0")
	localRepo.Git(t, "push", "origin", "v1.0.0", "--force")

	gitClient, err := client.New(remoteRepo.URL(), client.WithBasicAuth("testuser", "testpass123"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refs, err := gitClient.ListRefs(ctx)
	require.NoError(t, err, "ListRefs failed: %v", err)

	assert.NotEmpty(t, refs, "should have at least one reference")

	var (
		masterRef     *client.Ref
		testBranchRef *client.Ref
		tagRef        *client.Ref
	)

	for _, ref := range refs {
		switch ref.Name {
		case "refs/heads/main":
			masterRef = &ref
		case "refs/heads/test-branch":
			testBranchRef = &ref
		case "refs/tags/v1.0.0":
			tagRef = &ref
		}
	}

	require.NotNil(t, masterRef, "should have main branch")
	require.NotNil(t, testBranchRef, "should have test-branch")
	require.NotNil(t, tagRef, "should have v1.0.0 tag")

	for _, ref := range []*client.Ref{masterRef, testBranchRef, tagRef} {
		require.Len(t, ref.Hash, 40, "hash should be 40 characters")
	}
}
