package nanogit_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
		return
	}

	if os.Getenv("TEST_REPO") == "" || os.Getenv("TEST_TOKEN") == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO or TEST_TOKEN not set")
		return
	}

	client, err := nanogit.NewHTTPClient(
		os.Getenv("TEST_REPO"),
		nanogit.WithBasicAuth("git", os.Getenv("TEST_TOKEN")),
	)
	require.NoError(t, err)
	auth, err := client.IsAuthorized(context.Background())
	require.NoError(t, err)
	require.True(t, auth)

	exists, err := client.RepoExists(context.Background())
	require.NoError(t, err)
	require.True(t, exists)

	branchName := fmt.Sprintf("test-branch-%d", time.Now().Unix())
	mainRef, err := client.GetRef(context.Background(), "refs/heads/main")
	require.NoError(t, err)
	err = client.CreateRef(context.Background(), nanogit.Ref{
		Name: "refs/heads/" + branchName,
		Hash: mainRef.Hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err = client.DeleteRef(context.Background(), "refs/heads/"+branchName)
		require.NoError(t, err)
		refs, err := client.ListRefs(context.Background())
		require.NoError(t, err)
		require.NotContains(t, refs, nanogit.Ref{
			Name: "refs/heads/" + branchName,
		})
	})

	refs, err := client.ListRefs(context.Background())
	require.NoError(t, err)
	require.Contains(t, refs, nanogit.Ref{
		Name: "refs/heads/" + branchName,
		Hash: mainRef.Hash,
	})

	require.Contains(t, refs, nanogit.Ref{
		Name: "refs/heads/main",
		Hash: mainRef.Hash,
	})

	branchRef, err := client.GetRef(context.Background(), "refs/heads/"+branchName)
	require.NoError(t, err)

	writer, err := client.NewStagedWriter(context.Background(), branchRef)
	require.NoError(t, err)

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

	_, err = writer.DeleteTree(context.Background(), "")
	require.NoError(t, err)
	_, err = writer.Commit(context.Background(), "Delete everything", author, committer)
	require.NoError(t, err)

	exists, err = writer.BlobExists(context.Background(), "a/b/c/test.txt")
	require.NoError(t, err)
	require.False(t, exists)
	_, err = writer.GetTree(context.Background(), "a/b/c")
	require.Error(t, err)

	blobHash, err := writer.CreateBlob(context.Background(), "a/b/c/test.txt", []byte("test content"))
	require.NoError(t, err)

	tree, err := writer.GetTree(context.Background(), "a/b/c")
	require.NoError(t, err)
	require.Len(t, tree.Entries, 1)

	exists, err = writer.BlobExists(context.Background(), "a/b/c/test.txt")
	require.NoError(t, err)
	require.True(t, exists)

	commit, err := writer.Commit(context.Background(), "Add test file", author, committer)
	require.NoError(t, err)

	err = writer.Push(context.Background())
	require.NoError(t, err)

	branchRef, err = client.GetRef(context.Background(), "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, branchRef.Hash, commit.Hash)

	commit, err = client.GetCommit(context.Background(), commit.Hash)
	require.NoError(t, err)
	require.Equal(t, commit.Message, "Add test file")

	createdBlob, err := client.GetBlob(context.Background(), blobHash)
	require.NoError(t, err)
	require.Equal(t, string(createdBlob.Content), "test content")

	createdBlobByPath, err := client.GetBlobByPath(context.Background(), commit.Tree, "a/b/c/test.txt")
	require.NoError(t, err)
	require.Equal(t, string(createdBlobByPath.Content), "test content")

	// TODO: add check for other types of modifications and more commits in between
	// TODO: create a more complex tree
	// TODO: validate more fields
	compareCommits, err := client.CompareCommits(context.Background(), commit.Parent, commit.Hash)
	require.NoError(t, err)
	require.Len(t, compareCommits, 4)
	require.Equal(t, compareCommits[0].Path, "a")
	require.Equal(t, compareCommits[0].Status, protocol.FileStatusAdded)
	require.Equal(t, compareCommits[1].Path, "a/b")
	require.Equal(t, compareCommits[1].Status, protocol.FileStatusAdded)
	require.Equal(t, compareCommits[2].Path, "a/b/c")
	require.Equal(t, compareCommits[2].Status, protocol.FileStatusAdded)
	require.Equal(t, compareCommits[3].Path, "a/b/c/test.txt")
	require.Equal(t, compareCommits[3].Status, protocol.FileStatusAdded)

	flatTree, err := client.GetFlatTree(context.Background(), commit.Hash)
	require.NoError(t, err)
	require.Len(t, flatTree.Entries, 4)
	require.Equal(t, flatTree.Entries[0].Path, "a")
	require.Equal(t, flatTree.Entries[0].Type, protocol.ObjectTypeTree)
	require.Equal(t, flatTree.Entries[1].Path, "a/b")
	require.Equal(t, flatTree.Entries[1].Type, protocol.ObjectTypeTree)
	require.Equal(t, flatTree.Entries[2].Path, "a/b/c")
	require.Equal(t, flatTree.Entries[2].Type, protocol.ObjectTypeTree)
	require.Equal(t, flatTree.Entries[3].Path, "a/b/c/test.txt")
	require.Equal(t, flatTree.Entries[3].Type, protocol.ObjectTypeBlob)
	require.Equal(t, flatTree.Entries[3].Hash, blobHash)

	tree, err = client.GetTree(context.Background(), flatTree.Entries[2].Hash)
	require.NoError(t, err)
	require.Equal(t, tree.Hash, flatTree.Entries[2].Hash)
	require.Len(t, tree.Entries, 1)
	require.Equal(t, tree.Entries[0].Name, "test.txt")
	require.Equal(t, tree.Entries[0].Type, protocol.ObjectTypeBlob)
	require.Equal(t, tree.Entries[0].Hash, blobHash)

	treeByPath, err := client.GetTreeByPath(context.Background(), commit.Tree, "a/b/c")
	require.NoError(t, err)
	require.Equal(t, treeByPath.Hash, flatTree.Entries[2].Hash)
	require.Len(t, treeByPath.Entries, 1)
	require.Equal(t, treeByPath.Entries[0].Name, "test.txt")
	require.Equal(t, treeByPath.Entries[0].Type, protocol.ObjectTypeBlob)
	require.Equal(t, treeByPath.Entries[0].Hash, blobHash)

	treeByPath, err = writer.GetTree(context.Background(), "a/b/c")
	require.NoError(t, err)
	require.Equal(t, treeByPath.Hash, flatTree.Entries[2].Hash)
	require.Len(t, treeByPath.Entries, 1)
	require.Equal(t, treeByPath.Entries[0].Name, "test.txt")
	require.Equal(t, treeByPath.Entries[0].Type, protocol.ObjectTypeBlob)
	require.Equal(t, treeByPath.Entries[0].Hash, blobHash)

	blobHash, err = writer.UpdateBlob(context.Background(), "a/b/c/test.txt", []byte("updated content"))
	require.NoError(t, err)
	_, err = writer.Commit(context.Background(), "Update test file", author, committer)
	require.NoError(t, err)
	err = writer.Push(context.Background())
	require.NoError(t, err)
	blob, err := client.GetBlob(context.Background(), blobHash)
	require.NoError(t, err)
	require.Equal(t, string(blob.Content), "updated content")

	_, err = writer.DeleteBlob(context.Background(), "a/b/c/test.txt")
	require.NoError(t, err)

	deleteCommit, err := writer.Commit(context.Background(), "Delete test file", author, committer)
	require.NoError(t, err)
	err = writer.Push(context.Background())
	require.NoError(t, err)

	_, err = client.GetBlobByPath(context.Background(), deleteCommit.Tree, "a/b/c/test.txt")
	require.Error(t, err)

	branchRef, err = client.GetRef(context.Background(), "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, branchRef.Hash, deleteCommit.Hash)

	// TODO: Skip this does not work as expected for Github
	// commits, err := client.ListCommits(context.Background(), commit.Parent, nanogit.ListCommitsOptions{
	// 	Path: "a/b/c/test.txt",
	// })
	// require.NoError(t, err)
	// require.Len(t, commits, 1)
	// require.Equal(t, commits[0].Hash, commit.Hash)
}
