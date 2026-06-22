// Package gittest provides testing utilities for Git-based applications.
//
// It includes a lightweight Git server (using Gitea in testcontainers),
// local repository wrappers, and helper functions to set up test environments
// for Git operations.
//
// The package is client-agnostic - it provides connection information that
// you can use with any Git client (nanogit, go-git, or others).
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
//	// Initialize with remote - returns connection info
//	remote := repo
//	connInfo, _ := local.InitWithRemote(user, remote)
//
//	// Create your Git client from the connection info
//	// Example with nanogit:
//	// client, _ := nanogit.NewHTTPClient(connInfo.URL,
//	//     options.WithBasicAuth(connInfo.Username, connInfo.Password))
package gittest
