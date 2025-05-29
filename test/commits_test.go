//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetCommit(t *testing.T) {
	// set up remote repo
	logger := helpers.NewTestLogger(t)
	gitServer := helpers.NewGitServer(t, logger)
	user := gitServer.CreateUser(t)
	remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)

	// set up local repo
	local := helpers.NewLocalGitRepo(t, logger)
	local.Git(t, "config", "user.name", user.Username)
	local.Git(t, "config", "user.email", user.Email)
	local.Git(t, "remote", "add", "origin", remote.AuthURL())

	// Create initial commit
	local.CreateFile(t, "test.txt", "initial content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")
	initialCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	// Create second commit that modifies the file
	local.CreateFile(t, "test.txt", "modified content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Modify file")
	secondCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	// Create third commit that renames the file
	local.Git(t, "mv", "test.txt", "renamed.txt")
	local.CreateFile(t, "new.txt", "modified content")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Rename and add files")
	thirdCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	// Create and switch to main branch
	local.Git(t, "branch", "-M", "main")

	// Push commit
	local.Git(t, "push", "origin", "main", "--force")

	// Create client and get commit
	client, err := nanogit.NewHTTPClient(remote.AuthURL(), nanogit.WithBasicAuth(user.Username, user.Password))
	require.NoError(t, err)

	commit, err := client.GetCommit(context.Background(), initialCommitHash)
	require.NoError(t, err)

	// Verify commit details
	require.Equal(t, initialCommitHash, commit.Hash)
	require.Equal(t, hash.Zero, commit.Parent) // First commit has no parent
	require.Equal(t, user.Username, commit.Author.Name)
	require.Equal(t, user.Email, commit.Author.Email)
	require.NotZero(t, commit.Author.Time)
	require.Equal(t, user.Username, commit.Committer.Name)
	require.Equal(t, user.Email, commit.Committer.Email)
	require.NotZero(t, commit.Committer.Time)
	require.Equal(t, "Initial commit", commit.Message)

	// Check that commit times are recent (within 5 seconds)
	now := time.Now()
	require.InDelta(t, now.Unix(), commit.Committer.Time.Unix(), 5)
	require.InDelta(t, now.Unix(), commit.Author.Time.Unix(), 5)

	commit, err = client.GetCommit(context.Background(), secondCommitHash)
	require.NoError(t, err)

	// Verify commit details
	require.Equal(t, secondCommitHash, commit.Hash)
	require.Equal(t, initialCommitHash, commit.Parent)
	require.Equal(t, user.Username, commit.Author.Name)
	require.Equal(t, user.Email, commit.Author.Email)
	require.NotZero(t, commit.Author.Time)
	require.Equal(t, user.Username, commit.Committer.Name)
	require.Equal(t, user.Email, commit.Committer.Email)
	require.NotZero(t, commit.Committer.Time)
	require.Equal(t, "Modify file", commit.Message)

	// Check that commit times are recent (within 5 seconds)
	require.InDelta(t, now.Unix(), commit.Committer.Time.Unix(), 5)
	require.InDelta(t, now.Unix(), commit.Author.Time.Unix(), 5)

	commit, err = client.GetCommit(context.Background(), thirdCommitHash)
	require.NoError(t, err)

	// Verify commit details
	require.Equal(t, secondCommitHash, commit.Parent)
	require.Equal(t, user.Username, commit.Author.Name)
	require.Equal(t, user.Email, commit.Author.Email)
	require.NotZero(t, commit.Author.Time)
	require.Equal(t, user.Username, commit.Committer.Name)
	require.Equal(t, user.Email, commit.Committer.Email)
	require.NotZero(t, commit.Committer.Time)
	require.Equal(t, "Rename and add files", commit.Message)

	// Check that commit times are recent (within 5 seconds)
	require.InDelta(t, now.Unix(), commit.Committer.Time.Unix(), 5)
	require.InDelta(t, now.Unix(), commit.Author.Time.Unix(), 5)
}

