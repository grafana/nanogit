//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/suite"
)

// WriterTestSuite contains tests for writer operations
type WriterTestSuite struct {
	helpers.IntegrationTestSuite
}

// TestCreateBlobWithNewFile tests creating a blob with a new file
func (s *WriterTestSuite) TestCreateBlobWithNewFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial file
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.Git(s.T(), "add", "initial.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

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

	s.Logger.Info("Getting the current commit hash for the branch")
	currentHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	// Verify nothing to push before creating blob
	err = writer.Push(ctx)
	s.Error(err)
	s.ErrorIs(err, nanogit.ErrNothingToPush)

	// Verify nothing to commit before creating blob
	_, err = writer.Commit(ctx, "Add new file", author, committer)
	s.Error(err)
	s.ErrorIs(err, nanogit.ErrNothingToCommit)

	// Verify blob exists before creating it
	exists, err := writer.BlobExists(ctx, "new.txt")
	s.NoError(err)
	s.False(exists)

	_, err = writer.CreateBlob(ctx, "new.txt", newContent)
	s.NoError(err)
	commit, err := writer.Commit(ctx, "Add new file", author, committer)
	s.NoError(err)
	s.NotNil(commit)

	// Verify blob exists after creating it
	exists, err = writer.BlobExists(ctx, "new.txt")
	s.NoError(err)
	s.True(exists)

	s.Logger.Info("Pushing the commit")
	err = writer.Push(ctx)
	s.NoError(err)

	local.Git(s.T(), "pull")
	s.Equal(commit.Hash.String(), local.Git(s.T(), "rev-parse", "refs/heads/main"))

	s.Logger.Info("Verifying file content")
	content, err := os.ReadFile(filepath.Join(local.Path, "new.txt"))
	s.NoError(err)
	s.Equal(newContent, content)

	s.Logger.Info("Verifying commit message")
	commitMsg := local.Git(s.T(), "log", "-1", "--pretty=%B")
	s.Equal("Add new file", strings.TrimSpace(commitMsg))

	s.Logger.Info("Verifying author")
	commitAuthor := local.Git(s.T(), "log", "-1", "--pretty=%an <%ae>")
	s.Equal("Test Author <test@example.com>", strings.TrimSpace(commitAuthor))

	s.Logger.Info("Verifying committer")
	commitCommitter := local.Git(s.T(), "log", "-1", "--pretty=%cn <%ce>")
	s.Equal("Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

	s.Logger.Info("Verifying the ref was updated")
	hashAfterCommit := local.Git(s.T(), "rev-parse", "refs/heads/main")
	s.NotEqual(currentHash.String(), hashAfterCommit)

	s.Logger.Info("Verifying the initial file was not removed")
	otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.NotEqual(newContent, otherContent)
}

// TestCreateBlobWithNestedPath tests creating a blob with a nested path
func (s *WriterTestSuite) TestCreateBlobWithNestedPath() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial file
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.Git(s.T(), "add", "initial.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

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

	s.Logger.Info("Getting the current commit hash for the branch")
	currentHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	// Verify blob does not exist before creating it
	exists, err := writer.BlobExists(ctx, "dir/subdir/file.txt")
	s.NoError(err)
	s.False(exists)

	// Verify tree does not exist before creating it
	_, err = writer.GetTree(ctx, "dir")
	var pathNotFoundErr *nanogit.PathNotFoundError
	s.Error(err)
	s.ErrorAs(err, &pathNotFoundErr)
	s.Equal("dir", pathNotFoundErr.Path)

	_, err = writer.CreateBlob(ctx, "dir/subdir/file.txt", nestedContent)
	s.NoError(err)

	// Verify blob exists after creating it
	exists, err = writer.BlobExists(ctx, "dir/subdir/file.txt")
	s.NoError(err)
	s.True(exists)

	// Verify tree exists after creating it
	tree, err := writer.GetTree(ctx, "dir")
	s.NoError(err)
	s.NotNil(tree)
	s.Equal(1, len(tree.Entries))
	s.Equal("subdir", tree.Entries[0].Name)
	s.Equal(protocol.ObjectTypeTree, tree.Entries[0].Type)
	s.Equal(uint32(0o40000), tree.Entries[0].Mode)

	commit, err := writer.Commit(ctx, "Add nested file", author, committer)
	s.NoError(err)
	s.NotNil(commit)
	err = writer.Push(ctx)
	s.NoError(err)

	// Verify tree for subdir
	subdirTree, err := writer.GetTree(ctx, "dir/subdir")
	s.NoError(err)
	s.NotNil(subdirTree)
	s.Equal(1, len(subdirTree.Entries))
	s.Equal("file.txt", subdirTree.Entries[0].Name)
	s.Equal(protocol.ObjectTypeBlob, subdirTree.Entries[0].Type)
	s.Equal(uint32(0o100644), subdirTree.Entries[0].Mode)

	s.Logger.Info("Verifying using Git CLI")
	local.Git(s.T(), "pull")

	s.Logger.Info("Verifying commit hash")
	s.Equal(commit.Hash.String(), local.Git(s.T(), "rev-parse", "refs/heads/main"))

	s.Logger.Info("Verifying directory structure")
	dirInfo, err := os.Stat(filepath.Join(local.Path, "dir"))
	s.NoError(err)
	s.True(dirInfo.IsDir())

	subdirInfo, err := os.Stat(filepath.Join(local.Path, "dir/subdir"))
	s.NoError(err)
	s.True(subdirInfo.IsDir())

	s.Logger.Info("Verifying file content")
	content, err := os.ReadFile(filepath.Join(local.Path, "dir/subdir/file.txt"))
	s.NoError(err)
	s.Equal(nestedContent, content)

	s.Logger.Info("Verifying the initial file was not removed")
	otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.NotEqual(nestedContent, otherContent)

	s.Logger.Info("Verifying author")
	commitAuthor := local.Git(s.T(), "log", "-1", "--pretty=%an <%ae>")
	s.Equal("Test Author <test@example.com>", strings.TrimSpace(commitAuthor))

	s.Logger.Info("Verifying committer")
	commitCommitter := local.Git(s.T(), "log", "-1", "--pretty=%cn <%ce>")
	s.Equal("Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

	s.Logger.Info("Verifying the ref was updated")
	hashAfterCommit := local.Git(s.T(), "rev-parse", "refs/heads/main")
	s.NotEqual(currentHash.String(), hashAfterCommit)
}

// TestCreateBlobWithInvalidRef tests creating a blob with an invalid ref
func (s *WriterTestSuite) TestCreateBlobWithInvalidRef() {
	s.T().Parallel()

	remote, _ := s.CreateTestRepo()
	client := remote.Client(s.T())

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	_, err := client.NewStagedWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent", Hash: hash.Zero})
	s.Error(err)
	s.ErrorIs(err, nanogit.ErrObjectNotFound)
}

// TestUpdateBlobWithExistingFile tests updating a blob with an existing file
func (s *WriterTestSuite) TestUpdateBlobWithExistingFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial file plus file to be updated
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.CreateFile(s.T(), "tobeupdated.txt", "original content")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with files")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "refs/heads/main"))
	s.NoError(err)
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating ref writer")
	client := remote.Client(s.T())
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Verifying blob hash before update")
	oldBlobHash := local.Git(s.T(), "rev-parse", "HEAD:tobeupdated.txt")
	s.Logger.Info("Old blob hash", "hash", oldBlobHash)

	s.Logger.Info("Updating file content")
	updatedContent := []byte("Updated content")
	blobHash, err := writer.UpdateBlob(ctx, "tobeupdated.txt", updatedContent)
	s.NoError(err)
	s.NotNil(blobHash)

	s.Logger.Info("Blob hash changed", "old_hash", oldBlobHash, "new_hash", blobHash.String())
	s.NotEqual(oldBlobHash, blobHash.String())

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

	s.Logger.Info("Committing changes", "previous_commit", ref.Hash.String(), "ref", ref.Name)
	commit, err := writer.Commit(ctx, "Update test file", author, committer)
	s.NoError(err)
	s.NotNil(commit)

	s.Logger.Info("Pushing changes", "blob_hash", blobHash.String(), "commit", commit.Hash.String(), "previous_commit", ref.Hash.String(), "ref", ref.Name)
	err = writer.Push(ctx)
	s.NoError(err)

	s.Logger.Info("Pulling latest changes")
	local.Git(s.T(), "pull")

	s.Logger.Info("Verifying commit hash")
	s.Equal(commit.Hash.String(), local.Git(s.T(), "rev-parse", "HEAD"))

	s.Logger.Info("Verifying updated file content")
	content, err := os.ReadFile(filepath.Join(local.Path, "tobeupdated.txt"))
	s.NoError(err)
	s.Equal(updatedContent, content)

	s.Logger.Info("Verifying author and committer")
	commitAuthor := local.Git(s.T(), "log", "-1", "--pretty=%an <%ae>")
	s.Equal("Test Author <test@example.com>", strings.TrimSpace(commitAuthor))
	commitCommitter := local.Git(s.T(), "log", "-1", "--pretty=%cn <%ce>")
	s.Equal("Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))

	s.Logger.Info("Verifying initial file was preserved")
	otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.NotEqual(updatedContent, otherContent)
}

// TestUpdateBlobWithNestedFile tests updating a blob with a nested file
func (s *WriterTestSuite) TestUpdateBlobWithNestedFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial file plus nested file to be updated
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.CreateDirPath(s.T(), "dir/subdir")
	local.CreateFile(s.T(), "dir/subdir/tobeupdated.txt", "original nested content")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with nested file")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "refs/heads/main"))
	s.NoError(err)
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating ref writer")
	client := remote.Client(s.T())
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Updating nested file content")
	updatedContent := []byte("Updated nested content")
	blobHash, err := writer.UpdateBlob(ctx, "dir/subdir/tobeupdated.txt", updatedContent)
	s.NoError(err)
	s.NotNil(blobHash)

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

	s.Logger.Info("Committing changes", "previous_commit", ref.Hash.String(), "ref", ref.Name)
	commit, err := writer.Commit(ctx, "Update nested file", author, committer)
	s.NoError(err)
	s.NotNil(commit)

	s.Logger.Info("Pushing changes", "blob_hash", blobHash.String(), "commit", commit.Hash.String(), "previous_commit", ref.Hash.String(), "ref", ref.Name)
	err = writer.Push(ctx)
	s.NoError(err)

	s.Logger.Info("Pulling latest changes")
	local.Git(s.T(), "pull")

	s.Logger.Info("Verifying commit hash")
	s.Equal(commit.Hash.String(), local.Git(s.T(), "rev-parse", "HEAD"))

	s.Logger.Info("Verifying file content was updated")
	content, err := os.ReadFile(filepath.Join(local.Path, "dir/subdir/tobeupdated.txt"))
	s.NoError(err)
	s.Equal(updatedContent, content)

	s.Logger.Info("Verifying initial file was preserved")
	otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.NotEqual(updatedContent, otherContent)

	s.Logger.Info("Verifying author")
	commitAuthor := local.Git(s.T(), "log", "-1", "--pretty=%an <%ae>")
	s.Equal("Test Author <test@example.com>", strings.TrimSpace(commitAuthor))

	s.Logger.Info("Verifying committer")
	commitCommitter := local.Git(s.T(), "log", "-1", "--pretty=%cn <%ce>")
	s.Equal("Test Committer <test@example.com>", strings.TrimSpace(commitCommitter))
}

// TestUpdateBlobWithNonexistentFile tests updating a blob with a nonexistent file (should error)
func (s *WriterTestSuite) TestUpdateBlobWithNonexistentFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial file
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.Git(s.T(), "add", "initial.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHashStr := local.Git(s.T(), "rev-parse", "refs/heads/main")
	currentHash, err := hash.FromHex(currentHashStr)
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating a writer")
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Trying to update a nonexistent file")
	_, err = writer.UpdateBlob(ctx, "nonexistent.txt", []byte("should fail"))
	s.Error(err)
	s.ErrorIs(err, nanogit.ErrObjectNotFound)
}

// TestDeleteBlobWithExistingFile tests deleting an existing blob
func (s *WriterTestSuite) TestDeleteBlobWithExistingFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial files
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.CreateFile(s.T(), "tobedeleted.txt", "content to be deleted")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with files")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHashStr := local.Git(s.T(), "rev-parse", "refs/heads/main")
	currentHash, err := hash.FromHex(currentHashStr)
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating ref writer")
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Deleting existing file")
	treeHash, err := writer.DeleteBlob(ctx, "tobedeleted.txt")
	s.NoError(err)
	s.NotNil(treeHash)

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

	s.Logger.Info("Committing deletion")
	commit, err := writer.Commit(ctx, "Delete file", author, committer)
	s.NoError(err)
	s.NotNil(commit)

	s.Logger.Info("Pushing changes")
	err = writer.Push(ctx)
	s.NoError(err)

	s.Logger.Info("Pulling latest changes")
	local.Git(s.T(), "pull")

	s.Logger.Info("Verifying deleted file no longer exists")
	_, err = os.Stat(filepath.Join(local.Path, "tobedeleted.txt"))
	s.Error(err)
	s.True(os.IsNotExist(err))

	s.Logger.Info("Verifying initial file was preserved")
	initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.Equal([]byte("initial content"), initialContent)

	s.Logger.Info("Verifying commit hash")
	finalHash := local.Git(s.T(), "rev-parse", "HEAD")
	s.Equal(commit.Hash.String(), finalHash)
}

