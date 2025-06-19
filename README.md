# nanogit

[![License](https://img.shields.io/github/license/grafana/nanogit)](LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/grafana/nanogit)](https://goreportcard.com/report/github.com/grafana/nanogit)
[![GoDoc](https://godoc.org/github.com/grafana/nanogit?status.svg)](https://godoc.org/github.com/grafana/nanogit)
[![codecov](https://codecov.io/gh/grafana/nanogit/branch/main/graph/badge.svg)](https://codecov.io/gh/grafana/nanogit)

## Overview

nanogit is a lightweight Git implementation designed for cloud environments, with a focus on HTTPS-based operations. It provides a subset of Git functionality optimized for reading and writing Git objects over HTTPS.

## Features

- Support any HTTPS Git service that supports the Git Smart HTTP Protocol (version 2).
- Secure HTTPS-based operations for Git objects (blobs, commits, trees, deltas)
- Remote Git reference management via HTTPS
- File system operations over HTTPS
- Commit comparison and diffing capabilities
- Authentication support (Basic Auth and API tokens)
- SHA-1 repository compatibility

## Non-Goals

The following features are explicitly not supported:

- `git://` and Git-over-SSH protocols
- File protocol (local Git operations)
- Commit signing and signature verification
- Full Git clones
- Git hooks
- Git configuration management
- Direct .git directory access
- "Dumb" servers
- Complex permissions (all objects use mode 0644)

## Why nanogit?

While [go-git](https://github.com/go-git/go-git) is a mature Git implementation, nanogit is designed for cloud-native, multitenant environments requiring minimal, stateless operations.

| Feature        | nanogit                                                | go-git                 |
| -------------- | ------------------------------------------------------ | ---------------------- |
| Protocol       | HTTPS-only                                             | All protocols          |
| Storage        | Stateless, configurable storage (memory, disk, custom) | Local disk operations  |
| Scope          | Essential operations only                              | Full Git functionality |
| Use Case       | Cloud services, multitenant                            | General purpose        |
| Resource Usage | Minimal footprint                                      | Full Git features      |

Choose nanogit for lightweight cloud services requiring stateless operations and minimal resources. Use go-git when you need full Git functionality, local operations, or advanced features.

This are some of the performance differences between nanogit and go-git in some of the measured scenarios:

| Scenario                                      | Speed Improvement | Memory Usage Improvement                         |
| --------------------------------------------- | ----------------- | ------------------------------------------------ |
| **CreateFile (XL repo)**                      | 262x faster       | 195x less memory                                 |
| **UpdateFile (XL repo)**                      | 275x faster       | 180x less memory                                 |
| **DeleteFile (XL repo)**                      | 269x faster       | 212x less memory                                 |
| **BulkCreateFiles (1000 files, medium repo)** | 89x faster        | 24x more memory (regression, easy fix I believe) |
| **CompareCommits (XL repo)**                  | 111x faster       | 114x less memory                                 |
| **GetFlatTree (XL repo)**                     | 303x faster       | 137x less memory                                 |

For detailed performance metrics, see the [latest performance report](test/performance/LAST_REPORT.md).

## Getting Started

### Prerequisites

- Go 1.24 or later.
- Git (for development)

### Installation

```bash
go get github.com/grafana/nanogit
```

### Usage

```go
// Create client with authentication
client, err := nanogit.NewHTTPClient(
    "https://github.com/user/repo.git",
    options.WithBasicAuth("username", "token"),
)

// Get main branch and create staged writer
ref, err := client.GetRef(ctx, "refs/heads/main")
writer, err := client.NewStagedWriter(ctx, ref)

// Create and update files
writer.CreateBlob(ctx, "docs/new-feature.md", []byte("# New Feature"))
writer.UpdateBlob(ctx, "README.md", []byte("Updated content"))

// Commit changes with proper author/committer info
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
writer.Push(ctx)
```

## Storage and Caching

nanogit offers a flexible storage system for managing Git objects with multiple implementation options. Git objects (commits, trees, blobs) are immutable by design - once created, their content and hash remain constant. This immutability enables efficient caching and sharing of objects across repositories. Local caching of these objects is crucial for performance as it reduces network requests and speeds up common operations like diffing, merging, and history traversal.

### Flexibility on context caching

nanogit provides flexibility in how Git objects are cached through context-based storage configuration. This allows different parts of your application to use different storage strategies:

- Use in-memory storage for temporary operations
- Implement persistent storage for long-running processes
- Configure different storage backends per request
- Share storage across multiple operations
- Optimize storage based on specific use cases
- Scale storage independently of Git operations
- Persist Git objects across service restarts

The context-based approach enables fine-grained control over object caching while maintaining clean separation of concerns.

### In-Memory Storage

The default implementation uses an in-memory cache for Git objects, optimized for:

- Stateless operations requiring minimal resource footprint
- Temporary caching during Git operations
- High-performance read/write operations

### Custom Storage Implementations

The storage system is built with extensibility in mind through a well-defined interface. This allows you to:

- Implement custom storage backends (Redis, disk, etc.)
- Optimize storage for specific use cases
- Scale storage independently of the Git operations
- Persist Git objects across service restarts

To implement a custom storage backend, simply implement the `PackfileStorage` interface and put it using `storage.ToContext`.

## Testing

nanogit includes generated mocks for easy unit testing. The mocks are generated using [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) and provide comprehensive test doubles for both the `Client` and `StagedWriter` interfaces.

For detailed testing examples and instructions, see [CONTRIBUTING.md](CONTRIBUTING.md#testing-with-mocks). You can also find complete working examples in [mocks/example_test.go](mocks/example_test.go).

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

- [Git on the Server - The Protocols](https://git-scm.com/book/ms/v2/Git-on-the-Server-The-Protocols)
- [Git Protocol v2](https://git-scm.com/docs/protocol-v2)
- [Pack Protocol](https://git-scm.com/docs/pack-protocol)
- [Git HTTP Backend](https://git-scm.com/docs/git-http-backend)
- [HTTP Protocol](https://git-scm.com/docs/http-protocol)
- [Git Protocol HTTP](https://git-scm.com/docs/gitprotocol-http)
- [Git Protocol v2](https://git-scm.com/docs/gitprotocol-v2)
- [Git Protocol Pack](https://git-scm.com/docs/gitprotocol-pack)
- [Git Protocol Common](https://git-scm.com/docs/gitprotocol-common)

## Security

If you find a security vulnerability, please report it to <security@grafana.com>. For more information, see our [Security Policy](SECURITY.md).

## Support

- GitHub Issues: [Create an issue](https://github.com/grafana/nanogit/issues)
- Community: [Grafana Community Forums](https://community.grafana.com)

## Acknowledgments

- The Grafana team for their support and guidance
- The open source community for their valuable feedback and contributions

