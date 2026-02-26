# Testing Guide

This guide covers testing strategies and utilities for working with nanogit in your applications.

## Overview

nanogit provides the `testutil` package - a comprehensive testing toolkit that makes it easy to test Git operations in your applications. The gittest package includes:

- 🐳 **Containerized Git Server**: Gitea running in Docker via testcontainers
- 📁 **Local Repository Helpers**: Utilities for managing test repositories
- 🔧 **Flexible Setup**: Works with standard `testing` package and Ginkgo
- 🧹 **Automatic Cleanup**: Defer-friendly cleanup patterns

## Quick Start

### Installation

```bash
go get github.com/grafana/nanogit/gittest@latest
```

**Prerequisites:**
- Docker must be running
- Go 1.24 or later

### Basic Example

```go
package myapp_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/gittest"
	"github.com/stretchr/testify/require"
)

func TestMyGitFeature(t *testing.T) {
	ctx := context.Background()

	// Get a complete test environment in one call
	client, repo, local, user, cleanup, err := gittest.QuickSetup(ctx)
	require.NoError(t, err)
	defer cleanup()

	// Test your feature
	// - client: nanogit.Client for remote operations
	// - repo: Repository metadata and URLs
	// - local: Local git repository wrapper
	// - user: Test user credentials
}
```

## Testing Patterns

### Pattern 1: Quick Setup (Recommended)

Best for: Most test cases where you need a complete environment.

```go
func TestWithQuickSetup(t *testing.T) {
	ctx := context.Background()

	client, repo, local, user, cleanup, err := gittest.QuickSetup(ctx,
		gittest.WithQuickSetupLogger(gittest.NewTestLogger(t)),
	)
	require.NoError(t, err)
	defer cleanup()

	// Your test code here
}
```

**Pros:**
- One-line setup
- All components configured and ready
- Automatic cleanup

**Cons:**
- Less control over individual components
- May be overkill for simple tests

### Pattern 2: Manual Component Setup

Best for: Advanced scenarios where you need fine-grained control.

```go
func TestManualSetup(t *testing.T) {
	ctx := context.Background()

	// Create server with custom options
	server, err := gittest.NewServer(ctx,
		gittest.WithLogger(gittest.NewTestLogger(t)),
		gittest.WithTimeout(60*time.Second),
	)
	require.NoError(t, err)
	defer server.Cleanup()

	// Create user
	user, err := server.CreateUser(ctx)
	require.NoError(t, err)

	// Create specific repositories
	repo1, err := server.CreateRepo(ctx, "repo1", user)
	require.NoError(t, err)

	repo2, err := server.CreateRepo(ctx, "repo2", user)
	require.NoError(t, err)

	// Test with multiple repos
}
```

**Pros:**
- Full control over setup
- Can create multiple users/repos
- Fine-tune timeouts and configuration

**Cons:**
- More verbose
- Manual cleanup coordination

### Pattern 3: Server-Only Testing

Best for: Testing remote Git operations without local repositories.

```go
func TestRemoteOperations(t *testing.T) {
	ctx := context.Background()

	server, err := gittest.NewServer(ctx)
	require.NoError(t, err)
	defer server.Cleanup()

	user, err := server.CreateUser(ctx)
	require.NoError(t, err)

	repo, err := server.CreateRepo(ctx, "test", user)
	require.NoError(t, err)

	// Test nanogit client operations
	client, err := nanogit.NewHTTPClient(repo.AuthURL,
		options.WithBasicAuth(user.Username, user.Password))
	require.NoError(t, err)

	// Test remote operations
}
```

## Working with Local Repositories

The `LocalRepo` type provides high-level operations for working with local Git repositories:

### File Operations

```go
// Create files
err = local.CreateFile("README.md", "# My Project")
err = local.CreateFile("src/main.go", "package main")

// Update files
err = local.UpdateFile("README.md", "# Updated Project")

// Delete files
err = local.DeleteFile("old-file.txt")

// Create directories
err = local.CreateDirPath("src/internal/utils")
```

