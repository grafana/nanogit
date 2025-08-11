package integration_test

import (
	"os"
	"path/filepath"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Clone operations", func() {
	var (
		client nanogit.Client
		local  *LocalGitRepo
		user   *User
	)

	// Helper function to commit and push all changes
	commitAndPush := func(repo *LocalGitRepo, user *User, message string) {
		repo.Git("config", "user.name", user.Username)
		repo.Git("config", "user.email", user.Email)
		repo.Git("add", ".")
		repo.Git("commit", "-m", message)
		repo.Git("push", "-u", "origin", "main", "--force")
	}

	BeforeEach(func() {
		client, _, local, user = QuickSetup()
	})

	Context("Basic clone operations", func() {
		It("should clone a repository and write files to filesystem", func() {
			By("Setting up a test repository with multiple files")
			local.CreateFile("README.md", "# Test Repository")
			local.CreateFile("src/main.go", "package main\n\nfunc main() {}")
			local.CreateFile("docs/api.md", "# API Documentation")
			commitAndPush(local, user, "Initial commit")

			By("Cloning the repository")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())
			
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: ref.Hash,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Path).To(Equal(tempDir))
			Expect(result.Commit.Hash).To(Equal(ref.Hash))
			Expect(result.FilteredFiles).To(Equal(3))
			
			By("Verifying files were written to disk")
			content, err := os.ReadFile(filepath.Join(tempDir, "README.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("# Test Repository"))
		})

		It("should clone using a specific commit hash", func() {
			By("Setting up repository with multiple commits")
			local.CreateFile("first.txt", "first commit")
			commitAndPush(local, user, "First commit")
			
			firstRef, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())
			
			local.CreateFile("second.txt", "second commit")
			commitAndPush(local, user, "Second commit")

			By("Cloning using the first commit hash")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: firstRef.Hash,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Commit.Hash).To(Equal(firstRef.Hash))
			
			// Should have first.txt but not second.txt
			_, err = os.Stat(filepath.Join(tempDir, "first.txt"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(tempDir, "second.txt"))
			Expect(err).To(HaveOccurred()) // Should not exist
		})
	})

	Context("Path filtering", func() {
		BeforeEach(func() {
			By("Creating a repository with diverse file structure")
			local.CreateFile("README.md", "# Main readme")
			local.CreateFile("src/main.go", "package main")
			local.CreateFile("src/utils/helper.go", "package utils")
			local.CreateFile("docs/README.md", "# Documentation")
			local.CreateFile("tests/main_test.go", "package main_test")
			local.CreateFile("node_modules/package/index.js", "module.exports = {}")
			commitAndPush(local, user, "Create diverse structure")
		})

		It("should include only specified paths", func() {
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				IncludePaths: []string{"src/**", "docs/**"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.FilteredFiles).To(BeNumerically("<", result.TotalFiles))

			// Should include src and docs files
			_, err = os.Stat(filepath.Join(tempDir, "src", "main.go"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(tempDir, "docs", "README.md"))
			Expect(err).NotTo(HaveOccurred())

			// Should not include other files
			_, err = os.Stat(filepath.Join(tempDir, "README.md"))
			Expect(err).To(HaveOccurred())
		})

		It("should exclude specified paths", func() {
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				ExcludePaths: []string{"node_modules/**", "tests/**"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.FilteredFiles).To(BeNumerically("<", result.TotalFiles))

			// Should include main files
			_, err = os.Stat(filepath.Join(tempDir, "README.md"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(tempDir, "src", "main.go"))
			Expect(err).NotTo(HaveOccurred())

			// Should exclude specified directories
			_, err = os.Stat(filepath.Join(tempDir, "node_modules"))
			Expect(err).To(HaveOccurred())
			_, err = os.Stat(filepath.Join(tempDir, "tests"))
			Expect(err).To(HaveOccurred())
		})

		It("should prioritize exclude over include", func() {
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			tempDir := GinkgoT().TempDir()
			_, err = client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				IncludePaths: []string{"src/**", "tests/**"},
				ExcludePaths: []string{"tests/**"},
			})
			Expect(err).NotTo(HaveOccurred())

			// Should include src files
			_, err = os.Stat(filepath.Join(tempDir, "src", "main.go"))
			Expect(err).NotTo(HaveOccurred())

			// Should exclude tests files (exclude takes precedence)
			_, err = os.Stat(filepath.Join(tempDir, "tests"))
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Error handling", func() {
		It("should require a commit hash", func() {
			tempDir := GinkgoT().TempDir()
			_, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("commit hash is required"))
		})

		It("should handle non-existent commit hashes", func() {
			invalidHash := hash.MustFromHex("0000000000000000000000000000000000000000")
			tempDir := GinkgoT().TempDir()
			_, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: invalidHash,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get commit"))
		})

		It("should handle filtering that results in no files", func() {
			local.CreateFile("test.txt", "content")
			commitAndPush(local, user, "Test commit")

			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				ExcludePaths: []string{"**"}, // Exclude everything
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.FilteredFiles).To(Equal(0))
		})
	})
})