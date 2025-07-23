package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
		return
	}

	if os.Getenv("TEST_REPO") == "" || os.Getenv("TEST_TOKEN") == "" || os.Getenv("TEST_USER") == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO or TEST_TOKEN or TEST_USER or TEST_USER not set")
		return
	}

	ctx := log.ToContext(context.Background(), NewTestLogger(t.Logf))
	client, err := nanogit.NewHTTPClient(
		os.Getenv("TEST_REPO"),
		options.WithBasicAuth(os.Getenv("TEST_USER"), os.Getenv("TEST_TOKEN")),
	)
	require.NoError(t, err)
	auth, err := client.IsAuthorized(ctx)
	require.NoError(t, err)
	require.True(t, auth)

	exists, err := client.RepoExists(ctx)
	require.NoError(t, err)
	require.True(t, exists)

	branchName := fmt.Sprintf("test-branch-%d", time.Now().Unix())
	mainRef, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)
	err = client.CreateRef(ctx, nanogit.Ref{
		Name: "refs/heads/" + branchName,
		Hash: mainRef.Hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err = client.DeleteRef(ctx, "refs/heads/"+branchName)
		require.NoError(t, err)
		refs, err := client.ListRefs(ctx)
		require.NoError(t, err)
		require.NotContains(t, refs, nanogit.Ref{
			Name: "refs/heads/" + branchName,
		})
	})

	refs, err := client.ListRefs(ctx)
	require.NoError(t, err)
	require.Contains(t, refs, nanogit.Ref{
		Name: "refs/heads/" + branchName,
		Hash: mainRef.Hash,
	})

	require.Contains(t, refs, nanogit.Ref{
		Name: "refs/heads/main",
		Hash: mainRef.Hash,
	})

	branchRef, err := client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)

	writer, err := client.NewStagedWriter(ctx, branchRef)
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

	exists, err = writer.BlobExists(ctx, "a/b/c/test.txt")
	require.NoError(t, err)
	require.False(t, exists)
	_, err = writer.GetTree(ctx, "a/b/c")
	require.Error(t, err)

	blobHash, err := writer.CreateBlob(ctx, "a/b/c/test.txt", []byte("test content"))
	require.NoError(t, err)

	tree, err := writer.GetTree(ctx, "a/b/c")
	require.NoError(t, err)
	require.Len(t, tree.Entries, 1)

	exists, err = writer.BlobExists(ctx, "a/b/c/test.txt")
	require.NoError(t, err)
	require.True(t, exists)

	commit, err := writer.Commit(ctx, "Add test file", author, committer)
	require.NoError(t, err)

	err = writer.Push(ctx)
	require.NoError(t, err)

	branchRef, err = client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, commit.Hash, branchRef.Hash)

	commit, err = client.GetCommit(ctx, commit.Hash)
	require.NoError(t, err)
	require.Equal(t, "Add test file", commit.Message)

	createdBlob, err := client.GetBlob(ctx, blobHash)
	require.NoError(t, err)
	require.Equal(t, "test content", string(createdBlob.Content))

	createdBlobByPath, err := client.GetBlobByPath(ctx, commit.Tree, "a/b/c/test.txt")
	require.NoError(t, err)
	require.Equal(t, "test content", string(createdBlobByPath.Content))

	// TODO: add check for other types of modifications and more commits in between
	// TODO: create a more complex tree
	// TODO: validate more fields
	compareCommits, err := client.CompareCommits(ctx, commit.Parent, commit.Hash)
	require.NoError(t, err)
	require.Len(t, compareCommits, 4)
	require.Equal(t, "a", compareCommits[0].Path)
	require.Equal(t, protocol.FileStatusAdded, compareCommits[0].Status)
	require.Equal(t, "a/b", compareCommits[1].Path)
	require.Equal(t, protocol.FileStatusAdded, compareCommits[1].Status)
	require.Equal(t, "a/b/c", compareCommits[2].Path)
	require.Equal(t, protocol.FileStatusAdded, compareCommits[2].Status)
	require.Equal(t, "a/b/c/test.txt", compareCommits[3].Path)
	require.Equal(t, protocol.FileStatusAdded, compareCommits[3].Status)

	flatTree, err := client.GetFlatTree(ctx, commit.Hash)
	require.NoError(t, err)
	require.Len(t, flatTree.Entries, 4)
	require.Equal(t, "a", flatTree.Entries[0].Path)
	require.Equal(t, protocol.ObjectTypeTree, flatTree.Entries[0].Type)
	require.Equal(t, "a/b", flatTree.Entries[1].Path)
	require.Equal(t, protocol.ObjectTypeTree, flatTree.Entries[1].Type)
	require.Equal(t, "a/b/c", flatTree.Entries[2].Path)
	require.Equal(t, protocol.ObjectTypeTree, flatTree.Entries[2].Type)
	require.Equal(t, "a/b/c/test.txt", flatTree.Entries[3].Path)
	require.Equal(t, protocol.ObjectTypeBlob, flatTree.Entries[3].Type)
	require.Equal(t, flatTree.Entries[3].Hash, blobHash)

	tree, err = client.GetTree(ctx, flatTree.Entries[2].Hash)
	require.NoError(t, err)
	require.Equal(t, tree.Hash, flatTree.Entries[2].Hash)
	require.Len(t, tree.Entries, 1)
	require.Equal(t, "test.txt", tree.Entries[0].Name)
	require.Equal(t, protocol.ObjectTypeBlob, tree.Entries[0].Type)
	require.Equal(t, blobHash, tree.Entries[0].Hash)

	treeByPath, err := client.GetTreeByPath(ctx, commit.Tree, "a/b/c")
	require.NoError(t, err)
	require.Equal(t, flatTree.Entries[2].Hash, treeByPath.Hash)
	require.Len(t, treeByPath.Entries, 1)
	require.Equal(t, "test.txt", treeByPath.Entries[0].Name)
	require.Equal(t, protocol.ObjectTypeBlob, treeByPath.Entries[0].Type)
	require.Equal(t, blobHash, treeByPath.Entries[0].Hash)

	treeByPath, err = writer.GetTree(ctx, "a/b/c")
	require.NoError(t, err)
	require.Equal(t, flatTree.Entries[2].Hash, treeByPath.Hash)
	require.Len(t, treeByPath.Entries, 1)
	require.Equal(t, "test.txt", treeByPath.Entries[0].Name)
	require.Equal(t, protocol.ObjectTypeBlob, treeByPath.Entries[0].Type)
	require.Equal(t, blobHash, treeByPath.Entries[0].Hash)

	blobHash, err = writer.UpdateBlob(ctx, "a/b/c/test.txt", []byte("updated content"))
	require.NoError(t, err)
	updateCommit, err := writer.Commit(ctx, "Update test file", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)
	blob, err := client.GetBlob(ctx, blobHash)
	require.NoError(t, err)
	require.Equal(t, "updated content", string(blob.Content))

	_, err = writer.DeleteBlob(ctx, "a/b/c/test.txt")
	require.NoError(t, err)

	deleteCommit, err := writer.Commit(ctx, "Delete test file", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)

	_, err = client.GetBlobByPath(ctx, deleteCommit.Tree, "a/b/c/test.txt")
	require.Error(t, err)

	// Test MoveBlob operation
	_, err = writer.CreateBlob(ctx, "a/b/c/test.txt", []byte("content for move"))
	require.NoError(t, err)
	createAfterDeleteCommit, err := writer.Commit(ctx, "Recreate test file for move", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)

	// Move the file to a new location
	movedBlobHash, err := writer.MoveBlob(ctx, "a/b/c/test.txt", "a/moved-test.txt")
	require.NoError(t, err)
	moveCommit, err := writer.Commit(ctx, "Move test file", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)

	// Verify original location no longer exists
	_, err = client.GetBlobByPath(ctx, moveCommit.Tree, "a/b/c/test.txt")
	require.Error(t, err)

	// Verify new location exists with correct content
	movedBlob, err := client.GetBlobByPath(ctx, moveCommit.Tree, "a/moved-test.txt")
	require.NoError(t, err)
	require.Equal(t, "content for move", string(movedBlob.Content))
	require.Equal(t, movedBlobHash, movedBlob.Hash)

	// Verify the blob hash is the same (moved, not copied)
	retrievedMovedBlob, err := client.GetBlob(ctx, movedBlobHash)
	require.NoError(t, err)
	require.Equal(t, "content for move", string(retrievedMovedBlob.Content))

	// Test MoveTree operation
	_, err = writer.CreateBlob(ctx, "tree-test/subdir/file1.txt", []byte("tree file 1"))
	require.NoError(t, err)
	_, err = writer.CreateBlob(ctx, "tree-test/subdir/file2.txt", []byte("tree file 2"))
	require.NoError(t, err)
	_, err = writer.CreateBlob(ctx, "tree-test/file3.txt", []byte("tree file 3"))
	require.NoError(t, err)
	createTreeCommit, err := writer.Commit(ctx, "Create tree structure for move test", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)

	// Move the entire directory tree
	movedTreeHash, err := writer.MoveTree(ctx, "tree-test", "moved-tree")
	require.NoError(t, err)
	moveTreeCommit, err := writer.Commit(ctx, "Move directory tree", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)

	// Verify original tree location no longer exists
	_, err = client.GetTreeByPath(ctx, moveTreeCommit.Tree, "tree-test")
	require.Error(t, err)

	// Verify new tree location exists with correct structure
	movedTreeObj, err := client.GetTreeByPath(ctx, moveTreeCommit.Tree, "moved-tree")
	require.NoError(t, err)
	require.Equal(t, movedTreeHash, movedTreeObj.Hash)
	require.Len(t, movedTreeObj.Entries, 2) // subdir and file3.txt

	// Verify all files moved to new location with correct content
	movedFile1, err := client.GetBlobByPath(ctx, moveTreeCommit.Tree, "moved-tree/subdir/file1.txt")
	require.NoError(t, err)
	require.Equal(t, "tree file 1", string(movedFile1.Content))

	movedFile2, err := client.GetBlobByPath(ctx, moveTreeCommit.Tree, "moved-tree/subdir/file2.txt")
	require.NoError(t, err)
	require.Equal(t, "tree file 2", string(movedFile2.Content))

	movedFile3, err := client.GetBlobByPath(ctx, moveTreeCommit.Tree, "moved-tree/file3.txt")
	require.NoError(t, err)
	require.Equal(t, "tree file 3", string(movedFile3.Content))

	// Verify original file locations no longer exist
	_, err = client.GetBlobByPath(ctx, moveTreeCommit.Tree, "tree-test/subdir/file1.txt")
	require.Error(t, err)
	_, err = client.GetBlobByPath(ctx, moveTreeCommit.Tree, "tree-test/file3.txt")
	require.Error(t, err)

	branchRef, err = client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, moveTreeCommit.Hash, branchRef.Hash)

	// List commits without options
	commits, err := client.ListCommits(ctx, moveTreeCommit.Hash, nanogit.ListCommitsOptions{})
	require.NoError(t, err)
	require.Len(t, commits, 8)
	require.Equal(t, moveTreeCommit.Hash, commits[0].Hash)
	require.Equal(t, "Move directory tree", commits[0].Message)
	require.Equal(t, createTreeCommit.Hash, commits[1].Hash)
	require.Equal(t, "Create tree structure for move test", commits[1].Message)
	require.Equal(t, moveCommit.Hash, commits[2].Hash)
	require.Equal(t, "Move test file", commits[2].Message)
	require.Equal(t, createAfterDeleteCommit.Hash, commits[3].Hash)
	require.Equal(t, "Recreate test file for move", commits[3].Message)
	require.Equal(t, deleteCommit.Hash, commits[4].Hash)
	require.Equal(t, "Delete test file", commits[4].Message)
	require.Equal(t, updateCommit.Hash, commits[5].Hash)
	require.Equal(t, "Update test file", commits[5].Message)
	require.Equal(t, commit.Hash, commits[6].Hash)
	require.Equal(t, "Add test file", commits[6].Message)
	require.Equal(t, commit.Parent, commits[7].Hash)

	// List commits with path filter - should include move, create after delete, delete, update, and create
	commits, err = client.ListCommits(ctx, moveTreeCommit.Hash, nanogit.ListCommitsOptions{
		Path: "a/b/c/test.txt",
	})
	require.NoError(t, err)
	require.Len(t, commits, 4) // move (delete), create after delete, delete, update, create
	require.Equal(t, moveCommit.Hash, commits[0].Hash)
	require.Equal(t, "Move test file", commits[0].Message)
	require.Equal(t, createAfterDeleteCommit.Hash, commits[1].Hash)
	require.Equal(t, "Recreate test file for move", commits[1].Message)
	require.Equal(t, deleteCommit.Hash, commits[2].Hash)
	require.Equal(t, "Delete test file", commits[2].Message)
	require.Equal(t, updateCommit.Hash, commits[3].Hash)
	require.Equal(t, "Update test file", commits[3].Message)

	// List only last N commits for path
	// add a couple of commits in between
	_, err = writer.CreateBlob(ctx, "a/b/c/test2.txt", []byte("test content 2"))
	require.NoError(t, err)
	_, err = writer.Commit(ctx, "Add test file 2", author, committer)
	require.NoError(t, err)

	_, err = writer.CreateBlob(ctx, "a/b/c/test3.txt", []byte("test content 3"))
	require.NoError(t, err)
	lastCommit, err := writer.Commit(ctx, "Add test file 3", author, committer)
	require.NoError(t, err)
	err = writer.Push(ctx)
	require.NoError(t, err)

	commits, err = client.ListCommits(ctx, lastCommit.Hash, nanogit.ListCommitsOptions{
		PerPage: 2,
		Page:    1,
		Path:    "a/b/c/test.txt",
	})
	require.NoError(t, err)
	require.Len(t, commits, 2)
	require.Equal(t, moveCommit.Hash, commits[0].Hash)
	require.Equal(t, "Move test file", commits[0].Message)
	require.Equal(t, createAfterDeleteCommit.Hash, commits[1].Hash)
	require.Equal(t, "Recreate test file for move", commits[1].Message)
}

