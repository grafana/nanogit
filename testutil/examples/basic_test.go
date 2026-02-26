package examples

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/testutil"
	"github.com/stretchr/testify/require"
)

// TestBasicGitOperations demonstrates basic usage of testutil with the standard testing package.
func TestBasicGitOperations(t *testing.T) {
	ctx := context.Background()

	// Create server
	server, err := testutil.QuickServer(ctx,
		testutil.WithLogger(testutil.NewTestLogger(t)),
	)
	require.NoError(t, err, "failed to create server")
	defer server.Cleanup()

	// Create user and repo
	user, err := server.CreateUser(ctx)
	require.NoError(t, err, "failed to create user")

	repo, err := server.CreateRepo(ctx, "testrepo", user)
	require.NoError(t, err, "failed to create repo")

	// Create local repo
	local, err := testutil.NewLocalRepo(ctx, testutil.WithRepoLogger(testutil.NewTestLogger(t)))
	require.NoError(t, err, "failed to create local repo")
	defer local.Cleanup()

	// Initialize local repo and get client
	client, _, err := local.QuickInit(user, repo.AuthURL)
	require.NoError(t, err, "failed to initialize local repo")

	t.Logf("Test environment ready:")
	t.Logf("  Server: %s", repo.URL)
	t.Logf("  User: %s", user.Username)
	t.Logf("  Repo: %s", repo.Name)

	// Create and push a new file
	t.Log("Creating a new file")
	err = local.CreateFile("hello.txt", "Hello, World!")
	require.NoError(t, err, "failed to create file")

	_, err = local.Git("add", "hello.txt")
	require.NoError(t, err, "failed to add file")

	_, err = local.Git("commit", "-m", "Add hello.txt")
	require.NoError(t, err, "failed to commit file")

	_, err = local.Git("push", "origin", "main")
	require.NoError(t, err, "failed to push file")

	// Verify with nanogit client
	t.Log("Verifying push with nanogit client")
	ref, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err, "failed to get ref")
	require.NotNil(t, ref, "ref should not be nil")

	t.Logf("Successfully verified ref: %s", ref.Hash)

	// Update the file
	t.Log("Updating the file")
	err = local.UpdateFile("hello.txt", "Hello, nanogit!")
	require.NoError(t, err, "failed to update file")

	_, err = local.Git("add", "hello.txt")
	require.NoError(t, err, "failed to add updated file")

	_, err = local.Git("commit", "-m", "Update hello.txt")
	require.NoError(t, err, "failed to commit update")

	_, err = local.Git("push", "origin", "main")
	require.NoError(t, err, "failed to push update")

	// Verify the update
	t.Log("Verifying update with nanogit client")
	updatedRef, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err, "failed to get updated ref")
	require.NotNil(t, updatedRef, "updated ref should not be nil")
	require.NotEqual(t, ref.Hash, updatedRef.Hash, "ref should have changed after update")

	t.Logf("Successfully verified updated ref: %s", updatedRef.Hash)
}

// TestManualSetup demonstrates manual setup with more control over each component.
func TestManualSetup(t *testing.T) {
	ctx := context.Background()

	// Create server
	t.Log("Creating Git server")
	server, err := testutil.QuickServer(ctx,
		testutil.WithLogger(testutil.NewTestLogger(t)),
	)
	require.NoError(t, err, "failed to create server")
	defer server.Cleanup()

	t.Logf("Server ready at: %s", server.URL())

	// Create user
	t.Log("Creating test user")
	user, err := server.CreateUser(ctx)
	require.NoError(t, err, "failed to create user")
	t.Logf("User created: %s (%s)", user.Username, user.Email)

	// Create repository
	t.Log("Creating repository")
	repo, err := server.CreateRepo(ctx, "manual-test-repo", user)
	require.NoError(t, err, "failed to create repo")
	t.Logf("Repository created: %s", repo.Name)
	t.Logf("  Public URL: %s", repo.PublicURL())
	t.Logf("  Clone URL: %s", repo.CloneURL())

	// Create local repository
	t.Log("Creating local repository")
	local, err := testutil.NewLocalRepo(ctx,
		testutil.WithRepoLogger(testutil.NewTestLogger(t)),
	)
	require.NoError(t, err, "failed to create local repo")
	defer local.Cleanup()

	t.Logf("Local repository at: %s", local.Path)

	// Configure and initialize
	t.Log("Configuring local repository")
	_, err = local.Git("config", "user.name", user.Username)
	require.NoError(t, err, "failed to set user.name")

	_, err = local.Git("config", "user.email", user.Email)
	require.NoError(t, err, "failed to set user.email")

	_, err = local.Git("remote", "add", "origin", repo.AuthURL)
	require.NoError(t, err, "failed to add remote")

	// Create initial content
	t.Log("Creating initial content")
	err = local.CreateFile("README.md", "# Manual Test Repo\n\nThis is a test repository.")
	require.NoError(t, err, "failed to create README")

	_, err = local.Git("add", "README.md")
	require.NoError(t, err, "failed to add README")

	_, err = local.Git("commit", "-m", "Initial commit")
	require.NoError(t, err, "failed to commit")

	_, err = local.Git("branch", "-M", "main")
	require.NoError(t, err, "failed to rename branch")

	_, err = local.Git("push", "origin", "main")
	require.NoError(t, err, "failed to push")

	t.Log("Manual setup completed successfully!")
}

// TestMultipleFiles demonstrates working with multiple files and directories.
func TestMultipleFiles(t *testing.T) {
	ctx := context.Background()

	// Create server
	server, err := testutil.QuickServer(ctx,
		testutil.WithLogger(testutil.NewTestLogger(t)),
	)
	require.NoError(t, err)
	defer server.Cleanup()

	// Create user and repo
	user, err := server.CreateUser(ctx)
	require.NoError(t, err)

	repo, err := server.CreateRepo(ctx, "multifile-test", user)
	require.NoError(t, err)

	// Create local repo
	local, err := testutil.NewLocalRepo(ctx, testutil.WithRepoLogger(testutil.NewTestLogger(t)))
	require.NoError(t, err)
	defer local.Cleanup()

	// Initialize and get client
	client, _, err := local.QuickInit(user, repo.AuthURL)
	require.NoError(t, err)

	// Create a directory structure
	t.Log("Creating directory structure")
	err = local.CreateDirPath("src/main")
	require.NoError(t, err)

	err = local.CreateDirPath("src/test")
	require.NoError(t, err)

	// Create multiple files
	files := map[string]string{
		"src/main/app.go":    "package main\n\nfunc main() {}\n",
		"src/main/config.go": "package main\n\nconst Version = \"1.0.0\"\n",
		"src/test/app_test.go": "package main\n\nimport \"testing\"\n\n" +
			"func TestMain(t *testing.T) {}\n",
		"README.md": "# Multi-File Test\n",
		"go.mod":    "module example.com/test\n\ngo 1.24\n",
	}

	for path, content := range files {
		t.Logf("Creating file: %s", path)
		err = local.CreateFile(path, content)
		require.NoError(t, err, "failed to create %s", path)
	}

	// Add and commit all files
	t.Log("Adding and committing all files")
	_, err = local.Git("add", ".")
	require.NoError(t, err)

	_, err = local.Git("commit", "-m", "Add project structure")
	require.NoError(t, err)

	_, err = local.Git("push", "origin", "main")
	require.NoError(t, err)

	// Verify
	t.Log("Verifying with nanogit client")
	ref, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)
	require.NotNil(t, ref)

	t.Logf("Successfully pushed %d files", len(files)+1) // +1 for test.txt from QuickInit

	// Log the repository contents
	local.LogContents()
}
