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
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "new.txt", newContent)
		require.NoError(t, err)
		commit, err := writer.Commit(ctx, "Add new file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing the commit")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("")
		local.Git(t, "pull")
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

		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)
		_, err = writer.CreateBlob(ctx, "dir/subdir/file.txt", nestedContent)
		require.NoError(t, err)
		commit, err := writer.Commit(ctx, "Add nested file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Verifying using Git CLI")
		local.Git(t, "pull")

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

		_, err := client.NewStagedWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent", Hash: hash.Zero})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateBlob with existing file", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

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
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Verifying blob hash before update")
		oldBlobHash := local.Git(t, "rev-parse", "HEAD:tobeupdated.txt")
		logger.Info("Old blob hash", "hash", oldBlobHash)

		logger.Info("Updating file content")
		updatedContent := []byte("Updated content")
		blobHash, err := writer.UpdateBlob(ctx, "tobeupdated.txt", updatedContent)
		require.NoError(t, err)
		require.NotNil(t, blobHash)

		logger.Info("Blob hash changed", "old_hash", oldBlobHash, "new_hash", blobHash.String())
		require.NotEqual(t, oldBlobHash, blobHash.String())

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

		logger.Info("Committing changes", "previous_commit", ref.Hash.String(), "ref", ref.Name)
		commit, err := writer.Commit(ctx, "Update test file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		logger.Info("Pushing changes", "blob_hash", blobHash.String(), "commit", commit.Hash.String(), "previous_commit", ref.Hash.String(), "ref", ref.Name)
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Pull
		logger.Info("Getting git status before pull")
		local.Git(t, "fetch", "origin")
		status := local.Git(t, "status")
		logger.Info("Git status", "status", status)
		require.Contains(t, status, "Your branch is behind 'origin/main' by 1 commit")

		logger.Info("Checking repository state")
		logger.Info("Current branch", "branch", local.Git(t, "branch", "--show-current"))
		logger.Info("Remote tracking branch", "branch", local.Git(t, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"))
		logger.Info("Local commit", "hash", local.Git(t, "rev-parse", "HEAD"))
		logger.Info("Remote commit", "hash", local.Git(t, "rev-parse", "origin/main"))

		logger.Info("Checking for untracked files")
		untracked := local.Git(t, "ls-files", "--others", "--exclude-standard")
		logger.Info("Untracked files", "files", untracked)
		require.Empty(t, untracked, "Found untracked files: %s", untracked)

		logger.Info("Checking index state")
		indexFiles := local.Git(t, "ls-files", "--stage")
		logger.Info("Files in index", "files", indexFiles)

		logger.Info("Logging repository contents before pull")
		local.LogRepoContents(t)
		logger.Info("Pulling latest changes")
		local.Git(t, "pull")
		logger.Info("Logging repository contents after pull")
		local.LogRepoContents(t)

		// After pull
		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

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
		writer, err := client.NewStagedWriter(ctx, ref)
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

		logger.Info("Committing changes", "previous_commit", ref.Hash.String(), "ref", ref.Name)
		commit, err := writer.Commit(ctx, "Update nested file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)
		logger.Info("Pushing changes", "blob_hash", blobHash.String(), "commit", commit.Hash.String(), "previous_commit", ref.Hash.String(), "ref", ref.Name)
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Pulling
		logger.Info("Getting git status before pull")
		local.Git(t, "fetch", "origin")
		status := local.Git(t, "status")
		logger.Info("Git status", "status")
		require.Contains(t, status, "Your branch is behind 'origin/main' by 1 commit")
		logger.Info("Checking for untracked files")
		untracked := local.Git(t, "ls-files", "--others", "--exclude-standard")
		require.Empty(t, untracked, "Found untracked files: %s", untracked)

		logger.Info("Logging repository contents before pull")
		local.LogRepoContents(t)
		logger.Info("Pulling latest changes")
		local.Git(t, "pull")
		logger.Info("Logging repository contents after pull")
		local.LogRepoContents(t)

		// After pull
		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

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
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Trying to update a nonexistent file")
		_, err = writer.UpdateBlob(ctx, "nonexistent.txt", []byte("content"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blob at that path does not exist")
	})

	t.Run("DeleteBlob with existing file", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating a file to be deleted")
		fileContent := []byte("File to be deleted")
		local.CreateFile(t, "tobedeleted.txt", string(fileContent))

		logger.Info("Adding and committing the file to be deleted")
		local.Git(t, "add", "tobedeleted.txt")
		local.Git(t, "commit", "-m", "Add file to be deleted")
		local.Git(t, "push")

		logger.Info("Verifying file exists before deletion")
		_, err := os.Stat(filepath.Join(local.Path, "tobedeleted.txt"))
		require.NoError(t, err)

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Deleting the file")
		treeHash, err := writer.DeleteBlob(ctx, "tobedeleted.txt")
		require.NoError(t, err)
		require.NotNil(t, treeHash)

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

		logger.Info("Committing deletion", "previous_commit", ref.Hash.String(), "ref", ref.Name)
		commit, err := writer.Commit(ctx, "Delete test file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes", "tree_hash", treeHash.String(), "commit", commit.Hash.String(), "previous_commit", ref.Hash.String(), "ref", ref.Name)
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "pull")

		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		logger.Info("Verifying file was deleted")
		_, err = os.Stat(filepath.Join(local.Path, "tobedeleted.txt"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying other files were preserved")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, fileContent, otherContent)

		logger.Info("Verifying commit message")
		commitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		require.Equal(t, "Delete test file", strings.TrimSpace(commitMsg))

		logger.Info("Verifying author and committer")
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))
	})

	t.Run("DeleteBlob with nested file", func(t *testing.T) {
		logger, local, client, _ := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating nested file to be deleted")
		nestedContent := []byte("Nested file to be deleted")
		local.CreateDirPath(t, "dir/subdir")
		local.CreateFile(t, "dir/subdir/tobedeleted.txt", string(nestedContent))

		logger.Info("Adding and committing the nested file")
		local.Git(t, "add", "dir/subdir/tobedeleted.txt")
		local.Git(t, "commit", "-m", "Add nested file to be deleted")
		local.Git(t, "push")

		logger.Info("Verifying nested file exists before deletion")
		_, err := os.Stat(filepath.Join(local.Path, "dir/subdir/tobedeleted.txt"))
		require.NoError(t, err)

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Deleting the nested file")
		treeHash, err := writer.DeleteBlob(ctx, "dir/subdir/tobedeleted.txt")
		require.NoError(t, err)
		require.NotNil(t, treeHash)

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

		logger.Info("Committing deletion", "previous_commit", ref.Hash.String(), "ref", ref.Name)
		commit, err := writer.Commit(ctx, "Delete nested file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes", "tree_hash", treeHash.String(), "commit", commit.Hash.String(), "previous_commit", ref.Hash.String(), "ref", ref.Name)
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "pull")

		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		logger.Info("Verifying nested file was deleted")
		_, err = os.Stat(filepath.Join(local.Path, "dir/subdir/tobedeleted.txt"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying directory structure was removed")
		_, err = os.Stat(filepath.Join(local.Path, "dir"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying commit message")
		commitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		require.Equal(t, "Delete nested file", strings.TrimSpace(commitMsg))

		logger.Info("Verifying author and committer")
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		require.Equal(t, "Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		require.Equal(t, "Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))
	})

	t.Run("DeleteBlob with nonexistent file", func(t *testing.T) {
		logger, _, client, _ := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Getting current ref")
		ref, err := client.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)

		logger.Info("Creating a writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Trying to delete a nonexistent file")
		_, err = writer.DeleteBlob(ctx, "nonexistent.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blob at that path does not exist")
	})

	t.Run("DeleteBlob preserves other files in same directory", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating multiple files in same directory")
		file1Content := []byte("File 1 content")
		file2Content := []byte("File 2 content")
		local.CreateDirPath(t, "shared")
		local.CreateFile(t, "shared/file1.txt", string(file1Content))
		local.CreateFile(t, "shared/file2.txt", string(file2Content))

		logger.Info("Adding and committing all files")
		local.Git(t, "add", "shared/")
		local.Git(t, "commit", "-m", "Add files to shared directory")
		local.Git(t, "push")

		logger.Info("Verifying both files exist before deletion")
		_, err := os.Stat(filepath.Join(local.Path, "shared/file1.txt"))
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(local.Path, "shared/file2.txt"))
		require.NoError(t, err)

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Deleting only file1.txt")
		treeHash, err := writer.DeleteBlob(ctx, "shared/file1.txt")
		require.NoError(t, err)
		require.NotNil(t, treeHash)

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

		logger.Info("Committing deletion")
		commit, err := writer.Commit(ctx, "Delete file1.txt only", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "pull")

		logger.Info("Verifying file1.txt was deleted")
		_, err = os.Stat(filepath.Join(local.Path, "shared/file1.txt"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying file2.txt still exists")
		content, err := os.ReadFile(filepath.Join(local.Path, "shared/file2.txt"))
		require.NoError(t, err)
		require.Equal(t, file2Content, content)

		logger.Info("Verifying directory still exists")
		dirInfo, err := os.Stat(filepath.Join(local.Path, "shared"))
		require.NoError(t, err)
		require.True(t, dirInfo.IsDir())

		logger.Info("Verifying other files were preserved")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, file1Content, otherContent)
		require.NotEqual(t, file2Content, otherContent)
	})

	t.Run("DeleteTree with directory containing files", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating directory with files to be deleted")
		dir1Content := []byte("Directory 1 file content")
		file1Content := []byte("File 1 content")
		file2Content := []byte("File 2 content")
		local.CreateDirPath(t, "toberemoved")
		local.CreateFile(t, "toberemoved/file1.txt", string(file1Content))
		local.CreateFile(t, "toberemoved/file2.txt", string(file2Content))
		local.CreateFile(t, "preserved.txt", string(dir1Content))

		logger.Info("Adding and committing the directory with files")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Add directory with files to be deleted")
		local.Git(t, "push")

		logger.Info("Verifying directory and files exist before deletion")
		_, err := os.Stat(filepath.Join(local.Path, "toberemoved"))
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file1.txt"))
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file2.txt"))
		require.NoError(t, err)

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Deleting the entire directory")
		treeHash, err := writer.DeleteTree(ctx, "toberemoved")
		require.NoError(t, err)
		require.NotNil(t, treeHash)

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

		logger.Info("Committing directory deletion")
		commit, err := writer.Commit(ctx, "Delete entire directory", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "pull")

		logger.Info("Verifying commit hash")
		assert.Equal(t, commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		logger.Info("Verifying directory was completely deleted")
		_, err = os.Stat(filepath.Join(local.Path, "toberemoved"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying all files in directory were deleted")
		_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file1.txt"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file2.txt"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying other files were preserved")
		preservedContent, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
		require.NoError(t, err)
		require.Equal(t, dir1Content, preservedContent)

		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, file1Content, otherContent)
		require.NotEqual(t, file2Content, otherContent)

		logger.Info("Verifying commit message")
		commitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		require.Equal(t, "Delete entire directory", strings.TrimSpace(commitMsg))
	})

	t.Run("DeleteTree with nested directories", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating nested directory structure to be deleted")
		preservedContent := []byte("Preserved content")
		nested1Content := []byte("Nested 1 content")
		nested2Content := []byte("Nested 2 content")
		deepContent := []byte("Deep nested content")

		local.CreateDirPath(t, "toberemoved/subdir1")
		local.CreateDirPath(t, "toberemoved/subdir2/deep")
		local.CreateFile(t, "preserved.txt", string(preservedContent))
		local.CreateFile(t, "toberemoved/file.txt", string(nested1Content))
		local.CreateFile(t, "toberemoved/subdir1/nested.txt", string(nested2Content))
		local.CreateFile(t, "toberemoved/subdir2/deep/deep.txt", string(deepContent))

		logger.Info("Adding and committing the nested directory structure")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Add nested directory structure")
		local.Git(t, "push")

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Deleting the entire nested directory")
		treeHash, err := writer.DeleteTree(ctx, "toberemoved")
		require.NoError(t, err)
		require.NotNil(t, treeHash)

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

		logger.Info("Committing nested directory deletion")
		commit, err := writer.Commit(ctx, "Delete nested directory structure", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "pull")

		logger.Info("Verifying entire directory structure was deleted")
		_, err = os.Stat(filepath.Join(local.Path, "toberemoved"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying preserved file still exists")
		content, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
		require.NoError(t, err)
		require.Equal(t, preservedContent, content)

		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, nested1Content, otherContent)
	})

	t.Run("DeleteTree with nonexistent directory", func(t *testing.T) {
		logger, _, client, _ := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Getting current ref")
		ref, err := client.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)

		logger.Info("Creating a writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Trying to delete a nonexistent directory")
		_, err = writer.DeleteTree(ctx, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tree at that path does not exist")
	})

	t.Run("DeleteTree with file instead of directory", func(t *testing.T) {
		logger, local, client, _ := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating a file to test error case")
		fileContent := []byte("This is a file, not a directory")
		local.CreateFile(t, "testfile.txt", string(fileContent))
		local.Git(t, "add", "testfile.txt")
		local.Git(t, "commit", "-m", "Add test file")
		local.Git(t, "push")

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating a writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Trying to delete a file as if it were a directory")
		_, err = writer.DeleteTree(ctx, "testfile.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry at that path is not a tree")
	})

	t.Run("DeleteTree with subdirectory only", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

		logger.Info("Creating parent directory with subdirectories")
		parentFile := []byte("Parent file")
		subdir1File := []byte("Subdirectory 1 file")
		subdir2File := []byte("Subdirectory 2 file")

		local.CreateDirPath(t, "parent/subdir1")
		local.CreateDirPath(t, "parent/subdir2")
		local.CreateFile(t, "parent/parentfile.txt", string(parentFile))
		local.CreateFile(t, "parent/subdir1/file1.txt", string(subdir1File))
		local.CreateFile(t, "parent/subdir2/file2.txt", string(subdir2File))

		logger.Info("Adding and committing the directory structure")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Add parent with subdirectories")
		local.Git(t, "push")

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "refs/heads/main"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating ref writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		logger.Info("Deleting only subdir1, leaving subdir2 and parent")
		treeHash, err := writer.DeleteTree(ctx, "parent/subdir1")
		require.NoError(t, err)
		require.NotNil(t, treeHash)

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

		logger.Info("Committing subdirectory deletion")
		commit, err := writer.Commit(ctx, "Delete only subdir1", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling latest changes")
		local.Git(t, "pull")

		logger.Info("Verifying subdir1 was deleted")
		_, err = os.Stat(filepath.Join(local.Path, "parent/subdir1"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		logger.Info("Verifying parent directory still exists")
		parentContent, err := os.ReadFile(filepath.Join(local.Path, "parent/parentfile.txt"))
		require.NoError(t, err)
		require.Equal(t, parentFile, parentContent)

		logger.Info("Verifying subdir2 still exists")
		subdir2Content, err := os.ReadFile(filepath.Join(local.Path, "parent/subdir2/file2.txt"))
		require.NoError(t, err)
		require.Equal(t, subdir2File, subdir2Content)

		logger.Info("Verifying other files were preserved")
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEqual(t, subdir1File, otherContent)
	})

	t.Run("CreateBlob multiple files in different directories with one commit", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

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

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating staged writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		// Create multiple files in the same directory
		logger.Info("Creating multiple JSON files in config directory")
		config1Content := []byte(`{"database": {"host": "localhost", "port": 5432}}`)
		config2Content := []byte(`{"api": {"timeout": 30, "retries": 3}}`)
		config3Content := []byte(`{"logging": {"level": "info", "output": "stdout"}}`)

		_, err = writer.CreateBlob(ctx, "config/database.json", config1Content)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "config/api.json", config2Content)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "config/logging.json", config3Content)
		require.NoError(t, err)

		// Create files in different subdirectories
		logger.Info("Creating files in different subdirectories")
		dataContent := []byte(`{"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}`)
		schemaContent := []byte(`{"type": "object", "properties": {"name": {"type": "string"}}}`)

		_, err = writer.CreateBlob(ctx, "data/users.json", dataContent)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "schemas/user.json", schemaContent)
		require.NoError(t, err)

		logger.Info("Committing all files")
		commit, err := writer.Commit(ctx, "Add configuration and data files", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit)

		logger.Info("Pushing changes")
		err = writer.Push(ctx)
		require.NoError(t, err)

		logger.Info("Pulling and verifying")
		local.Git(t, "pull")

		// Verify directory structure
		logger.Info("Verifying directory structure")
		configDir, err := os.Stat(filepath.Join(local.Path, "config"))
		require.NoError(t, err)
		require.True(t, configDir.IsDir())

		dataDir, err := os.Stat(filepath.Join(local.Path, "data"))
		require.NoError(t, err)
		require.True(t, dataDir.IsDir())

		schemasDir, err := os.Stat(filepath.Join(local.Path, "schemas"))
		require.NoError(t, err)
		require.True(t, schemasDir.IsDir())

		// Verify all files exist with correct content
		logger.Info("Verifying file contents")
		content1, err := os.ReadFile(filepath.Join(local.Path, "config/database.json"))
		require.NoError(t, err)
		require.Equal(t, config1Content, content1)

		content2, err := os.ReadFile(filepath.Join(local.Path, "config/api.json"))
		require.NoError(t, err)
		require.Equal(t, config2Content, content2)

		content3, err := os.ReadFile(filepath.Join(local.Path, "config/logging.json"))
		require.NoError(t, err)
		require.Equal(t, config3Content, content3)

		dataFileContent, err := os.ReadFile(filepath.Join(local.Path, "data/users.json"))
		require.NoError(t, err)
		require.Equal(t, dataContent, dataFileContent)

		schemaFileContent, err := os.ReadFile(filepath.Join(local.Path, "schemas/user.json"))
		require.NoError(t, err)
		require.Equal(t, schemaContent, schemaFileContent)

		// Verify original file is preserved
		logger.Info("Verifying original file preserved")
		originalContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEmpty(t, originalContent)

		// Verify commit details
		logger.Info("Verifying commit details")
		finalHash := local.Git(t, "rev-parse", "HEAD")
		require.Equal(t, commit.Hash.String(), finalHash)

		commitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		require.Equal(t, "Add configuration and data files", strings.TrimSpace(commitMsg))
	})

	t.Run("CreateBlob multiple files in different directories across multiple commits", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

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

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating staged writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		// First commit: Create config files
		logger.Info("First commit: Creating configuration files")
		dbConfigContent := []byte(`{"host": "localhost", "port": 5432, "database": "myapp"}`)
		apiConfigContent := []byte(`{"baseUrl": "https://api.example.com", "timeout": 30}`)

		_, err = writer.CreateBlob(ctx, "config/database.json", dbConfigContent)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "config/api.json", apiConfigContent)
		require.NoError(t, err)

		commit1, err := writer.Commit(ctx, "Add database and API configuration", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit1)
		logger.Info("First commit created", "hash", commit1.Hash.String())

		// Second commit: Create documentation files
		logger.Info("Second commit: Creating documentation files")
		readmeContent := []byte(`# My Application\n\nThis is a sample application.`)
		apiDocsContent := []byte(`# API Documentation\n\n## Endpoints\n\n- GET /users`)

		_, err = writer.CreateBlob(ctx, "docs/README.md", readmeContent)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "docs/api.md", apiDocsContent)
		require.NoError(t, err)

		commit2, err := writer.Commit(ctx, "Add documentation files", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit2)
		logger.Info("Second commit created", "hash", commit2.Hash.String(), "parent", commit2.Parent.String())

		// Third commit: Create test and data files
		logger.Info("Third commit: Creating test and data files")
		testDataContent := []byte(`{"testUsers": [{"id": 1, "name": "Test User"}]}`)
		schemaContent := []byte(`{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`)

		_, err = writer.CreateBlob(ctx, "tests/data/users.json", testDataContent)
		require.NoError(t, err)

		_, err = writer.CreateBlob(ctx, "schemas/user.json", schemaContent)
		require.NoError(t, err)

		commit3, err := writer.Commit(ctx, "Add test data and schema files", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit3)
		logger.Info("Third commit created", "hash", commit3.Hash.String(), "parent", commit3.Parent.String())

		// Verify commit chain before push
		require.Equal(t, currentHash, commit1.Parent, "First commit should have initial commit as parent")
		require.Equal(t, commit1.Hash, commit2.Parent, "Second commit should have first commit as parent")
		require.Equal(t, commit2.Hash, commit3.Parent, "Third commit should have second commit as parent")

		// Push all commits at once
		logger.Info("Pushing all three commits")
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Pull and verify
		logger.Info("Pulling changes")
		local.Git(t, "pull")

		// Verify final commit hash
		logger.Info("Verifying final commit hash")
		finalHash := local.Git(t, "rev-parse", "HEAD")
		require.Equal(t, commit3.Hash.String(), finalHash, "HEAD should point to third commit")

		// Verify all commits exist in history
		logger.Info("Verifying commit history")
		commitHistory := local.Git(t, "log", "--oneline", "--format=%H %s")
		logger.Info("Commit history", "history", commitHistory)

		require.Contains(t, commitHistory, commit1.Hash.String(), "First commit should be in history")
		require.Contains(t, commitHistory, commit2.Hash.String(), "Second commit should be in history")
		require.Contains(t, commitHistory, commit3.Hash.String(), "Third commit should be in history")

		require.Contains(t, commitHistory, "Add database and API configuration", "First commit message should be in history")
		require.Contains(t, commitHistory, "Add documentation files", "Second commit message should be in history")
		require.Contains(t, commitHistory, "Add test data and schema files", "Third commit message should be in history")

		// Verify directory structure
		logger.Info("Verifying directory structure")
		configDir, err := os.Stat(filepath.Join(local.Path, "config"))
		require.NoError(t, err)
		require.True(t, configDir.IsDir())

		docsDir, err := os.Stat(filepath.Join(local.Path, "docs"))
		require.NoError(t, err)
		require.True(t, docsDir.IsDir())

		testsDir, err := os.Stat(filepath.Join(local.Path, "tests"))
		require.NoError(t, err)
		require.True(t, testsDir.IsDir())

		testDataDir, err := os.Stat(filepath.Join(local.Path, "tests/data"))
		require.NoError(t, err)
		require.True(t, testDataDir.IsDir())

		schemasDir, err := os.Stat(filepath.Join(local.Path, "schemas"))
		require.NoError(t, err)
		require.True(t, schemasDir.IsDir())

		// Verify all files exist with correct content
		logger.Info("Verifying file contents")
		dbConfig, err := os.ReadFile(filepath.Join(local.Path, "config/database.json"))
		require.NoError(t, err)
		require.Equal(t, dbConfigContent, dbConfig)

		apiConfig, err := os.ReadFile(filepath.Join(local.Path, "config/api.json"))
		require.NoError(t, err)
		require.Equal(t, apiConfigContent, apiConfig)

		readme, err := os.ReadFile(filepath.Join(local.Path, "docs/README.md"))
		require.NoError(t, err)
		require.Equal(t, readmeContent, readme)

		apiDocs, err := os.ReadFile(filepath.Join(local.Path, "docs/api.md"))
		require.NoError(t, err)
		require.Equal(t, apiDocsContent, apiDocs)

		testData, err := os.ReadFile(filepath.Join(local.Path, "tests/data/users.json"))
		require.NoError(t, err)
		require.Equal(t, testDataContent, testData)

		schema, err := os.ReadFile(filepath.Join(local.Path, "schemas/user.json"))
		require.NoError(t, err)
		require.Equal(t, schemaContent, schema)

		// Verify original file is preserved
		logger.Info("Verifying original file preserved")
		originalContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEmpty(t, originalContent)

		// Verify individual commits show correct files
		logger.Info("Verifying files in individual commits")
		commit1Files := strings.TrimSpace(local.Git(t, "ls-tree", "--name-only", "-r", commit1.Hash.String()))
		require.Contains(t, commit1Files, "config/database.json")
		require.Contains(t, commit1Files, "config/api.json")
		require.Contains(t, commit1Files, initCommitFile)
		require.NotContains(t, commit1Files, "docs/README.md")

		commit2Files := strings.TrimSpace(local.Git(t, "ls-tree", "--name-only", "-r", commit2.Hash.String()))
		require.Contains(t, commit2Files, "config/database.json")
		require.Contains(t, commit2Files, "config/api.json")
		require.Contains(t, commit2Files, "docs/README.md")
		require.Contains(t, commit2Files, "docs/api.md")
		require.Contains(t, commit2Files, initCommitFile)
		require.NotContains(t, commit2Files, "tests/data/users.json")

		commit3Files := strings.TrimSpace(local.Git(t, "ls-tree", "--name-only", "-r", commit3.Hash.String()))
		require.Contains(t, commit3Files, "config/database.json")
		require.Contains(t, commit3Files, "config/api.json")
		require.Contains(t, commit3Files, "docs/README.md")
		require.Contains(t, commit3Files, "docs/api.md")
		require.Contains(t, commit3Files, "tests/data/users.json")
		require.Contains(t, commit3Files, "schemas/user.json")
		require.Contains(t, commit3Files, initCommitFile)
	})

	t.Run("Multiple commits should all be visible after push", func(t *testing.T) {
		logger, local, client, initCommitFile := quickSetup(t)
		ctx := context.Background()
		logger.ForSubtest(t)

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

		logger.Info("Getting current ref")
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)
		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		logger.Info("Creating staged writer")
		writer, err := client.NewStagedWriter(ctx, ref)
		require.NoError(t, err)

		// Create first commit
		logger.Info("Creating first file and commit")
		_, err = writer.CreateBlob(ctx, "file1.txt", []byte("First file content"))
		require.NoError(t, err)
		commit1, err := writer.Commit(ctx, "Add first file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit1)
		logger.Info("First commit", "hash", commit1.Hash.String())

		// Create second commit
		logger.Info("Creating second file and commit")
		_, err = writer.CreateBlob(ctx, "file2.txt", []byte("Second file content"))
		require.NoError(t, err)
		commit2, err := writer.Commit(ctx, "Add second file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit2)
		logger.Info("Second commit", "hash", commit2.Hash.String(), "parent", commit2.Parent.String())

		// Create third commit
		logger.Info("Creating third file and commit")
		_, err = writer.CreateBlob(ctx, "file3.txt", []byte("Third file content"))
		require.NoError(t, err)
		commit3, err := writer.Commit(ctx, "Add third file", author, committer)
		require.NoError(t, err)
		require.NotNil(t, commit3)
		logger.Info("Third commit", "hash", commit3.Hash.String(), "parent", commit3.Parent.String())

		// Verify commit chain is correct
		require.Equal(t, currentHash, commit1.Parent, "First commit should have initial commit as parent")
		require.Equal(t, commit1.Hash, commit2.Parent, "Second commit should have first commit as parent")
		require.Equal(t, commit2.Hash, commit3.Parent, "Third commit should have second commit as parent")

		// Push all commits
		logger.Info("Pushing all commits")
		err = writer.Push(ctx)
		require.NoError(t, err)

		// Pull and verify
		logger.Info("Pulling changes")
		local.Git(t, "pull")

		// Verify final commit hash
		logger.Info("Verifying final commit hash")
		finalHash := local.Git(t, "rev-parse", "HEAD")
		require.Equal(t, commit3.Hash.String(), finalHash, "HEAD should point to third commit")

		// Verify all three commits exist in history
		logger.Info("Verifying commit history")
		commitHistory := local.Git(t, "log", "--oneline", "--format=%H %s")
		logger.Info("Commit history", "history", commitHistory)

		// Should contain all three commits
		require.Contains(t, commitHistory, commit1.Hash.String(), "First commit should be in history")
		require.Contains(t, commitHistory, commit2.Hash.String(), "Second commit should be in history")
		require.Contains(t, commitHistory, commit3.Hash.String(), "Third commit should be in history")

		// Verify commit messages are correct
		require.Contains(t, commitHistory, "Add first file", "First commit message should be in history")
		require.Contains(t, commitHistory, "Add second file", "Second commit message should be in history")
		require.Contains(t, commitHistory, "Add third file", "Third commit message should be in history")

		// Verify all files exist
		logger.Info("Verifying all files exist")
		content1, err := os.ReadFile(filepath.Join(local.Path, "file1.txt"))
		require.NoError(t, err)
		require.Equal(t, []byte("First file content"), content1)

		content2, err := os.ReadFile(filepath.Join(local.Path, "file2.txt"))
		require.NoError(t, err)
		require.Equal(t, []byte("Second file content"), content2)

		content3, err := os.ReadFile(filepath.Join(local.Path, "file3.txt"))
		require.NoError(t, err)
		require.Equal(t, []byte("Third file content"), content3)

		// Verify original file is preserved
		otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
		require.NoError(t, err)
		require.NotEmpty(t, otherContent)

		// Verify individual commits are reachable
		logger.Info("Verifying individual commits are reachable")
		commit1Files := strings.TrimSpace(local.Git(t, "ls-tree", "--name-only", commit1.Hash.String()))
		require.Contains(t, commit1Files, "file1.txt")
		require.Contains(t, commit1Files, initCommitFile)
		require.NotContains(t, commit1Files, "file2.txt")
		require.NotContains(t, commit1Files, "file3.txt")

		commit2Files := strings.TrimSpace(local.Git(t, "ls-tree", "--name-only", commit2.Hash.String()))
		require.Contains(t, commit2Files, "file1.txt")
		require.Contains(t, commit2Files, "file2.txt")
		require.Contains(t, commit2Files, initCommitFile)
		require.NotContains(t, commit2Files, "file3.txt")

		commit3Files := strings.TrimSpace(local.Git(t, "ls-tree", "--name-only", commit3.Hash.String()))
		require.Contains(t, commit3Files, "file1.txt")
		require.Contains(t, commit3Files, "file2.txt")
		require.Contains(t, commit3Files, "file3.txt")
		require.Contains(t, commit3Files, initCommitFile)
	})
}
