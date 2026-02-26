// Package gittest provides testing utilities for nanogit-based applications.
//
// It includes a lightweight Git server (using Gitea in testcontainers),
// local repository wrappers, and helper functions to set up test environments
// for Git operations.
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
//	repo, _ := server.CreateRepo(ctx, gittest.RandomRepoName(), user)
//
//	local, _ := gittest.NewLocalRepo(ctx)
//	defer local.Cleanup()
//
//	remote := repo
//	client, _ := local.InitWithRemote(user, remote)
//	// Now use client for testing
package gittest