// TestDeleteBlobWithNestedFile tests deleting a nested blob
func (s *WriterTestSuite) TestDeleteBlobWithNestedFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial files and nested file
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.CreateDirPath(s.T(), "dir/subdir")
	local.CreateFile(s.T(), "dir/subdir/tobedeleted.txt", "nested content to be deleted")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with nested file")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHashStr := local.Git(s.T(), "rev-parse", "refs/heads/main")
	currentHash, err := hash.FromHex(currentHashStr)
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating ref writer")
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Deleting nested file")
	treeHash, err := writer.DeleteBlob(ctx, "dir/subdir/tobedeleted.txt")
	s.NoError(err)
	s.NotNil(treeHash)

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

	s.Logger.Info("Committing nested file deletion")
	commit, err := writer.Commit(ctx, "Delete nested file", author, committer)
	s.NoError(err)
	s.NotNil(commit)

	s.Logger.Info("Pushing changes")
	err = writer.Push(ctx)
	s.NoError(err)

	s.Logger.Info("Pulling latest changes")
	local.Git(s.T(), "pull")

	s.Logger.Info("Verifying nested file was deleted")
	_, err = os.Stat(filepath.Join(local.Path, "dir/subdir/tobedeleted.txt"))
	s.Error(err)
	s.True(os.IsNotExist(err))

	s.Logger.Info("Verifying initial file was preserved")
	initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.Equal([]byte("initial content"), initialContent)

	s.Logger.Info("Verifying commit hash")
	finalHash := local.Git(s.T(), "rev-parse", "HEAD")
	s.Equal(commit.Hash.String(), finalHash)
}

