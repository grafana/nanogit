# nanogit

[![License](https://img.shields.io/github/license/grafana/nanogit)](LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/grafana/nanogit)](https://goreportcard.com/report/github.com/grafana/nanogit)
[![GoDoc](https://godoc.org/github.com/grafana/nanogit?status.svg)](https://godoc.org/github.com/grafana/nanogit)
[![codecov](https://codecov.io/gh/grafana/nanogit/branch/main/graph/badge.svg)](https://codecov.io/gh/grafana/nanogit)

## Overview

nanogit is a lightweight Git implementation designed for cloud environments, with a focus on HTTPS-based operations. It provides a subset of Git functionality optimized for reading and writing Git objects over HTTPS.

## Features

* Support any HTTPS Git service that supports the Git Smart HTTP Protocol (version 2).
* Secure HTTPS-based operations for Git objects (blobs, commits, trees, deltas)
* Remote Git reference management via HTTPS
* File system operations over HTTPS
* Commit comparison and diffing capabilities
* Authentication support (Basic Auth and API tokens)
* SHA-1 repository compatibility

## Non-Goals

The following features are explicitly not supported:

* `git://` and Git-over-SSH protocols
* File protocol (local Git operations)
* Commit signing and signature verification
* Full Git clones
* Git hooks
* Git configuration management
* Direct .git directory access
* "Dumb" servers
* Complex permissions (all objects use mode 0644)

## Why nanogit?

While [go-git](https://github.com/go-git/go-git) is a mature Git implementation, nanogit is designed for cloud-native, multitenant environments requiring minimal, stateless operations.

| Feature | nanogit | go-git |
|---------|---------|--------|
| Protocol | HTTPS-only | All protocols |
| Storage | Stateless, no local disk | Local disk operations |
| Scope | Essential operations only | Full Git functionality |
| Use Case | Cloud services, multitenant | General purpose |
| Resource Usage | Minimal footprint | Full Git features |

Choose nanogit for lightweight cloud services requiring stateless operations and minimal resources. Use go-git when you need full Git functionality, local operations, or advanced features.

## Getting Started

### Prerequisites

* Go 1.24 or later.
* Git (for development)

### Installation

```bash
go get github.com/grafana/nanogit
```

### Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/grafana/nanogit"
)

func main() {
    ctx := context.Background()

    // Create a new client with authentication
    client, err := nanogit.NewHTTPClient(
        "https://github.com/user/repo.git",
        nanogit.WithBasicAuth("username", "token"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Check if repository exists and is accessible
    authorized, err := client.IsAuthorized(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if !authorized {
        log.Fatal("Not authorized to access repository")
    }

    // Get the main branch reference
    ref, err := client.GetRef(ctx, "refs/heads/main")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Main branch points to: %s\n", ref.Hash.String())

    // Get the root tree from the latest commit
    commit, err := client.GetCommit(ctx, ref.Hash)
    if err != nil {
        log.Fatal(err)
    }

    // Read an existing file
    blob, err := client.GetBlobByPath(ctx, commit.Tree, "README.md")
    if err != nil {
        log.Printf("README.md not found: %v", err)
    } else {
        fmt.Printf("Current README.md content:\n%s\n", string(blob.Content))
    }

    // Create a staged writer to make changes
    writer, err := client.NewStagedWriter(ctx, ref)
    if err != nil {
        log.Fatal(err)
    }

    // Stage multiple changes
    // 1. Create a new file
    newFileContent := []byte("# New Feature\n\nThis is a new feature file.\n")
    _, err = writer.CreateBlob(ctx, "docs/new-feature.md", newFileContent)
    if err != nil {
        log.Fatal(err)
    }

    // 2. Update an existing file (if it exists)
    if blob != nil {
        updatedContent := append(blob.Content, []byte("\n## Recent Changes\n\nUpdated via nanogit!\n")...)
        _, err = writer.UpdateBlob(ctx, "README.md", updatedContent)
        if err != nil {
            log.Fatal(err)
        }
    }

    // 3. Create a configuration file
    configContent := []byte(`{
  "version": "1.0.0",
  "feature_enabled": true
}`)
    _, err = writer.CreateBlob(ctx, "config/settings.json", configContent)
    if err != nil {
        log.Fatal(err)
    }

    // Commit all staged changes
    author := nanogit.Author{
        Name:  "Automated Bot",
        Email: "bot@example.com",
        Time:  time.Now(),
    }
    
    commit, err = writer.Commit(ctx, "Add new feature and update documentation", author, author)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created commit: %s\n", commit.Hash.String())

    // Push changes to the remote repository
    err = writer.Push(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Successfully pushed changes to remote repository!")

    // List recent commits to verify our changes
    options := nanogit.ListCommitsOptions{
        PerPage: 5,
        Page:    1,
    }
    commits, err := client.ListCommits(ctx, commit.Hash, options)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nRecent commits:")
    for _, c := range commits {
        fmt.Printf("  %s: %s (by %s)\n", 
            c.Hash.String()[:8], 
            c.Message, 
            c.Author.Name)
    }
}
```

## Testing

nanogit includes generated mocks for easy unit testing. The mocks are generated using [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) and provide comprehensive test doubles for both the `Client` and `StagedWriter` interfaces.

For detailed testing examples and instructions, see [CONTRIBUTING.md](CONTRIBUTING.md#testing-with-mocks). You can also find complete working examples in [example_test.go](example_test.go).

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to submit pull requests, report issues, and set up your development environment.

## Code of Conduct

This project follows the [Grafana Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## License

This project is licensed under the [Apache License 2.0](LICENSE.md) - see the LICENSE file for details.

## Project Status

This project is currently in active development. While it's open source, it's important to note that it was initially created as part of a hackathon. We're working to make it production-ready, but please use it with appropriate caution.

## Resources

Want to learn how Git works? The following resources are useful:

* [Git on the Server - The Protocols](https://git-scm.com/book/ms/v2/Git-on-the-Server-The-Protocols)
* [Git Protocol v2](https://git-scm.com/docs/protocol-v2)
* [Pack Protocol](https://git-scm.com/docs/pack-protocol)
* [Git HTTP Backend](https://git-scm.com/docs/git-http-backend)
* [HTTP Protocol](https://git-scm.com/docs/http-protocol)
* [Git Protocol HTTP](https://git-scm.com/docs/gitprotocol-http)
* [Git Protocol v2](https://git-scm.com/docs/gitprotocol-v2)
* [Git Protocol Pack](https://git-scm.com/docs/gitprotocol-pack)
* [Git Protocol Common](https://git-scm.com/docs/gitprotocol-common)

## Security

If you find a security vulnerability, please report it to security@grafana.com. For more information, see our [Security Policy](SECURITY.md).

## Support

* GitHub Issues: [Create an issue](https://github.com/grafana/nanogit/issues)
* Community: [Grafana Community Forums](https://community.grafana.com)

## Acknowledgments

* The Grafana team for their support and guidance
* The open source community for their valuable feedback and contributions