### Git Operations

```go
// Execute git commands
output, err := local.Git("status")
output, err := local.Git("add", ".")
output, err := local.Git("commit", "-m", "Initial commit")
output, err := local.Git("push", "origin", "main")

// Commands that might fail
output, err := local.Git("merge", "feature")
if err != nil {
	// Handle merge conflict
}
```

### Quick Initialization

```go
// Set up repo with initial commit
client, fileName, err := local.QuickInit(user, repo.AuthURL)

// Now you have:
// - Configured user.name and user.email
// - Remote 'origin' added
// - Initial test.txt file committed
// - Main branch set up and pushed
// - Branch tracking configured
// - A configured nanogit client
```

## Logging Options

Choose the logging strategy that fits your test framework:

### No Logging (Default)

```go
// Fastest, cleanest output
client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx)
defer cleanup()
```

### Standard Testing Package

```go
client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx,
	gittest.WithQuickSetupLogger(gittest.NewTestLogger(t)),
)
defer cleanup()
```

### Ginkgo/Gomega

```go
client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx,
	gittest.WithQuickSetupLogger(gittest.NewWriterLogger(GinkgoWriter)),
)
defer cleanup()
```

### Colored Output

```go
client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx,
	gittest.WithQuickSetupLogger(gittest.NewColoredLogger(os.Stdout)),
)
defer cleanup()
```

## Testing with Ginkgo

testutil integrates seamlessly with Ginkgo:

```go
var _ = Describe("Git Operations", func() {
	var (
		ctx     context.Context
		client  nanogit.Client
		repo    *gittest.Repo
		local   *gittest.LocalRepo
		cleanup func()
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		client, repo, local, _, cleanup, err = gittest.QuickSetup(ctx,
			gittest.WithQuickSetupLogger(gittest.NewWriterLogger(GinkgoWriter)),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	It("should perform git operations", func() {
		// Your test code
	})
})
```

## Advanced Scenarios

### Testing Multiple Users

```go
func TestMultiUserCollaboration(t *testing.T) {
	ctx := context.Background()

	server, err := gittest.NewServer(ctx)
	require.NoError(t, err)
	defer server.Cleanup()

	// Create users
	alice, _ := server.CreateUser(ctx)
	bob, _ := server.CreateUser(ctx)

	// Create shared repository
	repo, _ := server.CreateRepo(ctx, "shared", alice)

	// Test collaboration scenarios
}
```

### Testing Merge Conflicts

```go
func TestMergeConflict(t *testing.T) {
	// Set up main branch
	_, repo, local1, user, cleanup, _ := gittest.QuickSetup(ctx)
	defer cleanup()

	// Create second local repo (same remote)
	local2, _ := gittest.NewLocalRepo(ctx)
	defer local2.Cleanup()

	// Configure local2
	local2.Git("config", "user.name", user.Username)
	local2.Git("config", "user.email", user.Email)
	local2.Git("clone", repo.AuthURL, ".")

	// Create conflicting changes
	local1.UpdateFile("data.txt", "Version A")
	local1.Git("add", "data.txt")
	local1.Git("commit", "-m", "Update A")
	local1.Git("push", "origin", "main")

	local2.UpdateFile("data.txt", "Version B")
	local2.Git("add", "data.txt")
	local2.Git("commit", "-m", "Update B")

	// This should fail with merge conflict
	output, err := local2.Git("push", "origin", "main")
	require.Error(t, err, "expected push to fail due to conflict")
	require.Contains(t, output, "rejected")
}
```

### Testing Large Repositories

```go
func TestLargeRepo(t *testing.T) {
	_, _, local, _, cleanup, _ := gittest.QuickSetup(ctx)
	defer cleanup()

	// Create many files
	for i := 0; i < 100; i++ {
		local.CreateFile(
			fmt.Sprintf("file_%d.txt", i),
			fmt.Sprintf("Content %d", i),
		)
	}

	local.Git("add", ".")
	local.Git("commit", "-m", "Add 100 files")
	local.Git("push", "origin", "main")

	// Test performance/behavior with large repos
}
```

