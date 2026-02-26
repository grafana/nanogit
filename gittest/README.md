# nanogit/gittest

Testing utilities for Git operations with nanogit. This package provides a lightweight Git server (using Gitea in testcontainers) and helper functions to set up test environments for Git operations.

## Features

- 🐳 **Containerized**: Uses testcontainers for isolated, reproducible tests
- 🔧 **Flexible**: Works with standard `testing` package and Ginkgo
- 🧹 **Clean**: Automatic cleanup with defer-friendly patterns
- 📝 **Logging**: Optional structured logging

## Installation

```bash
go get github.com/grafana/nanogit/gittest@latest
```

**Prerequisites:**
- Docker must be running (required by testcontainers)
- Go 1.24 or later

## Quick Start

### Basic Setup

Create test environment components:

```go
func TestGitOperations(t *testing.T) {
	ctx := context.Background()

	// Create server
	server, err := gittest.NewServer(ctx,
		gittest.WithLogger(gittest.NewTestLogger(t)),
	)
	require.NoError(t, err)
	defer server.Cleanup()

	// Create user
	user, err := server.CreateUser(ctx)
	require.NoError(t, err)

	// Create repository
	repo, err := server.CreateRepo(ctx, gittest.RandomRepoName(), user)
	require.NoError(t, err)

	// Create local repository
	local, err := gittest.NewLocalRepo(ctx,
		gittest.WithRepoLogger(gittest.NewTestLogger(t)),
	)
	require.NoError(t, err)
	defer local.Cleanup()

	// Use repo.AuthURL or repo.CloneURL() for authenticated access
	t.Logf("Repository URL: %s", repo.AuthURL)
}
```

## API Reference

### Server

The `Server` type represents a Gitea server running in a container.

```go
// Create a new server
server, err := gittest.NewServer(ctx, opts...)
defer server.Cleanup()

// Create a user
user, err := server.CreateUser(ctx)

// Create a repository
repo, err := server.CreateRepo(ctx, "myrepo", user)

// Generate access token
token, err := server.CreateToken(ctx, user.Username)

// Get server URL
url := server.URL()
```

**Server Options:**
- `WithLogger(logger)` - Set custom logger
- `WithTimeout(duration)` - Set container startup timeout
- `WithGiteaImage(image)` - Set Docker image name
- `WithGiteaVersion(version)` - Set Gitea version tag

### LocalRepo

The `LocalRepo` type wraps a local Git repository in a temporary directory.

```go
// Create a new local repository
local, err := gittest.NewLocalRepo(ctx, opts...)
defer local.Cleanup()

// File operations
err = local.CreateFile("path/to/file.txt", "content")
err = local.UpdateFile("path/to/file.txt", "new content")
err = local.DeleteFile("path/to/file.txt")
err = local.CreateDirPath("path/to/dir")

// Git operations
output, err := local.Git("add", ".")
output, err := local.Git("commit", "-m", "message")
output, err := local.Git("push", "origin", "main")

// Initialize with remote (config + initial commit + push)
remote := repo
client, err := local.InitWithRemote(user, remote)

// Debug helper
local.LogContents() // Prints directory tree
```

**LocalRepo Options:**
- `WithRepoLogger(logger)` - Set custom logger
- `WithTempDir(dir)` - Set parent temp directory
- `WithGitTrace()` - Enable Git protocol tracing (for debugging only)

### Types

#### User

```go
type User struct {
	Username string
	Email    string
	Password string
	Token    string // Generated access token (if applicable)
}
```

#### RemoteRepository

```go
type RemoteRepository struct {
	Name     string
	Owner    string
	URL      string // Public URL (no auth)
	AuthURL  string // Authenticated URL (with credentials)
	User     *User
}

// Get clone URL (same as AuthURL)
url := repo.CloneURL()

// Get public URL
url := repo.PublicURL()
```

### Logging

The `Logger` interface is minimal and flexible:

```go
type Logger interface {
	Logf(format string, args ...any)
}
```

**Built-in Loggers:**

```go
// No output (default)
logger := gittest.NoopLogger()

// Standard testing.T logger
logger := gittest.NewTestLogger(t)

// Writer-based (e.g., for Ginkgo)
logger := gittest.NewWriterLogger(ginkgo.GinkgoWriter)

// Colored output with emojis
logger := gittest.NewColoredLogger(os.Stdout)

// Structured logging (nanogit.Logger-compatible)
logger := gittest.NewStructuredLogger(gittest.NewTestLogger(t))
```

## Usage with Ginkgo

The package works seamlessly with Ginkgo:

```go
package mytest_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/grafana/nanogit/gittest"
)

var _ = Describe("Git Operations", func() {
	var (
		ctx    context.Context
		server *gittest.Server
		client nanogit.Client
		repo   *gittest.RemoteRepository
		local  *gittest.LocalRepo
		user   *gittest.User
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error

		// Create server
		server, err = gittest.NewServer(ctx,
			gittest.WithLogger(gittest.NewWriterLogger(GinkgoWriter)),
		)
		Expect(err).NotTo(HaveOccurred())

		// Create user and repo
		user, err = server.CreateUser(ctx)
		Expect(err).NotTo(HaveOccurred())

		repo, err = server.CreateRepo(ctx, gittest.RandomRepoName(), user)
		Expect(err).NotTo(HaveOccurred())

		// Create local repo
		local, err = gittest.NewLocalRepo(ctx,
			gittest.WithRepoLogger(gittest.NewWriterLogger(GinkgoWriter)),
		)
		Expect(err).NotTo(HaveOccurred())

		// Initialize and get client
		remote := repo
		client, err = local.InitWithRemote(user, remote)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if local != nil {
			_ = local.Cleanup()
		}
		if server != nil {
			_ = server.Cleanup()
		}
	})

	It("should clone and push", func() {
		// Your test code here
		Expect(client).NotTo(BeNil())
		Expect(repo).NotTo(BeNil())
	})
})
```

## Troubleshooting

### Docker not running

```
Error: Cannot connect to the Docker daemon
```

**Solution:** Start Docker Desktop or your Docker daemon.

### Port conflicts

```
Error: Bind for 0.0.0.0:3000 failed: port is already allocated
```

**Solution:** Testcontainers automatically assigns random ports, but if you see this, ensure no other Gitea instances are running.

### Timeout on container startup

```
Error: container did not start within timeout
```

**Solution:** Increase the timeout:

```go
server, err := gittest.NewServer(ctx,
	gittest.WithTimeout(60 * time.Second),
)
```

### Tests hang on cleanup

**Solution:** Ensure you're using contexts properly and calling cleanup methods:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

server, err := gittest.NewServer(ctx)
defer server.Cleanup()
```

## Contributing

Contributions are welcome! Please see the main [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## License

This package is part of nanogit and shares the same license. See [LICENSE](../LICENSE) for details.
