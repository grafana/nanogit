// Package testutil provides testing utilities for nanogit-based applications.
//
// It includes a lightweight Git server (using Gitea in testcontainers),
// local repository wrappers, and helper functions to quickly set up
// test environments for Git operations.
//
// # Basic Usage
//
//	server, cleanup, err := testutil.QuickServer(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer cleanup()
//
//	user, _ := server.CreateUser(ctx)
//	repo, _ := server.CreateRepo(ctx, "test", user)
//	// Use repo.AuthURL with your Git client
//
// # Quick Setup
//
// For complete test environment setup in one call:
//
//	client, repo, local, user, cleanup, err := testutil.QuickSetup(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer cleanup()
//
//	// Now you can use:
//	// - client: nanogit.Client connected to the test server
//	// - repo: Remote repository info (repo.AuthURL, etc.)
//	// - local: Local git repository wrapper
//	// - user: Test user with credentials
//
// For more examples, see the examples/ directory.
package testutil
