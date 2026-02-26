package integration_test

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delta Object Handling", func() {
	Context("when server sends deltified objects", func() {
		var (
			client nanogit.Client
			local  *LocalGitRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()
		})

		It("should handle ref-delta objects for modified files", func() {
			By("Creating a base file and committing it")
			baseContent := strings.Repeat("base content line\n", 100) // ~1.8KB
			_ = local.CreateFile("delta-test.txt", baseContent)
			gitNoError(local, "add", "delta-test.txt")
			gitNoError(local, "commit", "-m", "Initial commit with base content")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Making small modifications to the file multiple times")
			for i := 1; i <= 5; i++ {
				modifiedContent := strings.Replace(baseContent, "base content line", "modified content line", 1)
				baseContent = modifiedContent
				_ = local.UpdateFile("delta-test.txt", baseContent)
				gitNoError(local, "add", "delta-test.txt")
				gitNoError(local, "commit", "-m", "Modification "+string(rune('0'+i)))
			}
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing Git to repack with deltas")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the blob hash of the latest version")
			blobHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:delta-test.txt"))
			Expect(err).NotTo(HaveOccurred())

			By("Fetching the blob (may be sent as delta)")
			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob).NotTo(BeNil())
			Expect(blob.Hash).To(Equal(blobHash))

			By("Verifying the content matches")
			Expect(string(blob.Content)).To(Equal(baseContent))
			Expect(len(blob.Content)).To(BeNumerically(">", 1000))
		})

		It("should handle multiple deltified files in a single fetch", func() {
			By("Creating multiple similar files")
			baseTemplate := "File %d content: " + strings.Repeat("x", 500)
			fileCount := 10

			for i := 1; i <= fileCount; i++ {
				content := strings.Replace(baseTemplate, "%d", string(rune('0'+i)), 1)
				_ = local.CreateFile("file"+string(rune('0'+i))+".txt", content)
			}

			gitNoError(local, "add", ".")
			gitNoError(local, "commit", "-m", "Add multiple similar files")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing aggressive repacking to maximize deltification")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting all file hashes")
			var blobHashes []hash.Hash
			for i := 1; i <= fileCount; i++ {
				fileName := "file" + string(rune('0'+i)) + ".txt"
				hashStr := gitNoError(local, "rev-parse", "HEAD:"+fileName)
				blobHash, err := hash.FromHex(hashStr)
				Expect(err).NotTo(HaveOccurred())
				blobHashes = append(blobHashes, blobHash)
			}

			By("Fetching all blobs (some should be deltas)")
			for i, blobHash := range blobHashes {
				blob, err := client.GetBlob(ctx, blobHash)
				Expect(err).NotTo(HaveOccurred(), "Failed to fetch blob %d with hash %s", i+1, blobHash.String())
				Expect(blob).NotTo(BeNil())
				Expect(blob.Hash).To(Equal(blobHash))
				Expect(len(blob.Content)).To(BeNumerically(">", 500))
			}
		})

		It("should handle deltified tree objects", func() {
			By("Creating a directory structure")
			_ = local.CreateDirPath("dir1")
			for i := 1; i <= 5; i++ {
				_ = local.CreateFile("dir1/file"+string(rune('0'+i))+".txt", "content "+string(rune('0'+i)))
			}
			gitNoError(local, "add", ".")
			gitNoError(local, "commit", "-m", "Initial directory structure")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Modifying the directory structure slightly")
			_ = local.CreateFile("dir1/file6.txt", "content 6")
			gitNoError(local, "add", ".")
			gitNoError(local, "commit", "-m", "Add one more file")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing repacking")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the tree hash")
			treeHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())

			By("Fetching the tree (may be deltified)")
			tree, err := client.GetTree(ctx, treeHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())
			Expect(tree.Hash).To(Equal(treeHash))
			Expect(len(tree.Entries)).To(BeNumerically(">", 0))
		})

		It("should handle deltified commits", func() {
			By("Creating multiple commits with small changes")
			for i := 1; i <= 10; i++ {
				_ = local.CreateFile("commit-test-"+string(rune('0'+i))+".txt", "commit "+string(rune('0'+i)))
				gitNoError(local, "add", ".")
				gitNoError(local, "commit", "-m", "Commit number "+string(rune('0'+i)))
			}
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing repacking")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the commit hash")
			commitHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Fetching the commit (may be deltified)")
			commit, err := client.GetCommit(ctx, commitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())
			Expect(commit.Hash).To(Equal(commitHash))
			Expect(commit.Message).To(ContainSubstring("Commit number"))
		})

		It("should handle GetBlobByPath with deltified objects", func() {
			By("Creating a file and modifying it multiple times")
			baseContent := "Initial content\n" + strings.Repeat("line\n", 50)
			_ = local.CreateFile("path-test.txt", baseContent)
			gitNoError(local, "add", "path-test.txt")
			gitNoError(local, "commit", "-m", "Initial")
			gitNoError(local, "push", "origin", "main", "--force")

			for i := 1; i <= 3; i++ {
				baseContent += "Additional line " + string(rune('0'+i)) + "\n"
				_ = local.UpdateFile("path-test.txt", baseContent)
				gitNoError(local, "add", "path-test.txt")
				gitNoError(local, "commit", "-m", "Update "+string(rune('0'+i)))
			}
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing repacking")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the tree hash")
			treeHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())

			By("Fetching blob by path (underlying objects may be deltas)")
			blob, err := client.GetBlobByPath(ctx, treeHash, "path-test.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(blob).NotTo(BeNil())
			Expect(string(blob.Content)).To(Equal(baseContent))
		})

		It("should handle clone with deltified repository", func() {
			By("Creating a realistic repository structure")
			_ = local.CreateDirPath("src")
			_ = local.CreateDirPath("docs")

			// Create base files
			for i := 1; i <= 5; i++ {
				_ = local.CreateFile("src/file"+string(rune('0'+i))+".go", "package main\n\nfunc main() {\n\t// Version "+string(rune('0'+i))+"\n}\n")
				_ = local.CreateFile("docs/doc"+string(rune('0'+i))+".md", "# Documentation "+string(rune('0'+i))+"\n\nContent here.\n")
			}

			gitNoError(local, "add", ".")
			gitNoError(local, "commit", "-m", "Initial structure")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Making incremental changes to create delta opportunities")
			for i := 1; i <= 5; i++ {
				_ = local.UpdateFile("src/file1.go", "package main\n\nfunc main() {\n\t// Modified version "+string(rune('0'+i))+"\n}\n")
				gitNoError(local, "add", ".")
				gitNoError(local, "commit", "-m", "Update iteration "+string(rune('0'+i)))
			}
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing aggressive repacking")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Getting the main branch commit hash")
			commitHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Cloning the repository (should handle all deltas)")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: commitHash,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			By("Verifying all files were cloned correctly")
			// Check src files exist
			for i := 1; i <= 5; i++ {
				fileName := "src/file" + string(rune('0'+i)) + ".go"
				clonedPath := filepath.Join(tempDir, fileName)
				_, err := os.Stat(clonedPath)
				Expect(err).NotTo(HaveOccurred(), "File %s should exist", fileName)

				// Verify content matches (git show may add trailing newline, so check content is present)
				clonedData, err := os.ReadFile(clonedPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(clonedData)).To(BeNumerically(">", 0), "File should have content")
				Expect(string(clonedData)).To(ContainSubstring("package main"), "File should contain Go code")
			}

			// Check docs files exist
			for i := 1; i <= 5; i++ {
				fileName := "docs/doc" + string(rune('0'+i)) + ".md"
				clonedPath := filepath.Join(tempDir, fileName)
				_, err := os.Stat(clonedPath)
				Expect(err).NotTo(HaveOccurred(), "File %s should exist", fileName)

				// Verify content matches
				clonedData, err := os.ReadFile(clonedPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(clonedData)).To(BeNumerically(">", 0), "File should have content")
				Expect(string(clonedData)).To(ContainSubstring("Documentation"), "File should contain docs")
			}
		})
	})

	Context("edge cases with deltas", func() {
		var (
			client nanogit.Client
			local  *LocalGitRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()
		})

		It("should handle empty file deltification", func() {
			By("Creating and modifying an empty file")
			_ = local.CreateFile("empty.txt", "")
			gitNoError(local, "add", "empty.txt")
			gitNoError(local, "commit", "-m", "Add empty file")

			_ = local.UpdateFile("empty.txt", "now has content")
			gitNoError(local, "add", "empty.txt")
			gitNoError(local, "commit", "-m", "Add content")

			_ = local.UpdateFile("empty.txt", "")
			gitNoError(local, "add", "empty.txt")
			gitNoError(local, "commit", "-m", "Empty again")

			gitNoError(local, "push", "origin", "main", "--force")
			gitNoError(local, "repack", "-a", "-d", "-f")

			By("Fetching the empty file blob")
			blobHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:empty.txt"))
			Expect(err).NotTo(HaveOccurred())

			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob.Content).To(BeEmpty())
		})

		It("should handle large file with deltas", func() {
			By("Creating a large file")
			largeContent := strings.Repeat("Large file content line with some variation\n", 1000) // ~47KB
			_ = local.CreateFile("large.txt", largeContent)
			gitNoError(local, "add", "large.txt")
			gitNoError(local, "commit", "-m", "Add large file")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Making a small modification to the large file")
			modifiedContent := largeContent + "One more line\n"
			_ = local.UpdateFile("large.txt", modifiedContent)
			gitNoError(local, "add", "large.txt")
			gitNoError(local, "commit", "-m", "Small modification")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Forcing repacking")
			gitNoError(local, "repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			gitNoError(local, "push", "origin", "main", "--force")

			By("Fetching the modified blob")
			blobHash, err := hash.FromHex(gitNoError(local, "rev-parse", "HEAD:large.txt"))
			Expect(err).NotTo(HaveOccurred())

			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying content matches expected")
			Expect(string(blob.Content)).To(Equal(modifiedContent))
			Expect(len(blob.Content)).To(Equal(len(modifiedContent)), "Content length should match")
			Expect(len(blob.Content)).To(BeNumerically(">", 40000), "Should be a large file")
		})
	})
})
