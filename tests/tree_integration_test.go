package integration_test

import (
	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/gittest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Trees", func() {
	Context("GetFlatTree operations", func() {
		var (
			client     nanogit.Client
			local      *gittest.LocalRepo
			commitHash hash.Hash
			getHash    func(string) hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository with directory structure")
			client, _, local, _ = QuickSetup()

			By("Creating a directory structure with files")
			Expect(local.CreateDirPath("dir1")).To(Succeed())
			Expect(local.CreateDirPath("dir2")).To(Succeed())
			Expect(local.CreateFile("dir1/file1.txt", "content1")).To(Succeed())
			Expect(local.CreateFile("dir1/file2.txt", "content2")).To(Succeed())
			Expect(local.CreateFile("dir2/file3.txt", "content3")).To(Succeed())
			Expect(local.CreateFile("root.txt", "root content")).To(Succeed())

			By("Adding and committing the files")
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial commit with tree structure")
			Expect(err).NotTo(HaveOccurred())

			By("Creating and switching to main branch")
			_, err = local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the commit hash")
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up hash helper function")
			getHash = func(path string) hash.Hash {
				out, err := local.Git("rev-parse", "HEAD:"+path)
				Expect(err).NotTo(HaveOccurred())
				h, err := hash.FromHex(out)
				Expect(err).NotTo(HaveOccurred())
				return h
			}
		})

		It("should retrieve flat tree structure successfully", func() {
			tree, err := client.GetFlatTree(ctx, commitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())

			wantEntries := []nanogit.FlatTreeEntry{
				{
					Name: "test.txt",
					Path: "test.txt",
					Mode: 33188, // 100644 in octal
					Hash: getHash("test.txt"),
					Type: protocol.ObjectTypeBlob,
				},
				{
					Name: "root.txt",
					Path: "root.txt",
					Mode: 33188, // 100644 in octal
					Hash: getHash("root.txt"),
					Type: protocol.ObjectTypeBlob,
				},
				{
					Name: "dir1",
					Path: "dir1",
					Mode: 16384, // 040000 in octal
					Hash: getHash("dir1"),
					Type: protocol.ObjectTypeTree,
				},
				{
					Name: "file1.txt",
					Path: "dir1/file1.txt",
					Mode: 33188, // 100644 in octal
					Hash: getHash("dir1/file1.txt"),
					Type: protocol.ObjectTypeBlob,
				},
				{
					Name: "file2.txt",
					Path: "dir1/file2.txt",
					Mode: 33188, // 100644 in octal
					Hash: getHash("dir1/file2.txt"),
					Type: protocol.ObjectTypeBlob,
				},
				{
					Name: "dir2",
					Path: "dir2",
					Mode: 16384, // 040000 in octal
					Hash: getHash("dir2"),
					Type: protocol.ObjectTypeTree,
				},
				{
					Name: "file3.txt",
					Path: "dir2/file3.txt",
					Mode: 33188, // 100644 in octal
					Hash: getHash("dir2/file3.txt"),
					Type: protocol.ObjectTypeBlob,
				},
			}

			Expect(tree.Entries).To(HaveLen(len(wantEntries)))
			// Check that all expected entries are present by comparing each one
			for _, expectedEntry := range wantEntries {
				found := false
				for _, actualEntry := range tree.Entries {
					if actualEntry.Name == expectedEntry.Name &&
						actualEntry.Path == expectedEntry.Path &&
						actualEntry.Mode == expectedEntry.Mode &&
						actualEntry.Type == expectedEntry.Type &&
						actualEntry.Hash.Is(expectedEntry.Hash) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Expected entry not found: %+v", expectedEntry)
			}
		})

		It("should handle non-existent hash", func() {
			nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetFlatTree(ctx, nonExistentHash)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})
	})

	Context("GetFlatTree complex structure", func() {
		var (
			client     nanogit.Client
			local      *gittest.LocalRepo
			commitHash hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository with complex directory structure")
			client, _, local, _ = QuickSetup()

			By("Creating a complex directory structure with files")
			local.CreateDirPath("dir1/subdir1/subsubdir1")
			local.CreateDirPath("dir1/subdir2")
			local.CreateDirPath("dir2/subdir1")
			local.CreateDirPath("dir3")

			local.CreateFile("dir1/subdir1/subsubdir1/file1.txt", "content1")
			local.CreateFile("dir1/subdir1/file2.txt", "content2")
			local.CreateFile("dir1/subdir2/file3.txt", "content3")
			local.CreateFile("dir2/subdir1/file4.txt", "content4")
			local.CreateFile("dir2/file5.txt", "content5")
			local.CreateFile("dir3/file6.txt", "content6")
			local.CreateFile("root.txt", "root content")

			By("Adding and committing the files")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with complex tree structure")

			By("Creating and switching to main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "origin", "main", "--force")

			By("Getting the tree hash")
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve flat tree structure successfully", func() {
			flatTree, err := client.GetFlatTree(ctx, commitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(flatTree).NotTo(BeNil())

			expectedEntries := []struct {
				path string
				mode uint32
				typ  protocol.ObjectType
			}{
				{"dir1", 16384, protocol.ObjectTypeTree},
				{"dir1/subdir1", 16384, protocol.ObjectTypeTree},
				{"dir1/subdir1/file2.txt", 33188, protocol.ObjectTypeBlob},
				{"dir1/subdir1/subsubdir1", 16384, protocol.ObjectTypeTree},
				{"dir1/subdir1/subsubdir1/file1.txt", 33188, protocol.ObjectTypeBlob},
				{"dir1/subdir2", 16384, protocol.ObjectTypeTree},
				{"dir1/subdir2/file3.txt", 33188, protocol.ObjectTypeBlob},
				{"dir2", 16384, protocol.ObjectTypeTree},
				{"dir2/file5.txt", 33188, protocol.ObjectTypeBlob},
				{"dir2/subdir1", 16384, protocol.ObjectTypeTree},
				{"dir2/subdir1/file4.txt", 33188, protocol.ObjectTypeBlob},
				{"dir3", 16384, protocol.ObjectTypeTree},
				{"dir3/file6.txt", 33188, protocol.ObjectTypeBlob},
				{"root.txt", 33188, protocol.ObjectTypeBlob},
				{"test.txt", 33188, protocol.ObjectTypeBlob},
			}

			Expect(flatTree.Entries).To(HaveLen(len(expectedEntries)))
			for i, expected := range expectedEntries {
				actual := flatTree.Entries[i]
				Expect(actual.Path).To(Equal(expected.path))
				Expect(actual.Mode).To(Equal(expected.mode))
				Expect(actual.Type).To(Equal(expected.typ))
			}
		})
	})
	Context("GetTree operations", func() {
		var (
			client   nanogit.Client
			local    *gittest.LocalRepo
			treeHash hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository with directory structure")
			client, _, local, _ = QuickSetup()

			By("Creating a directory structure with files")
			local.CreateDirPath("dir1")
			local.CreateDirPath("dir2")
			local.CreateFile("dir1/file1.txt", "content1")
			local.CreateFile("dir1/file2.txt", "content2")
			local.CreateFile("dir2/file3.txt", "content3")
			local.CreateFile("root.txt", "root content")

			By("Adding and committing the files")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with tree structure")

			By("Creating and switching to main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "origin", "main", "--force")

			By("Getting the tree hash")
			output, err := local.Git("rev-parse", "HEAD^{tree}")
			Expect(err).NotTo(HaveOccurred())
			treeHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve tree structure successfully", func() {
			tree, err := client.GetTree(ctx, treeHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())

			expectedEntryNames := []string{"test.txt", "root.txt", "dir1", "dir2"}
			Expect(tree.Entries).To(HaveLen(len(expectedEntryNames)))

			entryNames := make([]string, len(tree.Entries))
			for i, entry := range tree.Entries {
				entryNames[i] = entry.Name
			}
			Expect(entryNames).To(ConsistOf(expectedEntryNames))
		})

		It("should handle non-existent hash", func() {
			nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetTree(ctx, nonExistentHash)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})
	})

	Context("GetTreeByPath operations", func() {
		var (
			client   nanogit.Client
			local    *gittest.LocalRepo
			treeHash hash.Hash
			getHash  func(string) hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository with directory structure")
			client, _, local, _ = QuickSetup()

			By("Creating a directory structure with files")
			local.CreateDirPath("dir1")
			local.CreateDirPath("dir2")
			local.CreateFile("dir1/file1.txt", "content1")
			local.CreateFile("dir1/file2.txt", "content2")
			local.CreateFile("dir2/file3.txt", "content3")
			local.CreateFile("root.txt", "root content")

			By("Adding and committing the files")
			local.Git("add", ".")
			local.Git("commit", "-m", "Initial commit with tree structure")

			By("Creating and switching to main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "origin", "main", "--force")

			By("Getting the tree hash")
			output, err := local.Git("rev-parse", "HEAD^{tree}")
			Expect(err).NotTo(HaveOccurred())
			treeHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up hash helper function")
			getHash = func(path string) hash.Hash {
				out, err := local.Git("rev-parse", "HEAD:"+path)
				Expect(err).NotTo(HaveOccurred())
				h, err := hash.FromHex(out)
				Expect(err).NotTo(HaveOccurred())
				return h
			}
		})

		It("should get root tree with empty path", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())

			entryNames := make([]string, len(tree.Entries))
			for i, entry := range tree.Entries {
				entryNames[i] = entry.Name
			}
			Expect(entryNames).To(ConsistOf("test.txt", "root.txt", "dir1", "dir2"))
		})
		It("should fail if treeHash does not exist", func() {
			nonexistentHash := hash.MustFromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
			_, err := client.GetTreeByPath(ctx, nonexistentHash, "dir2")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})
		It("should fail if path component is empty", func() {
			_, err := client.GetTreeByPath(ctx, treeHash, "dir1//file1.txt")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("path component is empty"))
		})

		It("should get root tree with dot path", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, ".")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())

			entryNames := make([]string, len(tree.Entries))
			for i, entry := range tree.Entries {
				entryNames[i] = entry.Name
			}
			Expect(entryNames).To(ConsistOf("test.txt", "root.txt", "dir1", "dir2"))
		})

		It("should get dir1 subdirectory", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, "dir1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())

			Expect(tree.Entries).To(HaveLen(2)) // file1.txt, file2.txt
			entryNames := make([]string, len(tree.Entries))
			for i, entry := range tree.Entries {
				entryNames[i] = entry.Name
			}
			Expect(entryNames).To(ConsistOf("file1.txt", "file2.txt"))
			Expect(tree.Hash.Is(getHash("dir1"))).To(BeTrue())
		})

		It("should get dir2 subdirectory", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, "dir2")
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).NotTo(BeNil())

			Expect(tree.Entries).To(HaveLen(1)) // file3.txt
			Expect(tree.Entries[0].Name).To(Equal("file3.txt"))
			Expect(tree.Hash.Is(getHash("dir2"))).To(BeTrue())
		})

		It("should handle nonexistent path", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, "nonexistent")
			Expect(err).To(HaveOccurred())
			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(errors.As(err, &pathNotFoundErr)).To(BeTrue())
			Expect(tree).To(BeNil())
		})

		It("should handle path to file instead of directory", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, "root.txt")
			Expect(err).To(HaveOccurred())
			var unexpectedTypeErr *nanogit.UnexpectedObjectTypeError
			Expect(errors.As(err, &unexpectedTypeErr)).To(BeTrue())
			Expect(tree).To(BeNil())
		})

		It("should handle nested nonexistent path", func() {
			tree, err := client.GetTreeByPath(ctx, treeHash, "dir1/nonexistent")
			Expect(err).To(HaveOccurred())
			var pathNotFoundErr *nanogit.PathNotFoundError
			Expect(errors.As(err, &pathNotFoundErr)).To(BeTrue())
			Expect(tree).To(BeNil())
		})
	})

	Context("GetFlatTree with fallback fetch", func() {
		var (
			client     nanogit.Client
			local      *gittest.LocalRepo
			commitHash hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Creating a complex nested directory structure")
			// Create multiple levels of nested directories with files
			local.CreateDirPath("level1/level2a/level3a")
			local.CreateDirPath("level1/level2a/level3b")
			local.CreateDirPath("level1/level2b/level3c")
			local.CreateDirPath("level1/level2b/level3d/level4")

			// Add files at various levels
			local.CreateFile("level1/file1.txt", "content at level1")
			local.CreateFile("level1/level2a/file2a.txt", "content at level2a")
			local.CreateFile("level1/level2a/level3a/file3a.txt", "content at level3a")
			local.CreateFile("level1/level2a/level3b/file3b.txt", "content at level3b")
			local.CreateFile("level1/level2b/file2b.txt", "content at level2b")
			local.CreateFile("level1/level2b/level3c/file3c.txt", "content at level3c")
			local.CreateFile("level1/level2b/level3d/file3d.txt", "content at level3d")
			local.CreateFile("level1/level2b/level3d/level4/file4.txt", "deep content at level4")

			By("Committing the complex structure")
			local.Git("add", ".")
			local.Git("commit", "-m", "Complex nested tree structure")
			local.Git("branch", "-M", "main")
			local.Git("push", "origin", "main", "--force")

			By("Getting the commit hash")
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve deeply nested tree structure with fallback if needed", func() {
			By("Fetching the flat tree for the entire repository")
			flatTree, err := client.GetFlatTree(ctx, commitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(flatTree).NotTo(BeNil())

			By("Verifying all directories and files are present")
			// We should have 4 directories (level2a, level2b, level3a, level3b, level3c, level3d, level4)
			// and 8 files in the flat tree
			Expect(len(flatTree.Entries)).To(BeNumerically(">", 10))

			By("Verifying specific deep paths are accessible")
			foundDeepFile := false
			foundLevel3aFile := false
			for _, entry := range flatTree.Entries {
				if entry.Path == "level1/level2b/level3d/level4/file4.txt" {
					foundDeepFile = true
					Expect(entry.Type).To(Equal(protocol.ObjectTypeBlob))
				}
				if entry.Path == "level1/level2a/level3a/file3a.txt" {
					foundLevel3aFile = true
					Expect(entry.Type).To(Equal(protocol.ObjectTypeBlob))
				}
			}
			Expect(foundDeepFile).To(BeTrue(), "Deep file should be found in flat tree")
			Expect(foundLevel3aFile).To(BeTrue(), "Level3a file should be found in flat tree")

			By("Verifying tree structure integrity")
			// Count directories and files
			var dirCount, fileCount int
			for _, entry := range flatTree.Entries {
				switch entry.Type {
				case protocol.ObjectTypeTree:
					dirCount++
				case protocol.ObjectTypeBlob:
					fileCount++
				}
			}
			Expect(dirCount).To(BeNumerically(">=", 7), "Should have at least 7 directories")
			Expect(fileCount).To(BeNumerically(">=", 8), "Should have at least 8 files")
		})
	})

	// Integration test for the submodule (gitlink mode 0o160000) bug fix.
	// Submodule entries have mode 0o160000, which has the 0o40000 bit set.
	// Before the fix, the bitmask check (FileMode&0o40000 != 0) would incorrectly
	// classify submodule entries as trees, causing "not our ref" errors when
	// nanogit tried to fetch the submodule commit hash as a tree object.
	Context("GetFlatTree with submodule", func() {
		var (
			client     nanogit.Client
			local      *gittest.LocalRepo
			commitHash hash.Hash
		)

		BeforeEach(func() {
			By("Setting up main repository")
			var remote *gittest.RemoteRepository
			var user *gittest.User
			client, remote, local, user = QuickSetup()

			By("Creating a second repository to use as a submodule source")
			subRepo, err := gitServer.CreateRepo(ctx, "subrepo", user)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up the submodule source repository with content")
			subLocal, err := gittest.NewLocalRepo(ctx,
				gittest.WithRepoLogger(logger),
				gittest.WithGitTrace(),
			)
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("config", "user.name", user.Username)
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("config", "user.email", user.Email)
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("remote", "add", "origin", subRepo.AuthURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(subLocal.CreateFile("lib.txt", "library content")).To(Succeed())
			_, err = subLocal.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("commit", "-m", "Initial submodule commit")
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Adding files to the main repository")
			err = local.CreateDirPath("src")
			Expect(err).NotTo(HaveOccurred())
			err = local.CreateFile("src/main.txt", "main content")
			Expect(err).NotTo(HaveOccurred())
			err = local.CreateFile("README.md", "readme")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add source files")
			Expect(err).NotTo(HaveOccurred())

			By("Adding the submodule to the main repository")
			_ = remote // main remote is used by QuickInit
			_, err = local.Git("submodule", "add", subRepo.AuthURL, "external/lib")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add submodule")
			Expect(err).NotTo(HaveOccurred())

			By("Pushing the main repository")
			_, err = local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve flat tree excluding submodule entry without errors", func() {
			By("Fetching the flat tree for the commit with a submodule")
			flatTree, err := client.GetFlatTree(ctx, commitHash)
			Expect(err).NotTo(HaveOccurred(), "GetFlatTree should succeed on repos with submodules")
			Expect(flatTree).NotTo(BeNil())

			By("Verifying the submodule entry is NOT in the flat tree")
			paths := make([]string, len(flatTree.Entries))
			for i, e := range flatTree.Entries {
				paths[i] = e.Path
			}
			Expect(paths).NotTo(ContainElement("external/lib"),
				"submodule entry should be skipped in flat tree")

			By("Verifying regular files and directories are still present")
			Expect(paths).To(ContainElement("README.md"))
			Expect(paths).To(ContainElement("src"))
			Expect(paths).To(ContainElement("src/main.txt"))
			Expect(paths).To(ContainElement(".gitmodules"))

			By("Verifying no entries exist inside the submodule path")
			for _, entry := range flatTree.Entries {
				Expect(entry.Path).NotTo(HavePrefix("external/lib/"),
					"flat tree should not contain entries inside the submodule")
			}
		})
	})

	Context("CompareCommits with submodule", func() {
		var (
			client              nanogit.Client
			local               *gittest.LocalRepo
			beforeSubmoduleHash hash.Hash
			afterSubmoduleHash  hash.Hash
		)

		BeforeEach(func() {
			By("Setting up main repository")
			var remote *gittest.RemoteRepository
			var user *gittest.User
			client, remote, local, user = QuickSetup()

			By("Creating initial commit in main repo")
			err := local.CreateFile("README.md", "readme")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial commit")
			Expect(err).NotTo(HaveOccurred())
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			beforeSubmoduleHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a second repository to use as a submodule source")
			subRepo, err := gitServer.CreateRepo(ctx, "subrepo-compare", user)
			Expect(err).NotTo(HaveOccurred())

			subLocal, err := gittest.NewLocalRepo(ctx,
				gittest.WithRepoLogger(logger),
				gittest.WithGitTrace(),
			)
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("config", "user.name", user.Username)
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("config", "user.email", user.Email)
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("remote", "add", "origin", subRepo.AuthURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(subLocal.CreateFile("lib.txt", "library content")).To(Succeed())
			_, err = subLocal.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("commit", "-m", "Initial submodule commit")
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = subLocal.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Adding the submodule to the main repository")
			_ = remote // main remote is used by QuickInit
			_, err = local.Git("submodule", "add", subRepo.AuthURL, "vendor/lib")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add submodule")
			Expect(err).NotTo(HaveOccurred())
			output, err = local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			afterSubmoduleHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Pushing the main repository")
			_, err = local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should compare commits across submodule addition without errors", func() {
			By("Comparing the commit before and after submodule addition")
			changes, err := client.CompareCommits(ctx, beforeSubmoduleHash, afterSubmoduleHash)
			Expect(err).NotTo(HaveOccurred(), "CompareCommits should succeed when submodule is added")
			Expect(changes).NotTo(BeEmpty())

			By("Verifying the submodule entry is NOT in the changes (skipped)")
			for _, change := range changes {
				Expect(change.Path).NotTo(Equal("vendor/lib"),
					"submodule entry should be skipped from commit comparison")
			}

			By("Verifying .gitmodules is reported as an added file")
			paths := make([]string, len(changes))
			for i, c := range changes {
				paths[i] = c.Path
			}
			Expect(paths).To(ContainElement(".gitmodules"),
				".gitmodules should appear as a changed file when a submodule is added")
		})
	})
})
