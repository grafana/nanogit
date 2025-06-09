package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
)

// TestBlobOperations tests creating, updating, and deleting blobs with different scenarios
func (s *IntegrationTestSuite) TestBlobOperations() {
	// Common test fixtures
	testAuthor := nanogit.Author{
		Name:  "Test Author",
		Email: "test@example.com",
		Time:  time.Now(),
	}
	testCommitter := nanogit.Committer{
		Name:  "Test Committer",
		Email: "test@example.com",
		Time:  time.Now(),
	}

	// Helper to verify author and committer in commit
	verifyCommitAuthorship := func(t *testing.T, local *helpers.LocalGitRepo) {
		commitAuthor := local.Git(t, "log", "-1", "--pretty=%an <%ae>")
		s.Equal("Test Author <test@example.com>", strings.TrimSpace(commitAuthor))

		commitCommitter := local.Git(t, "log", "-1", "--pretty=%cn <%ce>")
		s.Equal("Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))
	}

	// Helper to create writer from current HEAD
	createWriterFromHead := func(ctx context.Context, t *testing.T, client nanogit.Client, local *helpers.LocalGitRepo) (nanogit.StagedWriter, *hash.Hash) {
		currentHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		s.NoError(err)

		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		writer, err := client.NewStagedWriter(ctx, ref)
		s.NoError(err)

		return writer, &currentHash
	}

	// CREATE BLOB OPERATIONS
	s.Run("new file", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		client, _, local := s.GitServer.TestRepo(t)

		writer, currentHash := createWriterFromHead(ctx, t, client, local)

		newContent := []byte("new content")
		fileName := "new.txt"
		commitMsg := "Add new file"

		// Verify empty state before creating blob
		err := writer.Push(ctx)
		s.Error(err)
		s.ErrorIs(err, nanogit.ErrNothingToPush)

		_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.Error(err)
		s.ErrorIs(err, nanogit.ErrNothingToCommit)

		exists, err := writer.BlobExists(ctx, fileName)
		s.NoError(err)
		s.False(exists)

		// Create blob and commit
		_, err = writer.CreateBlob(ctx, fileName, newContent)
		s.NoError(err)

		exists, err = writer.BlobExists(ctx, fileName)
		s.NoError(err)
		s.True(exists)

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		content, err := os.ReadFile(filepath.Join(local.Path, fileName))
		s.NoError(err)
		s.Equal(newContent, content)

		actualCommitMsg := local.Git(t, "log", "-1", "--pretty=%B")
		s.Equal(commitMsg, strings.TrimSpace(actualCommitMsg))

		verifyCommitAuthorship(t, local)

		hashAfterCommit := local.Git(t, "rev-parse", "refs/heads/main")
		s.NotEqual(currentHash.String(), hashAfterCommit)

		// Verify initial file preserved
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
		s.NoError(err)
		s.NotEqual(newContent, otherContent)
	})

	s.Run("nested", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		client, _, local := s.GitServer.TestRepo(t)

		writer, currentHash := createWriterFromHead(ctx, t, client, local)
		nestedContent := []byte("nested content")
		nestedPath := "dir/subdir/file.txt"
		commitMsg := "Add nested file"

		// Verify nested path doesn't exist
		exists, err := writer.BlobExists(ctx, nestedPath)
		s.NoError(err)
		s.False(exists)

		_, err = writer.GetTree(ctx, "dir")
		var pathNotFoundErr *nanogit.PathNotFoundError
		s.Error(err)
		s.ErrorAs(err, &pathNotFoundErr)
		s.Equal("dir", pathNotFoundErr.Path)

		// Create nested blob
		_, err = writer.CreateBlob(ctx, nestedPath, nestedContent)
		s.NoError(err)

		exists, err = writer.BlobExists(ctx, nestedPath)
		s.NoError(err)
		s.True(exists)

		// Verify tree structure created
		tree, err := writer.GetTree(ctx, "dir")
		s.NoError(err)
		s.NotNil(tree)
		s.Len(tree.Entries, 1)
		s.Equal("subdir", tree.Entries[0].Name)
		s.Equal(protocol.ObjectTypeTree, tree.Entries[0].Type)
		s.Equal(uint32(0o40000), tree.Entries[0].Mode)

		subdirTree, err := writer.GetTree(ctx, "dir/subdir")
		s.NoError(err)
		s.NotNil(subdirTree)
		s.Len(subdirTree.Entries, 1)
		s.Equal("file.txt", subdirTree.Entries[0].Name)
		s.Equal(protocol.ObjectTypeBlob, subdirTree.Entries[0].Type)
		s.Equal(uint32(0o100644), subdirTree.Entries[0].Mode)

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "refs/heads/main"))

		// Verify directory structure
		dirInfo, err := os.Stat(filepath.Join(local.Path, "dir"))
		s.NoError(err)
		s.True(dirInfo.IsDir())

		subdirInfo, err := os.Stat(filepath.Join(local.Path, "dir/subdir"))
		s.NoError(err)
		s.True(subdirInfo.IsDir())

		content, err := os.ReadFile(filepath.Join(local.Path, nestedPath))
		s.NoError(err)
		s.Equal(nestedContent, content)

		verifyCommitAuthorship(t, local)

		hashAfterCommit := local.Git(t, "rev-parse", "refs/heads/main")
		s.NotEqual(currentHash.String(), hashAfterCommit)

		// Verify initial file preserved
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
		s.NoError(err)
		s.NotEqual(nestedContent, otherContent)
	})

	s.Run("invalid ref", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		client, _, _ := s.GitServer.TestRepo(t)

		_, err := client.NewStagedWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent", Hash: hash.Zero})
		s.Error(err)
		s.ErrorIs(err, nanogit.ErrObjectNotFound)
	})

	// UPDATE BLOB OPERATIONS
	s.Run("update file", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		remote, _ := s.CreateTestRepo()
		local := remote.Local(t)

		// Create and commit initial file plus file to be updated
		local.CreateFile(t, "initial.txt", "initial content")
		local.CreateFile(t, "tobeupdated.txt", "original content")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Initial commit with files")
		local.Git(t, "push", "-u", "origin", "main", "--force")

		client := remote.Client(t)
		writer, _ := createWriterFromHead(ctx, t, client, local)

		fileName := "tobeupdated.txt"
		updatedContent := []byte("Updated content")
		commitMsg := "Update test file"

		// Verify blob hash before update
		oldBlobHash := local.Git(t, "rev-parse", "HEAD:"+fileName)

		blobHash, err := writer.UpdateBlob(ctx, fileName, updatedContent)
		s.NoError(err)
		s.NotNil(blobHash)
		s.NotEqual(oldBlobHash, blobHash.String())

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		content, err := os.ReadFile(filepath.Join(local.Path, fileName))
		s.NoError(err)
		s.Equal(updatedContent, content)

		verifyCommitAuthorship(t, local)

		// Verify initial file was preserved
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
		s.NoError(err)
		s.NotEqual(updatedContent, otherContent)
	})

	s.Run("update nested", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		remote, _ := s.CreateTestRepo()
		local := remote.Local(t)

		// Create and commit initial file plus nested file to be updated
		local.CreateFile(t, "initial.txt", "initial content")
		local.CreateDirPath(t, "dir/subdir")
		local.CreateFile(t, "dir/subdir/tobeupdated.txt", "original nested content")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Initial commit with nested file")
		local.Git(t, "push", "-u", "origin", "main", "--force")

		client := remote.Client(t)
		writer, _ := createWriterFromHead(ctx, t, client, local)

		nestedPath := "dir/subdir/tobeupdated.txt"
		updatedContent := []byte("Updated nested content")
		commitMsg := "Update nested file"

		blobHash, err := writer.UpdateBlob(ctx, nestedPath, updatedContent)
		s.NoError(err)
		s.NotNil(blobHash)

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		content, err := os.ReadFile(filepath.Join(local.Path, nestedPath))
		s.NoError(err)
		s.Equal(updatedContent, content)

		verifyCommitAuthorship(t, local)

		// Verify initial file was preserved
		otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
		s.NoError(err)
		s.NotEqual(updatedContent, otherContent)
	})

	s.Run("update missing", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		client, _, local := s.GitServer.TestRepo(t)

		// Create and commit initial file
		local.CreateFile(t, "initial.txt", "initial content")
		local.Git(t, "add", "initial.txt")
		local.Git(t, "commit", "-m", "Initial commit")
		local.Git(t, "push", "-u", "origin", "main", "--force")

		writer, _ := createWriterFromHead(ctx, t, client, local)

		_, err := writer.UpdateBlob(ctx, "nonexistent.txt", []byte("should fail"))
		s.Error(err)
		s.ErrorIs(err, nanogit.ErrObjectNotFound)
	})

	// DELETE BLOB OPERATIONS
	s.Run("delete file", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		remote, _ := s.CreateTestRepo()
		local := remote.Local(t)

		// Create and commit initial files
		local.CreateFile(t, "initial.txt", "initial content")
		local.CreateFile(t, "tobedeleted.txt", "content to be deleted")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Initial commit with files")
		local.Git(t, "push", "-u", "origin", "main", "--force")

		client := remote.Client(t)
		writer, _ := createWriterFromHead(ctx, t, client, local)

		fileName := "tobedeleted.txt"
		commitMsg := "Delete file"

		treeHash, err := writer.DeleteBlob(ctx, fileName)
		s.NoError(err)
		s.NotNil(treeHash)

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		// Verify deleted file no longer exists
		_, err = os.Stat(filepath.Join(local.Path, fileName))
		s.Error(err)
		s.True(os.IsNotExist(err))

		// Verify initial file was preserved
		initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
		s.NoError(err)
		s.Equal([]byte("initial content"), initialContent)

		verifyCommitAuthorship(t, local)
	})

	s.Run("delete nested", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		remote, _ := s.CreateTestRepo()
		local := remote.Local(t)

		// Create and commit initial files and nested file
		local.CreateFile(t, "initial.txt", "initial content")
		local.CreateDirPath(t, "dir/subdir")
		local.CreateFile(t, "dir/subdir/tobedeleted.txt", "nested content to be deleted")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Initial commit with nested file")
		local.Git(t, "push", "-u", "origin", "main", "--force")

		client := remote.Client(t)
		writer, _ := createWriterFromHead(ctx, t, client, local)

		nestedPath := "dir/subdir/tobedeleted.txt"
		commitMsg := "Delete nested file"

		treeHash, err := writer.DeleteBlob(ctx, nestedPath)
		s.NoError(err)
		s.NotNil(treeHash)

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		// Verify nested file was deleted
		_, err = os.Stat(filepath.Join(local.Path, nestedPath))
		s.Error(err)
		s.True(os.IsNotExist(err))

		// Verify initial file was preserved
		initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
		s.NoError(err)
		s.Equal([]byte("initial content"), initialContent)

		verifyCommitAuthorship(t, local)
	})

	s.Run("delete missing", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		client, _, local := s.GitServer.TestRepo(t)
		writer, _ := createWriterFromHead(ctx, t, client, local)

		_, err := writer.DeleteBlob(ctx, "nonexistent.txt")
		s.Error(err)
		s.ErrorIs(err, nanogit.ErrObjectNotFound)
	})

	s.Run("preserve others", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Setting up remote repository")
		client, _, local := s.GitServer.TestRepo(t)

		// Create and commit multiple files in same directory
		local.CreateFile(t, "initial.txt", "initial content")
		local.CreateDirPath(t, "shared")
		local.CreateFile(t, "shared/tobedeleted.txt", "content to be deleted")
		local.CreateFile(t, "shared/tobepreserved.txt", "content to be preserved")
		local.Git(t, "add", ".")
		local.Git(t, "commit", "-m", "Initial commit with shared directory")
		local.Git(t, "push", "-u", "origin", "main", "--force")

		writer, _ := createWriterFromHead(ctx, t, client, local)

		deletePath := "shared/tobedeleted.txt"
		preservePath := "shared/tobepreserved.txt"
		commitMsg := "Delete one file from shared directory"

		treeHash, err := writer.DeleteBlob(ctx, deletePath)
		s.NoError(err)
		s.NotNil(treeHash)

		commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
		s.NoError(err)
		s.NotNil(commit)

		err = writer.Push(ctx)
		s.NoError(err)

		// Verify results
		local.Git(t, "pull")
		s.Equal(commit.Hash.String(), local.Git(t, "rev-parse", "HEAD"))

		// Verify deleted file no longer exists
		_, err = os.Stat(filepath.Join(local.Path, deletePath))
		s.Error(err)
		s.True(os.IsNotExist(err))

		// Verify preserved file still exists in same directory
		preservedContent, err := os.ReadFile(filepath.Join(local.Path, preservePath))
		s.NoError(err)
		s.Equal([]byte("content to be preserved"), preservedContent)

		// Verify initial file was preserved
		initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
		s.NoError(err)
		s.Equal([]byte("initial content"), initialContent)

		verifyCommitAuthorship(t, local)
	})
}

// The remaining complex tests (DeleteTree operations and multi-commit scenarios)
// are candidates for future refactoring to suite methods. For now, we focus on the basic
// blob operations that have been successfully converted to IntegrationTestSuite methods.

// TODO: add the preview scenarios complete one
