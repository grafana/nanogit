package integration_test

import (
	"fmt"
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
		It("should clone a simple repository", func() {
			By("Setting up a test repository with some files")
			local.CreateFile("README.md", "# Test Repository")
			local.CreateFile("src/main.go", "package main\n\nfunc main() {}")
			local.CreateFile("docs/api.md", "# API Documentation")
			commitAndPush(local, user, "Initial commit with multiple files")

			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning the repository without any filters")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: ref.Hash,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			By("Verifying the clone result")
			Expect(result.Commit).NotTo(BeNil())
			Expect(result.FlatTree).NotTo(BeNil())
			Expect(result.TotalFiles).To(Equal(3))
			Expect(result.FilteredFiles).To(Equal(3))

			By("Verifying all files are present")
			fileMap := make(map[string]bool)
			for _, entry := range result.FlatTree.Entries {
				fileMap[entry.Path] = true
			}
			Expect(fileMap).To(HaveKey("README.md"))
			Expect(fileMap).To(HaveKey("src/main.go"))
			Expect(fileMap).To(HaveKey("docs/api.md"))
		})

		It("should require a commit hash to be provided", func() {
			By("Setting up a test repository")
			local.CreateFile("test.txt", "test content")
			commitAndPush(local, user, "Test commit")

			By("Trying to clone without specifying a hash")
			tempDir := GinkgoT().TempDir()
			_, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("commit hash is required"))
		})

		It("should write files to filesystem when path is specified", func() {
			By("Setting up a test repository with multiple files")
			local.CreateFile("README.md", "# Test Repository")
			local.CreateFile("src/main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello World\")\n}")
			local.CreateFile("docs/api.md", "# API Documentation\n\nThis is the API documentation.")
			commitAndPush(local, user, "Initial commit with files")

			By("Creating a temporary directory for cloning")
			tempDir := GinkgoT().TempDir()
			clonePath := filepath.Join(tempDir, "cloned-repo")

			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning the repository to filesystem")
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: clonePath,
				Hash: ref.Hash,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			By("Verifying files were written to disk")
			// Check README.md
			readmeContent, err := os.ReadFile(filepath.Join(clonePath, "README.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(readmeContent)).To(Equal("# Test Repository"))

			// Check src/main.go
			mainContent, err := os.ReadFile(filepath.Join(clonePath, "src", "main.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainContent)).To(Equal("package main\n\nfunc main() {\n\tfmt.Println(\"Hello World\")\n}"))

			// Check docs/api.md
			docsContent, err := os.ReadFile(filepath.Join(clonePath, "docs", "api.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(docsContent)).To(Equal("# API Documentation\n\nThis is the API documentation."))

			By("Verifying directory structure was created correctly")
			_, err = os.Stat(filepath.Join(clonePath, "src"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(clonePath, "docs"))
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the result contains the correct path")
			Expect(result.Path).To(Equal(clonePath))
		})

		It("should clone using a commit hash instead of ref", func() {
			By("Setting up a test repository")
			local.CreateFile("test.txt", "test content from commit")
			commitAndPush(local, user, "Test commit for hash cloning")

			By("Getting the current commit hash")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())
			commitHash := ref.Hash

			By("Cloning using the commit hash directly")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: commitHash, // Use Hash field instead of Ref
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			By("Verifying the file was cloned correctly")
			content, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("test content from commit"))

			By("Verifying the commit hash matches")
			Expect(result.Commit.Hash).To(Equal(commitHash))
		})

		It("should clone from a specific commit when provided with hash", func() {
			By("Setting up a test repository with two commits")
			local.CreateFile("first.txt", "first commit")
			commitAndPush(local, user, "First commit")
			
			firstRef, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())
			firstHash := firstRef.Hash
			
			local.CreateFile("second.txt", "second commit")
			commitAndPush(local, user, "Second commit")

			By("Cloning using the first commit hash")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: firstHash, // First commit
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying it used the first commit")
			Expect(result.Commit.Hash).To(Equal(firstHash))
			
			// Should have first.txt but not second.txt
			_, err = os.Stat(filepath.Join(tempDir, "first.txt"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(tempDir, "second.txt"))
			Expect(err).To(HaveOccurred()) // Should not exist
		})

	})

	Context("Path filtering", func() {
		BeforeEach(func() {
			By("Creating a complex directory structure")
			files := map[string]string{
				"README.md":                     "# Main readme",
				"package.json":                  `{"name": "test"}`,
				"src/main.go":                   "package main",
				"src/utils/helper.go":           "package utils",
				"src/api/handler.go":            "package api",
				"docs/README.md":                "# Documentation",
				"docs/api/endpoints.md":         "# API Endpoints",
				"tests/main_test.go":            "package main_test",
				"tests/utils/helper_test.go":    "package utils_test",
				"node_modules/package/index.js": "module.exports = {}",
				"vendor/lib/library.go":         "package lib",
				"build/output.bin":              "binary content",
				"tmp/cache.tmp":                 "cache data",
			}

			for path, content := range files {
				local.CreateFile(path, content)
			}
			commitAndPush(local, user, "Create complex directory structure")
		})

		It("should include only specified paths with IncludePaths", func() {
			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning with include patterns for src and docs")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				IncludePaths: []string{"src/**", "docs/**"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying only included paths are present")
			Expect(result.FilteredFiles).To(BeNumerically("<", result.TotalFiles))

			fileMap := make(map[string]bool)
			for _, entry := range result.FlatTree.Entries {
				fileMap[entry.Path] = true
			}

			// Should include src and docs files
			Expect(fileMap).To(HaveKey("src/main.go"))
			Expect(fileMap).To(HaveKey("src/utils/helper.go"))
			Expect(fileMap).To(HaveKey("src/api/handler.go"))
			Expect(fileMap).To(HaveKey("docs/README.md"))
			Expect(fileMap).To(HaveKey("docs/api/endpoints.md"))

			// Should not include other files
			Expect(fileMap).NotTo(HaveKey("README.md"))
			Expect(fileMap).NotTo(HaveKey("package.json"))
			Expect(fileMap).NotTo(HaveKey("tests/main_test.go"))
			Expect(fileMap).NotTo(HaveKey("node_modules/package/index.js"))
		})

		It("should exclude specified paths with ExcludePaths", func() {
			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning with exclude patterns for node_modules, vendor, build, and tmp")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				ExcludePaths: []string{"node_modules/**", "vendor/**", "build/**", "tmp/**"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying excluded paths are not present")
			Expect(result.FilteredFiles).To(BeNumerically("<", result.TotalFiles))

			fileMap := make(map[string]bool)
			for _, entry := range result.FlatTree.Entries {
				fileMap[entry.Path] = true
			}

			// Should include main files
			Expect(fileMap).To(HaveKey("README.md"))
			Expect(fileMap).To(HaveKey("src/main.go"))
			Expect(fileMap).To(HaveKey("docs/README.md"))
			Expect(fileMap).To(HaveKey("tests/main_test.go"))

			// Should exclude specified directories
			Expect(fileMap).NotTo(HaveKey("node_modules/package/index.js"))
			Expect(fileMap).NotTo(HaveKey("vendor/lib/library.go"))
			Expect(fileMap).NotTo(HaveKey("build/output.bin"))
			Expect(fileMap).NotTo(HaveKey("tmp/cache.tmp"))
		})

		It("should prioritize ExcludePaths over IncludePaths", func() {
			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning with both include and exclude patterns where they conflict")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				IncludePaths: []string{"src/**", "tests/**"},
				ExcludePaths: []string{"tests/**"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying exclude takes precedence")
			fileMap := make(map[string]bool)
			for _, entry := range result.FlatTree.Entries {
				fileMap[entry.Path] = true
			}

			// Should include src files
			Expect(fileMap).To(HaveKey("src/main.go"))
			Expect(fileMap).To(HaveKey("src/utils/helper.go"))

			// Should exclude tests files (exclude takes precedence)
			Expect(fileMap).NotTo(HaveKey("tests/main_test.go"))
			Expect(fileMap).NotTo(HaveKey("tests/utils/helper_test.go"))

			// Should not include other files (not in include list)
			Expect(fileMap).NotTo(HaveKey("README.md"))
			Expect(fileMap).NotTo(HaveKey("docs/README.md"))
		})

		It("should handle glob patterns correctly", func() {
			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning with specific glob patterns")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				IncludePaths: []string{"*.md", "src/*.go"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying glob patterns work correctly")
			fileMap := make(map[string]bool)
			for _, entry := range result.FlatTree.Entries {
				fileMap[entry.Path] = true
			}

			// Should include top-level .md files
			Expect(fileMap).To(HaveKey("README.md"))

			// Should include .go files in src directory
			Expect(fileMap).To(HaveKey("src/main.go"))

			// Should not include .md files in subdirectories (not matching pattern)
			Expect(fileMap).NotTo(HaveKey("docs/README.md"))
			Expect(fileMap).NotTo(HaveKey("docs/api/endpoints.md"))

			// Should not include .go files in subdirectories (not matching pattern)
			Expect(fileMap).NotTo(HaveKey("src/utils/helper.go"))
		})
	})

	Context("Edge cases and error handling", func() {
		It("should require a path to be specified", func() {
			By("Trying to clone without specifying a path")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Clone(ctx, nanogit.CloneOptions{
				Hash: ref.Hash,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("clone path is required"))
		})

		It("should handle non-existent commit hashes gracefully", func() {
			By("Trying to clone with a non-existent commit hash")
			invalidHash := hash.MustFromHex("0000000000000000000000000000000000000000")
			_, err := client.Clone(ctx, nanogit.CloneOptions{
				Path: "/tmp/test",
				Hash: invalidHash,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get commit"))
		})

		It("should handle empty repositories", func() {
			By("Creating an empty remote repository")
			emptyClient, _, emptyLocal, emptyUser := QuickSetup()

			By("Creating a single commit to make the repo valid")
			emptyLocal.CreateFile("empty.txt", "")
			commitAndPush(emptyLocal, emptyUser, "Initial empty commit")

			By("Getting the commit hash from the main branch")
			ref, err := emptyClient.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning the empty repository")
			tempDir := GinkgoT().TempDir()
			result, err := emptyClient.Clone(ctx, nanogit.CloneOptions{
				Path: tempDir,
				Hash: ref.Hash,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.FilteredFiles).To(Equal(1))
		})

		It("should handle filtering that results in no files", func() {
			By("Setting up a repository")
			local.CreateFile("test.txt", "content")
			commitAndPush(local, user, "Test commit")

			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning with filters that exclude all files")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				ExcludePaths: []string{"**"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.FilteredFiles).To(Equal(0))
			Expect(len(result.FlatTree.Entries)).To(Equal(0))
		})
	})

	Context("Performance considerations", func() {
		It("should work efficiently with large file structures", func() {
			By("Creating a larger directory structure")
			dirs := []string{"frontend/src", "frontend/public", "backend/src", "backend/tests", "docs/user", "docs/dev"}
			for _, dir := range dirs {
				for i := 0; i < 10; i++ {
					fileName := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
					local.CreateFile(fileName, fmt.Sprintf("Content for %s", fileName))
				}
			}
			commitAndPush(local, user, "Create large structure")

			By("Getting the commit hash from the main branch")
			ref, err := client.GetRef(ctx, "main")
			Expect(err).NotTo(HaveOccurred())

			By("Cloning with selective filtering")
			tempDir := GinkgoT().TempDir()
			result, err := client.Clone(ctx, nanogit.CloneOptions{
				Path:         tempDir,
				Hash:         ref.Hash,
				IncludePaths: []string{"backend/**"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the filtering is efficient and correct")
			Expect(result.FilteredFiles).To(BeNumerically("<", result.TotalFiles))
			Expect(result.FilteredFiles).To(Equal(20)) // 10 files each in backend/src and backend/tests

			// Verify only backend files are included
			for _, entry := range result.FlatTree.Entries {
				Expect(entry.Path).To(HavePrefix("backend/"))
			}
		})
	})
})
