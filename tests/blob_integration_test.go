package integration_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
			gitNoError(local, "add", "blob.txt")
			gitNoError(local, "commit", "-m", "Initial commit")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting blob hash")
			blobHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:blob.txt"))
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
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
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
			gitNoError(local, "add", "blob.txt")
			gitNoError(local, "commit", "-m", "Initial commit")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the commit hash")
			var err error
			rootHash, err = hash.FromHex(gitNoError(local, "rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get blob by path with existing file", func() {
			testContent := []byte("test content")

			By("Getting blob by path")
			file, err := client.GetBlobByPath(ctx, rootHash, "blob.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Content).To(Equal(testContent))

			By("Verifying hash matches Git CLI")
			fileHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:blob.txt"))
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
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})
		It("should fail when path ends with slash", func() {
			By("Attempting to get a file with a path ending in slash")
			_, err := client.GetBlobByPath(ctx, rootHash, "blob.txt/")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid path: ends with slash"))
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
			gitNoError(local, "add", ".")
			gitNoError(local, "commit", "-m", "Initial commit with nested structure")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the commit hash")
			var err error
			rootHash, err = hash.FromHex(gitNoError(local, "rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get root file", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "root.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("root file content"))

			expectedHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:root.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should get file in first level directory", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "dir1/file1.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("dir1 file content"))

			expectedHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:dir1/file1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should get deeply nested file", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "dir1/subdir1/nested.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("deeply nested content"))

			expectedHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:dir1/subdir1/nested.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Hash).To(Equal(expectedHash))
		})

		It("should get file in different directory", func() {
			file, err := client.GetBlobByPath(ctx, rootHash, "dir2/file2.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(file.Content)).To(Equal("dir2 file content"))

			expectedHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:dir2/file2.txt"))
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
		It("should ignore empty parts of the path", func() {
			// These paths should all resolve to the same file
			paths := []string{
				"dir1//file1.txt",
				"//dir1/file1.txt",
			}
			expectedContent := "dir1 file content"
			expectedHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:dir1/file1.txt"))
			Expect(err).NotTo(HaveOccurred())

			for _, p := range paths {
				file, err := client.GetBlobByPath(ctx, rootHash, p)
				Expect(err).NotTo(HaveOccurred(), "path: %q", p)
				Expect(string(file.Content)).To(Equal(expectedContent), "path: %q", p)
				Expect(file.Hash).To(Equal(expectedHash), "path: %q", p)
			}
		})

		It("should fail when path points to directory instead of file", func() {
			_, err := client.GetBlobByPath(ctx, rootHash, "dir1")
			Expect(err).To(HaveOccurred())

			var unexpectedTypeErr *nanogit.UnexpectedObjectTypeError
			Expect(err).To(BeAssignableToTypeOf(unexpectedTypeErr))
		})
	})

	Context("Large blob operations", func() {
		var (
			client nanogit.Client
			local  *LocalGitRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()
		})

		It("should handle extra large dashboard file (3.7MB)", func() {
			By("Reading the xlarge dashboard from the repository")
			dashboardPath := filepath.Join("..", "perf", "cmd", "generate_dashboards", "generated_dashboards", "xlarge-dashboard.json")
			dashboardContent, err := os.ReadFile(dashboardPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(dashboardContent)).To(BeNumerically(">", 3000000), "Dashboard should be larger than 3MB")

			By("Creating and committing the large dashboard file")
			local.CreateFile("xlarge-dashboard.json", string(dashboardContent))
			gitNoError(local, "add", "xlarge-dashboard.json")
			gitNoError(local, "commit", "-m", "Add xlarge dashboard for blob testing")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting blob hash for the large file")
			blobHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:xlarge-dashboard.json"))
			Expect(err).NotTo(HaveOccurred())

			By("Testing GetBlob with large blob")
			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob.Content).To(Equal(dashboardContent))
			Expect(blob.Hash).To(Equal(blobHash))
			Expect(len(blob.Content)).To(BeNumerically(">", 3000000), "Retrieved blob should maintain size")

			By("Testing GetBlobByPath with large blob")
			rootHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())

			file, err := client.GetBlobByPath(ctx, rootHash, "xlarge-dashboard.json")
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Content).To(Equal(dashboardContent))
			Expect(file.Hash).To(Equal(blobHash))
			Expect(len(file.Content)).To(BeNumerically(">", 3000000), "Retrieved file should maintain size")

			By("Verifying the content is valid JSON dashboard")
			contentStr := string(file.Content)
			Expect(contentStr).To(ContainSubstring("\"title\""))
			Expect(contentStr).To(ContainSubstring("\"panels\""))
			Expect(contentStr).To(ContainSubstring("\"templating\""))
			Expect(contentStr).To(Or(ContainSubstring("xlarge"), ContainSubstring("Xlarge")))
		})
	})
})
