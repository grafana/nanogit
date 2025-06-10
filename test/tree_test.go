package integration_test

import (
	"context"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// TestGetFlatTree tests getting a flat tree structure with all entries including nested ones
func (s *IntegrationTestSuite) TestGetFlatTree() {
	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	s.Logger.Info("Creating a directory structure with files")
	local.CreateDirPath(s.T(), "dir1")
	local.CreateDirPath(s.T(), "dir2")
	local.CreateFile(s.T(), "dir1/file1.txt", "content1")
	local.CreateFile(s.T(), "dir1/file2.txt", "content2")
	local.CreateFile(s.T(), "dir2/file3.txt", "content3")
	local.CreateFile(s.T(), "root.txt", "root content")

	s.Logger.Info("Adding and committing the files")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with tree structure")

	s.Logger.Info("Creating and switching to main branch")
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Getting the commit hash")
	commitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	client := remote.Client(s.T())

	s.Logger.Info("Helper to get the hash for a given path (file or directory)")
	getHash := func(path string) hash.Hash {
		out := local.Git(s.T(), "rev-parse", "HEAD:"+path)
		h, err := hash.FromHex(out)
		s.NoError(err)
		return h
	}

	s.Run("successful flat tree retrieval", func() {

		s.Logger.Info("Testing GetFlatTree")
		tree, err := client.GetFlatTree(context.Background(), commitHash)
		s.NoError(err)
		s.NotNil(tree)

		s.Logger.Info("Defining expected entries with correct hashes")
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

		s.Logger.Info("Verifying tree structure")
		s.Len(tree.Entries, len(wantEntries))

		s.Logger.Info("Comparing entries using ElementsMatch")
		s.ElementsMatch(wantEntries, tree.Entries, "Tree entries do not match expected values")
	})

	s.Run("non-existent hash", func() {

		s.Logger.Info("Testing GetFlatTree with non-existent hash")
		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		s.NoError(err)
		_, err = client.GetFlatTree(context.Background(), nonExistentHash)
		s.Error(err)
		s.Contains(err.Error(), "not our ref")
	})
}

// TestGetTree tests getting a tree structure with only direct children
func (s *IntegrationTestSuite) TestGetTree() {
	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	s.Logger.Info("Creating a directory structure with files")
	local.CreateDirPath(s.T(), "dir1")
	local.CreateDirPath(s.T(), "dir2")
	local.CreateFile(s.T(), "dir1/file1.txt", "content1")
	local.CreateFile(s.T(), "dir1/file2.txt", "content2")
	local.CreateFile(s.T(), "dir2/file3.txt", "content3")
	local.CreateFile(s.T(), "root.txt", "root content")

	s.Logger.Info("Adding and committing the files")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with tree structure")

	s.Logger.Info("Creating and switching to main branch")
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Getting the tree hash")
	treeHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD^{tree}"))
	s.NoError(err)

	client := remote.Client(s.T())

	s.Logger.Info("Helper to get the hash for a given path (file or directory)")
	getHash := func(path string) hash.Hash {
		out := local.Git(s.T(), "rev-parse", "HEAD:"+path)
		h, err := hash.FromHex(out)
		s.NoError(err)
		return h
	}

	s.Run("successful tree retrieval", func() {

		s.Logger.Info("Testing GetTree")
		tree, err := client.GetTree(context.Background(), treeHash)
		s.NoError(err)
		s.NotNil(tree)

		s.Logger.Info("Verifying tree entries (direct children only)")
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

		s.Logger.Info("Verifying tree structure")
		s.Len(tree.Entries, len(expectedEntries))

		for _, entry := range tree.Entries {
			expected, exists := expectedEntries[entry.Name]
			s.True(exists, "Unexpected entry: %s", entry.Name)
			s.Equal(expected.Mode, entry.Mode)
			s.Equal(expected.Type, entry.Type)
			s.Equal(expected.Hash, entry.Hash)
		}
	})

	s.Run("non-existent hash", func() {

		s.Logger.Info("Testing GetTree with non-existent hash")
		nonExistentHash, err := hash.FromHex("b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0")
		s.NoError(err)
		_, err = client.GetTree(context.Background(), nonExistentHash)
		s.Error(err)
		s.Contains(err.Error(), "not our ref")
	})
}

// TestGetTreeByPath tests getting trees by path with various scenarios
func (s *IntegrationTestSuite) TestGetTreeByPath() {
	s.Logger.Info("Setting up remote repository")
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	s.Logger.Info("Creating a directory structure with files")
	local.CreateDirPath(s.T(), "dir1")
	local.CreateDirPath(s.T(), "dir2")
	local.CreateFile(s.T(), "dir1/file1.txt", "content1")
	local.CreateFile(s.T(), "dir1/file2.txt", "content2")
	local.CreateFile(s.T(), "dir2/file3.txt", "content3")
	local.CreateFile(s.T(), "root.txt", "root content")

	s.Logger.Info("Adding and committing the files")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Initial commit with tree structure")

	s.Logger.Info("Creating and switching to main branch")
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Getting the tree hash")
	treeHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD^{tree}"))
	s.NoError(err)

	client := remote.Client(s.T())

	s.Logger.Info("Helper to get the hash for a given path (file or directory)")
	getHash := func(path string) hash.Hash {
		out := local.Git(s.T(), "rev-parse", "HEAD:"+path)
		h, err := hash.FromHex(out)
		s.NoError(err)
		return h
	}

	testCases := []struct {
		name          string
		path          string
		expectedError interface{}
		verifyFunc    func(tree *nanogit.Tree)
	}{
		{
			name: "get root tree with empty path",
			path: "",
			verifyFunc: func(tree *nanogit.Tree) {
				s.Len(tree.Entries, 3) // root.txt, dir1, dir2
				entryNames := make([]string, len(tree.Entries))
				for i, entry := range tree.Entries {
					entryNames[i] = entry.Name
				}
				s.ElementsMatch([]string{"root.txt", "dir1", "dir2"}, entryNames)
			},
		},
		{
			name: "get root tree with dot path",
			path: ".",
			verifyFunc: func(tree *nanogit.Tree) {
				s.Len(tree.Entries, 3)
			},
		},
		{
			name: "get dir1 subdirectory",
			path: "dir1",
			verifyFunc: func(tree *nanogit.Tree) {
				s.Len(tree.Entries, 2) // file1.txt, file2.txt
				entryNames := make([]string, len(tree.Entries))
				for i, entry := range tree.Entries {
					entryNames[i] = entry.Name
				}
				s.ElementsMatch([]string{"file1.txt", "file2.txt"}, entryNames)
				s.Equal(getHash("dir1"), tree.Hash)
			},
		},
		{
			name: "get dir2 subdirectory",
			path: "dir2",
			verifyFunc: func(tree *nanogit.Tree) {
				s.Len(tree.Entries, 1) // file3.txt
				s.Equal("file3.txt", tree.Entries[0].Name)
				s.Equal(getHash("dir2"), tree.Hash)
			},
		},
		{
			name:          "nonexistent path",
			path:          "nonexistent",
			expectedError: &nanogit.PathNotFoundError{},
		},
		{
			name:          "path to file instead of directory",
			path:          "root.txt",
			expectedError: &nanogit.UnexpectedObjectTypeError{},
		},
		{
			name:          "nested nonexistent path",
			path:          "dir1/nonexistent",
			expectedError: &nanogit.PathNotFoundError{},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tree, err := client.GetTreeByPath(context.Background(), treeHash, tc.path)
			if tc.expectedError != nil {
				s.Error(err)
				s.ErrorAs(err, &tc.expectedError)
				s.Nil(tree)
				return
			}

			s.NoError(err)
			s.NotNil(tree)
			tc.verifyFunc(tree)
		})
	}
}
