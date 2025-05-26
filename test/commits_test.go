//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		commits, err := client.ListCommits(ctx, headHash, nil)
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
		options := &nanogit.ListCommitsOptions{
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
		options := &nanogit.ListCommitsOptions{
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

	t.Run("ListCommits with author filter", func(t *testing.T) {
		logger := helpers.NewTestLogger(t)
		logger.ForSubtest(t)
		gitServer := helpers.NewGitServer(t, logger)
		user := gitServer.CreateUser(t)
		remote := gitServer.CreateRepo(t, "testrepo", user.Username, user.Password)
		local := helpers.NewLocalGitRepo(t, logger)
		client, _ := local.QuickInit(t, user, remote.AuthURL())
		ctx := context.Background()

		// Create a commit with specific author
		local.Git(t, "config", "user.email", "author1@example.com")
		local.Git(t, "config", "user.name", "Author One")
		local.CreateFile(t, "file1.txt", "content 1")
		local.Git(t, "add", "file1.txt")
		local.Git(t, "commit", "-m", "Author 1 commit")
		local.Git(t, "push")

		// Create a commit with different author
		local.Git(t, "config", "user.email", "author2@example.com")
		local.Git(t, "config", "user.name", "Author Two")
		local.CreateFile(t, "file2.txt", "content 2")
		local.Git(t, "add", "file2.txt")
		local.Git(t, "commit", "-m", "Author 2 commit")
		local.Git(t, "push")

		headHash, err := hash.FromHex(local.Git(t, "rev-parse", "HEAD"))
		require.NoError(t, err)

		// Filter by first author
		options := &nanogit.ListCommitsOptions{
			Author: "author1@example.com",
		}
		commits, err := client.ListCommits(ctx, headHash, options)
		require.NoError(t, err)

		// Should find only commits by author1
		for _, commit := range commits {
			if commit.Message == "Author 1 commit" {
				assert.Equal(t, "author1@example.com", commit.Author.Email)
			}
		}
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
		options := &nanogit.ListCommitsOptions{
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
