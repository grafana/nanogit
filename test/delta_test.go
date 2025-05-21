//go:build integration

package integration_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CompareCommits(t *testing.T) {
	// set up remote repo
	gitServer := helpers.NewGitServer(t)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	// set up local repo
	local := helpers.NewLocalGitRepo(t)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	// Create initial commit with a file
	local.CreateFile(t, "test.txt", "initial content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")
	initialCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	// Create second commit that modifies the file
	local.CreateFile(t, "test.txt", "modified content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Modify file")
	modifiedCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	// Create third commit that renames and adds files
	local.Git(t, "mv", "test.txt", "renamed.txt")
	local.CreateFile(t, "new.txt", "modified content")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Rename and add files")
	renamedCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	// Push all commits
	local.Git(t, "push", "origin", "main", "--force")

	// Create client
	logger := helpers.NewTestLogger(t)
	client, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test comparing initial and modified commits
	changes, err := client.CompareCommits(ctx, initialCommitHash, modifiedCommitHash)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "test.txt", changes[0].Path)
	assert.Equal(t, protocol.FileStatusModified, changes[0].Status)

	// Get the file hashes for verification
	initialFileHash, err := hash.FromHex(local.Git(t, "rev-parse", initialCommitHash.String()+":test.txt"))
	require.NoError(t, err)
	modifiedFileHash, err := hash.FromHex(local.Git(t, "rev-parse", modifiedCommitHash.String()+":test.txt"))
	require.NoError(t, err)

	assert.Equal(t, initialFileHash, changes[0].OldHash)
	assert.Equal(t, modifiedFileHash, changes[0].Hash)

	// Test comparing modified and renamed commits
	changes, err = client.CompareCommits(ctx, modifiedCommitHash, renamedCommitHash)
	require.NoError(t, err)
	require.Len(t, changes, 2)

	// Sort changes by path for consistent testing
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	assert.Equal(t, "new.txt", changes[0].Path)
	assert.Equal(t, protocol.FileStatusAdded, changes[0].Status)

	assert.Equal(t, "test.txt", changes[1].Path)
	assert.Equal(t, protocol.FileStatusDeleted, changes[1].Status)
}