## Performance Considerations

### Container Startup Time

The first test that starts a Gitea container will take ~5-10 seconds while Docker pulls and starts the image. Subsequent tests reuse the pulled image and start faster.

**Optimization strategies:**

1. **Reuse servers across tests** (for test suites):

```go
var server *gittest.Server

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error
	server, err = gittest.NewServer(ctx)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	server.Cleanup()
	os.Exit(code)
}
```

2. **Use table-driven tests**:

```go
func TestGitOperations(t *testing.T) {
	// Single setup
	client, _, local, _, cleanup, _ := gittest.QuickSetup(ctx)
	defer cleanup()

	tests := []struct {
		name string
		op   func(t *testing.T)
	}{
		{"create file", func(t *testing.T) { /* test */ }},
		{"update file", func(t *testing.T) { /* test */ }},
		{"delete file", func(t *testing.T) { /* test */ }},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.op)
	}
}
```

### Parallel Testing

Be careful with parallel tests:

```go
func TestParallel(t *testing.T) {
	t.Run("test1", func(t *testing.T) {
		t.Parallel()
		// Each parallel test needs its own setup
		client, _, _, _, cleanup, _ := gittest.QuickSetup(context.Background())
		defer cleanup()
	})

	t.Run("test2", func(t *testing.T) {
		t.Parallel()
		client, _, _, _, cleanup, _ := gittest.QuickSetup(context.Background())
		defer cleanup()
	})
}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run tests
        run: go test -v ./...
```

### GitLab CI

```yaml
test:
  image: golang:1.24
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
  script:
    - go test -v ./...
```

## Troubleshooting

### Docker Connection Issues

**Problem:** `Cannot connect to the Docker daemon`

**Solution:** Ensure Docker is running:
```bash
docker ps
```

### Container Startup Timeout

**Problem:** `container did not start within timeout`

**Solution:** Increase timeout:
```go
server, err := gittest.NewServer(ctx,
	gittest.WithTimeout(60*time.Second),
)
```

### Port Conflicts

**Problem:** `port is already allocated`

**Solution:** testcontainers uses random ports by default. If you see this, check for zombie containers:
```bash
docker ps -a | grep gitea
docker rm -f <container-id>
```

### Tests Hang on Cleanup

**Problem:** Tests hang or don't terminate

**Solution:** Use context with timeout:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx)
defer cleanup()
```

### Missing Cleanup

**Problem:** Containers left running after tests

**Solution:** Always use defer with cleanup functions:
```go
// GOOD
client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx)
defer cleanup()  // Will run even if test fails

// BAD
client, _, _, _, cleanup, _ := gittest.QuickSetup(ctx)
cleanup()  // Only runs if test succeeds
```

## Best Practices

### ✅ DO

- **Always defer cleanup**: Use `defer cleanup()` immediately after setup
- **Use contexts**: Pass proper contexts with timeouts
- **Log strategically**: Enable logging for failing tests, disable for passing ones
- **Test error paths**: Don't just test the happy path
- **Isolate tests**: Each test should be independent

### ❌ DON'T

- **Don't share state**: Avoid global variables or shared repositories
- **Don't skip cleanup**: Always clean up resources
- **Don't ignore errors**: Check all error returns
- **Don't hardcode URLs**: Use the provided repo URLs
- **Don't assume timing**: Git operations can be slow; don't use time.Sleep

## Examples

For complete working examples, see:

- [gittest/examples/basic_test.go](../gittest/examples/basic_test.go) - Standard testing examples
- [gittest/examples/ginkgo_test.go](../gittest/examples/ginkgo_test.go) - Ginkgo integration
- [gittest/README.md](../gittest/README.md) - Detailed API documentation

## API Reference

For complete API documentation, see:

```bash
go doc github.com/grafana/nanogit/gittest
```

Or visit: [pkg.go.dev/github.com/grafana/nanogit/gittest](https://pkg.go.dev/github.com/grafana/nanogit/gittest)