func TestClient_CompareCommits(t *testing.T) {
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

	logger.Info("Creating initial commit with a file")
	local.CreateFile(t, "test.txt", "initial content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Initial commit")
	initialCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	logger.Info("Creating second commit that modifies the file")
	local.CreateFile(t, "test.txt", "modified content")
	local.Git(t, "add", "test.txt")
	local.Git(t, "commit", "-m", "Modify file")
	modifiedCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	logger.Info("Creating third commit that renames and adds files")
	local.Git(t, "mv", "test.txt", "renamed.txt")
	local.CreateFile(t, "new.txt", "modified content")
	local.Git(t, "add", ".")
	local.Git(t, "commit", "-m", "Rename and add files")
	renamedCommitHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
	require.NoError(t, err)

	logger.Info("Setting up main branch and pushing changes")
	local.Git(t, "branch", "-M", "main")

	logger.Info("Pushing all commits")
	local.Git(t, "push", "origin", "main", "--force")

	logger.Info("Debug output: print remote URL and commit hashes")
	t.Logf("Remote URL: %s", remote.AuthURL())
	t.Logf("Initial commit hash: %s", initialCommitHash)
	t.Logf("Modified commit hash: %s", modifiedCommitHash)
	t.Logf("Renamed commit hash: %s", renamedCommitHash)

	logger.Info("Manually checking if the commit exists on the remote")
	t.Log("Running git ls-remote to verify the commit")
	local.Git(t, "ls-remote", remote.AuthURL())

	logger.Info("Creating client")
	client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Getting the file hashes for verification")
	initialFileHash, err := hash.FromHex(local.Git(t, "rev-parse", initialCommitHash.String()+":test.txt"))
	require.NoError(t, err)
	modifiedFileHash, err := hash.FromHex(local.Git(t, "rev-parse", modifiedCommitHash.String()+":test.txt"))
	require.NoError(t, err)

	t.Run("compare initial and modified commits", func(t *testing.T) {
		logger.ForSubtest(t)

		changes, err := client.CompareCommits(ctx, initialCommitHash, modifiedCommitHash)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		assert.Equal(t, "test.txt", changes[0].Path)
		assert.Equal(t, protocol.FileStatusModified, changes[0].Status)

		assert.Equal(t, initialFileHash, changes[0].OldHash)
		assert.Equal(t, modifiedFileHash, changes[0].Hash)
	})

	t.Run("compare modified and renamed commits", func(t *testing.T) {
		logger.ForSubtest(t)

		changes, err := client.CompareCommits(ctx, modifiedCommitHash, renamedCommitHash)
		require.NoError(t, err)
		require.Len(t, changes, 3)

		assert.Equal(t, "new.txt", changes[0].Path)
		assert.Equal(t, protocol.FileStatusAdded, changes[0].Status)

		assert.Equal(t, "renamed.txt", changes[1].Path)
		assert.Equal(t, protocol.FileStatusAdded, changes[1].Status)

		assert.Equal(t, "test.txt", changes[2].Path)
		assert.Equal(t, protocol.FileStatusDeleted, changes[2].Status)
	})

	t.Run("compare renamed and modified commits in inverted direction", func(t *testing.T) {
		logger.ForSubtest(t)

		changes, err := client.CompareCommits(ctx, renamedCommitHash, modifiedCommitHash)
		require.NoError(t, err)
		require.Len(t, changes, 3)

		assert.Equal(t, "new.txt", changes[0].Path)
		assert.Equal(t, protocol.FileStatusDeleted, changes[0].Status)

		assert.Equal(t, "renamed.txt", changes[1].Path)
		assert.Equal(t, protocol.FileStatusDeleted, changes[1].Status)

		assert.Equal(t, "test.txt", changes[2].Path)
		assert.Equal(t, protocol.FileStatusAdded, changes[2].Status)
	})

	t.Run("compare modified and initial commits in inverted direction", func(t *testing.T) {
		logger.ForSubtest(t)

		changes, err := client.CompareCommits(ctx, modifiedCommitHash, initialCommitHash)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		assert.Equal(t, "test.txt", changes[0].Path)
		assert.Equal(t, protocol.FileStatusModified, changes[0].Status)
		assert.Equal(t, modifiedFileHash, changes[0].OldHash)
		assert.Equal(t, initialFileHash, changes[0].Hash)
	})
}

