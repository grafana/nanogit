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

	t.Run("CreateBlob with new file", func(t *testing.T) {
		// Pull latest changes before starting the test
		local.Git(t, "pull", "origin", "main")

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
			Hash: currentHash,
		}
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "new.txt", newContent)
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

	t.Run("CreateBlob with nested path", func(t *testing.T) {
		// Pull latest changes before starting the test
		local.Git(t, "pull", "origin", "main")

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
			Hash: currentHash,
		}

		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)
		_, err = writer.CreateBlob(ctx, "dir/subdir/file.txt", nestedContent)
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

	t.Run("CreateBlob with invalid ref", func(t *testing.T) {
		_, err := client.NewRefWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})

	t.Run("UpdateBlob with existing file", func(t *testing.T) {
		// Pull latest changes before starting the test
		local.Git(t, "clean", "-fd")
		local.Git(t, "pull")

		// Create a new file to be updated
		newContent := []byte("New file content")
		local.CreateFile(t, "tobeupdated.txt", string(newContent))

		// Add and commit the file to be updated
		local.Git(t, "add", "tobeupdated.txt")
		local.Git(t, "commit", "-m", "Add file to be updated")
		local.Git(t, "push")

		// Get current ref
		ref, err := client.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)

		// Create a writer
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		// Update the test file
		updatedContent := []byte("Updated content")
		blobHash, err := writer.UpdateBlob(ctx, "tobeupdated.txt", updatedContent)
		require.NoError(t, err)
		require.NotNil(t, blobHash)

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

		// Commit the changes
		commit, err := writer.Commit(ctx, "Update test file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Clean up any untracked files before pulling
		local.Git(t, "clean", "-fd")
		local.Git(t, "pull")

		// Verify commit hash
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		// Verify file content was updated
		content, err := os.ReadFile(filepath.Join(local.Path, "tobeupdated.txt"))
		require.NoError(t, err)
		require.Equal(t, updatedContent, content)

		// Verify author and committer
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

		// Verify the test file was not removed
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
		require.NoError(t, err)
		require.Equal(t, testContent, otherContent)
	})
	t.Run("UpdateBlob with nested file", func(t *testing.T) {
		// Pull latest changes before starting the test
		local.Git(t, "clean", "-fd")
		local.Git(t, "pull")

		// Create a new file to be updated
		newContent := []byte("New file content")
		local.CreateDirPath(t, "dir/subdir")
		local.CreateFile(t, "dir/subdir/tobeupdated.txt", string(newContent))

		// Add and commit the file to be updated
		local.Git(t, "add", "dir/subdir/tobeupdated.txt")
		local.Git(t, "commit", "-m", "Add file to be updated")
		local.Git(t, "push", "origin", "main")

		// Get current ref
		ref, err := client.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)

		// Create a writer
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		// Update the nested file
		updatedContent := []byte("Updated nested content")
		blobHash, err := writer.UpdateBlob(ctx, "dir/subdir/tobeupdated.txt", updatedContent)
		require.NoError(t, err)
		require.NotNil(t, blobHash)

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

		// Commit the changes
		commit, err := writer.Commit(ctx, "Update nested file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Verify using Git CLI
		// Clean up any untracked files before pulling
		local.Git(t, "clean", "-fd")
		local.Git(t, "pull")

		// Verify commit hash
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		// Verify file content was updated
		content, err := os.ReadFile(filepath.Join(local.Path, "dir/subdir/tobeupdated.txt"))
		require.NoError(t, err)
		require.Equal(t, updatedContent, content)

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
	})

	t.Run("UpdateBlob with nonexistent file", func(t *testing.T) {
		// Get current ref
		ref, err := client.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)

		// Create a writer
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		// Try to update a nonexistent file
		_, err = writer.UpdateBlob(ctx, "nonexistent.txt", []byte("content"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blob at that path does not exist")
	})
}
