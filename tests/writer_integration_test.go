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

			logger.Info("Verifying all files in directory were deleted")
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

			logger.Info("Creating nested directory structure to be deleted")
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

			logger.Info("Adding and committing the nested directory structure")
			local.Git("add", ".")
			local.Git("commit", "-m", "Add nested directory structure")
			local.Git("push")

			logger.Info("Getting current ref")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			logger.Info("Creating ref writer")
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Deleting the entire nested directory")
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

			logger.Info("Committing nested directory deletion")
			commit, err := writer.Commit(ctx, "Delete nested directory structure", author, committer)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			logger.Info("Pushing changes")
			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Pulling latest changes")
			local.Git("pull")

			logger.Info("Verifying entire directory structure was deleted")
			_, err = os.Stat(filepath.Join(local.Path, "toberemoved"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			logger.Info("Verifying preserved file still exists")
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

			logger.Info("Getting current ref")
			ref, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Creating a writer")
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Trying to delete a nonexistent directory")
			_, err = writer.DeleteTree(ctx, "nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&nanogit.PathNotFoundError{Path: "nonexistent"}))
		})

		It("should fail to delete tree when path is a file", func() {
			client, _, local, _ := QuickSetup()
			ctx := ctx

			logger.Info("Creating a file to test error case")
			fileContent := []byte("This is a file, not a directory")
			local.CreateFile("testfile.txt", string(fileContent))
			local.Git("add", "testfile.txt")
			local.Git("commit", "-m", "Add test file")
			local.Git("push")

			fileHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:testfile.txt"))
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Getting current ref")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			logger.Info("Creating a writer")
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Trying to delete a file as if it were a directory")
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

			logger.Info("Creating parent directory with subdirectories")
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

			logger.Info("Getting current ref")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "refs/heads/main"))
			Expect(err).NotTo(HaveOccurred())
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			logger.Info("Creating ref writer")
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Deleting only subdir1, leaving subdir2 and parent")
			treeHash, err := writer.DeleteTree(ctx, "parent/subdir1")
			Expect(err).NotTo(HaveOccurred())
			Expect(treeHash).NotTo(BeNil())

			logger.Info("Committing subdirectory deletion")
			commit, err := writer.Commit(ctx, "Delete only subdir1", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			logger.Info("Pushing changes")
			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Pulling latest changes")
			local.Git("pull")

			logger.Info("Verifying subdir1 was deleted")
			_, err = os.Stat(filepath.Join(local.Path, "parent/subdir1"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			logger.Info("Verifying parent directory still exists")
			parentContent, err := os.ReadFile(filepath.Join(local.Path, "parent/parentfile.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(parentContent).To(Equal(parentFile))

			logger.Info("Verifying subdir2 still exists")
			subdir2Content, err := os.ReadFile(filepath.Join(local.Path, "parent/subdir2/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(subdir2Content).To(Equal(subdir2File))

			logger.Info("Verifying other files were preserved")
			otherContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherContent).NotTo(Equal(subdir1File))
		})

		Context("Complex scenarios", func() {
			It("should create multiple files in different directories with one commit", func() {
				client, _, local, _ := QuickSetup()
				ctx := ctx

				logger.Info("Getting current ref")
				currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
				Expect(err).NotTo(HaveOccurred())
				ref := nanogit.Ref{
					Name: "refs/heads/main",
					Hash: currentHash,
				}

				logger.Info("Creating staged writer")
				writer, err := client.NewStagedWriter(ctx, ref)
				Expect(err).NotTo(HaveOccurred())

				// Create multiple files in the same directory
				logger.Info("Creating multiple JSON files in config directory")
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
				logger.Info("Creating files in different subdirectories")
				dataContent := []byte(`{"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}`)
				schemaContent := []byte(`{"type": "object", "properties": {"name": {"type": "string"}}}`)

				_, err = writer.CreateBlob(ctx, "data/users.json", dataContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "schemas/user.json", schemaContent)
				Expect(err).NotTo(HaveOccurred())

				logger.Info("Committing all files")
				commit, err := writer.Commit(ctx, "Add configuration and data files", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit).NotTo(BeNil())

				logger.Info("Pushing changes")
				err = writer.Push(ctx)
				Expect(err).NotTo(HaveOccurred())

				logger.Info("Pulling and verifying")
				local.Git("pull")

				// Verify directory structure
				logger.Info("Verifying directory structure")
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
				logger.Info("Verifying file contents")
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
				logger.Info("Verifying original file preserved")
				originalContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(originalContent).NotTo(BeEmpty())

				// Verify commit details
				logger.Info("Verifying commit details")
				finalHash := local.Git("rev-parse", "HEAD")
				Expect(finalHash).To(Equal(commit.Hash.String()))

				commitMsg := local.Git("log", "-1", "--pretty=%B")
				Expect(commitMsg).To(Equal("Add configuration and data files"))
			})

			It("should create multiple files in different directories across multiple commits", func() {
				client, _, local, _ := QuickSetup()
				ctx := ctx

				logger.Info("Getting current ref")
				currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
				Expect(err).NotTo(HaveOccurred())
				ref := nanogit.Ref{
					Name: "refs/heads/main",
					Hash: currentHash,
				}

				logger.Info("Creating staged writer")
				writer, err := client.NewStagedWriter(ctx, ref)
				Expect(err).NotTo(HaveOccurred())

				// First commit: Create config files
				logger.Info("First commit: Creating configuration files")
				dbConfigContent := []byte(`{"host": "localhost", "port": 5432, "database": "myapp"}`)
				apiConfigContent := []byte(`{"baseUrl": "https://api.example.com", "timeout": 30}`)

				_, err = writer.CreateBlob(ctx, "config/database.json", dbConfigContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "config/api.json", apiConfigContent)
				Expect(err).NotTo(HaveOccurred())

				commit1, err := writer.Commit(ctx, "Add database and API configuration", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit1).NotTo(BeNil())
				logger.Info("First commit created", "hash", commit1.Hash.String())

				// Second commit: Create documentation files
				logger.Info("Second commit: Creating documentation files")
				readmeContent := []byte(`# My Application\n\nThis is a sample application.`)
				apiDocsContent := []byte(`# API Documentation\n\n## Endpoints\n\n- GET /users`)

				_, err = writer.CreateBlob(ctx, "docs/README.md", readmeContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "docs/api.md", apiDocsContent)
				Expect(err).NotTo(HaveOccurred())

				commit2, err := writer.Commit(ctx, "Add documentation files", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit2).NotTo(BeNil())
				logger.Info("Second commit created", "hash", commit2.Hash.String(), "parent", commit2.Parent.String())

				// Third commit: Create test and data files
				logger.Info("Third commit: Creating test and data files")
				testDataContent := []byte(`{"testUsers": [{"id": 1, "name": "Test User"}]}`)
				schemaContent := []byte(`{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`)

				_, err = writer.CreateBlob(ctx, "tests/data/users.json", testDataContent)
				Expect(err).NotTo(HaveOccurred())

				_, err = writer.CreateBlob(ctx, "schemas/user.json", schemaContent)
				Expect(err).NotTo(HaveOccurred())

				commit3, err := writer.Commit(ctx, "Add test data and schema files", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit3).NotTo(BeNil())
				logger.Info("Third commit created", "hash", commit3.Hash.String(), "parent", commit3.Parent.String())

				// Verify commit chain before push
				Expect(currentHash).To(Equal(commit1.Parent))
				Expect(commit1.Hash).To(Equal(commit2.Parent))
				Expect(commit2.Hash).To(Equal(commit3.Parent))

				// Push all commits at once
				logger.Info("Pushing all three commits")
				err = writer.Push(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Pull and verify
				logger.Info("Pulling changes")
				local.Git("pull")

				// Verify final commit hash
				logger.Info("Verifying final commit hash")
				finalHash := local.Git("rev-parse", "HEAD")
				Expect(finalHash).To(Equal(commit3.Hash.String()))

				// Verify all commits exist in history
				logger.Info("Verifying commit history")
				commitHistory := local.Git("log", "--oneline", "--format=%H %s")
				logger.Info("Commit history", "history", commitHistory)

				Expect(commitHistory).To(ContainSubstring(commit1.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit2.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit3.Hash.String()))

				Expect(commitHistory).To(ContainSubstring("Add database and API configuration"))
				Expect(commitHistory).To(ContainSubstring("Add documentation files"))
				Expect(commitHistory).To(ContainSubstring("Add test data and schema files"))

				// Verify directory structure
				logger.Info("Verifying directory structure")
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
				logger.Info("Verifying file contents")
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
				logger.Info("Verifying original file preserved")
				originalContent, err := os.ReadFile(filepath.Join(local.Path, "test.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(originalContent).NotTo(BeEmpty())

				// Verify individual commits show correct files
				logger.Info("Verifying files in individual commits")
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

				logger.Info("Getting current ref")
				currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
				Expect(err).NotTo(HaveOccurred())
				ref := nanogit.Ref{
					Name: "refs/heads/main",
					Hash: currentHash,
				}

				logger.Info("Creating staged writer")
				writer, err := client.NewStagedWriter(ctx, ref)
				Expect(err).NotTo(HaveOccurred())

				// Create first commit
				logger.Info("Creating first file and commit")
				_, err = writer.CreateBlob(ctx, "file1.txt", []byte("First file content"))
				Expect(err).NotTo(HaveOccurred())
				commit1, err := writer.Commit(ctx, "Add first file", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit1).NotTo(BeNil())
				logger.Info("First commit", "hash", commit1.Hash.String())

				// Create second commit
				logger.Info("Creating second file and commit")
				_, err = writer.CreateBlob(ctx, "file2.txt", []byte("Second file content"))
				Expect(err).NotTo(HaveOccurred())
				commit2, err := writer.Commit(ctx, "Add second file", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit2).NotTo(BeNil())
				logger.Info("Second commit", "hash", commit2.Hash.String(), "parent", commit2.Parent.String())

				// Create third commit
				logger.Info("Creating third file and commit")
				_, err = writer.CreateBlob(ctx, "file3.txt", []byte("Third file content"))
				Expect(err).NotTo(HaveOccurred())
				commit3, err := writer.Commit(ctx, "Add third file", testAuthor, testCommitter)
				Expect(err).NotTo(HaveOccurred())
				Expect(commit3).NotTo(BeNil())
				logger.Info("Third commit", "hash", commit3.Hash.String(), "parent", commit3.Parent.String())

				// Verify commit chain is correct
				Expect(currentHash).To(Equal(commit1.Parent))
				Expect(commit1.Hash).To(Equal(commit2.Parent))
				Expect(commit2.Hash).To(Equal(commit3.Parent))

				// Push all commits
				logger.Info("Pushing all commits")
				err = writer.Push(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Pull and verify
				logger.Info("Pulling changes")
				local.Git("pull")

				// Verify final commit hash
				logger.Info("Verifying final commit hash")
				finalHash := local.Git("rev-parse", "HEAD")
				Expect(finalHash).To(Equal(commit3.Hash.String()))

				// Verify all three commits exist in history
				logger.Info("Verifying commit history")
				commitHistory := local.Git("log", "--oneline", "--format=%H %s")
				logger.Info("Commit history", "history", commitHistory)

				// Should contain all three commits
				Expect(commitHistory).To(ContainSubstring(commit1.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit2.Hash.String()))
				Expect(commitHistory).To(ContainSubstring(commit3.Hash.String()))

				// Verify commit messages are correct
				Expect(commitHistory).To(ContainSubstring("Add first file"))
				Expect(commitHistory).To(ContainSubstring("Add second file"))
				Expect(commitHistory).To(ContainSubstring("Add third file"))

				// Verify all files exist
				logger.Info("Verifying all files exist")
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
				logger.Info("Verifying individual commits are reachable")
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
	})
})
