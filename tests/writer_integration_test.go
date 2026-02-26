package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Writer Operations", func() {
	var (
		testAuthor = nanogit.Author{
			Name:  "Test Author",
			Email: "test@example.com",
			Time:  time.Now(),
		}
		testCommitter = nanogit.Committer{
			Name:  "Test Committer",
			Email: "test@example.com",
			Time:  time.Now(),
		}
	)

	// Helper to verify author and committer in commit
	verifyCommitAuthorship := func(local *LocalGitRepo) {
		commitAuthor := local.Git("log", "-1", "--pretty=%an <%ae>")
		Expect(strings.TrimSpace(commitAuthor)).To(Equal("Test Author <test@example.com>"))

		commitCommitter := local.Git("log", "-1", "--pretty=%cn <%ce>")
		Expect(strings.TrimSpace(commitCommitter)).To(Equal("Test Committer <test@example.com>"))
	}

	// Helper to create writer from current HEAD
	createWriterFromHead := func(ctx context.Context, client nanogit.Client, local *LocalGitRepo) (nanogit.StagedWriter, *hash.Hash) {
		currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
		Expect(err).NotTo(HaveOccurred())

		ref := nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		}

		writer, err := client.NewStagedWriter(ctx, ref)
		Expect(err).NotTo(HaveOccurred())

		return writer, &currentHash
	}

	Context("Create Blob Operations", func() {
		It("should create a new file", func() {
			client, _, local, _ := QuickSetup()

			writer, currentHash := createWriterFromHead(ctx, client, local)

			newContent := []byte("new content")
			fileName := "new.txt"
			commitMsg := "Add new file"

			// Verify empty state before creating blob
			err := writer.Push(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToPush))

			_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToCommit))

			exists, err := writer.BlobExists(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Create blob and commit
			_, err = writer.CreateBlob(ctx, fileName, newContent)
			Expect(err).NotTo(HaveOccurred())

			exists, err = writer.BlobExists(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull", "origin", "main")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "refs/heads/main")))

			content, err := os.ReadFile(filepath.Join(local.Path, fileName))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(newContent))

			actualCommitMsg := local.Git("log", "-1", "--pretty=%B")
			Expect(strings.TrimSpace(actualCommitMsg)).To(Equal(commitMsg))

			verifyCommitAuthorship(local)

			hashAfterCommit := local.Git("rev-parse", "refs/heads/main")
			Expect(hashAfterCommit).NotTo(Equal(currentHash.String()))

			// Verify initial file preserved
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(newContent))
		})
		It("should fail to create a file if it already exists", func() {
			client, _, local, _ := QuickSetup()
			fileName := "test.txt"
			originalContent := []byte("original content")
			// Ensure the file exists in the repo
			local.CreateFile(fileName, string(originalContent))
			local.Git("add", fileName)
			local.Git("commit", "-m", "Add test.txt")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)
			// Try to create the same file again
			_, err := writer.CreateBlob(ctx, fileName, []byte("new content"))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectAlreadyExists))
		})

		It("should create a nested file", func() {
			client, _, local, _ := QuickSetup()

			writer, currentHash := createWriterFromHead(ctx, client, local)
			nestedContent := []byte("nested content")
			nestedPath := "dir/subdir/file.txt"
			commitMsg := "Add nested file"

			// Verify nested path doesn't exist
			exists, err := writer.BlobExists(ctx, nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			_, err = writer.GetTree(ctx, "dir")
			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(pathNotFoundErr))
			if ok := errors.As(err, &pathNotFoundErr); ok {
				Expect(pathNotFoundErr.Path).To(Equal("dir"))
			} else {
				Fail(fmt.Sprintf("Expected PathNotFoundError, got %T", err))
			}

			// Create nested blob
			_, err = writer.CreateBlob(ctx, nestedPath, nestedContent)
			Expect(err).NotTo(HaveOccurred())

			exists, err = writer.BlobExists(ctx, nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify tree structure created
			tree, err := writer.GetTree(ctx, "dir")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())
			Expect(tree.Entries).To(HaveLen(1))
			Expect(tree.Entries[0].Name).To(Equal("subdir"))
			Expect(tree.Entries[0].Type).To(Equal(protocol.ObjectTypeTree))
			Expect(tree.Entries[0].Mode).To(Equal(uint32(0o40000)))

			subdirTree, err := writer.GetTree(ctx, "dir/subdir")
			Expect(err).NotTo(HaveOccurred())
			Expect(subdirTree).NotTo(BeNil())
			Expect(subdirTree.Entries).To(HaveLen(1))
			Expect(subdirTree.Entries[0].Name).To(Equal("file.txt"))
			Expect(subdirTree.Entries[0].Type).To(Equal(protocol.ObjectTypeBlob))
			Expect(subdirTree.Entries[0].Mode).To(Equal(uint32(0o100644)))

			// Verify GetTree fails for a file
			_, err = writer.GetTree(ctx, "dir/subdir/file.txt")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrUnexpectedObjectType)).To(BeTrue())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "refs/heads/main")))

			// Verify directory structure
			dirInfo, err := os.Stat(filepath.Join(local.Path, "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(dirInfo.IsDir()).To(BeTrue())

			subdirInfo, err := os.Stat(filepath.Join(local.Path, "dir/subdir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(subdirInfo.IsDir()).To(BeTrue())

			content, err := os.ReadFile(filepath.Join(local.Path, nestedPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(nestedContent))

			verifyCommitAuthorship(local)

			hashAfterCommit := local.Git("rev-parse", "refs/heads/main")
			Expect(hashAfterCommit).NotTo(Equal(currentHash.String()))

			// Verify initial file preserved
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(nestedContent))
		})

		It("should handle invalid ref", func() {
			client, _, _, _ := QuickSetup()

			_, err := client.NewStagedWriter(ctx, nanogit.Ref{Name: "refs/heads/nonexistent", Hash: hash.Zero})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))
		})
		It("should return ErrEmptyPath for BlobExists and CreateBlob if path is empty", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			exists, err := writer.BlobExists(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyPath))
			Expect(exists).To(BeFalse())

			_, err = writer.CreateBlob(ctx, "", []byte("test"))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyPath))
		})
	})

	Context("Update Blob Operations", func() {
		It("should update an existing file", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial file plus file to be updated
			local.CreateFile("initial.txt", "initial content")
			local.CreateFile("tobeupdated.txt", "original content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with files")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			fileName := "tobeupdated.txt"
			updatedContent := []byte("Updated content")
			commitMsg := "Update test file"

			// Verify blob hash before update
			oldBlobHash := local.Git("rev-parse", "HEAD:"+fileName)

			blobHash, err := writer.UpdateBlob(ctx, fileName, updatedContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())
			Expect(blobHash.String()).NotTo(Equal(oldBlobHash))

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			content, err := os.ReadFile(filepath.Join(local.Path, fileName))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(updatedContent))

			verifyCommitAuthorship(local)

			// Verify initial file was preserved
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(updatedContent))
		})
		It("should fail with ErrEmptyPath if the path is empty", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.UpdateBlob(ctx, "", []byte("some content"))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrEmptyPath)).To(BeTrue())
		})

		It("should update a nested file", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial file plus nested file to be updated
			local.CreateFile("initial.txt", "initial content")
			local.CreateDirPath("dir/subdir")
			local.CreateFile("dir/subdir/tobeupdated.txt", "original nested content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with nested file")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			nestedPath := "dir/subdir/tobeupdated.txt"
			updatedContent := []byte("Updated nested content")
			commitMsg := "Update nested file"

			blobHash, err := writer.UpdateBlob(ctx, nestedPath, updatedContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			content, err := os.ReadFile(filepath.Join(local.Path, nestedPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(updatedContent))

			verifyCommitAuthorship(local)

			// Verify initial file was preserved
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(updatedContent))
		})

		It("should handle updating missing file", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial file
			local.CreateFile("initial.txt", "initial content")
			local.Git("add", "initial.txt")
			local.Git("commit", "-m", "Initial commit")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.UpdateBlob(ctx, "nonexistent.txt", []byte("should fail"))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))
		})
	})

	Context("Delete Blob Operations", func() {
		It("should delete an existing file", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial files
			local.CreateFile("initial.txt", "initial content")
			local.CreateFile("tobedeleted.txt", "content to be deleted")
			local.Git("add", ".")
			local.Git("branch", "-M", "main")
			local.Git("commit", "-m", "Initial commit with files")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			fileName := "tobedeleted.txt"
			commitMsg := "Delete file"

			treeHash, err := writer.DeleteBlob(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			// Verify deleted file no longer exists
			_, err = os.Stat(filepath.Join(local.Path, fileName))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Verify initial file was preserved
			initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(initialContent).To(Equal([]byte("initial content")))

			verifyCommitAuthorship(local)
		})

		It("should delete a nested file", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial files and nested file
			local.CreateFile("initial.txt", "initial content")
			local.CreateDirPath("dir/subdir")
			local.CreateFile("dir/subdir/tobedeleted.txt", "nested content to be deleted")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with nested file")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			nestedPath := "dir/subdir/tobedeleted.txt"
			commitMsg := "Delete nested file"

			treeHash, err := writer.DeleteBlob(ctx, nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			// Verify nested file was deleted
			_, err = os.Stat(filepath.Join(local.Path, nestedPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Verify initial file was preserved
			initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(initialContent).To(Equal([]byte("initial content")))

			verifyCommitAuthorship(local)
		})
		It("should fail with ErrEmptyPath if the path is empty", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.DeleteBlob(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrEmptyPath)).To(BeTrue())
		})

		It("should fail with ErrUnexpectedObjectType if trying to delete a tree as a blob", func() {
			client, _, local, _ := QuickSetup()
			local.CreateDirPath("dir")
			local.CreateFile("dir/file.txt", "file content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Add directory and file")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			// Try to delete the directory as if it were a blob
			_, err := writer.DeleteBlob(ctx, "dir")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrUnexpectedObjectType)).To(BeTrue())
		})

		It("should handle deleting missing file", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.DeleteBlob(ctx, "nonexistent.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))
		})

		It("should preserve other files when deleting from shared directory", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit multiple files in same directory
			local.CreateFile("initial.txt", "initial content")
			local.CreateDirPath("shared")
			local.CreateFile("shared/tobedeleted.txt", "content to be deleted")
			local.CreateFile("shared/tobepreserved.txt", "content to be preserved")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with shared directory")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			deletePath := "shared/tobedeleted.txt"
			preservePath := "shared/tobepreserved.txt"
			commitMsg := "Delete one file from shared directory"

			treeHash, err := writer.DeleteBlob(ctx, deletePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			// Verify deleted file no longer exists
			_, err = os.Stat(filepath.Join(local.Path, deletePath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Verify preserved file still exists in same directory
			preservedContent, err := os.ReadFile(filepath.Join(local.Path, preservePath))
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedContent).To(Equal([]byte("content to be preserved")))

			// Verify initial file was preserved
			initialContent, err := os.ReadFile(filepath.Join(local.Path, "initial.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(initialContent).To(Equal([]byte("initial content")))

			verifyCommitAuthorship(local)
		})
	})

	Context("Move Blob Operations", func() {
		It("should move a file to a new location", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial files
			local.CreateFile("source.txt", "content to move")
			local.CreateFile("other.txt", "other content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with files")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "source.txt"
			destPath := "destination.txt"
			commitMsg := "Move file"

			// Verify source exists and destination doesn't
			exists, err := writer.BlobExists(ctx, srcPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = writer.BlobExists(ctx, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Perform move
			blobHash, err := writer.MoveBlob(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())

			// Verify source no longer exists and destination exists
			exists, err = writer.BlobExists(ctx, srcPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			exists, err = writer.BlobExists(ctx, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			// Source file should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Destination file should exist with original content
			content, err := os.ReadFile(filepath.Join(local.Path, destPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal([]byte("content to move")))

			// Other file should be preserved
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "other.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).To(Equal([]byte("other content")))

			verifyCommitAuthorship(local)
		})

		It("should move a file to a nested directory", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial file
			local.CreateFile("source.txt", "nested content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "source.txt"
			destPath := "dir/subdir/moved.txt"
			commitMsg := "Move file to nested directory"

			// Perform move
			blobHash, err := writer.MoveBlob(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())

			// Verify tree structure was created
			tree, err := writer.GetTree(ctx, "dir")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())
			Expect(tree.Entries).To(HaveLen(1))
			Expect(tree.Entries[0].Name).To(Equal("subdir"))
			Expect(tree.Entries[0].Type).To(Equal(protocol.ObjectTypeTree))

			subdirTree, err := writer.GetTree(ctx, "dir/subdir")
			Expect(err).NotTo(HaveOccurred())
			Expect(subdirTree).NotTo(BeNil())
			Expect(subdirTree.Entries).To(HaveLen(1))
			Expect(subdirTree.Entries[0].Name).To(Equal("moved.txt"))
			Expect(subdirTree.Entries[0].Type).To(Equal(protocol.ObjectTypeBlob))

			_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Source should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Destination should exist with correct content
			content, err := os.ReadFile(filepath.Join(local.Path, destPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal([]byte("nested content")))

			// Directory structure should exist
			dirInfo, err := os.Stat(filepath.Join(local.Path, "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(dirInfo.IsDir()).To(BeTrue())

			subdirInfo, err := os.Stat(filepath.Join(local.Path, "dir/subdir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(subdirInfo.IsDir()).To(BeTrue())
		})

		It("should move a file from nested to root directory", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit nested file
			local.CreateDirPath("dir/subdir")
			local.CreateFile("dir/subdir/nested.txt", "nested file content")
			local.CreateFile("preserved.txt", "preserved content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with nested file")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "dir/subdir/nested.txt"
			destPath := "moved-to-root.txt"
			commitMsg := "Move nested file to root"

			// Perform move
			blobHash, err := writer.MoveBlob(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())

			_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Source should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Destination should exist with correct content
			content, err := os.ReadFile(filepath.Join(local.Path, destPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal([]byte("nested file content")))

			// Preserved file should still exist
			preservedContent, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedContent).To(Equal([]byte("preserved content")))
		})

		It("should fail to move if source doesn't exist", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveBlob(ctx, "nonexistent.txt", "destination.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))
		})

		It("should fail to move if destination already exists", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit files
			local.CreateFile("source.txt", "source content")
			local.CreateFile("destination.txt", "destination content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveBlob(ctx, "source.txt", "destination.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectAlreadyExists))
		})

		It("should fail to move if source and destination are the same", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit file
			local.CreateFile("file.txt", "content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveBlob(ctx, "file.txt", "file.txt")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source and destination paths are the same"))
		})

		It("should fail to move if source is empty path", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveBlob(ctx, "", "destination.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyPath))
		})

		It("should fail to move if destination is empty path", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveBlob(ctx, "source.txt", "")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyPath))
		})

		It("should fail to move if source is a tree", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit directory
			local.CreateDirPath("dir")
			local.CreateFile("dir/file.txt", "content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveBlob(ctx, "dir", "moved-dir")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrUnexpectedObjectType)).To(BeTrue())
		})

		It("should move multiple files in same commit", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial files
			local.CreateFile("file1.txt", "content 1")
			local.CreateFile("file2.txt", "content 2")
			local.CreateFile("preserved.txt", "preserved")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			// Move both files
			_, err := writer.MoveBlob(ctx, "file1.txt", "moved1.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.MoveBlob(ctx, "file2.txt", "dir/moved2.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Move multiple files", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Original files should not exist
			_, err = os.Stat(filepath.Join(local.Path, "file1.txt"))
			Expect(os.IsNotExist(err)).To(BeTrue())
			_, err = os.Stat(filepath.Join(local.Path, "file2.txt"))
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Moved files should exist with correct content
			content1, err := os.ReadFile(filepath.Join(local.Path, "moved1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("content 1")))

			content2, err := os.ReadFile(filepath.Join(local.Path, "dir/moved2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("content 2")))

			// Preserved file should still exist
			preservedContent, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedContent).To(Equal([]byte("preserved")))
		})
	})

	Context("Move Tree Operations", func() {
		It("should move a directory to a new location", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial directory structure
			local.CreateDirPath("source-dir")
			local.CreateFile("source-dir/file1.txt", "content 1")
			local.CreateFile("source-dir/file2.txt", "content 2")
			local.CreateFile("other.txt", "other content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with directory")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "source-dir"
			destPath := "dest-dir"
			commitMsg := "Move directory"

			// Verify source tree exists
			srcTree, err := writer.GetTree(ctx, srcPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(srcTree.Entries).To(HaveLen(2))

			// Perform move
			treeHash, err := writer.MoveTree(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			// Verify source no longer exists and destination exists
			_, err = writer.GetTree(ctx, srcPath)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))

			destTree, err := writer.GetTree(ctx, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(destTree.Entries).To(HaveLen(2))
			Expect(destTree.Hash).To(Equal(treeHash))

			// Verify files exist at new location
			exists, err := writer.BlobExists(ctx, "dest-dir/file1.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = writer.BlobExists(ctx, "dest-dir/file2.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify files don't exist at old location
			exists, err = writer.BlobExists(ctx, "source-dir/file1.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Source directory should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Destination directory should exist with correct files
			destInfo, err := os.Stat(filepath.Join(local.Path, destPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(destInfo.IsDir()).To(BeTrue())

			content1, err := os.ReadFile(filepath.Join(local.Path, "dest-dir/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("content 1")))

			content2, err := os.ReadFile(filepath.Join(local.Path, "dest-dir/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("content 2")))

			// Other file should be preserved
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "other.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).To(Equal([]byte("other content")))

			verifyCommitAuthorship(local)
		})

		It("should move a nested directory structure", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit nested directory structure
			local.CreateDirPath("parent/source-dir/subdir")
			local.CreateFile("parent/source-dir/file1.txt", "nested content 1")
			local.CreateFile("parent/source-dir/subdir/file2.txt", "nested content 2")
			local.CreateFile("parent/preserved.txt", "preserved content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with nested structure")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "parent/source-dir"
			destPath := "parent/moved-dir"
			commitMsg := "Move nested directory"

			// Perform move
			treeHash, err := writer.MoveTree(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			// Verify tree structure at destination
			destTree, err := writer.GetTree(ctx, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(destTree.Entries).To(HaveLen(2)) // file1.txt and subdir

			subdirTree, err := writer.GetTree(ctx, "parent/moved-dir/subdir")
			Expect(err).NotTo(HaveOccurred())
			Expect(subdirTree.Entries).To(HaveLen(1)) // file2.txt

			_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Source should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Destination should exist with correct structure
			content1, err := os.ReadFile(filepath.Join(local.Path, "parent/moved-dir/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("nested content 1")))

			content2, err := os.ReadFile(filepath.Join(local.Path, "parent/moved-dir/subdir/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("nested content 2")))

			// Preserved file should still exist
			preservedContent, err := os.ReadFile(filepath.Join(local.Path, "parent/preserved.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedContent).To(Equal([]byte("preserved content")))
		})

		It("should move a directory to a new nested location", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit directory at root
			local.CreateDirPath("root-dir")
			local.CreateFile("root-dir/file.txt", "root file content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "root-dir"
			destPath := "new/nested/location"
			commitMsg := "Move directory to nested location"

			// Perform move
			treeHash, err := writer.MoveTree(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			// Verify nested tree structure was created
			newTree, err := writer.GetTree(ctx, "new")
			Expect(err).NotTo(HaveOccurred())
			Expect(newTree.Entries).To(HaveLen(1))
			Expect(newTree.Entries[0].Name).To(Equal("nested"))

			nestedTree, err := writer.GetTree(ctx, "new/nested")
			Expect(err).NotTo(HaveOccurred())
			Expect(nestedTree.Entries).To(HaveLen(1))
			Expect(nestedTree.Entries[0].Name).To(Equal("location"))

			locationTree, err := writer.GetTree(ctx, destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(locationTree.Entries).To(HaveLen(1))
			Expect(locationTree.Entries[0].Name).To(Equal("file.txt"))

			_, err = writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Source should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Destination should exist with correct content
			content, err := os.ReadFile(filepath.Join(local.Path, "new/nested/location/file.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal([]byte("root file content")))

			// Directory structure should exist
			for _, dirPath := range []string{"new", "new/nested", "new/nested/location"} {
				dirInfo, err := os.Stat(filepath.Join(local.Path, dirPath))
				Expect(err).NotTo(HaveOccurred())
				Expect(dirInfo.IsDir()).To(BeTrue())
			}
		})

		It("should fail to move if source doesn't exist", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveTree(ctx, "nonexistent", "destination")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))
		})

		It("should fail to move if destination already exists", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit directories
			local.CreateDirPath("source")
			local.CreateFile("source/file.txt", "source content")
			local.CreateDirPath("destination")
			local.CreateFile("destination/file.txt", "destination content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveTree(ctx, "source", "destination")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectAlreadyExists))
		})

		It("should fail to move if source and destination are the same", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit directory
			local.CreateDirPath("directory")
			local.CreateFile("directory/file.txt", "content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveTree(ctx, "directory", "directory")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source and destination paths are the same"))
		})

		It("should fail to move if source is empty path", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveTree(ctx, "", "destination")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyPath))
		})

		It("should fail to move if destination is empty path", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveTree(ctx, "source", "")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyPath))
		})

		It("should fail to move if source is a blob", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit file
			local.CreateFile("file.txt", "content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.MoveTree(ctx, "file.txt", "moved-file")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrUnexpectedObjectType)).To(BeTrue())
		})

		It("should move multiple directories in same commit", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit multiple directories
			local.CreateDirPath("dir1")
			local.CreateFile("dir1/file1.txt", "content 1")
			local.CreateDirPath("dir2")
			local.CreateFile("dir2/file2.txt", "content 2")
			local.CreateFile("preserved.txt", "preserved")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			// Move both directories
			_, err := writer.MoveTree(ctx, "dir1", "moved1")
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.MoveTree(ctx, "dir2", "nested/moved2")
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Move multiple directories", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Original directories should not exist
			_, err = os.Stat(filepath.Join(local.Path, "dir1"))
			Expect(os.IsNotExist(err)).To(BeTrue())
			_, err = os.Stat(filepath.Join(local.Path, "dir2"))
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Moved directories should exist with correct content
			content1, err := os.ReadFile(filepath.Join(local.Path, "moved1/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("content 1")))

			content2, err := os.ReadFile(filepath.Join(local.Path, "nested/moved2/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("content 2")))

			// Preserved file should still exist
			preservedContent, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedContent).To(Equal([]byte("preserved")))
		})

		It("should move directory with complex nested structure", func() {
			client, _, local, _ := QuickSetup()

			// Create complex directory structure
			local.CreateDirPath("complex/sub1/deep1")
			local.CreateDirPath("complex/sub1/deep2")
			local.CreateDirPath("complex/sub2")
			local.CreateFile("complex/root-file.txt", "root content")
			local.CreateFile("complex/sub1/sub1-file.txt", "sub1 content")
			local.CreateFile("complex/sub1/deep1/deep1-file.txt", "deep1 content")
			local.CreateFile("complex/sub1/deep2/deep2-file.txt", "deep2 content")
			local.CreateFile("complex/sub2/sub2-file.txt", "sub2 content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Create complex structure")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			srcPath := "complex"
			destPath := "moved-complex"

			// Perform move
			_, err := writer.MoveTree(ctx, srcPath, destPath)
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Move complex directory structure", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			local.Git("pull")

			// Source should not exist
			_, err = os.Stat(filepath.Join(local.Path, srcPath))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			// All files should exist at destination with correct content
			testFiles := map[string]string{
				"moved-complex/root-file.txt":             "root content",
				"moved-complex/sub1/sub1-file.txt":        "sub1 content",
				"moved-complex/sub1/deep1/deep1-file.txt": "deep1 content",
				"moved-complex/sub1/deep2/deep2-file.txt": "deep2 content",
				"moved-complex/sub2/sub2-file.txt":        "sub2 content",
			}

			for filePath, expectedContent := range testFiles {
				content, err := os.ReadFile(filepath.Join(local.Path, filePath))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal(expectedContent))
			}

			// All directories should exist
			testDirs := []string{
				"moved-complex",
				"moved-complex/sub1",
				"moved-complex/sub1/deep1",
				"moved-complex/sub1/deep2",
				"moved-complex/sub2",
			}

			for _, dirPath := range testDirs {
				dirInfo, err := os.Stat(filepath.Join(local.Path, dirPath))
				Expect(err).NotTo(HaveOccurred())
				Expect(dirInfo.IsDir()).To(BeTrue())
			}
		})
	})

	Context("Delete Tree Operations", func() {
		It("should delete the entire tree with empty path", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial files and directories
			local.CreateFile("file1.txt", "file 1 content")
			local.CreateDirPath("dir/subdir")
			local.CreateFile("dir/file2.txt", "file 2 content")
			local.CreateFile("dir/subdir/file3.txt", "file 3 content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with files and directories")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			commitMsg := "Delete entire tree (empty path)"

			// Delete the entire tree by passing empty path
			treeHash, err := writer.DeleteTree(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull changes to local repo
			local.Git("pull")

			// The repository should now be empty (no files or directories except .git)
			files, err := os.ReadDir(local.Path)
			Expect(err).NotTo(HaveOccurred())

			// Only .git directory should remain
			var nonGitFiles []string
			for _, f := range files {
				if f.Name() != ".git" {
					nonGitFiles = append(nonGitFiles, f.Name())
				}
			}
			Expect(nonGitFiles).To(BeEmpty())

			verifyCommitAuthorship(local)
		})
		It("should delete the entire tree with dot path", func() {
			client, _, local, _ := QuickSetup()

			// Create and commit initial files and directories
			local.CreateFile("file1.txt", "file 1 content")
			local.CreateDirPath("dir/subdir")
			local.CreateFile("dir/file2.txt", "file 2 content")
			local.CreateFile("dir/subdir/file3.txt", "file 3 content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with files and directories")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			commitMsg := "Delete entire tree (dot path)"

			// Delete the entire tree by passing dot path
			treeHash, err := writer.DeleteTree(ctx, ".")
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull changes to local repo
			local.Git("pull")

			// The repository should now be empty (no files or directories except .git)
			files, err := os.ReadDir(local.Path)
			Expect(err).NotTo(HaveOccurred())

			// Only .git directory should remain
			var nonGitFiles []string
			for _, f := range files {
				if f.Name() != ".git" {
					nonGitFiles = append(nonGitFiles, f.Name())
				}
			}
			Expect(nonGitFiles).To(BeEmpty())

			verifyCommitAuthorship(local)
		})
		It("should delete a tree structure containing files", func() {
			client, _, local, _ := QuickSetup()
			initCommitFile := "test.txt"

			By("Creating directory with files to be deleted")
			dir1Content := []byte("Directory 1 file content")
			file1Content := []byte("File 1 content")
			file2Content := []byte("File 2 content")
			local.CreateDirPath("toberemoved")
			local.CreateFile("toberemoved/file1.txt", string(file1Content))
			local.CreateFile("toberemoved/file2.txt", string(file2Content))
			local.CreateFile("preserved.txt", string(dir1Content))

			By("Adding and committing the directory with files")
			local.Git("add", ".")
			local.Git("commit", "-m", "Add directory with files to be deleted")
			local.Git("push")

			By("Verifying directory and files exist before deletion")
			_, err := os.Stat(filepath.Join(local.Path, "toberemoved"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file2.txt"))
			Expect(err).NotTo(HaveOccurred())

			By("Getting current ref")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			By("Creating ref writer")
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the entire directory")
			treeHash, err := writer.DeleteTree(ctx, "toberemoved")
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			By("Committing directory deletion")
			commit, err := writer.Commit(ctx, "Delete entire directory", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			By("Pushing changes")
			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			By("Pulling latest changes")
			local.Git("pull")

			By("Verifying commit hash")
			Expect(commit.Hash.String()).To(Equal(local.Git("rev-parse", "HEAD")))

			By("Verifying directory was completely deleted")
			_, err = os.Stat(filepath.Join(local.Path, "toberemoved"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file1.txt"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
			_, err = os.Stat(filepath.Join(local.Path, "toberemoved/file2.txt"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("Verifying other files were preserved")
			preservedContent, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedContent).To(Equal(dir1Content))

			otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(file1Content))
			Expect(otherContent).NotTo(Equal(file2Content))

			By("Verifying commit message")
			commitMsg := local.Git("log", "-1", "--pretty=%B")
			Expect(strings.TrimSpace(commitMsg)).To(Equal("Delete entire directory"))
		})

		It("should delete a tree structure with nested directories", func() {
			client, _, local, _ := QuickSetup()
			ctx := ctx

			preservedContent := []byte("Preserved content")
			nested1Content := []byte("Nested 1 content")
			nested2Content := []byte("Nested 2 content")
			deepContent := []byte("Deep nested content")

			local.CreateDirPath("toberemoved/subdir1")
			local.CreateDirPath("toberemoved/subdir2/deep")
			local.CreateFile("preserved.txt", string(preservedContent))
			local.CreateFile("toberemoved/file.txt", string(nested1Content))
			local.CreateFile("toberemoved/subdir1/nested.txt", string(nested2Content))
			local.CreateFile("toberemoved/subdir2/deep/deep.txt", string(deepContent))

			local.Git("add", ".")
			local.Git("commit", "-m", "Add nested directory structure")
			local.Git("push")

			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			treeHash, err := writer.DeleteTree(ctx, "toberemoved")
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

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

			commit, err := writer.Commit(ctx, "Delete nested directory structure", author, committer)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			local.Git("pull")

			_, err = os.Stat(filepath.Join(local.Path, "toberemoved"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			content, err := os.ReadFile(filepath.Join(local.Path, "preserved.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(preservedContent))

			otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(nested1Content))
		})

		It("should fail to delete tree of missing directory", func() {
			client, _, _, _ := QuickSetup()
			ctx := ctx

			ref, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())

			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.DeleteTree(ctx, "nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&nanogit.PathNotFoundError{Path: "nonexistent"}))
		})

		It("should fail to delete tree when path is a file", func() {
			client, _, local, _ := QuickSetup()
			ctx := ctx

			fileContent := []byte("This is a file, not a directory")
			local.CreateFile("testfile.txt", string(fileContent))
			local.Git("add", "testfile.txt")
			local.Git("commit", "-m", "Add test file")
			local.Git("push")

			fileHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:testfile.txt"))
			Expect(err).NotTo(HaveOccurred())

			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.DeleteTree(ctx, "testfile.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&nanogit.UnexpectedObjectTypeError{
				ObjectID:     fileHash,
				ExpectedType: protocol.ObjectTypeTree,
				ActualType:   protocol.ObjectTypeBlob,
			}))
		})

		It("should delete a tree structure with subdirectory only", func() {
			client, _, local, _ := QuickSetup()
			ctx := ctx

			parentFile := []byte("Parent file")
			subdir1File := []byte("Subdirectory 1 file")
			subdir2File := []byte("Subdirectory 2 file")

			local.CreateDirPath("parent/subdir1")
			local.CreateDirPath("parent/subdir2")
			local.CreateFile("parent/parentfile.txt", string(parentFile))
			local.CreateFile("parent/subdir1/file1.txt", string(subdir1File))
			local.CreateFile("parent/subdir2/file2.txt", string(subdir2File))

			By("Adding and committing the directory structure")
			local.Git("add", ".")
			local.Git("commit", "-m", "Add parent with subdirectories")
			local.Git("push")

			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			treeHash, err := writer.DeleteTree(ctx, "parent/subdir1")
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(ctx, "Delete only subdir1", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			local.Git("pull")

			_, err = os.Stat(filepath.Join(local.Path, "parent/subdir1"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			parentContent, err := os.ReadFile(filepath.Join(local.Path, "parent/parentfile.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(parentContent).To(Equal(parentFile))

			subdir2Content, err := os.ReadFile(filepath.Join(local.Path, "parent/subdir2/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(subdir2Content).To(Equal(subdir2File))

			otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(subdir1File))
		})

		Context("Complex scenarios", func() {
			It("should create multiple files in different directories with one commit", func() {
				client, _, local, _ := QuickSetup()
				ctx := ctx

				currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
				Expect(err).NotTo(HaveOccurred())
				ref := nanogit.Ref{
					Name: "refs/heads/main",
					Hash: currentHash,
				}

				writer, err := client.NewStagedWriter(ctx, ref)
				Expect(err).NotTo(HaveOccurred())

				// Create multiple files in the same directory
				config1Content := []byte(`{"database": {"host": "localhost", "port": 5432}}`)
				config2Content := []byte(`{"api": {"timeout": 30, "retries": 3}}`)
				config3Content := []byte(`{"logging": {"level": "info", "output": "stdout"}}`)

				_, err = writer.CreateBlob(ctx, "config/database.json", config1Content)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "config/api.json", config2Content)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "config/logging.json", config3Content)
				Expect(err).NotTo(HaveOccurred())

				// Create files in different subdirectories
				dataContent := []byte(`{"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}`)
				schemaContent := []byte(`{"type": "object", "properties": {"name": {"type": "string"}}}`)

				_, err = writer.CreateBlob(ctx, "data/users.json", dataContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "schemas/user.json", schemaContent)
				Expect(err).NotTo(HaveOccurred())

				commit, err := writer.Commit(ctx, "Add configuration and data files", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit).NotTo(BeNil())

				err = writer.Push(ctx)
				Expect(err).NotTo(HaveOccurred())

				local.Git("pull")

				// Verify directory structure
				configDir, err := os.Stat(filepath.Join(local.Path, "config"))
				Expect(err).NotTo(HaveOccurred())
				Expect(configDir.IsDir()).To(BeTrue())

				dataDir, err := os.Stat(filepath.Join(local.Path, "data"))
				Expect(err).NotTo(HaveOccurred())
				Expect(dataDir.IsDir()).To(BeTrue())

				schemasDir, err := os.Stat(filepath.Join(local.Path, "schemas"))
				Expect(err).NotTo(HaveOccurred())
				Expect(schemasDir.IsDir()).To(BeTrue())

				// Verify all files exist with correct content
				content1, err := os.ReadFile(filepath.Join(local.Path, "config/database.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content1).To(Equal(config1Content))

				content2, err := os.ReadFile(filepath.Join(local.Path, "config/api.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content2).To(Equal(config2Content))

				content3, err := os.ReadFile(filepath.Join(local.Path, "config/logging.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content3).To(Equal(config3Content))

				dataFileContent, err := os.ReadFile(filepath.Join(local.Path, "data/users.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(dataFileContent).To(Equal(dataContent))

				schemaFileContent, err := os.ReadFile(filepath.Join(local.Path, "schemas/user.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(schemaFileContent).To(Equal(schemaContent))

				// Verify original file is preserved
				originalContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(originalContent).NotTo(BeEmpty())

				// Verify commit details
				finalHash := local.Git("rev-parse", "HEAD")
				Expect(finalHash).To(Equal(commit.Hash.String()))

				commitMsg := local.Git("log", "-1", "--pretty=%B")
				Expect(commitMsg).To(Equal("Add configuration and data files"))
			})

			It("should create multiple files in different directories across multiple commits", func() {
				client, _, local, _ := QuickSetup()
				ctx := ctx

				currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
				Expect(err).NotTo(HaveOccurred())
				ref := nanogit.Ref{
					Name: "refs/heads/main",
					Hash: currentHash,
				}

				writer, err := client.NewStagedWriter(ctx, ref)
				Expect(err).NotTo(HaveOccurred())

				// First commit: Create config files
				dbConfigContent := []byte(`{"host": "localhost", "port": 5432, "database": "myapp"}`)
				apiConfigContent := []byte(`{"baseUrl": "https://api.example.com", "timeout": 30}`)

				_, err = writer.CreateBlob(ctx, "config/database.json", dbConfigContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "config/api.json", apiConfigContent)
				Expect(err).NotTo(HaveOccurred())

				commit1, err := writer.Commit(ctx, "Add database and API configuration", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit1).NotTo(BeNil())

				// Second commit: Create documentation files
				readmeContent := []byte(`# My Application\n\nThis is a sample application.`)
				apiDocsContent := []byte(`# API Documentation\n\n## Endpoints\n\n- GET /users`)

				_, err = writer.CreateBlob(ctx, "docs/README.md", readmeContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "docs/api.md", apiDocsContent)
				Expect(err).NotTo(HaveOccurred())

				commit2, err := writer.Commit(ctx, "Add documentation files", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit2).NotTo(BeNil())

				// Third commit: Create test and data files
				testDataContent := []byte(`{"testUsers": [{"id": 1, "name": "Test User"}]}`)
				schemaContent := []byte(`{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`)

				_, err = writer.CreateBlob(ctx, "tests/data/users.json", testDataContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "schemas/user.json", schemaContent)
				Expect(err).NotTo(HaveOccurred())

				commit3, err := writer.Commit(ctx, "Add test data and schema files", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit3).NotTo(BeNil())

				// Verify commit chain before push
				Expect(currentHash).To(Equal(commit1.Parent))
				Expect(commit1.Hash).To(Equal(commit2.Parent))
				Expect(commit2.Hash).To(Equal(commit3.Parent))

				// Push all commits at once
				err = writer.Push(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Pull and verify
				local.Git("pull")

				// Verify final commit hash
				finalHash := local.Git("rev-parse", "HEAD")
				Expect(finalHash).To(Equal(commit3.Hash.String()))

				// Verify all commits exist in history
				commitHistory := local.Git("log", "--oneline", "--format=%H %s")

				Expect(commitHistory).To(ContainSubstring(commit1.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit2.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit3.Hash.String()))

				Expect(commitHistory).To(ContainSubstring("Add database and API configuration"))
				Expect(commitHistory).To(ContainSubstring("Add documentation files"))
				Expect(commitHistory).To(ContainSubstring("Add test data and schema files"))

				// Verify directory structure
				configDir, err := os.Stat(filepath.Join(local.Path, "config"))
				Expect(err).NotTo(HaveOccurred())
				Expect(configDir.IsDir()).To(BeTrue())

				docsDir, err := os.Stat(filepath.Join(local.Path, "docs"))
				Expect(err).NotTo(HaveOccurred())
				Expect(docsDir.IsDir()).To(BeTrue())

				testsDir, err := os.Stat(filepath.Join(local.Path, "tests"))
				Expect(err).NotTo(HaveOccurred())
				Expect(testsDir.IsDir()).To(BeTrue())

				testDataDir, err := os.Stat(filepath.Join(local.Path, "tests/data"))
				Expect(err).NotTo(HaveOccurred())
				Expect(testDataDir.IsDir()).To(BeTrue())

				schemasDir, err := os.Stat(filepath.Join(local.Path, "schemas"))
				Expect(err).NotTo(HaveOccurred())
				Expect(schemasDir.IsDir()).To(BeTrue())

				// Verify all files exist with correct content
				dbConfig, err := os.ReadFile(filepath.Join(local.Path, "config/database.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(dbConfig).To(Equal(dbConfigContent))

				apiConfig, err := os.ReadFile(filepath.Join(local.Path, "config/api.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(apiConfig).To(Equal(apiConfigContent))

				readme, err := os.ReadFile(filepath.Join(local.Path, "docs/README.md"))
				Expect(err).NotTo(HaveOccurred())
				Expect(readme).To(Equal(readmeContent))

				apiDocs, err := os.ReadFile(filepath.Join(local.Path, "docs/api.md"))
				Expect(err).NotTo(HaveOccurred())
				Expect(apiDocs).To(Equal(apiDocsContent))

				testData, err := os.ReadFile(filepath.Join(local.Path, "tests/data/users.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(testData).To(Equal(testDataContent))

				schema, err := os.ReadFile(filepath.Join(local.Path, "schemas/user.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(schema).To(Equal(schemaContent))

				// Verify original file is preserved
				originalContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(originalContent).NotTo(BeEmpty())

				// Verify individual commits show correct files
				commit1Files := strings.TrimSpace(local.Git("ls-tree", "--name-only", "-r", commit1.Hash.String()))
				Expect(commit1Files).To(ContainSubstring("config/database.json"))
				Expect(commit1Files).To(ContainSubstring("config/api.json"))
				Expect(commit1Files).To(ContainSubstring("test.txt"))
				Expect(commit1Files).NotTo(ContainSubstring("docs/README.md"))

				commit2Files := strings.TrimSpace(local.Git("ls-tree", "--name-only", "-r", commit2.Hash.String()))
				Expect(commit2Files).To(ContainSubstring("config/database.json"))
				Expect(commit2Files).To(ContainSubstring("config/api.json"))
				Expect(commit2Files).To(ContainSubstring("docs/README.md"))
				Expect(commit2Files).To(ContainSubstring("docs/api.md"))
				Expect(commit2Files).To(ContainSubstring("test.txt"))
				Expect(commit2Files).NotTo(ContainSubstring("tests/data/users.json"))

				commit3Files := strings.TrimSpace(local.Git("ls-tree", "--name-only", "-r", commit3.Hash.String()))
				Expect(commit3Files).To(ContainSubstring("config/database.json"))
				Expect(commit3Files).To(ContainSubstring("config/api.json"))
				Expect(commit3Files).To(ContainSubstring("docs/README.md"))
				Expect(commit3Files).To(ContainSubstring("docs/api.md"))
				Expect(commit3Files).To(ContainSubstring("tests/data/users.json"))
				Expect(commit3Files).To(ContainSubstring("schemas/user.json"))
				Expect(commit3Files).To(ContainSubstring("test.txt"))
			})

			It("should create multiple commits and all be visible after push", func() {
				client, _, local, _ := QuickSetup()
				ctx := ctx

				currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
				Expect(err).NotTo(HaveOccurred())
				ref := nanogit.Ref{
					Name: "refs/heads/main",
					Hash: currentHash,
				}

				writer, err := client.NewStagedWriter(ctx, ref)
				Expect(err).NotTo(HaveOccurred())

				// Create first commit
				_, err = writer.CreateBlob(ctx, "file1.txt", []byte("First file content"))
				Expect(err).NotTo(HaveOccurred())
				commit1, err := writer.Commit(ctx, "Add first file", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit1).NotTo(BeNil())

				// Create second commit
				_, err = writer.CreateBlob(ctx, "file2.txt", []byte("Second file content"))
				Expect(err).NotTo(HaveOccurred())
				commit2, err := writer.Commit(ctx, "Add second file", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit2).NotTo(BeNil())

				// Create third commit
				_, err = writer.CreateBlob(ctx, "file3.txt", []byte("Third file content"))
				Expect(err).NotTo(HaveOccurred())
				commit3, err := writer.Commit(ctx, "Add third file", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit3).NotTo(BeNil())

				// Verify commit chain is correct
				Expect(currentHash).To(Equal(commit1.Parent))
				Expect(commit1.Hash).To(Equal(commit2.Parent))
				Expect(commit2.Hash).To(Equal(commit3.Parent))

				// Push all commits
				err = writer.Push(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Pull and verify
				local.Git("pull")

				// Verify final commit hash
				finalHash := local.Git("rev-parse", "HEAD")
				Expect(finalHash).To(Equal(commit3.Hash.String()))

				// Verify all three commits exist in history
				commitHistory := local.Git("log", "--oneline", "--format=%H %s")

				// Should contain all three commits
				Expect(commitHistory).To(ContainSubstring(commit1.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit2.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit3.Hash.String()))

				// Verify commit messages are correct
				Expect(commitHistory).To(ContainSubstring("Add first file"))
				Expect(commitHistory).To(ContainSubstring("Add second file"))
				Expect(commitHistory).To(ContainSubstring("Add third file"))

				// Verify all files exist
				content1, err := os.ReadFile(filepath.Join(local.Path, "file1.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content1).To(Equal([]byte("First file content")))

				content2, err := os.ReadFile(filepath.Join(local.Path, "file2.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content2).To(Equal([]byte("Second file content")))

				content3, err := os.ReadFile(filepath.Join(local.Path, "file3.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(content3).To(Equal([]byte("Third file content")))

				initCommitFile := "test.txt"
				// Verify original file is preserved
				otherContent, err := os.ReadFile(filepath.Join(local.Path, initCommitFile))
				Expect(err).NotTo(HaveOccurred())
				Expect(otherContent).NotTo(BeEmpty())

				// Verify individual commits are reachable
				commit1Files := strings.TrimSpace(local.Git("ls-tree", "--name-only", commit1.Hash.String()))
				Expect(commit1Files).To(ContainSubstring("file1.txt"))
				Expect(commit1Files).To(ContainSubstring(initCommitFile))
				Expect(commit1Files).NotTo(ContainSubstring("file2.txt"))
				Expect(commit1Files).NotTo(ContainSubstring("file3.txt"))

				commit2Files := strings.TrimSpace(local.Git("ls-tree", "--name-only", commit2.Hash.String()))
				Expect(commit2Files).To(ContainSubstring("file1.txt"))
				Expect(commit2Files).To(ContainSubstring("file2.txt"))
				Expect(commit2Files).To(ContainSubstring(initCommitFile))
				Expect(commit2Files).NotTo(ContainSubstring("file3.txt"))

				commit3Files := strings.TrimSpace(local.Git("ls-tree", "--name-only", commit3.Hash.String()))
				Expect(commit3Files).To(ContainSubstring("file1.txt"))
				Expect(commit3Files).To(ContainSubstring("file2.txt"))
				Expect(commit3Files).To(ContainSubstring("file3.txt"))
				Expect(commit3Files).To(ContainSubstring(initCommitFile))
			})
		})
	})
	Context("Commit Operations", func() {
		It("should fail to commit if the tree is empty", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			_, err := writer.Commit(ctx, "Empty commit", testAuthor, testCommitter)
			Expect(err).To(HaveOccurred())
		})
		It("should fail to commit if the message is empty", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			// Add a file so the tree is not empty
			_, err := writer.CreateBlob(ctx, "file.txt", []byte("content"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "", testAuthor, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrEmptyCommitMessage))
		})

		It("should fail to commit if the author is missing email", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			// Add a file so the tree is not empty
			_, err := writer.CreateBlob(ctx, "file.txt", []byte("content"))
			Expect(err).NotTo(HaveOccurred())

			author := testAuthor
			author.Email = ""
			_, err = writer.Commit(ctx, "Missing author email", author, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrInvalidAuthor))
		})

		It("should fail to commit if the author is missing name", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			// Add a file so the tree is not empty
			_, err := writer.CreateBlob(ctx, "file.txt", []byte("content"))
			Expect(err).NotTo(HaveOccurred())

			author := testAuthor
			author.Name = ""
			_, err = writer.Commit(ctx, "Missing author name", author, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrInvalidAuthor)).To(BeTrue())
		})

		It("should fail to commit if the committer is missing email", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			// Add a file so the tree is not empty
			_, err := writer.CreateBlob(ctx, "file.txt", []byte("content"))
			Expect(err).NotTo(HaveOccurred())

			committer := testCommitter
			committer.Email = ""
			_, err = writer.Commit(ctx, "Missing committer email", testAuthor, committer)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrInvalidAuthor)).To(BeTrue())
		})

		It("should fail to commit if the committer is missing name", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(ctx, client, local)

			// Add a file so the tree is not empty
			_, err := writer.CreateBlob(ctx, "file.txt", []byte("content"))
			Expect(err).NotTo(HaveOccurred())

			committer := testCommitter
			committer.Name = ""
			_, err = writer.Commit(ctx, "Missing committer name", testAuthor, committer)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrInvalidAuthor)).To(BeTrue())
		})

		It("should update only the relevant branch of the tree when updating a deep file", func() {
			client, _, local, _ := QuickSetup()
			// Create a deeper nested directory structure:
			// root/
			//   dir1/
			//     subdir1/
			//       file1.txt
			//   dir2/
			//     subdir2/
			//       file2.txt
			//   file3.txt
			local.CreateDirPath("dir1/subdir1")
			local.CreateDirPath("dir2/subdir2")
			local.CreateFile("dir1/subdir1/file1.txt", "original file1")
			local.CreateFile("dir2/subdir2/file2.txt", "original file2")
			local.CreateFile("file3.txt", "original file3")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial deep nested commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			// Get the tree hashes before update
			rootTreeHashBefore := local.Git("rev-parse", "HEAD^{tree}")
			dir1TreeHashBefore := local.Git("rev-parse", "HEAD:dir1")
			subdir1TreeHashBefore := local.Git("rev-parse", "HEAD:dir1/subdir1")
			dir2TreeHashBefore := local.Git("rev-parse", "HEAD:dir2")
			subdir2TreeHashBefore := local.Git("rev-parse", "HEAD:dir2/subdir2")

			// Update a deep file: dir1/subdir1/file1.txt
			newContent := []byte("updated file1 content")
			_, err := writer.UpdateBlob(ctx, "dir1/subdir1/file1.txt", newContent)
			Expect(err).NotTo(HaveOccurred())

			commitMsg := "Update dir1/subdir1/file1.txt"
			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull changes to local repo to verify
			local.Git("pull", "origin", "main")

			// Get the tree hashes after update
			rootTreeHashAfter := local.Git("rev-parse", "HEAD^{tree}")
			dir1TreeHashAfter := local.Git("rev-parse", "HEAD:dir1")
			subdir1TreeHashAfter := local.Git("rev-parse", "HEAD:dir1/subdir1")
			dir2TreeHashAfter := local.Git("rev-parse", "HEAD:dir2")
			subdir2TreeHashAfter := local.Git("rev-parse", "HEAD:dir2/subdir2")

			// Only the relevant branch (dir1/subdir1) should have changed
			Expect(subdir1TreeHashAfter).NotTo(Equal(subdir1TreeHashBefore))
			Expect(dir1TreeHashAfter).NotTo(Equal(dir1TreeHashBefore))
			Expect(rootTreeHashAfter).NotTo(Equal(rootTreeHashBefore))
			Expect(subdir2TreeHashAfter).To(Equal(subdir2TreeHashBefore))
			Expect(dir2TreeHashAfter).To(Equal(dir2TreeHashBefore))

			// file3.txt should be unchanged
			content, err := os.ReadFile(filepath.Join(local.Path, "file3.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("original file3"))

			// dir2/subdir2/file2.txt should be unchanged
			content, err = os.ReadFile(filepath.Join(local.Path, "dir2/subdir2/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("original file2"))

			// dir1/subdir1/file1.txt should be updated
			content, err = os.ReadFile(filepath.Join(local.Path, "dir1/subdir1/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("updated file1 content"))
		})
		It("should delete a file deep in the tree and only update relevant trees", func() {
			client, _, local, _ := QuickSetup()

			// Setup: create a deep directory structure with files
			local.CreateDirPath("dir1/subdir1")
			local.CreateDirPath("dir2/subdir2")
			local.CreateFile("dir1/subdir1/file1.txt", "original file1")
			local.CreateFile("dir2/subdir2/file2.txt", "original file2")
			local.CreateFile("file3.txt", "original file3")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with deep structure")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			// Get the tree hashes before deletion
			rootTreeHashBefore := local.Git("rev-parse", "HEAD^{tree}")
			dir1TreeHashBefore := local.Git("rev-parse", "HEAD:dir1")
			subdir1TreeHashBefore := local.Git("rev-parse", "HEAD:dir1/subdir1")
			dir2TreeHashBefore := local.Git("rev-parse", "HEAD:dir2")
			subdir2TreeHashBefore := local.Git("rev-parse", "HEAD:dir2/subdir2")

			// Delete a deep file: dir1/subdir1/file1.txt
			_, err := writer.DeleteBlob(ctx, "dir1/subdir1/file1.txt")
			Expect(err).NotTo(HaveOccurred())

			commitMsg := "Delete dir1/subdir1/file1.txt"
			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull changes to local repo to verify
			local.Git("pull", "origin", "main")

			// Get the tree hashes after deletion
			rootTreeHashAfter := local.Git("rev-parse", "HEAD^{tree}")
			dir1TreeHashAfter := local.Git("rev-parse", "HEAD:dir1")
			subdir1TreeHashAfter := local.Git("rev-parse", "HEAD:dir1/subdir1")
			dir2TreeHashAfter := local.Git("rev-parse", "HEAD:dir2")
			subdir2TreeHashAfter := local.Git("rev-parse", "HEAD:dir2/subdir2")

			// Only the relevant branch (dir1/subdir1) should have changed
			Expect(rootTreeHashAfter).NotTo(Equal(rootTreeHashBefore))
			Expect(subdir1TreeHashAfter).NotTo(Equal(subdir1TreeHashBefore))
			Expect(dir1TreeHashAfter).NotTo(Equal(dir1TreeHashBefore))
			Expect(subdir2TreeHashAfter).To(Equal(subdir2TreeHashBefore))
			Expect(dir2TreeHashAfter).To(Equal(dir2TreeHashBefore))

			// file3.txt should be unchanged
			content, err := os.ReadFile(filepath.Join(local.Path, "file3.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("original file3"))

			// dir2/subdir2/file2.txt should be unchanged
			content, err = os.ReadFile(filepath.Join(local.Path, "dir2/subdir2/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("original file2"))

			// dir1/subdir1/file1.txt should be deleted
			_, err = os.Stat(filepath.Join(local.Path, "dir1/subdir1/file1.txt"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("should create a file deep in nested directories and only update relevant trees", func() {
			client, _, local, _ := QuickSetup()

			// Setup: create a multi-level directory structure
			local.CreateDirPath("dirA/dirB")
			local.CreateDirPath("dirC/dirD")
			local.CreateFile("dirA/dirB/fileA.txt", "original A")
			local.CreateFile("dirC/dirD/fileC.txt", "original C")
			local.CreateFile("rootfile.txt", "root content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial nested structure")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			writer, _ := createWriterFromHead(ctx, client, local)

			// Get tree hashes before
			rootTreeHashBefore := local.Git("rev-parse", "HEAD^{tree}")
			dirAHashBefore := local.Git("rev-parse", "HEAD:dirA")
			dirBHashBefore := local.Git("rev-parse", "HEAD:dirA/dirB")
			dirCHashBefore := local.Git("rev-parse", "HEAD:dirC")
			dirDHashBefore := local.Git("rev-parse", "HEAD:dirC/dirD")

			// Create a new file deep in dirA/dirB
			newFilePath := "dirA/dirB/newdeep.txt"
			newContent := []byte("deep content")
			blobHash, err := writer.CreateBlob(ctx, newFilePath, newContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())

			commitMsg := "Add deep file"
			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull changes to local repo to verify
			local.Git("pull", "origin", "main")

			commitHash := local.Git("rev-parse", "HEAD")
			Expect(commitHash).To(Equal(commit.Hash.String()))

			// Get tree hashes after
			rootTreeHashAfter := local.Git("rev-parse", "HEAD^{tree}")
			dirAHashAfter := local.Git("rev-parse", "HEAD:dirA")
			dirBHashAfter := local.Git("rev-parse", "HEAD:dirA/dirB")
			dirCHashAfter := local.Git("rev-parse", "HEAD:dirC")
			dirDHashAfter := local.Git("rev-parse", "HEAD:dirC/dirD")

			// Only the relevant branch (dirA/dirB) and its parents should have changed
			Expect(rootTreeHashAfter).NotTo(Equal(rootTreeHashBefore))
			Expect(dirBHashAfter).NotTo(Equal(dirBHashBefore))
			Expect(dirAHashAfter).NotTo(Equal(dirAHashBefore))
			Expect(dirCHashAfter).To(Equal(dirCHashBefore))
			Expect(dirDHashAfter).To(Equal(dirDHashBefore))

			// The new file should exist with correct content
			content, err := os.ReadFile(filepath.Join(local.Path, newFilePath))
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(newContent))

			// Unrelated files should be unchanged
			content, err = os.ReadFile(filepath.Join(local.Path, "dirC/dirD/fileC.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("original C"))

			content, err = os.ReadFile(filepath.Join(local.Path, "dirA/dirB/fileA.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("original A"))

			content, err = os.ReadFile(filepath.Join(local.Path, "rootfile.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("root content"))
		})
	})

	Describe("Empty tree repository", func() {
		var (
			writer nanogit.StagedWriter
			local  *LocalGitRepo
		)

		BeforeEach(func() {
			var client nanogit.Client
			client, _, local, _ = QuickSetup()
			// Remove all commits from the main branch in the local repository.
			// This is done by creating an orphan branch and force-updating main to point to it.
			// This simulates an "empty" repository state for the main branch.
			// Create an empty tree object, commit it, and update the main ref.
			local.Git("checkout", "--orphan", "someorphan")
			emptyTreeHash := local.Git("mktree") // No input = empty tree
			commitHash := local.Git("commit-tree", emptyTreeHash, "-m", "Empty commit")
			local.Git("update-ref", "refs/heads/main", commitHash)
			local.Git("push", "origin", "main", "--force")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())

			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			writer, err = client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should not find any blobs in an empty repository", func() {
			exists, err := writer.BlobExists(ctx, "text.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should fail to commit if there are no staged changes", func() {
			_, err := writer.Commit(ctx, "Empty commit", testAuthor, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToCommit))
		})

		It("should fail to push if there are no staged objects", func() {
			err := writer.Push(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToPush))
		})
		It("should create a new tree when creating a file in an empty repository", func() {
			fileName := "hello.txt"
			content := []byte("Hello, world!")

			// The file should not exist yet
			exists, err := writer.BlobExists(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Create the blob
			_, err = writer.CreateBlob(ctx, fileName, content)
			Expect(err).NotTo(HaveOccurred())

			// Now the blob should exist
			exists, err = writer.BlobExists(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Commit the change
			commitMsg := "Add hello.txt to empty repo"
			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			// Push the commit
			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull and verify the file exists in the local repo
			local.Git("pull", "origin", "main")
			filePath := filepath.Join(local.Path, fileName)
			readContent, err := os.ReadFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(readContent).To(Equal(content))

			// The tree for the root should now contain the file
			treeHash := local.Git("rev-parse", "HEAD^{tree}")
			Expect(treeHash).NotTo(BeEmpty())
			lsTree := local.Git("ls-tree", "--name-only", treeHash)
			Expect(strings.TrimSpace(lsTree)).To(Equal(fileName))
		})
	})

	Describe("Empty branch repository", func() {
		var (
			writer nanogit.StagedWriter
			local  *LocalGitRepo
		)

		BeforeEach(func() {
			var client nanogit.Client
			client, _, local, _ = QuickSetup()
			// Delete the branch on the remote and locally, then recreate it as an orphan (no history).
			local.Git("checkout", "--orphan", "temp-empty-branch")
			local.Git("rm", "-rf", ".")
			local.Git("commit", "--allow-empty", "-m", "Empty commit")
			local.Git("branch", "-D", "main")
			local.Git("branch", "-m", "main")
			local.Git("push", "origin", "--force", "--set-upstream", "main")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())

			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			writer, err = client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should not find any blobs in an empty repository", func() {
			exists, err := writer.BlobExists(ctx, "text.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should fail to commit if there are no staged changes", func() {
			_, err := writer.Commit(ctx, "Empty commit", testAuthor, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToCommit))
		})

		It("should fail to push if there are no staged objects", func() {
			err := writer.Push(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToPush))
		})
		It("should create a new tree when creating a file in an empty repository", func() {
			fileName := "hello.txt"
			content := []byte("Hello, world!")

			// The file should not exist yet
			exists, err := writer.BlobExists(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Create the blob
			_, err = writer.CreateBlob(ctx, fileName, content)
			Expect(err).NotTo(HaveOccurred())

			// Now the blob should exist
			exists, err = writer.BlobExists(ctx, fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Commit the change
			commitMsg := "Add hello.txt to empty repo"
			commit, err := writer.Commit(ctx, commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			// Push the commit
			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Pull and verify the file exists in the local repo
			local.Git("pull", "origin", "main")
			filePath := filepath.Join(local.Path, fileName)
			readContent, err := os.ReadFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(readContent).To(Equal(content))

			// The tree for the root should now contain the file
			treeHash := local.Git("rev-parse", "HEAD^{tree}")
			Expect(treeHash).NotTo(BeEmpty())
			lsTree := local.Git("ls-tree", "--name-only", treeHash)
			Expect(strings.TrimSpace(lsTree)).To(Equal(fileName))
		})
	})

	Context("Tree Sorting Regression Test", func() {
		It("should create files in a directory structure that previously caused tree sorting errors", func() {
			client, _, local, _ := QuickSetup()

			// Recreate the exact tree structure from the bug report
			// Based on the repository structure that caused the "treeNotSorted" error
			treeStructure := map[string]string{
				"another-one.json":                                             `{"type": "dashboard", "title": "Another One"}`,
				"dir1/dir2/dir3/deepinside.json":                               `{"deep": "content"}`,
				"example.json":                                                 `{"example": "data"}`,
				"finaltest/folder/bounds.json":                                 `{"bounds": true}`,
				"finaltest/folder/nested/timeline.json":                        `{"timeline": "data"}`,
				"finaltest/heatmap.json":                                       `{"heatmap": "visualization"}`,
				"grafana/fix-migration-test.json":                              `{"migration": "test"}`,
				"grafana/new-dashboard-2025-06-16-YCsUH.json":                  `{"dashboard": "2025-06-16"}`,
				"grafana/new-dashboard.json":                                   `{"dashboard": "new"}`,
				"legacy/legacy-dashboard-inside.json":                          `{"legacy": "inside"}`,
				"legacy-dashboard.json":                                        `{"legacy": "root"}`,
				"one/two/three/inside.json":                                    `{"nested": "deep"}`,
				"README.md":                                                    `# Test Repository`,
				"repofolder/in-repo-folder.json":                               `{"repo": "folder"}`,
				"test2/heatmap.json":                                           `{"test2": "heatmap"}`,
				"testagain/heatmap.json":                                       `{"testagain": "heatmap"}`,
				"testloop/__folder_not_found/aei9icv4lhywwa/bounds.json":       `{"testloop": "bounds"}`,
				"testloop/__folder_not_found/bei9j85bufu2oa/timeline.json":     `{"testloop": "timeline"}`,
				"testloop/heatmap.json":                                        `{"testloop": "heatmap"}`,
				"testo2/folder/bounds.json":                                    `{"testo2": "bounds"}`,
				"testo2/folder/nested/timeline.json":                           `{"testo2": "timeline"}`,
				"testo2/heatmap.json":                                          `{"testo2": "heatmap"}`,
				"whatever/DemoFolder/DemoDeeperFolder/inside-deep-folder.json": `{"demo": "deep"}`,
				"whatever/DemoFolder/inside-demo-folder.json":                  `{"demo": "folder"}`,
				"whatever/git-sync-demo-dashboard.json":                        `{"git": "sync"}`,
			}

			// Create all files in the structure first
			for filePath, content := range treeStructure {
				local.CreateDirPath(filepath.Dir(filePath))
				local.CreateFile(filePath, content)
			}
			local.Git("add", ".")
			local.Git("commit", "-m", "Create complex tree structure")
			local.Git("branch", "-M", "robertoonboarding")
			local.Git("push", "-u", "origin", "robertoonboarding", "--force")

			// Get current ref for the robertoonboarding branch
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/robertoonboarding"))
			Expect(err).NotTo(HaveOccurred())

			ref := nanogit.Ref{
				Name: "refs/heads/robertoonboarding",
				Hash: currentHash,
			}

			// Create the staged writer
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			// This is the exact file creation that was causing the tree sorting error
			newFileName := "robertoonboarding/new-dashboard-2025-06-25-hmYyl.json"
			newFileContent := []byte(`{
				"dashboard": {
					"title": "New Dashboard 2025-06-25",
					"tags": ["robertoonboarding"],
					"timezone": "browser",
					"panels": [],
					"time": {
						"from": "now-6h",
						"to": "now"
					},
					"timepicker": {},
					"templating": {
						"list": []
					},
					"annotations": {
						"list": []
					},
					"refresh": "5s",
					"schemaVersion": 16,
					"version": 0
				}
			}`)

			// Create the problematic file in the robertoonboarding directory
			_, err = writer.CreateBlob(ctx, newFileName, newFileContent)
			Expect(err).NotTo(HaveOccurred())

			// Commit the change
			commit, err := writer.Commit(ctx, "some commit", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			// This push should NOT fail with "treeNotSorted" error anymore
			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify the file was created successfully
			local.Git("pull", "origin", "robertoonboarding")

			// Check that the new file exists
			filePath := filepath.Join(local.Path, newFileName)
			content, err := os.ReadFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal(newFileContent))

			// Verify the commit was successful
			latestHash := local.Git("rev-parse", "HEAD")
			Expect(latestHash).To(Equal(commit.Hash.String()))

			// Verify all original files are still present
			for originalPath := range treeStructure {
				_, err := os.Stat(filepath.Join(local.Path, originalPath))
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify the tree structure is correctly sorted by checking git fsck passes
			fsckOutput := local.Git("fsck", "--full")
			Expect(fsckOutput).NotTo(ContainSubstring("treeNotSorted"))
			Expect(fsckOutput).NotTo(ContainSubstring("not properly sorted"))
		})
	})
})
