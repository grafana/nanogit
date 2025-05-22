//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	t.Run("CreateFile with new file", func(t *testing.T) {
		newContent := []byte("new content")
		author := nanogit.Author{
			Name:  "Test Author",
			Email: "test@example.com",
			Time:  time.Now(),
		}
		committer := nanogit.Committer{
			Name:  "Test Committer",
			Email: "test@example.com",
			Time:  time.Now(),
		}

		// Get the current commit hash for the branch
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)

		// Pass the ref with both Name and Hash
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash.String(),
		}
		writer, err := client.GetRefWriter(ctx, ref)
		require.NoError(t, err)

		_, err = writer.CreateFile(ctx, "new.txt", newContent)
		require.NoError(t, err)
		commit, err := writer.Commit(ctx, "Add new file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		// push the commit
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Verify using Git CLI
		local.Git(t, "pull", "origin", "main")
		// verify commit hash
		require.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		// verify file content
		content, err := os.ReadFile(filepath.Join(local.Path, "new.txt"))
		require.NoError(t, err)
		require.Equal(t, newContent, content)

		// Verify commit message
		commitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		require.Equal(t, "Add new file", strings.TrimSpace(commitMsg))

		// Verify author
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		// Verify committer
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

		// Verify the ref was updated
		hashAfterCommit := local.Git(t, "rev-parse", "refs/heads/main")
		require.NotEqual(t, currentHash.String(), hashAfterCommit)

		// Verify file content
		content, err = os.ReadFile(filepath.Join(local.Path, "new.txt"))
		require.NoError(t, err)
		require.Equal(t, newContent, content)
		// Verify the test file was not removed
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
		require.NoError(t, err)
		require.Equal(t, testContent, otherContent)
	})

	t.Run("CreateFile with nested path", func(t *testing.T) {
		nestedContent := []byte("nested content")
		author := nanogit.Author{
			Name:  "Test Author",
			Email: "test@example.com",
			Time:  time.Now(),
		}
		committer := nanogit.Committer{
			Name:  "Test Committer",
			Email: "test@example.com",
			Time:  time.Now(),
		}

		// Get the current commit hash for the branch
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)

		// Pass the ref with both Name and Hash
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash.String(),
		}

		writer, err := client.GetRefWriter(ctx, ref)
		require.NoError(t, err)
		_, err = writer.CreateFile(ctx, "dir/subdir/file.txt", nestedContent)
		require.NoError(t, err)
		commit, err := writer.Commit(ctx, "Add nested file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Verify using Git CLI
		local.Git(t, "pull", "origin", "main")

		// verify commit hash
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		// Verify directory structure
		dirInfo, err := os.Stat(filepath.Join(local.Path, "dir"))
		require.NoError(t, err)
		require.True(t, dirInfo.IsDir())

		subdirInfo, err := os.Stat(filepath.Join(local.Path, "dir/subdir"))
		require.NoError(t, err)
		require.True(t, subdirInfo.IsDir())

		// Verify file content
		content, err := os.ReadFile(filepath.Join(local.Path, "dir/subdir/file.txt"))
		require.NoError(t, err)
		require.Equal(t, nestedContent, content)

		// Verify the test file was not removed
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
		require.NoError(t, err)
		require.Equal(t, testContent, otherContent)

		// Verify author
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		// Verify committer
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

		// Verify the ref was updated
		hashAfterCommit := local.Git(t, "rev-parse", "refs/heads/main")
		require.NotEqual(t, currentHash.String(), hashAfterCommit)
	})

	t.Run("CreateFile with invalid ref", func(t *testing.T) {
		_, err := client.GetRefWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})
}
