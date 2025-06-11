package nanogit_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/internal/testhelpers"
	"github.com/grafana/nanogit/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Providers Suite")
}

var _ = Describe("Providers", func() {
	var client nanogit.Client

	BeforeEach(func() {
		By("Getting credentials from environment")
		repo := os.Getenv("TEST_REPO")
		token := os.Getenv("TEST_TOKEN")
		Expect(repo).NotTo(BeEmpty(), "TEST_REPO environment variable must be set")
		Expect(token).NotTo(BeEmpty(), "TEST_TOKEN environment variable must be set")

		By("Creating GitHub client")
		var err error
		client, err = nanogit.NewHTTPClient(
			fmt.Sprintf("https://github.com/%s.git", repo),
			nanogit.WithBasicAuth("git", token),
			nanogit.WithLogger(testhelpers.NewTestLogger()),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should perform end-to-end operations", func() {
		By("Checking if client is authorized")
		auth, err := client.IsAuthorized(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(auth).To(BeTrue())

		By("Checking if repository exists")
		exists, err := client.RepoExists(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())

		By("Creating and switching to a new branch")
		branchName := fmt.Sprintf("test-branch-%d", time.Now().Unix())
		mainRef, err := client.GetRef(context.Background(), "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())
		err = client.CreateRef(context.Background(), nanogit.Ref{
			Name: "refs/heads/" + branchName,
			Hash: mainRef.Hash,
		})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			By("Deleting branch")
			err = client.DeleteRef(context.Background(), "refs/heads/"+branchName)
			Expect(err).NotTo(HaveOccurred())
			refs, err := client.ListRefs(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(refs).NotTo(ContainElement(nanogit.Ref{
				Name: "refs/heads/" + branchName,
			}))
		})

		By("Listing refs")
		refs, err := client.ListRefs(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(refs).To(ContainElement(nanogit.Ref{
			Name: "refs/heads/" + branchName,
			Hash: mainRef.Hash,
		}))

		Expect(refs).To(ContainElement(nanogit.Ref{
			Name: "refs/heads/main",
			Hash: mainRef.Hash,
		}))

		By("Getting branch ref")
		branchRef, err := client.GetRef(context.Background(), "refs/heads/"+branchName)
		Expect(err).NotTo(HaveOccurred())

		By("Creating staged writer")
		writer, err := client.NewStagedWriter(context.Background(), branchRef)
		Expect(err).NotTo(HaveOccurred())

		author := nanogit.Author{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Time:  time.Now(),
		}
		committer := nanogit.Committer{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Time:  time.Now(),
		}

		By("Deleting everything")
		_, err = writer.DeleteTree(context.Background(), "")
		Expect(err).NotTo(HaveOccurred())
		_, err = writer.Commit(context.Background(), "Delete everything", author, committer)
		Expect(err).NotTo(HaveOccurred())

		By("Creating blob")
		exists, err = writer.BlobExists(context.Background(), "a/b/c/test.txt")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
		_, err = writer.GetTree(context.Background(), "a/b/c")
		Expect(err).To(HaveOccurred())

		blobHash, err := writer.CreateBlob(context.Background(), "a/b/c/test.txt", []byte("test content"))
		Expect(err).NotTo(HaveOccurred())

		tree, err := writer.GetTree(context.Background(), "a/b/c")
		Expect(err).NotTo(HaveOccurred())
		Expect(tree.Entries).To(HaveLen(1))

		exists, err = writer.BlobExists(context.Background(), "a/b/c/test.txt")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())

		commit, err := writer.Commit(context.Background(), "Add test file", author, committer)
		Expect(err).NotTo(HaveOccurred())

		By("Pushing changes")
		err = writer.Push(context.Background())
		Expect(err).NotTo(HaveOccurred())

		By("Getting branch ref")
		branchRef, err = client.GetRef(context.Background(), "refs/heads/"+branchName)
		Expect(err).NotTo(HaveOccurred())
		Expect(branchRef.Hash).To(Equal(commit.Hash))

		By("Getting commit")
		commit, err = client.GetCommit(context.Background(), commit.Hash)
		Expect(err).NotTo(HaveOccurred())
		Expect(commit.Message).To(Equal("Add test file"))

		By("Getting blob by hash")
		createdBlob, err := client.GetBlob(context.Background(), blobHash)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(createdBlob.Content)).To(Equal("test content"))

		By("Getting blob by path")
		createdBlobByPath, err := client.GetBlobByPath(context.Background(), commit.Tree, "a/b/c/test.txt")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(createdBlobByPath.Content)).To(Equal("test content"))

		By("Comparing commits")
		// TODO: add check for other types of modifications and more commits in between
		// TODO: create a more complex tree
		// TODO: validate more fields
		compareCommits, err := client.CompareCommits(context.Background(), commit.Parent, commit.Hash)
		Expect(err).NotTo(HaveOccurred())
		Expect(compareCommits).To(HaveLen(4))
		Expect(compareCommits[0].Path).To(Equal("a"))
		Expect(compareCommits[0].Status).To(Equal(protocol.FileStatusAdded))
		Expect(compareCommits[1].Path).To(Equal("a/b"))
		Expect(compareCommits[1].Status).To(Equal(protocol.FileStatusAdded))
		Expect(compareCommits[2].Path).To(Equal("a/b/c"))
		Expect(compareCommits[2].Status).To(Equal(protocol.FileStatusAdded))
		Expect(compareCommits[3].Path).To(Equal("a/b/c/test.txt"))
		Expect(compareCommits[3].Status).To(Equal(protocol.FileStatusAdded))

		By("Getting flat tree")
		flatTree, err := client.GetFlatTree(context.Background(), commit.Hash)
		Expect(err).NotTo(HaveOccurred())
		Expect(flatTree.Entries).To(HaveLen(4))
		Expect(flatTree.Entries[0].Path).To(Equal("a"))
		Expect(flatTree.Entries[0].Type).To(Equal(protocol.ObjectTypeTree))
		Expect(flatTree.Entries[1].Path).To(Equal("a/b"))
		Expect(flatTree.Entries[1].Type).To(Equal(protocol.ObjectTypeTree))
		Expect(flatTree.Entries[2].Path).To(Equal("a/b/c"))
		Expect(flatTree.Entries[2].Type).To(Equal(protocol.ObjectTypeTree))
		Expect(flatTree.Entries[3].Path).To(Equal("a/b/c/test.txt"))
		Expect(flatTree.Entries[3].Type).To(Equal(protocol.ObjectTypeBlob))
		Expect(flatTree.Entries[3].Hash).To(Equal(blobHash))

		By("Getting tree by hash")
		tree, err = client.GetTree(context.Background(), flatTree.Entries[2].Hash)
		Expect(err).NotTo(HaveOccurred())
		Expect(tree.Hash).To(Equal(flatTree.Entries[2].Hash))
		Expect(tree.Entries).To(HaveLen(1))
		Expect(tree.Entries[0].Name).To(Equal("test.txt"))
		Expect(tree.Entries[0].Type).To(Equal(protocol.ObjectTypeBlob))
		Expect(tree.Entries[0].Hash).To(Equal(blobHash))

		By("Getting tree by path")
		treeByPath, err := client.GetTreeByPath(context.Background(), commit.Tree, "a/b/c")
		Expect(err).NotTo(HaveOccurred())
		Expect(treeByPath.Hash).To(Equal(flatTree.Entries[2].Hash))
		Expect(treeByPath.Entries).To(HaveLen(1))
		Expect(treeByPath.Entries[0].Name).To(Equal("test.txt"))
		Expect(treeByPath.Entries[0].Type).To(Equal(protocol.ObjectTypeBlob))
		Expect(treeByPath.Entries[0].Hash).To(Equal(blobHash))

		By("Getting tree by path")
		treeByPath, err = writer.GetTree(context.Background(), "a/b/c")
		Expect(err).NotTo(HaveOccurred())
		Expect(treeByPath.Hash).To(Equal(flatTree.Entries[2].Hash))
		Expect(treeByPath.Entries).To(HaveLen(1))
		Expect(treeByPath.Entries[0].Name).To(Equal("test.txt"))
		Expect(treeByPath.Entries[0].Type).To(Equal(protocol.ObjectTypeBlob))
		Expect(treeByPath.Entries[0].Hash).To(Equal(blobHash))

		By("Updating blob")
		blobHash, err = writer.UpdateBlob(context.Background(), "a/b/c/test.txt", []byte("updated content"))
		Expect(err).NotTo(HaveOccurred())
		_, err = writer.Commit(context.Background(), "Update test file", author, committer)
		Expect(err).NotTo(HaveOccurred())
		err = writer.Push(context.Background())
		Expect(err).NotTo(HaveOccurred())
		blob, err := client.GetBlob(context.Background(), blobHash)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(blob.Content)).To(Equal("updated content"))

		By("Deleting blob")
		_, err = writer.DeleteBlob(context.Background(), "a/b/c/test.txt")
		Expect(err).NotTo(HaveOccurred())

		deleteCommit, err := writer.Commit(context.Background(), "Delete test file", author, committer)
		Expect(err).NotTo(HaveOccurred())
		err = writer.Push(context.Background())
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetBlobByPath(context.Background(), deleteCommit.Tree, "a/b/c/test.txt")
		Expect(err).To(HaveOccurred())

		branchRef, err = client.GetRef(context.Background(), "refs/heads/"+branchName)
		Expect(err).NotTo(HaveOccurred())
		Expect(branchRef.Hash).To(Equal(deleteCommit.Hash))

		// TODO: Skip this does not work as expected for Github
		// commits, err := client.ListCommits(context.Background(), commit.Parent, nanogit.ListCommitsOptions{
		// 	Path: "a/b/c/test.txt",
		// })
		// Expect(err).NotTo(HaveOccurred())
		// Expect(commits).To(HaveLen(1))
		// Expect(commits[0].Hash).To(Equal(commit.Hash))
	})
})
