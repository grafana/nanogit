package nanogit_test

import (
	"errors"
	"fmt"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Blobs", func() {
	Context("GetBlob operations", func() {
		var (
			client nanogit.Client
			local  *LocalGitRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()
		})

		It("should get blob with valid hash", func() {
			By("Creating and committing test file")
			testContent := []byte("test content")
			local.CreateFile("blob.txt", string(testContent))
			local.Git("add", "blob.txt")
			local.Git("commit", "-m", "Initial commit")
			local.Git("push", "origin", "main", "--force")

			By("Getting blob hash")
			blobHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:blob.txt"))
			Expect(err).NotTo(HaveOccurred())

			By("Testing GetBlob with valid hash")
			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob.Content).To(Equal(testContent))
			Expect(blob.Hash).To(Equal(blobHash))
		})

		It("should fail to get blob with non-existent hash", func() {
			By("Testing GetBlob with non-existent hash")
			nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetBlob(ctx, nonExistentHash)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0"))
		})
	})

	Context("GetBlobByPath operations", func() {
		var (
			client   nanogit.Client
			local    *LocalGitRepo
			rootHash hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Creating and committing test file")
			testContent := []byte("test content")
			local.CreateFile("blob.txt", string(testContent))
			local.Git("add", "blob.txt")
			local.Git("commit", "-m", "Initial commit")
			local.Git("push", "origin", "main", "--force")

			By("Getting the commit hash")
			var err error
			rootHash, err = hash.FromHex(local.Git("rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get blob by path with existing file", func() {
			testContent := []byte("test content")

			By("Getting blob by path")
			file, err := client.GetBlobByPath(ctx, rootHash, "blob.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Content).To(Equal(testContent))

			By("Verifying hash matches Git CLI")
			fileHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:blob.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(fileHash))
		})

		It("should fail to get blob by path with non-existent file", func() {
			By("Attempting to get non-existent file")
			_, err := client.GetBlobByPath(ctx, rootHash, "nonexistent.txt")
			Expect(err).To(HaveOccurred())

			By("Verifying correct error type")
			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(err).To(BeAssignableToTypeOf(pathNotFoundErr))
			if ok := errors.As(err, &pathNotFoundErr); ok {
				Expect(pathNotFoundErr.Path).To(Equal("nonexistent.txt"))
			} else {
				Fail(fmt.Sprintf("Expected PathNotFoundError, got %T", err))
			}
		})

		It("should fail to get blob by path with non-existent hash", func() {
			By("Testing with non-existent commit hash")
			nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetBlobByPath(ctx, nonExistentHash, "blob.txt")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not our ref"))
		})
	})

	Context("GetBlobByPath with nested directories", func() {
		var (
			client   nanogit.Client
			local    *LocalGitRepo
			rootHash hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Creating nested directory structure with files")
			local.CreateDirPath("dir1/subdir1")
			local.CreateDirPath("dir1/subdir2")
			local.CreateDirPath("dir2")

			// Create files at various levels
			local.CreateFile("root.txt", "root file content")
			local.CreateFile("dir1/file1.txt", "dir1 file content")
			local.CreateFile("dir1/subdir1/nested.txt", "deeply nested content")
			local.CreateFile("dir2/file2.txt", "dir2 file content")

			By("Adding and committing all files")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with nested structure")
			local.Git("push", "origin", "main", "--force")

			By("Getting the commit hash")
			var err error
			rootHash, err = hash.FromHex(local.Git("rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get root file", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "root.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("root file content"))

			expectedHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:root.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should get file in first level directory", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "dir1/file1.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("dir1 file content"))

			expectedHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:dir1/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should get deeply nested file", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "dir1/subdir1/nested.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("deeply nested content"))

			expectedHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:dir1/subdir1/nested.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should get file in different directory", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "dir2/file2.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("dir2 file content"))

			expectedHash, err := hash.FromHex(local.Git("rev-parse", "HEAD:dir2/file2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should fail with nonexistent file in existing directory", func() {
			_, err := client.GetBlobByPath(ctx, rootHash, "dir1/nonexistent.txt")
			Expect(err).To(HaveOccurred())

			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(err).To(BeAssignableToTypeOf(pathNotFoundErr))
		})

		It("should fail with file in nonexistent directory", func() {
			_, err := client.GetBlobByPath(ctx, rootHash, "nonexistent/file.txt")
			Expect(err).To(HaveOccurred())

			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(err).To(BeAssignableToTypeOf(pathNotFoundErr))
		})

		It("should fail with empty path", func() {
			_, err := client.GetBlobByPath(ctx, rootHash, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty path"))
		})

		It("should fail when path points to directory instead of file", func() {
			_, err := client.GetBlobByPath(ctx, rootHash, "dir1")
			Expect(err).To(HaveOccurred())

			var unexpectedTypeErr *nanogit.UnexpectedObjectTypeError
			Expect(err).To(BeAssignableToTypeOf(unexpectedTypeErr))
		})
	})
})
