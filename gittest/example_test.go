package gittest_test

import (
	"context"
	"fmt"
	"log"

	"github.com/grafana/nanogit/gittest"
)

// Example demonstrates basic usage of testutil to set up a Git server and repository.
func Example() {
	ctx := context.Background()

	// Create a Git server (Gitea in testcontainers)
	server, err := gittest.NewServer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Cleanup()

	// Create a test user
	user, err := server.CreateUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a repository
	repo, err := server.CreateRepo(ctx, "myrepo", user)
	if err != nil {
		log.Fatal(err)
	}

	// Create a local repository wrapper
	local, err := gittest.NewLocalRepo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer local.Cleanup()

	// Initialize local repo and get a nanogit client
	client, err := local.QuickInit(user, repo.AuthURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server URL: %s\n", server.URL())
	fmt.Printf("User: %s\n", user.Username)
	fmt.Printf("Repo: %s\n", repo.Name)
	fmt.Printf("Client connected: %v\n", client != nil)
}

// ExampleNewServer demonstrates creating a Git server with custom options.
func ExampleNewServer() {
	ctx := context.Background()

	// Create server with custom logger
	server, err := gittest.NewServer(ctx,
		gittest.WithLogger(gittest.NoopLogger()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Cleanup()

	fmt.Printf("Server running at: %s\n", server.URL())
}

// ExampleServer_CreateUser demonstrates creating a test user.
func ExampleServer_CreateUser() {
	ctx := context.Background()

	server, err := gittest.NewServer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Cleanup()

	// Create a user with auto-generated credentials
	user, err := server.CreateUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created user: %s\n", user.Username)
	fmt.Printf("Email: %s\n", user.Email)
	// user.Password and user.Token are also available
}

// ExampleServer_CreateRepo demonstrates creating a repository.
func ExampleServer_CreateRepo() {
	ctx := context.Background()

	server, err := gittest.NewServer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Cleanup()

	user, err := server.CreateUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a repository owned by the user
	repo, err := server.CreateRepo(ctx, "myproject", user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Repository URL: %s\n", repo.URL)
	fmt.Printf("Clone URL (with auth): %s\n", repo.AuthURL)
}

// ExampleNewLocalRepo demonstrates creating a local Git repository wrapper.
func ExampleNewLocalRepo() {
	ctx := context.Background()

	// Create a local repo in a temporary directory
	local, err := gittest.NewLocalRepo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer local.Cleanup()

	// Create a file
	err = local.CreateFile("README.md", "# My Project")
	if err != nil {
		log.Fatal(err)
	}

	// Run git commands
	_, err = local.Git("add", "README.md")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Local repository created")
}

// ExampleLocalRepo_QuickInit demonstrates initializing a local repo with a remote.
func ExampleLocalRepo_QuickInit() {
	ctx := context.Background()

	// Set up server and repo
	server, err := gittest.NewServer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Cleanup()

	user, err := server.CreateUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	repo, err := server.CreateRepo(ctx, "test", user)
	if err != nil {
		log.Fatal(err)
	}

	// Create and initialize local repo
	local, err := gittest.NewLocalRepo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer local.Cleanup()

	// QuickInit configures git user, creates initial commit, and returns a client
	client, err := local.QuickInit(user, repo.AuthURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Initialized with remote: %s\n", repo.Name)
	fmt.Printf("Client ready: %v\n", client != nil)
}

// ExampleNewTestLogger demonstrates using a test logger.
func ExampleNewTestLogger() {
	// In a real test function:
	// logger := gittest.NewTestLogger(t)

	// For this example, we'll show the pattern
	fmt.Println("Use gittest.NewTestLogger(t) in test functions")
	fmt.Println("It logs to testing.T.Logf()")
}

// ExampleNewColoredLogger demonstrates using colored output.
func ExampleNewColoredLogger() {
	// logger := gittest.NewColoredLogger(os.Stdout)

	fmt.Println("NewColoredLogger provides colored output with emojis")
	fmt.Println("Great for visual feedback during development")
}
