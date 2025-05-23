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

func quickSetup(t *testing.T) (*helpers.TestLogger, *helpers.LocalGitRepo, nanogit.Client, string) {
	logger := helpers.NewTestLogger(t)
	logger.Info("Setting up remote and local repository")
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)
	local := helpers.NewLocalGitRepo(t, logger)
	client, initCommitFile := local.QuickInit(t, user, remote.AuthURL())

	return logger, local, client, initCommitFile
}

func TestClient_Writer(t *testing.T) {
	t.Run("CreateBlob with new file", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()

		logger.ForSubtest(t)

		logger.Info("Pulling latest changes before starting the test")
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

		logger.Info("Getting the current commit hash for the branch")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

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

		logger.Info("Pushing the commit")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Verifying using Git CLI")
		local.Git(t, "pull", "origin", "main")
		require.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		logger.Info("Verifying file content")
		content, err := os.ReadFile(filepath.Join(local.Path, "new.txt"))
		require.NoError(t, err)
		require.Equal(t, newContent, content)

		logger.Info("Verifying commit message")
		commitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		require.Equal(t, "Add new file", strings.TrimSpace(commitMsg))

		logger.Info("Verifying author")
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))

		logger.Info("Verifying committer")
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

		logger.Info("Verifying the ref was updated")
		hashAfterCommit := local.Git(t, "rev-parse", "refs/heads/main")
		require.NotEqual(t, currentHash.String(), hashAfterCommit)

		logger.Info("Verifying file content")
		content, err = os.ReadFile(filepath.Join(local.Path, "new.txt"))
		require.NoError(t, err)
		require.Equal(t, newContent, content)

		logger.Info("Verifying the test file was not removed")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, newContent, otherContent)
	})

	t.Run("CreateBlob with nested path", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Pulling latest changes before starting the test")
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

		logger.Info("Getting the current commit hash for the branch")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

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

		logger.Info("Verifying using Git CLI")
		local.Git(t, "pull", "origin", "main")

		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		logger.Info("Verifying directory structure")
		dirInfo, err := os.Stat(filepath.Join(local.Path, "dir"))
		require.NoError(t, err)
		require.True(t, dirInfo.IsDir())

		subdirInfo, err := os.Stat(filepath.Join(local.Path, "dir/subdir"))
		require.NoError(t, err)
		require.True(t, subdirInfo.IsDir())

		logger.Info("Verifying file content")
		content, err := os.ReadFile(filepath.Join(local.Path, "dir/subdir/file.txt"))
		require.NoError(t, err)
		require.Equal(t, nestedContent, content)

		logger.Info("Verifying the test file was not removed")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, nestedContent, otherContent)

		logger.Info("Verifying author")
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))

		logger.Info("Verifying committer")
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

		logger.Info("Verifying the ref was updated")
		hashAfterCommit := local.Git(t, "rev-parse", "refs/heads/main")
		require.NotEqual(t, currentHash.String(), hashAfterCommit)
	})

	t.Run("CreateBlob with invalid ref", func(t *testing.T) {
		logger, _, client, _ := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		_, err := client.NewRefWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})

	t.Run("UpdateBlob with existing file", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Cleaning working directory")
		local.Git(t, "clean", "-fd")
		local.Git(t, "reset", "--hard")
		local.Git(t, "pull")

		logger.Info("Creating initial file to be updated")
		newContent := []byte("New file content")
		local.CreateFile(t, "tobeupdated.txt", string(newContent))

		logger.Info("Committing initial file")
		local.Git(t, "add", "tobeupdated.txt")
		local.Git(t, "commit", "-m", "Add file to be updated")
		local.Git(t, "push")

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Updating file content")
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

		logger.Info("Committing changes")
		commit, err := writer.Commit(ctx, "Update test file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "clean", "-fd")
		local.Git(t, "reset", "--hard")
		local.Git(t, "pull")

		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		logger.Info("Verifying updated file content")
		content, err := os.ReadFile(filepath.Join(local.Path, "tobeupdated.txt"))
		require.NoError(t, err)
		require.Equal(t, updatedContent, content)

		logger.Info("Verifying author and committer")
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

		logger.Info("Verifying test file was preserved")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, newContent, otherContent)
	})
	t.Run("UpdateBlob with nested file", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Ensuring clean working directory")
		local.Git(t, "clean", "-fd")
		local.Git(t, "reset", "--hard")
		local.Git(t, "pull")

		logger.Info("Creating new file to be updated")
		newContent := []byte("New file content")
		local.CreateDirPath(t, "dir/subdir")
		local.CreateFile(t, "dir/subdir/tobeupdated.txt", string(newContent))

		logger.Info("Adding and committing the file to be updated")
		local.Git(t, "add", "dir/subdir/tobeupdated.txt")
		local.Git(t, "commit", "-m", "Add file to be updated")
		local.Git(t, "push")

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Updating nested file content")
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

		logger.Info("Committing changes")
		commit, err := writer.Commit(ctx, "Update nested file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Cleaning up and pulling changes")
		local.Git(t, "clean", "-fd")
		local.Git(t, "reset", "--hard")
		local.Git(t, "pull")

		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		logger.Info("Verifying file content was updated")
		content, err := os.ReadFile(filepath.Join(local.Path, "dir/subdir/tobeupdated.txt"))
		require.NoError(t, err)
		require.Equal(t, updatedContent, content)

		logger.Info("Verifying test file was preserved")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, newContent, otherContent)

		logger.Info("Verifying author")
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		logger.Info("Verifying committer")
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))
	})

	t.Run("UpdateBlob with nonexistent file", func(t *testing.T) {
		logger, _, client, _ := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Getting current ref")
		ref, err := client.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)

		logger.Info("Creating a writer")
		writer, err := client.NewRefWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Trying to update a nonexistent file")
		_, err = writer.UpdateBlob(ctx, "nonexistent.txt", []byte("content"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blob at that path does not exist")
	})
}
