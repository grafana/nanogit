package integration_test

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol/hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delta Object Handling", func() {
	Context("when server sends deltified objects", func() {
		var (
			client nanogit.Client
			local  *gittest.LocalRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()
		})

		It("should handle ref-delta objects for modified files", func() {
			By("Creating a base file and committing it")
			baseContent := strings.Repeat("base content line\n", 100) // ~1.8KB
			Expect(local.CreateFile("delta-test.txt", baseContent)).To(Succeed())
			_, err := local.Git("add", "delta-test.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial commit with base content")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Making small modifications to the file multiple times")
			for i := 1; i <= 5; i++ {
				modifiedContent := strings.Replace(baseContent, "base content line", "modified content line", 1)
				baseContent = modifiedContent
				Expect(local.UpdateFile("delta-test.txt", baseContent)).To(Succeed())
				_, err = local.Git("add", "delta-test.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Modification "+string(rune('0'+i)))
				Expect(err).NotTo(HaveOccurred())
			}
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing Git to repack with deltas")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the blob hash of the latest version")
			output, err := local.Git("rev-parse", "HEAD:delta-test.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(output)
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
				Expect(local.CreateFile("file"+string(rune('0'+i))+".txt", content)).To(Succeed())
			}

			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add multiple similar files")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing aggressive repacking to maximize deltification")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting all file hashes")
			var blobHashes []hash.Hash
			for i := 1; i <= fileCount; i++ {
				fileName := "file" + string(rune('0'+i)) + ".txt"
				hashStr, err := local.Git("rev-parse", "HEAD:"+fileName)
				Expect(err).NotTo(HaveOccurred())
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
			Expect(local.CreateDirPath("dir1")).To(Succeed())
			for i := 1; i <= 5; i++ {
				Expect(local.CreateFile("dir1/file"+string(rune('0'+i))+".txt", "content "+string(rune('0'+i)))).To(Succeed())
			}
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial directory structure")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Modifying the directory structure slightly")
			Expect(local.CreateFile("dir1/file6.txt", "content 6")).To(Succeed())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add one more file")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing repacking")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the tree hash")
			output, err := local.Git("rev-parse", "HEAD^{tree}")
			Expect(err).NotTo(HaveOccurred())
			treeHash, err := hash.FromHex(output)
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
				Expect(local.CreateFile("commit-test-"+string(rune('0'+i))+".txt", "commit "+string(rune('0'+i)))).To(Succeed())
				_, err := local.Git("add", ".")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Commit number "+string(rune('0'+i)))
				Expect(err).NotTo(HaveOccurred())
			}
			_, err := local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing repacking")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the commit hash")
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err := hash.FromHex(output)
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
			Expect(local.CreateFile("path-test.txt", baseContent)).To(Succeed())
			_, err := local.Git("add", "path-test.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			for i := 1; i <= 3; i++ {
				baseContent += "Additional line " + string(rune('0'+i)) + "\n"
				Expect(local.UpdateFile("path-test.txt", baseContent)).To(Succeed())
				_, err = local.Git("add", "path-test.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Update "+string(rune('0'+i)))
				Expect(err).NotTo(HaveOccurred())
			}
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing repacking")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the tree hash")
			output, err := local.Git("rev-parse", "HEAD^{tree}")
			Expect(err).NotTo(HaveOccurred())
			treeHash, err := hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Fetching blob by path (underlying objects may be deltas)")
			blob, err := client.GetBlobByPath(ctx, treeHash, "path-test.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(blob).NotTo(BeNil())
			Expect(string(blob.Content)).To(Equal(baseContent))
		})

		It("should handle clone with deltified repository", func() {
			By("Creating a realistic repository structure")
			Expect(local.CreateDirPath("src")).To(Succeed())
			Expect(local.CreateDirPath("docs")).To(Succeed())

			// Create base files
			for i := 1; i <= 5; i++ {
				Expect(local.CreateFile("src/file"+string(rune('0'+i))+".go", "package main\n\nfunc main() {\n\t// Version "+string(rune('0'+i))+"\n}\n")).To(Succeed())
				Expect(local.CreateFile("docs/doc"+string(rune('0'+i))+".md", "# Documentation "+string(rune('0'+i))+"\n\nContent here.\n")).To(Succeed())
			}

			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial structure")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Making incremental changes to create delta opportunities")
			for i := 1; i <= 5; i++ {
				Expect(local.UpdateFile("src/file1.go", "package main\n\nfunc main() {\n\t// Modified version "+string(rune('0'+i))+"\n}\n")).To(Succeed())
				_, err := local.Git("add", ".")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Update iteration "+string(rune('0'+i)))
				Expect(err).NotTo(HaveOccurred())
			}
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing aggressive repacking")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the main branch commit hash")
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err := hash.FromHex(output)
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
			local  *gittest.LocalRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()
		})

		It("should handle empty file deltification", func() {
			By("Creating and modifying an empty file")
			Expect(local.CreateFile("empty.txt", "")).To(Succeed())
			_, err := local.Git("add", "empty.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add empty file")
			Expect(err).NotTo(HaveOccurred())

			Expect(local.UpdateFile("empty.txt", "now has content")).To(Succeed())
			_, err = local.Git("add", "empty.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add content")
			Expect(err).NotTo(HaveOccurred())

			Expect(local.UpdateFile("empty.txt", "")).To(Succeed())
			_, err = local.Git("add", "empty.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Empty again")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("repack", "-a", "-d", "-f")
			Expect(err).NotTo(HaveOccurred())

			By("Fetching the empty file blob")
			output, err := local.Git("rev-parse", "HEAD:empty.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob.Content).To(BeEmpty())
		})

		It("should handle large file with deltas", func() {
			By("Creating a large file")
			largeContent := strings.Repeat("Large file content line with some variation\n", 1000) // ~47KB
			Expect(local.CreateFile("large.txt", largeContent)).To(Succeed())
			_, err := local.Git("add", "large.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add large file")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Making a small modification to the large file")
			modifiedContent := largeContent + "One more line\n"
			Expect(local.UpdateFile("large.txt", modifiedContent)).To(Succeed())
			_, err = local.Git("add", "large.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Small modification")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Forcing repacking")
			_, err = local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Fetching the modified blob")
			output, err := local.Git("rev-parse", "HEAD:large.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(output)
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
