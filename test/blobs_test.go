package integration_test

import (
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// TestBasicBlobOperations tests basic blob operations like GetBlob
func (s *IntegrationTestSuite) TestGetBlob() {
	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	s.Logger.Info("Creating and committing test file")
	testContent := []byte("test content")
	local.CreateFile(s.T(), "test.txt", string(testContent))
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")

	s.Logger.Info("Setting up main branch and pushing changes")
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Getting blob hash", "file", "test.txt")
	blobHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD:test.txt"))
	s.NoError(err)

	s.Logger.Info("Creating client")
	client := remote.Client(s.T())

	s.Run("GetBlob with valid hash", func() {

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Testing GetBlob with valid hash", "hash", blobHash)
		blob, err := client.GetBlob(ctx, blobHash)
		s.NoError(err)
		s.Equal(testContent, blob.Content)
		s.Equal(blobHash, blob.Hash)
	})

	s.Run("GetBlob with non-existent hash", func() {

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Testing GetBlob with non-existent hash")
		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		s.NoError(err)
		_, err = client.GetBlob(ctx, nonExistentHash)
		s.Error(err)
		s.Contains(err.Error(), "not our ref b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
	})
}

// TestGetBlobByPath tests getting blobs by file paths
func (s *IntegrationTestSuite) TestGetBlobByPath() {
	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	s.Logger.Info("Creating and committing test file")
	testContent := []byte("test content")
	local.CreateFile(s.T(), "test.txt", string(testContent))
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")

	s.Logger.Info("Setting up main branch and pushing changes")
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Tracking current branch")
	local.Git(s.T(), "branch", "--set-upstream-to=origin/main", "main")

	client := remote.Client(s.T())

	s.Logger.Info("Getting the commit hash")
	commitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	s.Run("GetBlobByPath with existing file", func() {

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		file, err := client.GetBlobByPath(ctx, commitHash, "test.txt")
		if err != nil {
			s.T().Logf("Failed to get file with hash %s and path %s: %v", commitHash, "test.txt", err)
		}
		s.NoError(err)
		s.Equal(testContent, file.Content)

		fileHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD:test.txt"))
		s.NoError(err)
		s.Equal(fileHash, file.Hash)
	})

	s.Run("GetBlobByPath with non-existent file", func() {

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		_, err := client.GetBlobByPath(ctx, commitHash, "nonexistent.txt")
		s.Error(err)
		// Check for structured PathNotFoundError
		var pathNotFoundErr *nanogit.PathNotFoundError
		s.ErrorAs(err, &pathNotFoundErr)
		s.Equal("nonexistent.txt", pathNotFoundErr.Path)
	})

	s.Run("GetBlobByPath with non-existent hash", func() {

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		s.NoError(err)
		_, err = client.GetBlobByPath(ctx, nonExistentHash, "test.txt")
		s.Error(err)
		s.Contains(err.Error(), "not our ref")
	})
}

// TestGetBlobByPathNestedDirectories tests GetBlobByPath with nested directory structures
func (s *IntegrationTestSuite) TestGetBlobByPathNestedDirectories() {
	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	s.Logger.Info("Creating nested directory structure with files")
	local.CreateDirPath(s.T(), "dir1/subdir1")
	local.CreateDirPath(s.T(), "dir1/subdir2")
	local.CreateDirPath(s.T(), "dir2")

	// Create files at various levels
	rootContent := []byte("root file content")
	local.CreateFile(s.T(), "root.txt", string(rootContent))

	dir1Content := []byte("dir1 file content")
	local.CreateFile(s.T(), "dir1/file1.txt", string(dir1Content))

	nestedContent := []byte("deeply nested content")
	local.CreateFile(s.T(), "dir1/subdir1/nested.txt", string(nestedContent))

	dir2Content := []byte("dir2 file content")
	local.CreateFile(s.T(), "dir2/file2.txt", string(dir2Content))

	s.Logger.Info("Adding and committing all files")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with nested structure")

	s.Logger.Info("Setting up main branch and pushing changes")
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "origin", "main", "--force")

	client := remote.Client(s.T())

	s.Logger.Info("Getting the commit hash")
	commitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	tests := []struct {
		name        string
		path        string
		expected    []byte
		expectedErr interface{}
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
		s.Run(tt.name, func() {

			ctx, cancel := s.CreateContext(s.StandardTimeout())
			defer cancel()

			file, err := client.GetBlobByPath(ctx, commitHash, tt.path)
			if tt.expectedErr != nil {
				s.Error(err)
				s.ErrorAs(err, &tt.expectedErr)
				s.Nil(file)
				return
			}

			s.NoError(err, "Failed to get file for path: %s", tt.path)
			s.Equal(tt.expected, file.Content)

			// Verify the hash matches what Git CLI returns
			expectedHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD:"+tt.path))
			s.NoError(err)
			s.Equal(expectedHash, file.Hash)
		})
	}
}
