//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Refs(t *testing.T) {
	logger := helpers.NewTestLogger(t)
	logger.Info("Setting up remote repository")
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user)
	logger.Info("Setting up local repository")
	local := remote.Local(t)

	logger.Info("Committing something")
	local.CreateFile(t, "test.txt", "test content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")
	firstCommit, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "-u", "origin", "main", "--force")

	logger.Info("Creating test branch")
	local.Git(t, "branch", "test-branch")
	local.Git(t, "push", "origin", "test-branch", "--force")

	logger.Info("Creating v1.0.0 tag")
	local.Git(t, "tag", "v1.0.0")
	local.Git(t, "push", "origin", "v1.0.0", "--force")

	gitClient := remote.Client(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refs, err := gitClient.ListRefs(ctx)
	require.NoError(t, err, "ListRefs failed: %v", err)
	require.Len(t, refs, 4, "should have 4 references")

	wantRefs := []nanogit.Ref{
		{Name: "HEAD", Hash: firstCommit},
		{Name: "refs/heads/main", Hash: firstCommit},
		{Name: "refs/heads/test-branch", Hash: firstCommit},
		{Name: "refs/tags/v1.0.0", Hash: firstCommit},
	}

	assert.ElementsMatch(t, wantRefs, refs)

	logger.Info("Getting refs one by one")
	for _, ref := range wantRefs {
		ref, err := gitClient.GetRef(ctx, ref.Name)
		require.NoError(t, err, "GetRef failed: %v", err)
		assert.Equal(t, ref.Name, ref.Name)
		assert.Equal(t, firstCommit, ref.Hash)
	}

	logger.Info("Getting ref with non-existent ref")
	_, err = gitClient.GetRef(ctx, "refs/heads/non-existent")
	var notFoundErr *nanogit.RefNotFoundError
	require.True(t, errors.As(err, &notFoundErr))
	require.Equal(t, "refs/heads/non-existent", notFoundErr.RefName)

	logger.Info("Creating ref with new-branch")
	err = gitClient.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/new-branch", Hash: firstCommit})
	require.NoError(t, err)

	logger.Info("Getting ref with new-branch")
	ref, err := gitClient.GetRef(ctx, "refs/heads/new-branch")
	require.NoError(t, err)
	assert.Equal(t, firstCommit, ref.Hash)

	logger.Info("Creating a new commit")
	local.Git(t, "commit", "--allow-empty", "-m", "new commit")
	newHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Updating ref to point to new commit")
	err = gitClient.UpdateRef(ctx, nanogit.Ref{Name: "refs/heads/new-branch", Hash: newHash})
	require.NoError(t, err)

	logger.Info("Getting ref and verifying it points to new commit")
	ref, err = gitClient.GetRef(ctx, "refs/heads/new-branch")
	require.NoError(t, err)
	assert.Equal(t, newHash, ref.Hash)

	logger.Info("Deleting ref with new-branch")
	err = gitClient.DeleteRef(ctx, "refs/heads/new-branch")
	require.NoError(t, err)

	logger.Info("Getting ref with new-branch should fail")
	_, err = gitClient.GetRef(ctx, "refs/heads/new-branch")
	var notFoundErr2 *nanogit.RefNotFoundError
	require.True(t, errors.As(err, &notFoundErr2))
	require.Equal(t, "refs/heads/new-branch", notFoundErr2.RefName)

	logger.Info("Creating tag with v2.0.0")
	err = gitClient.CreateRef(ctx, nanogit.Ref{Name: "refs/tags/v2.0.0", Hash: firstCommit})
	require.NoError(t, err)

	logger.Info("Getting ref with new tag")
	ref, err = gitClient.GetRef(ctx, "refs/tags/v2.0.0")
	require.NoError(t, err)
	assert.Equal(t, firstCommit, ref.Hash)

	logger.Info("Deleting tag with v2.0.0")
	err = gitClient.DeleteRef(ctx, "refs/tags/v2.0.0")
	require.NoError(t, err)

	logger.Info("Getting ref with new tag should fail")
	_, err = gitClient.GetRef(ctx, "refs/tags/v2.0.0")
	var notFoundErr3 *nanogit.RefNotFoundError
	require.True(t, errors.As(err, &notFoundErr3))
	require.Equal(t, "refs/tags/v2.0.0", notFoundErr3.RefName)
}