func TestClient_ListCommits(t *testing.T) {
	t.Run("ListCommits basic functionality", func(t *testing.T) {
		logger := helpers.NewTestLogger(t)
		logger.ForSubtest(t)
		gitServer := helpers.NewGitServer(t, logger)
		user := gitServer.CreateUser(t)
		remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)
		local := helpers.NewLocalGitRepo(t, logger)
		client, _ := local.QuickInit(t, user, remote.AuthURL())
		ctx := context.Background()

		// Create several commits to test with
		local.CreateFile(t, "file1.txt", "content 1")
		local.Git(t, "add", "file1.txt")
		local.Git(t, "commit", "-m", "Add file1")
		local.Git(t, "push")

		local.CreateFile(t, "file2.txt", "content 2")
		local.Git(t, "add", "file2.txt")
		local.Git(t, "commit", "-m", "Add file2")
		local.Git(t, "push")

		local.CreateFile(t, "file3.txt", "content 3")
		local.Git(t, "add", "file3.txt")
		local.Git(t, "commit", "-m", "Add file3")
		local.Git(t, "push")

		// Get the current HEAD commit
		headHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

		// Test basic listing without options
		commits, err := client.ListCommits(ctx, headHash, nanogit.ListCommitsOptions{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(commits), 3, "Should have at least 3 commits")

		// Verify commits are in reverse chronological order
		for i := 1; i < len(commits); i++ {
			assert.True(t, commits[i-1].Time().After(commits[i].Time()) || commits[i-1].Time().Equal(commits[i].Time()),
				"Commits should be in reverse chronological order")
		}

		// Verify the latest commit message
		assert.Equal(t, "Add file3", commits[0].Message)
	})

	t.Run("ListCommits with pagination", func(t *testing.T) {
		logger := helpers.NewTestLogger(t)
		logger.ForSubtest(t)
		gitServer := helpers.NewGitServer(t, logger)
		user := gitServer.CreateUser(t)
		remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)
		local := helpers.NewLocalGitRepo(t, logger)
		client, _ := local.QuickInit(t, user, remote.AuthURL())
		ctx := context.Background()

		// Create multiple commits
		for i := 1; i <= 5; i++ {
			local.CreateFile(t, fmt.Sprintf("file%d.txt", i), fmt.Sprintf("content %d", i))
			local.Git(t, "add", fmt.Sprintf("file%d.txt", i))
			local.Git(t, "commit", "-m", fmt.Sprintf("Add file%d", i))
			local.Git(t, "push")
		}

		headHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

		// Test first page with 2 items per page
		options := nanogit.ListCommitsOptions{
			PerPage: 2,
			Page:    1,
		}
		commits, err := client.ListCommits(ctx, headHash, options)
		require.NoError(t, err)
		assert.Equal(t, 2, len(commits))
		assert.Equal(t, "Add file5", commits[0].Message)
		assert.Equal(t, "Add file4", commits[1].Message)

		// Test second page
		options.Page = 2
		commits, err = client.ListCommits(ctx, headHash, options)
		require.NoError(t, err)
		assert.Equal(t, 2, len(commits))
		assert.Equal(t, "Add file3", commits[0].Message)
		assert.Equal(t, "Add file2", commits[1].Message)
	})

	t.Run("ListCommits with path filter", func(t *testing.T) {
		logger := helpers.NewTestLogger(t)
		logger.ForSubtest(t)
		gitServer := helpers.NewGitServer(t, logger)
		user := gitServer.CreateUser(t)
		remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)
		local := helpers.NewLocalGitRepo(t, logger)
		client, _ := local.QuickInit(t, user, remote.AuthURL())
		ctx := context.Background()

		// Create commits affecting different paths
		local.CreateDirPath(t, "docs")
		local.CreateFile(t, "docs/readme.md", "readme content")
		local.Git(t, "add", "docs/readme.md")
		local.Git(t, "commit", "-m", "Add docs")
		local.Git(t, "push")

		local.CreateDirPath(t, "src")
		local.CreateFile(t, "src/main.go", "main content")
		local.Git(t, "add", "src/main.go")
		local.Git(t, "commit", "-m", "Add main")
		local.Git(t, "push")

		local.CreateFile(t, "docs/guide.md", "guide content")
		local.Git(t, "add", "docs/guide.md")
		local.Git(t, "commit", "-m", "Add guide")
		local.Git(t, "push")

		headHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

		// Filter commits affecting docs/ directory
		options := nanogit.ListCommitsOptions{
			Path: "docs",
		}
		commits, err := client.ListCommits(ctx, headHash, options)
		require.NoError(t, err)

		// Should find commits that affect docs directory
		found := 0
		for _, commit := range commits {
			if commit.Message == "Add docs" || commit.Message == "Add guide" {
				found++
			}
		}
		assert.GreaterOrEqual(t, found, 2, "Should find commits affecting docs directory")

		// Filter commits affecting specific file
		options.Path = "src/main.go"
		commits, err = client.ListCommits(ctx, headHash, options)
		require.NoError(t, err)

		// Should find the commit that added main.go
		found = 0
		for _, commit := range commits {
			if commit.Message == "Add main" {
				found++
			}
		}
		assert.GreaterOrEqual(t, found, 1, "Should find commit affecting src/main.go")
	})

	t.Run("ListCommits with time filters", func(t *testing.T) {
		logger := helpers.NewTestLogger(t)
		logger.ForSubtest(t)
		gitServer := helpers.NewGitServer(t, logger)
		user := gitServer.CreateUser(t)
		remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)
		local := helpers.NewLocalGitRepo(t, logger)
		client, _ := local.QuickInit(t, user, remote.AuthURL())
		ctx := context.Background()

		// Create first commit
		local.CreateFile(t, "file1.txt", "content 1")
		local.Git(t, "add", "file1.txt")
		local.Git(t, "commit", "-m", "Old commit")
		local.Git(t, "push")

		// Wait a bit and record time
		time.Sleep(2 * time.Second)
		midTime := time.Now()
		time.Sleep(2 * time.Second)

		// Create second commit
		local.CreateFile(t, "file2.txt", "content 2")
		local.Git(t, "add", "file2.txt")
		local.Git(t, "commit", "-m", "New commit")
		local.Git(t, "push")

		headHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

		// Filter commits since midTime (should get only the new commit)
		options := nanogit.ListCommitsOptions{
			Since: midTime,
		}
		commits, err := client.ListCommits(ctx, headHash, options)
		require.NoError(t, err)

		// Should find at least the new commit
		found := false
		for _, commit := range commits {
			if commit.Message == "New commit" {
				found = true
				assert.True(t, commit.Time().After(midTime), "Commit should be after the since time")
			}
		}
		assert.True(t, found, "Should find the new commit")
	})
}
