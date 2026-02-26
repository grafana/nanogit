# testutil Examples

This directory contains working examples demonstrating how to use the testutil package.

## Prerequisites

- Docker must be running
- Go 1.24 or later

## Running the Examples

```bash
# Run all examples
go test -v

# Run a specific example
go test -v -run TestBasicGitOperations
```

## Available Examples

### basic_test.go

Demonstrates basic usage with the standard `testing` package:
- Creating a test server
- Setting up a repository
- Performing Git operations
- Verifying results with nanogit client

### ginkgo_test.go

Shows integration with Ginkgo BDD testing framework:
- Using testutil with Ginkgo's BeforeEach/AfterEach
- Structured test organization
- Colored output with GinkgoWriter

## What to Expect

Each test will:
1. Start a Gitea container (takes ~5-10 seconds first time)
2. Create a test user and repository
3. Perform Git operations
4. Clean up automatically

You should see output like:

```
=== RUN   TestBasicGitOperations
🚀 Starting Gitea server container...
✅ Gitea server ready at http://localhost:32768
👤 Creating test user 'testuser-1234567890abcd'...
✅ Test user 'testuser-1234567890abcd' created successfully
📦 Creating repository 'testrepo-1234567890abcd'...
✅ Repository 'testrepo-1234567890abcd' created successfully
...
--- PASS: TestBasicGitOperations (12.34s)
PASS
```

## Troubleshooting

If tests fail:

1. **Check Docker is running**: `docker ps`
2. **Check for port conflicts**: Stop other Gitea instances
3. **Increase timeout**: If tests timeout, your system may need more time to pull/start containers
4. **Clean up containers**: `docker ps -a | grep gitea` and remove old containers if needed

## Learning from Examples

These examples serve as:
- **Templates**: Copy and adapt for your own tests
- **Documentation**: See the package in action
- **Validation**: Verify your setup works correctly

For more details, see the [main README](../README.md).