// TestDeleteBlobWithNonexistentFile tests deleting a nonexistent blob (should error)
func (s *WriterTestSuite) TestDeleteBlobWithNonexistentFile() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit initial file
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.Git(s.T(), "add", "initial.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHashStr := local.Git(s.T(), "rev-parse", "refs/heads/main")
	currentHash, err := hash.FromHex(currentHashStr)
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating ref writer")
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Trying to delete nonexistent file")
	_, err = writer.DeleteBlob(ctx, "nonexistent.txt")
	s.Error(err)
	s.ErrorIs(err, nanogit.ErrObjectNotFound)
}

// TestDeleteBlobPreservesOtherFiles tests that deleting a blob preserves other files in the same directory
func (s *WriterTestSuite) TestDeleteBlobPreservesOtherFiles() {
	s.T().Parallel()

	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create and commit multiple files in same directory
	local.CreateFile(s.T(), "initial.txt", "initial content")
	local.CreateDirPath(s.T(), "shared")
	local.CreateFile(s.T(), "shared/tobedeleted.txt", "content to be deleted")
	local.CreateFile(s.T(), "shared/tobepreserved.txt", "content to be preserved")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with shared directory")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	s.Logger.Info("Getting current ref")
	currentHashStr := local.Git(s.T(), "rev-parse", "refs/heads/main")
	currentHash, err := hash.FromHex(currentHashStr)
	s.NoError(err)

	client := remote.Client(s.T())
	ref := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: currentHash,
	}

	s.Logger.Info("Creating ref writer")
	writer, err := client.NewStagedWriter(ctx, ref)
	s.NoError(err)

	s.Logger.Info("Deleting one file from shared directory")
	treeHash, err := writer.DeleteBlob(ctx, "shared/tobedeleted.txt")
	s.NoError(err)
	s.NotNil(treeHash)

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

	s.Logger.Info("Committing selective deletion")
	commit, err := writer.Commit(ctx, "Delete one file from shared directory", author, committer)
	s.NoError(err)
	s.NotNil(commit)

	s.Logger.Info("Pushing changes")
	err = writer.Push(ctx)
	s.NoError(err)

	s.Logger.Info("Pulling latest changes")
	local.Git(s.T(), "pull")

	s.Logger.Info("Verifying deleted file no longer exists")
	_, err = os.Stat(filepath.Join(local.Path, "shared/tobedeleted.txt"))
	s.Error(err)
	s.True(os.IsNotExist(err))

	s.Logger.Info("Verifying preserved file still exists in same directory")
	preservedContent, err := os.ReadFile(filepath.Join(local.Path, "shared/tobepreserved.txt"))
	s.NoError(err)
	s.Equal([]byte("content to be preserved"), preservedContent)

	s.Logger.Info("Verifying initial file was preserved")
	initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
	s.NoError(err)
	s.Equal([]byte("initial content"), initialContent)

	s.Logger.Info("Verifying commit hash")
	finalHash := local.Git(s.T(), "rev-parse", "HEAD")
	s.Equal(commit.Hash.String(), finalHash)
}

// The remaining complex tests (DeleteTree operations and multi-commit scenarios)
// are candidates for future refactoring to suite methods. For now, we focus on the basic
// blob operations that have been successfully converted to WriterTestSuite methods.

// TestWriterTestSuite runs the writer test suite
func TestWriterTestSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}
