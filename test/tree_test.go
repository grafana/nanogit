//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetFlatTree(t *testing.T) {
	logger := helpers.NewTestLogger(t)
	logger.Info("Setting up remote repository")
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	logger.Info("Setting up local repository")
	local := helpers.NewLocalGitRepo(t, logger)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	logger.Info("Creating a directory structure with files")
	local.CreateDirPath(t, "dir1")
	local.CreateDirPath(t, "dir2")
	local.CreateFile(t, "dir1/file1.txt", "content1")
	local.CreateFile(t, "dir1/file2.txt", "content2")
	local.CreateFile(t, "dir2/file3.txt", "content3")
	local.CreateFile(t, "root.txt", "root content")

	logger.Info("Adding and committing the files")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Initial commit with tree structure")

	logger.Info("Creating and switching to main branch")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Getting the tree hash")
	treeHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD^{tree}"))
	require.NoError(t, err)

	client, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Testing GetTree")
	tree, err := client.GetFlatTree(ctx, treeHash)
	require.NoError(t, err)
	require.NotNil(t, tree)

	logger.Info("Helper to get the hash for a given path (file or directory)")
	getHash := func(path string) hash.Hash {
		out := local.Git(t, "rev-parse", "HEAD:"+path)
		h, err := hash.FromHex(out)
		require.NoError(t, err)
		return h
	}

	logger.Info("Defining expected entries with correct hashes")
	wantEntries := []nanogit.FlatTreeEntry{
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

	logger.Info("Verifying tree structure")
	require.Len(t, tree.Entries, len(wantEntries))

	logger.Info("Comparing entries using EqualElements")
	assert.ElementsMatch(t, wantEntries, tree.Entries, "Tree entries do not match expected values")

	logger.Info("Testing GetTree with non-existent hash")
	nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
	require.NoError(t, err)
	_, err = client.GetFlatTree(ctx, nonExistentHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not our ref")
}

func TestClient_GetTree(t *testing.T) {
	logger := helpers.NewTestLogger(t)
	logger.Info("Setting up remote repository")
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	logger.Info("Setting up local repository")
	local := helpers.NewLocalGitRepo(t, logger)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	logger.Info("Creating a directory structure with files")
	local.CreateDirPath(t, "dir1")
	local.CreateDirPath(t, "dir2")
	local.CreateFile(t, "dir1/file1.txt", "content1")
	local.CreateFile(t, "dir1/file2.txt", "content2")
	local.CreateFile(t, "dir2/file3.txt", "content3")
	local.CreateFile(t, "root.txt", "root content")

	logger.Info("Adding and committing the files")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Initial commit with tree structure")

	logger.Info("Creating and switching to main branch")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Getting the tree hash")
	treeHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD^{tree}"))
	require.NoError(t, err)

	client, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Testing GetTree")
	tree, err := client.GetTree(ctx, treeHash)
	require.NoError(t, err)
	require.NotNil(t, tree)

	logger.Info("Helper to get the hash for a given path (file or directory)")
	getHash := func(path string) hash.Hash {
		out := local.Git(t, "rev-parse", "HEAD:"+path)
		h, err := hash.FromHex(out)
		require.NoError(t, err)
		return h
	}

	logger.Info("Verifying tree entries (direct children only)")
	expectedEntries := map[string]nanogit.TreeEntry{
		"root.txt": {
			Name: "root.txt",
			Mode: 33188, // 100644 in octal
			Hash: getHash("root.txt"),
			Type: protocol.ObjectTypeBlob,
		},
		"dir1": {
			Name: "dir1",
			Mode: 16384, // 040000 in octal
			Hash: getHash("dir1"),
			Type: protocol.ObjectTypeTree,
		},
		"dir2": {
			Name: "dir2",
			Mode: 16384, // 040000 in octal
			Hash: getHash("dir2"),
			Type: protocol.ObjectTypeTree,
		},
	}

	logger.Info("Verifying tree structure")
	require.Len(t, tree.Entries, len(expectedEntries))

	for _, entry := range tree.Entries {
		expected, exists := expectedEntries[entry.Name]
		require.True(t, exists, "Unexpected entry: %s", entry.Name)
		assert.Equal(t, expected.Mode, entry.Mode)
		assert.Equal(t, expected.Type, entry.Type)
		assert.Equal(t, expected.Hash, entry.Hash)
	}

	logger.Info("Testing GetTree with non-existent hash")
	nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
	require.NoError(t, err)
	_, err = client.GetTree(ctx, nonExistentHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not our ref")
}

func TestClient_GetTreeByPath(t *testing.T) {
	logger := helpers.NewTestLogger(t)
	logger.Info("Setting up remote repository")
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	logger.Info("Setting up local repository")
	local := helpers.NewLocalGitRepo(t, logger)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	logger.Info("Creating a directory structure with files")
	local.CreateDirPath(t, "dir1")
	local.CreateDirPath(t, "dir2")
	local.CreateFile(t, "dir1/file1.txt", "content1")
	local.CreateFile(t, "dir1/file2.txt", "content2")
	local.CreateFile(t, "dir2/file3.txt", "content3")
	local.CreateFile(t, "root.txt", "root content")

	logger.Info("Adding and committing the files")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Initial commit with tree structure")

	logger.Info("Creating and switching to main branch")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Getting the tree hash")
	treeHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD^{tree}"))
	require.NoError(t, err)

	client, err := nanogit.NewClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Helper to get the hash for a given path (file or directory)")
	getHash := func(path string) hash.Hash {
		out := local.Git(t, "rev-parse", "HEAD:"+path)
		h, err := hash.FromHex(out)
		require.NoError(t, err)
		return h
	}

	testCases := []struct {
		name          string
		path          string
		expectedError string
		verifyFunc    func(tree *nanogit.Tree)
	}{
		{
			name: "get root tree with empty path",
			path: "",
			verifyFunc: func(tree *nanogit.Tree) {
				require.Len(t, tree.Entries, 3) // root.txt, dir1, dir2
				entryNames := make([]string, len(tree.Entries))
				for i, entry := range tree.Entries {
					entryNames[i] = entry.Name
				}
				assert.ElementsMatch(t, []string{"root.txt", "dir1", "dir2"}, entryNames)
			},
		},
		{
			name: "get root tree with dot path",
			path: ".",
			verifyFunc: func(tree *nanogit.Tree) {
				require.Len(t, tree.Entries, 3)
			},
		},
		{
			name: "get dir1 subdirectory",
			path: "dir1",
			verifyFunc: func(tree *nanogit.Tree) {
				require.Len(t, tree.Entries, 2) // file1.txt, file2.txt
				entryNames := make([]string, len(tree.Entries))
				for i, entry := range tree.Entries {
					entryNames[i] = entry.Name
				}
				assert.ElementsMatch(t, []string{"file1.txt", "file2.txt"}, entryNames)
				assert.Equal(t, getHash("dir1"), tree.Hash)
			},
		},
		{
			name: "get dir2 subdirectory",
			path: "dir2",
			verifyFunc: func(tree *nanogit.Tree) {
				require.Len(t, tree.Entries, 1) // file3.txt
				assert.Equal(t, "file3.txt", tree.Entries[0].Name)
				assert.Equal(t, getHash("dir2"), tree.Hash)
			},
		},
		{
			name:          "nonexistent path",
			path:          "nonexistent",
			expectedError: "path component 'nonexistent' not found",
		},
		{
			name:          "path to file instead of directory",
			path:          "root.txt",
			expectedError: "path component 'root.txt' is not a directory",
		},
		{
			name:          "nested nonexistent path",
			path:          "dir1/nonexistent",
			expectedError: "path component 'nonexistent' not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.ForSubtest(t)

			tree, err := client.GetTreeByPath(ctx, treeHash, tc.path)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, tree)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tree)
			tc.verifyFunc(tree)
		})
	}
}