func TestProvidersLargeBlob(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
		return
	}

	if os.Getenv("TEST_REPO") == "" || os.Getenv("TEST_TOKEN") == "" || os.Getenv("TEST_USER") == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO or TEST_TOKEN or TEST_USER not set")
		return
	}

	ctx := log.ToContext(context.Background(), NewTestLogger(t.Logf))
	client, err := nanogit.NewHTTPClient(
		os.Getenv("TEST_REPO"),
		options.WithBasicAuth(os.Getenv("TEST_USER"), os.Getenv("TEST_TOKEN")),
	)
	require.NoError(t, err)

	auth, err := client.IsAuthorized(ctx)
	require.NoError(t, err)
	require.True(t, auth)

	exists, err := client.RepoExists(ctx)
	require.NoError(t, err)
	require.True(t, exists)

	// Read the xlarge dashboard file
	dashboardPath := filepath.Join("..", "perf", "cmd", "generate_dashboards", "generated_dashboards", "xlarge-dashboard.json")
	dashboardContent, err := os.ReadFile(dashboardPath)
	require.NoError(t, err)
	require.Greater(t, len(dashboardContent), 3000000, "Dashboard should be larger than 3MB")

	branchName := fmt.Sprintf("test-large-blob-%d", time.Now().Unix())
	mainRef, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)

	err = client.CreateRef(ctx, nanogit.Ref{
		Name: "refs/heads/" + branchName,
		Hash: mainRef.Hash,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = client.DeleteRef(ctx, "refs/heads/"+branchName)
		require.NoError(t, err)
	})

	branchRef, err := client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)

	writer, err := client.NewStagedWriter(ctx, branchRef)
	require.NoError(t, err)

	author := nanogit.Author{
		Name:  "Test User",
		Email: "test@example.com",
		Time:  time.Now(),
	}
	committer := nanogit.Committer{
		Name:  "Test User",
		Email: "test@example.com",
		Time:  time.Now(),
	}

	t.Log("Creating large blob (3.7MB dashboard)...")
	blobHash, err := writer.CreateBlob(ctx, "xlarge-dashboard.json", dashboardContent)
	require.NoError(t, err)

	t.Log("Committing large blob...")
	commit, err := writer.Commit(ctx, "Add xlarge dashboard for large blob testing", author, committer)
	require.NoError(t, err)

	t.Log("Pushing to remote repository...")
	err = writer.Push(ctx)
	require.NoError(t, err)

	t.Log("Verifying commit was pushed...")
	branchRef, err = client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, commit.Hash, branchRef.Hash)

	t.Log("Testing GetBlob with large blob...")
	retrievedBlob, err := client.GetBlob(ctx, blobHash)
	require.NoError(t, err)
	require.Equal(t, dashboardContent, retrievedBlob.Content)
	require.Equal(t, blobHash, retrievedBlob.Hash)
	require.Greater(t, len(retrievedBlob.Content), 3000000, "Retrieved blob should maintain size")

	t.Log("Testing GetBlobByPath with large blob...")
	retrievedFile, err := client.GetBlobByPath(ctx, commit.Tree, "xlarge-dashboard.json")
	require.NoError(t, err)
	require.Equal(t, dashboardContent, retrievedFile.Content)
	require.Equal(t, blobHash, retrievedFile.Hash)
	require.Greater(t, len(retrievedFile.Content), 3000000, "Retrieved file should maintain size")

	t.Log("Verifying content is valid JSON dashboard...")
	contentStr := string(retrievedFile.Content)
	require.Contains(t, contentStr, "\"title\"")
	require.Contains(t, contentStr, "\"panels\"")
	require.Contains(t, contentStr, "\"templating\"")
	require.True(t,
		contentStr == string(dashboardContent) &&
			(len(contentStr) > 3000000),
		"Content should match original and be large")

	t.Logf("Successfully tested large blob operations with %d bytes", len(retrievedFile.Content))
}
