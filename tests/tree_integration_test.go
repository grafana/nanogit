package integration_test

import (
	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Trees", func() {
	Context("GetFlatTree operations", func() {
		var (
			client     nanogit.Client
			local      *LocalGitRepo
			commitHash hash.Hash
			getHash    func(string) hash.Hash
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

			By("Getting the commit hash")
			var err error
			commitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Setting up hash helper function")
			getHash = func(path string) hash.Hash {
				out := local.Git("rev-parse", "HEAD:"+path)
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
			local      *LocalGitRepo
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
			var err error
			commitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
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
			local    *LocalGitRepo
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
			var err error
			treeHash, err = hash.FromHex(local.Git("rev-parse", "HEAD^{tree}"))
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
			local    *LocalGitRepo
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
			var err error
			treeHash, err = hash.FromHex(local.Git("rev-parse", "HEAD^{tree}"))
			Expect(err).NotTo(HaveOccurred())

			By("Setting up hash helper function")
			getHash = func(path string) hash.Hash {
				out := local.Git("rev-parse", "HEAD:"+path)
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
})
