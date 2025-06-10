package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
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
	verifyCommitAuthorship := func(local *helpers.LocalGitRepo) {
		commitAuthor := local.Git("log", "-1", "--pretty=%an <%ae>")
		Expect(strings.TrimSpace(commitAuthor)).To(Equal("Test Author <test@example.com>"))

		commitCommitter := local.Git("log", "-1", "--pretty=%cn <%ce>")
		Expect(strings.TrimSpace(commitCommitter)).To(Equal("Test Committer <test@example.com>"))
	}

	// Helper to create writer from current HEAD
	createWriterFromHead := func(ctx context.Context, client nanogit.Client, local *helpers.LocalGitRepo) (nanogit.StagedWriter, *hash.Hash) {
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

			writer, currentHash := createWriterFromHead(context.Background(), client, local)

			newContent := []byte("new content")
			fileName := "new.txt"
			commitMsg := "Add new file"

			// Verify empty state before creating blob
			err := writer.Push(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToPush))

			_, err = writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrNothingToCommit))

			exists, err := writer.BlobExists(context.Background(), fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Create blob and commit
			_, err = writer.CreateBlob(context.Background(), fileName, newContent)
			Expect(err).NotTo(HaveOccurred())

			exists, err = writer.BlobExists(context.Background(), fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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

		It("should create a nested file", func() {
			client, _, local, _ := QuickSetup()

			writer, currentHash := createWriterFromHead(context.Background(), client, local)
			nestedContent := []byte("nested content")
			nestedPath := "dir/subdir/file.txt"
			commitMsg := "Add nested file"

			// Verify nested path doesn't exist
			exists, err := writer.BlobExists(context.Background(), nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			_, err = writer.GetTree(context.Background(), "dir")
			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(pathNotFoundErr))
			Expect(err.(*nanogit.PathNotFoundError).Path).To(Equal("dir"))

			// Create nested blob
			_, err = writer.CreateBlob(context.Background(), nestedPath, nestedContent)
			Expect(err).NotTo(HaveOccurred())

			exists, err = writer.BlobExists(context.Background(), nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify tree structure created
			tree, err := writer.GetTree(context.Background(), "dir")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())
			Expect(tree.Entries).To(HaveLen(1))
			Expect(tree.Entries[0].Name).To(Equal("subdir"))
			Expect(tree.Entries[0].Type).To(Equal(protocol.ObjectTypeTree))
			Expect(tree.Entries[0].Mode).To(Equal(uint32(0o40000)))

			subdirTree, err := writer.GetTree(context.Background(), "dir/subdir")
			Expect(err).NotTo(HaveOccurred())
			Expect(subdirTree).NotTo(BeNil())
			Expect(subdirTree.Entries).To(HaveLen(1))
			Expect(subdirTree.Entries[0].Name).To(Equal("file.txt"))
			Expect(subdirTree.Entries[0].Type).To(Equal(protocol.ObjectTypeBlob))
			Expect(subdirTree.Entries[0].Mode).To(Equal(uint32(0o100644)))

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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

			_, err := client.NewStagedWriter(context.Background(), nanogit.Ref{Name: "refs/heads/nonexistent", Hash: hash.Zero})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(nanogit.ErrObjectNotFound))
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

			writer, _ := createWriterFromHead(context.Background(), client, local)

			fileName := "tobeupdated.txt"
			updatedContent := []byte("Updated content")
			commitMsg := "Update test file"

			// Verify blob hash before update
			oldBlobHash := local.Git("rev-parse", "HEAD:"+fileName)

			blobHash, err := writer.UpdateBlob(context.Background(), fileName, updatedContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())
			Expect(blobHash.String()).NotTo(Equal(oldBlobHash))

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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

			writer, _ := createWriterFromHead(context.Background(), client, local)

			nestedPath := "dir/subdir/tobeupdated.txt"
			updatedContent := []byte("Updated nested content")
			commitMsg := "Update nested file"

			blobHash, err := writer.UpdateBlob(context.Background(), nestedPath, updatedContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobHash).NotTo(BeNil())

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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

			writer, _ := createWriterFromHead(context.Background(), client, local)

			_, err := writer.UpdateBlob(context.Background(), "nonexistent.txt", []byte("should fail"))
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

			writer, _ := createWriterFromHead(context.Background(), client, local)

			fileName := "tobedeleted.txt"
			commitMsg := "Delete file"

			treeHash, err := writer.DeleteBlob(context.Background(), fileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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

			writer, _ := createWriterFromHead(context.Background(), client, local)

			nestedPath := "dir/subdir/tobedeleted.txt"
			commitMsg := "Delete nested file"

			treeHash, err := writer.DeleteBlob(context.Background(), nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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

		It("should handle deleting missing file", func() {
			client, _, local, _ := QuickSetup()
			writer, _ := createWriterFromHead(context.Background(), client, local)

			_, err := writer.DeleteBlob(context.Background(), "nonexistent.txt")
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

			writer, _ := createWriterFromHead(context.Background(), client, local)

			deletePath := "shared/tobedeleted.txt"
			preservePath := "shared/tobepreserved.txt"
			commitMsg := "Delete one file from shared directory"

			treeHash, err := writer.DeleteBlob(context.Background(), deletePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			commit, err := writer.Commit(context.Background(), commitMsg, testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			err = writer.Push(context.Background())
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
})

// The remaining complex tests (DeleteTree operations and multi-commit scenarios)
// are candidates for future refactoring to suite methods. For now, we focus on the basic
// blob operations that have been successfully converted to IntegrationTestSuite methods.

// TODO: add the preview scenarios complete one
