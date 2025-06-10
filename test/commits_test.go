package integration_test

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// TestGetCommit tests retrieving individual commits
func (s *IntegrationTestSuite) TestGetCommit() {
	// Set up remote repo
	s.Logger.Info("Setting up remote repository")
	client, remote, local := s.TestRepo()

	initialCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	// Create create file commit
	s.Logger.Info("Creating create file commit")
	local.CreateFile(s.T(), "new.txt", "initial content")
	local.Git(s.T(), "add", "new.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	createFileCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	// Create second commit that modifies the file
	s.Logger.Info("Creating modify file commit")
	local.CreateFile(s.T(), "new.txt", "modified content")
	local.Git(s.T(), "add", "new.txt")
	local.Git(s.T(), "commit", "-m", "Modify file")
	modifyFileCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	// Create third commit that renames the file
	s.Logger.Info("Creating rename file commit")
	local.Git(s.T(), "mv", "new.txt", "renamed.txt")
	local.CreateFile(s.T(), "new.txt", "modified content")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Rename and add files")
	renameCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	// Push commit
	local.Git(s.T(), "push", "origin", "main", "--force")

	user := remote.User

	s.Run("initial commit", func() {
		commit, err := client.GetCommit(context.Background(), initialCommitHash)
		s.NoError(err)

		// Verify commit details
		s.Equal(initialCommitHash, commit.Hash)
		s.Equal(hash.Zero, commit.Parent) // First commit has no parent
		s.Equal(user.Username, commit.Author.Name)
		s.Equal(user.Email, commit.Author.Email)
		s.NotZero(commit.Author.Time)

		// Check that commit times are recent (within 5 seconds)
		now := time.Now()
		s.InDelta(now.Unix(), commit.Committer.Time.Unix(), 5)
		s.InDelta(now.Unix(), commit.Author.Time.Unix(), 5)
	})

	s.Run("create file commit", func() {
		commit, err := client.GetCommit(context.Background(), createFileCommitHash)
		s.NoError(err)

		// Verify commit details
		s.Equal(createFileCommitHash, commit.Hash)
		s.Equal(initialCommitHash, commit.Parent) // First commit has no parent
		s.Equal(user.Username, commit.Author.Name)
		s.Equal(user.Email, commit.Author.Email)
		s.NotZero(commit.Author.Time)
		s.Equal(user.Username, commit.Committer.Name)
		s.Equal(user.Email, commit.Committer.Email)
		s.NotZero(commit.Committer.Time)
		s.Equal("Initial commit", commit.Message)

		// Check that commit times are recent (within 5 seconds)
		now := time.Now()
		s.InDelta(now.Unix(), commit.Committer.Time.Unix(), 5)
		s.InDelta(now.Unix(), commit.Author.Time.Unix(), 5)
	})

	s.Run("modify file commit", func() {
		s.T().Skip("Skipping modify file commit test")

		commit, err := client.GetCommit(context.Background(), modifyFileCommitHash)
		s.NoError(err)

		// Verify commit details
		s.Equal(modifyFileCommitHash, commit.Hash)
		s.Equal(createFileCommitHash, commit.Parent)
		s.Equal(user.Username, commit.Author.Name)
		s.Equal(user.Email, commit.Author.Email)
		s.NotZero(commit.Author.Time)
		s.Equal(user.Username, commit.Committer.Name)
		s.Equal(user.Email, commit.Committer.Email)
		s.NotZero(commit.Committer.Time)
		s.Equal("Modify file", commit.Message)

		// Check that commit times are recent (within 5 seconds)
		now := time.Now()
		s.InDelta(now.Unix(), commit.Committer.Time.Unix(), 5)
		s.InDelta(now.Unix(), commit.Author.Time.Unix(), 5)
	})

	s.Run("rename file commit", func() {
		commit, err := client.GetCommit(context.Background(), renameCommitHash)
		s.NoError(err)

		// Verify commit details
		s.Equal(modifyFileCommitHash, commit.Parent)
		s.Equal(user.Username, commit.Author.Name)
		s.Equal(user.Email, commit.Author.Email)
		s.NotZero(commit.Author.Time)
		s.Equal(user.Username, commit.Committer.Name)
		s.Equal(user.Email, commit.Committer.Email)
		s.NotZero(commit.Committer.Time)
		s.Equal("Rename and add files", commit.Message)

		// Check that commit times are recent (within 5 seconds)
		now := time.Now()
		s.InDelta(now.Unix(), commit.Committer.Time.Unix(), 5)
		s.InDelta(now.Unix(), commit.Author.Time.Unix(), 5)
	})
}

// TestCompareCommits tests comparing commits to see changes
func (s *IntegrationTestSuite) TestCompareCommits() {
	s.Logger.Info("Setting up remote repository")
	client, _, local := s.TestRepo()

	s.Logger.Info("Creating initial commit with a file")
	local.CreateFile(s.T(), "test.txt", "initial content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	initialCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	s.Logger.Info("Creating second commit that modifies the file")
	local.CreateFile(s.T(), "test.txt", "modified content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Modify file")
	modifiedCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	s.Logger.Info("Creating third commit that renames and adds files")
	local.Git(s.T(), "mv", "test.txt", "renamed.txt")
	local.CreateFile(s.T(), "new.txt", "modified content")
	local.Git(s.T(), "add", ".")
	local.Git(s.T(), "commit", "-m", "Rename and add files")
	renamedCommitHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
	s.NoError(err)

	s.Logger.Info("Pushing all commits")
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Getting the file hashes for verification")
	initialFileHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", initialCommitHash.String()+":test.txt"))
	s.NoError(err)
	modifiedFileHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", modifiedCommitHash.String()+":test.txt"))
	s.NoError(err)

	s.Run("compare initial and modified commits", func() {
		changes, err := client.CompareCommits(context.Background(), initialCommitHash, modifiedCommitHash)
		s.NoError(err)
		s.Len(changes, 1)
		s.Equal("test.txt", changes[0].Path)
		s.Equal(protocol.FileStatusModified, changes[0].Status)

		s.Equal(initialFileHash, changes[0].OldHash)
		s.Equal(modifiedFileHash, changes[0].Hash)
	})

	s.Run("compare modified and renamed commits", func() {
		changes, err := client.CompareCommits(context.Background(), modifiedCommitHash, renamedCommitHash)
		s.NoError(err)
		s.Len(changes, 3)

		s.Equal("new.txt", changes[0].Path)
		s.Equal(protocol.FileStatusAdded, changes[0].Status)

		s.Equal("renamed.txt", changes[1].Path)
		s.Equal(protocol.FileStatusAdded, changes[1].Status)

		s.Equal("test.txt", changes[2].Path)
		s.Equal(protocol.FileStatusDeleted, changes[2].Status)
	})

	s.Run("compare renamed and modified commits in inverted direction", func() {
		changes, err := client.CompareCommits(context.Background(), renamedCommitHash, modifiedCommitHash)
		s.NoError(err)
		s.Len(changes, 3)

		s.Equal("new.txt", changes[0].Path)
		s.Equal(protocol.FileStatusDeleted, changes[0].Status)

		s.Equal("renamed.txt", changes[1].Path)
		s.Equal(protocol.FileStatusDeleted, changes[1].Status)

		s.Equal("test.txt", changes[2].Path)
		s.Equal(protocol.FileStatusAdded, changes[2].Status)
	})

	s.Run("compare modified and initial commits in inverted direction", func() {
		changes, err := client.CompareCommits(context.Background(), modifiedCommitHash, initialCommitHash)
		s.NoError(err)
		s.Len(changes, 1)
		s.Equal("test.txt", changes[0].Path)
		s.Equal(protocol.FileStatusModified, changes[0].Status)
		s.Equal(modifiedFileHash, changes[0].OldHash)
		s.Equal(initialFileHash, changes[0].Hash)
	})
}

// TestListCommits tests listing commits with various options and filters
func (s *IntegrationTestSuite) TestListCommits() {
	s.Run("ListCommits basic functionality", func() {

		client, _, local := s.TestRepo()

		// Create several commits to test with
		local.CreateFile(s.T(), "file1.txt", "content 1")
		local.Git(s.T(), "add", "file1.txt")
		local.Git(s.T(), "commit", "-m", "Add file1")
		local.Git(s.T(), "push", "-u", "origin", "main", "--force")

		local.CreateFile(s.T(), "file2.txt", "content 2")
		local.Git(s.T(), "add", "file2.txt")
		local.Git(s.T(), "commit", "-m", "Add file2")
		local.Git(s.T(), "push", "origin", "main")

		local.CreateFile(s.T(), "file3.txt", "content 3")
		local.Git(s.T(), "add", "file3.txt")
		local.Git(s.T(), "commit", "-m", "Add file3")
		local.Git(s.T(), "push", "origin", "main")

		// Get the current HEAD commit
		headHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
		s.NoError(err)

		// Test basic listing without options
		commits, err := client.ListCommits(context.Background(), headHash, nanogit.ListCommitsOptions{})
		s.NoError(err)
		s.GreaterOrEqual(len(commits), 3, "Should have at least 3 commits")

		// Verify commits are in reverse chronological order
		for i := 1; i < len(commits); i++ {
			s.True(commits[i-1].Time().After(commits[i].Time()) || commits[i-1].Time().Equal(commits[i].Time()),
				"Commits should be in reverse chronological order")
		}

		// Verify the latest commit message
		s.Equal("Add file3", commits[0].Message)
	})

	s.Run("ListCommits with pagination", func() {
		client, _, local := s.TestRepo()

		// Create multiple commits
		for i := 1; i <= 5; i++ {
			local.CreateFile(s.T(), fmt.Sprintf("file%d.txt", i), fmt.Sprintf("content %d", i))
			local.Git(s.T(), "add", fmt.Sprintf("file%d.txt", i))
			local.Git(s.T(), "commit", "-m", fmt.Sprintf("Add file%d", i))
			if i == 1 {
				local.Git(s.T(), "push", "-u", "origin", "main", "--force")
			} else {
				local.Git(s.T(), "push", "origin", "main")
			}
		}

		headHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
		s.NoError(err)

		// Test first page with 2 items per page
		options := nanogit.ListCommitsOptions{
			PerPage: 2,
			Page:    1,
		}

		commits, err := client.ListCommits(context.Background(), headHash, options)
		s.NoError(err)
		s.Equal(2, len(commits))
		s.Equal("Add file5", commits[0].Message)
		s.Equal("Add file4", commits[1].Message)

		// Test second page
		options.Page = 2
		commits, err = client.ListCommits(context.Background(), headHash, options)
		s.NoError(err)
		s.Equal(2, len(commits))
		s.Equal("Add file3", commits[0].Message)
		s.Equal("Add file2", commits[1].Message)
	})

	s.Run("ListCommits with path filter", func() {

		client, _, local := s.TestRepo()

		// Create commits affecting different paths
		local.CreateDirPath(s.T(), "docs")
		local.CreateFile(s.T(), "docs/readme.md", "readme content")
		local.Git(s.T(), "add", "docs/readme.md")
		local.Git(s.T(), "commit", "-m", "Add docs")
		local.Git(s.T(), "branch", "-M", "main")
		local.Git(s.T(), "push", "-u", "origin", "main", "--force")

		local.CreateDirPath(s.T(), "src")
		local.CreateFile(s.T(), "src/main.go", "main content")
		local.Git(s.T(), "add", "src/main.go")
		local.Git(s.T(), "commit", "-m", "Add main")
		local.Git(s.T(), "push", "origin", "main")

		local.CreateFile(s.T(), "docs/guide.md", "guide content")
		local.Git(s.T(), "add", "docs/guide.md")
		local.Git(s.T(), "commit", "-m", "Add guide")
		local.Git(s.T(), "push", "origin", "main")

		headHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
		s.NoError(err)

		// Filter commits affecting docs/ directory
		options := nanogit.ListCommitsOptions{
			Path: "docs",
		}
		commits, err := client.ListCommits(context.Background(), headHash, options)
		s.NoError(err)

		// Should find commits that affect docs directory
		found := 0
		for _, commit := range commits {
			if commit.Message == "Add docs" || commit.Message == "Add guide" {
				found++
			}
		}
		s.GreaterOrEqual(found, 2, "Should find commits affecting docs directory")

		// Filter commits affecting specific file
		options.Path = "src/main.go"
		commits, err = client.ListCommits(context.Background(), headHash, options)
		s.NoError(err)

		// Should find the commit that added main.go
		found = 0
		for _, commit := range commits {
			if commit.Message == "Add main" {
				found++
			}
		}
		s.GreaterOrEqual(found, 1, "Should find commit affecting src/main.go")
	})

	s.Run("ListCommits with time filters", func() {
		client, _, local := s.TestRepo()

		// Create first commit
		local.CreateFile(s.T(), "file1.txt", "content 1")
		local.Git(s.T(), "add", "file1.txt")
		local.Git(s.T(), "commit", "-m", "Old commit")
		local.Git(s.T(), "branch", "-M", "main")
		local.Git(s.T(), "push", "-u", "origin", "main", "--force")

		// Wait a bit and record time
		time.Sleep(2 * time.Second)
		midTime := time.Now()
		time.Sleep(2 * time.Second)

		// Create second commit
		local.CreateFile(s.T(), "file2.txt", "content 2")
		local.Git(s.T(), "add", "file2.txt")
		local.Git(s.T(), "commit", "-m", "New commit")
		local.Git(s.T(), "push", "origin", "main")

		headHash, err := hash.FromHex(local.Git(s.T(), "rev-parse", "HEAD"))
		s.NoError(err)

		// Filter commits since midTime (should get only the new commit)
		options := nanogit.ListCommitsOptions{
			Since: midTime,
		}
		commits, err := client.ListCommits(context.Background(), headHash, options)
		s.NoError(err)

		// Should find at least the new commit
		found := false
		for _, commit := range commits {
			if commit.Message == "New commit" {
				found = true
				s.True(commit.Time().After(midTime), "Commit should be after the since time")
			}
		}
		s.True(found, "Should find the new commit")
	})
}
