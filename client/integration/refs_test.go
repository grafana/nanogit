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
	defer gitServer.Cleanup(t)
	localRepo := helpers.NewLocalGitRepo(t)
	defer localRepo.Cleanup(t)

	gitServer.CreateUser(t, "testuser", "test@example.com", "testpass123")
	repoURL, authRepoURL := gitServer.CreateRepo(t, "testrepo", "testuser", "testpass123")
	// TODO: Simplify this
	localRepo.Run(t, "init")
	localRepo.Run(t, "config", "user.name", "testuser")
	localRepo.Run(t, "config", "user.email", "test@example.com")
	localRepo.CreateFile(t, "test.txt", "test content")
	localRepo.Run(t, "add", "test.txt")
	localRepo.Run(t, "commit", "-m", "Initial commit")
	localRepo.Run(t, "branch", "-M", "main")
	localRepo.Run(t, "branch", "test-branch")
	localRepo.Run(t, "tag", "v1.0.0")

	// Push the repository to Gitea
	t.Log("Pushing to Gitea...")
	localRepo.Run(t, "remote", "add", "origin", authRepoURL)
	localRepo.Run(t, "push", "-u", "origin", "main", "--force")
	localRepo.Run(t, "push", "origin", "test-branch", "--force")
	localRepo.Run(t, "push", "origin", "v1.0.0", "--force")

	gitClient, err := client.New(repoURL, client.WithBasicAuth("testuser", "testpass123"))
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
