// Package testutil provides testing utilities for nanogit-based applications.
//
// It includes a lightweight Git server (using Gitea in testcontainers),
// local repository wrappers, and helper functions to quickly set up
// test environments for Git operations.
//
// # Basic Usage
//
//	server, err := gittest.NewServer(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer server.Cleanup()
//
//	user, _ := server.CreateUser(ctx)
//	repo, _ := server.CreateRepo(ctx, "test", user)
//
//	// Create local repo and initialize
//	local, _ := gittest.NewLocalRepo(ctx)
//	defer local.Cleanup()
//
//	client, _, _ := local.QuickInit(user, repo.AuthURL)
//	// Now use client for testing
//
// For more examples, see the examples/ directory.
package gittest
