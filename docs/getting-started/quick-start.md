# Quick Start

This guide will help you get started with nanogit quickly.

## Basic Operations

### Creating a Client

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/grafana/nanogit"
    "github.com/grafana/nanogit/options"
)

func main() {
    ctx := context.Background()

    // Create client with authentication
    client, err := nanogit.NewHTTPClient(
        "https://github.com/user/repo.git",
        options.WithBasicAuth("username", "token"),
    )
    if err != nil {
        panic(err)
    }
}
```

### Reading Files

```go
// Get the main branch reference
ref, err := client.GetRef(ctx, "refs/heads/main")
if err != nil {
    panic(err)
}

// Read a file
content, err := client.GetBlob(ctx, ref.Hash, "README.md")
if err != nil {
    panic(err)
}

fmt.Println(string(content))
```

### Writing Files

```go
// Create a staged writer
writer, err := client.NewStagedWriter(ctx, ref)
if err != nil {
    panic(err)
}

// Create a new file
err = writer.CreateBlob(ctx, "docs/new-feature.md", []byte("# New Feature"))
if err != nil {
    panic(err)
}

// Update an existing file
err = writer.UpdateBlob(ctx, "README.md", []byte("Updated content"))
if err != nil {
    panic(err)
}

// Commit changes
author := nanogit.Author{
    Name:  "John Doe",
    Email: "john@example.com",
    Time:  time.Now(),
}
committer := nanogit.Committer{
    Name:  "Deploy Bot",
    Email: "deploy@example.com",
    Time:  time.Now(),
}

commit, err := writer.Commit(ctx, "Add feature and update docs", author, committer)
if err != nil {
    panic(err)
}

// Push to remote
err = writer.Push(ctx)
if err != nil {
    panic(err)
}

fmt.Printf("Committed: %s\n", commit.Hash)
```

### Cloning a Repository

```go
// Get the commit to clone
ref, err := client.GetRef(ctx, "main")
if err != nil {
    panic(err)
}

// Clone with path filtering
result, err := client.Clone(ctx, nanogit.CloneOptions{
    Path:         "/tmp/my-repo",
    Hash:         ref.Hash,
    IncludePaths: []string{"src/**", "docs/**"},
    ExcludePaths: []string{"*.tmp", "node_modules/**"},
    BatchSize:    50,  // Batch fetch for better performance
    Concurrency:  8,   // Use 8 concurrent workers
})
if err != nil {
    panic(err)
}

fmt.Printf("Cloned %d of %d files to %s\n",
    result.FilteredFiles, result.TotalFiles, result.Path)
```

**Clone Features:**
- **Path filtering**: Use glob patterns to include/exclude specific files and directories
- **Filesystem output**: Automatically writes filtered files to specified local path
- **Shallow clones**: Fetch only the latest commit to minimize bandwidth
- **Branch isolation**: Clone only specific branches to reduce transfer time
- **CI optimized**: Perfect for build environments with no persistent storage

## Authentication Options

### Basic Auth (GitHub Token)

```go
client, err := nanogit.NewHTTPClient(
    repo,
    options.WithBasicAuth("username", "ghp_token"),
)
```

### GitLab Personal Access Token

```go
client, err := nanogit.NewHTTPClient(
    repo,
    options.WithBasicAuth("oauth2", "gitlab_pat_token"),
)
```

### No Authentication (Public Repos)

```go
client, err := nanogit.NewHTTPClient(repo)
```

## Performance Optimization

### Clone Performance

**BatchSize** - Controls how many blobs to fetch in a single network request:

- **Value 0 or 1**: Fetches blobs individually (backward compatible, default behavior)
- **Values > 1**: Enables batch fetching, reducing network round trips by 50-70%
- Recommended value: 20-100 depending on average blob size and network conditions

**Concurrency** - Controls parallel blob fetching:

- **Value 0 or 1**: Sequential fetching (backward compatible, default behavior)
- **Values > 1**: Enables concurrent fetching using worker pools
- Recommended value: 4-10 depending on network conditions and server capacity
- Can improve performance by 2-3x on high-latency networks

**Performance Impact**: Combined optimization (BatchSize=50, Concurrency=8) can achieve 5-10x speedup compared to default sequential fetching, making it ideal for CI/CD environments and large repository operations.

```go
// Optimized clone for production
result, err := client.Clone(ctx, nanogit.CloneOptions{
    Path:        "/tmp/repo",
    Hash:        ref.Hash,
    BatchSize:   50,  // Fetch 50 blobs per request
    Concurrency: 8,   // 8 concurrent workers
})
```

### Writer Storage Modes

nanogit provides flexible writing modes to optimize memory usage:

```go
// Auto mode (default) - smart memory/disk switching
writer, err := client.NewStagedWriter(ctx, ref)

// Memory mode - maximum performance
writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithMemoryStorage())

// Disk mode - minimal memory usage for bulk operations
writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithDiskStorage())
```

Learn more about [Storage Architecture](../architecture/storage.md) and [Performance](../architecture/performance.md).

## Retry Mechanism

nanogit includes a pluggable retry mechanism, making operations more robust against transient network errors and server issues.

### Basic Retry Usage

```go
import "github.com/grafana/nanogit/retry"

ctx := context.Background()

// Enable retries with default exponential backoff
retrier := retry.NewExponentialBackoffRetrier()
ctx = retry.ToContext(ctx, retrier)

// All HTTP operations will now use retry logic
client, err := nanogit.NewHTTPClient(repo)
ref, err := client.GetRef(ctx, "main")
```

### Custom Retry Configuration

```go
// Configure retry behavior for production
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(5).                    // Retry up to 5 times
    WithInitialDelay(200 * time.Millisecond). // Start with 200ms delay
    WithMaxDelay(10 * time.Second).        // Cap at 10 seconds
    WithJitter()                           // Add random jitter

ctx = retry.ToContext(ctx, retrier)
```

**What gets retried:**
- Network errors (connection refused, timeouts)
- 5xx server errors (for GET requests)
- Temporary errors

**What does NOT get retried:**
- 4xx client errors (bad requests, auth failures)
- Context cancellation
- POST request 5xx errors (request body limitation)

Learn more about the [Retry Mechanism](../architecture/retry.md).

## Next Steps

- **[API Reference (GoDoc)](https://pkg.go.dev/github.com/grafana/nanogit)** - Complete API reference with all methods
- **[Storage Architecture](../architecture/storage.md)** - Pluggable storage and writing modes
- **[Retry Mechanism](../architecture/retry.md)** - Pluggable retry mechanism for robust operations
- **[Performance](../architecture/performance.md)** - Performance characteristics and benchmarks
- **[Architecture Overview](../architecture/overview.md)** - Core design principles
