# Testing Guide

Testing utilities for nanogit applications using the `gittest` package.

## Installation

```bash
go get github.com/grafana/nanogit/gittest@latest
```

**Prerequisites:** Docker must be running.

## Quick Start

```go
package myapp_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

func TestGitOperations(t *testing.T) {
	ctx := context.Background()

	// Create Git server
	server, err := gittest.NewServer(ctx)
	require.NoError(t, err)
	defer server.Cleanup()

	// Create user and repository
	user, err := server.CreateUser(ctx)
	require.NoError(t, err)

	repo, err := server.CreateRepo(ctx, gittest.RandomRepoName(), user)
	require.NoError(t, err)

	// Create local repository
	local, err := gittest.NewLocalRepo(ctx)
	require.NoError(t, err)
	defer local.Cleanup()

	// Initialize with remote and get connection info
	remote := repo
	connInfo, err := local.InitWithRemote(user, remote)
	require.NoError(t, err)

	// Create nanogit client from connection info
	client, err := nanogit.NewHTTPClient(connInfo.URL,
		options.WithBasicAuth(connInfo.Username, connInfo.Password))
	require.NoError(t, err)

	// Test your feature
	err = local.CreateFile("test.txt", "content")
	require.NoError(t, err)
}
```

## Core API

### Server

```go
server, err := gittest.NewServer(ctx,
	gittest.WithLogger(gittest.NewTestLogger(t)),
	gittest.WithTimeout(60*time.Second),
)
defer server.Cleanup()

user, err := server.CreateUser(ctx)
repo, err := server.CreateRepo(ctx, gittest.RandomRepoName(), user)
token, err := server.CreateToken(ctx, user.Username)
```

### Local Repository

```go
local, err := gittest.NewLocalRepo(ctx,
	gittest.WithRepoLogger(gittest.NewTestLogger(t)),
)
defer local.Cleanup()

// Initialize with remote - returns connection info
remote := repo
connInfo, err := local.InitWithRemote(user, remote)

// Create your Git client from the connection info
client, err := nanogit.NewHTTPClient(connInfo.URL,
	options.WithBasicAuth(connInfo.Username, connInfo.Password))

err = local.CreateFile("path/file.txt", "content")
err = local.UpdateFile("path/file.txt", "new content")
err = local.DeleteFile("path/file.txt")
output, err := local.Git("status")
```

## Logging

```go
// No logging (default)
server, err := gittest.NewServer(ctx)

// With standard testing logger
server, err := gittest.NewServer(ctx,
	gittest.WithLogger(gittest.NewTestLogger(t)),
)

// With structured logging
baseLogger := gittest.NewWriterLogger(os.Stdout)
structuredLogger := gittest.NewStructuredLogger(baseLogger)
structuredLogger.Info("message", "key", "value")
```

## Ginkgo Integration

```go
var _ = Describe("Git Operations", func() {
	var (
		ctx    context.Context
		server *gittest.Server
		local  *gittest.LocalRepo
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error

		server, err = gittest.NewServer(ctx,
			gittest.WithLogger(gittest.NewWriterLogger(GinkgoWriter)),
		)
		Expect(err).NotTo(HaveOccurred())

		local, err = gittest.NewLocalRepo(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if local != nil {
			Expect(local.Cleanup()).To(Succeed())
		}
		if server != nil {
			Expect(server.Cleanup()).To(Succeed())
		}
	})

	It("should work", func() {
		// Test code
	})
})
```

## Best Practices

- Always use `defer cleanup()` immediately after resource creation
- Use `gittest.RandomRepoName()` for unique repository names
- Pass contexts with timeouts for long-running tests
- Reuse servers across test suites via `TestMain` for performance

## Documentation

- [gittest/README.md](../gittest/README.md) - Complete package documentation
- [pkg.go.dev](https://pkg.go.dev/github.com/grafana/nanogit/gittest) - API reference
