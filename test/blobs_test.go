//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Blobs(t *testing.T) {
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

	logger.Info("Creating and committing test file")
	testContent := []byte("test content")
	local.CreateFile(t, "test.txt", string(testContent))
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Getting blob hash", "file", "test.txt")
	blobHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD:test.txt"))
	require.NoError(t, err)

	logger.Info("Creating client")
	client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetBlob with valid hash", func(t *testing.T) {
		logger.Info("Testing GetBlob with valid hash", "hash", blobHash)
		blob, err := client.GetBlob(ctx, blobHash)
		require.NoError(t, err)
		assert.Equal(t, testContent, blob.Content)
		assert.Equal(t, blobHash, blob.Hash)
	})

	t.Run("GetBlob with non-existent hash", func(t *testing.T) {
		logger.Info("Testing GetBlob with non-existent hash")
		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		require.NoError(t, err)
		_, err = client.GetBlob(ctx, nonExistentHash)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
	})
}

func TestClient_GetBlobByPath(t *testing.T) {
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

	logger.Info("Creating and committing test file")
	testContent := []byte("test content")
	local.CreateFile(t, "test.txt", string(testContent))
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Tracking current branch")
	local.Git(t, "branch", "--set-upstream-to=origin/main", "main")

	client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Getting the commit hash")
	commitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	t.Run("GetBlobByPath with existing file", func(t *testing.T) {
		logger.ForSubtest(t)

		file, err := client.GetBlobByPath(ctx, commitHash, "test.txt")
		if err != nil {
			t.Logf("Failed to get file with hash %s and path %s: %v", commitHash, "test.txt", err)
		}
		require.NoError(t, err)
		assert.Equal(t, testContent, file.Content)

		fileHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD:test.txt"))
		require.NoError(t, err)
		assert.Equal(t, fileHash, file.Hash)
	})

	t.Run("GetBlobByPath with non-existent file", func(t *testing.T) {
		logger.ForSubtest(t)

		_, err := client.GetBlobByPath(ctx, commitHash, "nonexistent.txt")
		require.Error(t, err)
		// Check for structured PathNotFoundError
		var pathNotFoundErr *nanogit.PathNotFoundError
		require.ErrorAs(t, err, &pathNotFoundErr)
		assert.Equal(t, "nonexistent.txt", pathNotFoundErr.Path)
	})

	t.Run("GetBlobByPath with non-existent hash", func(t *testing.T) {
		logger.ForSubtest(t)

		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		require.NoError(t, err)
		_, err = client.GetBlobByPath(ctx, nonExistentHash, "test.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not our ref")
	})
}

func TestClient_GetBlobByPath_NestedDirectories(t *testing.T) {
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

	logger.Info("Creating nested directory structure with files")
	local.CreateDirPath(t, "dir1/subdir1")
	local.CreateDirPath(t, "dir1/subdir2")
	local.CreateDirPath(t, "dir2")

	// Create files at various levels
	rootContent := []byte("root file content")
	local.CreateFile(t, "root.txt", string(rootContent))

	dir1Content := []byte("dir1 file content")
	local.CreateFile(t, "dir1/file1.txt", string(dir1Content))

	nestedContent := []byte("deeply nested content")
	local.CreateFile(t, "dir1/subdir1/nested.txt", string(nestedContent))

	dir2Content := []byte("dir2 file content")
	local.CreateFile(t, "dir2/file2.txt", string(dir2Content))

	logger.Info("Adding and committing all files")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Initial commit with nested structure")

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")
	local.Git(t, "push", "origin", "main", "--force")

	client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Getting the commit hash")
	commitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		expected    []byte
		expectedErr error
	}{
		{
			name:        "root file",
			path:        "root.txt",
			expected:    rootContent,
			expectedErr: nil,
		},
		{
			name:        "file in first level directory",
			path:        "dir1/file1.txt",
			expected:    dir1Content,
			expectedErr: nil,
		},
		{
			name:        "deeply nested file",
			path:        "dir1/subdir1/nested.txt",
			expected:    nestedContent,
			expectedErr: nil,
		},
		{
			name:        "file in different directory",
			path:        "dir2/file2.txt",
			expected:    dir2Content,
			expectedErr: nil,
		},
		{
			name:        "nonexistent file in existing directory",
			path:        "dir1/nonexistent.txt",
			expected:    nil,
			expectedErr: &nanogit.PathNotFoundError{},
		},
		{
			name:        "file in nonexistent directory",
			path:        "nonexistent/file.txt",
			expected:    nil,
			expectedErr: &nanogit.PathNotFoundError{},
		},
		{
			name:        "empty path",
			path:        "",
			expected:    nil,
			expectedErr: &nanogit.PathNotFoundError{},
		},
		{
			name:        "path pointing to directory instead of file",
			path:        "dir1",
			expected:    nil,
			expectedErr: &nanogit.UnexpectedObjectTypeError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.ForSubtest(t)

			file, err := client.GetBlobByPath(ctx, commitHash, tt.path)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorAs(t, err, &tt.expectedErr)
				assert.Nil(t, file)
				return
			}

			require.NoError(t, err, "Failed to get file for path: %s", tt.path)
			assert.Equal(t, tt.expected, file.Content)

			// Verify the hash matches what Git CLI returns
			expectedHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD:"+tt.path))
			require.NoError(t, err)
			assert.Equal(t, expectedHash, file.Hash)
		})
	}
}
