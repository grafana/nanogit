# nanogit

[![GitHub Release](https://img.shields.io/github/v/release/grafana/nanogit)](https://github.com/grafana/nanogit/releases)
[![License](https://img.shields.io/github/license/grafana/nanogit)](https://github.com/grafana/nanogit/blob/main/LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/grafana/nanogit)](https://goreportcard.com/report/github.com/grafana/nanogit)
[![GoDoc](https://godoc.org/github.com/grafana/nanogit?status.svg)](https://godoc.org/github.com/grafana/nanogit)
[![codecov](https://codecov.io/gh/grafana/nanogit/branch/main/graph/badge.svg)](https://codecov.io/gh/grafana/nanogit)

## Overview

nanogit is a lightweight, cloud-native Git implementation designed for applications that need efficient Git operations over HTTPS without the complexity and resource overhead of traditional Git implementations.

## Features

- **HTTPS-only Git operations** - Works with any Git service supporting Smart HTTP Protocol v2 (GitHub, GitLab, Bitbucket, etc.), eliminating the need for SSH key management in cloud environments

- **Stateless architecture** - No local .git directory dependency, making it perfect for serverless functions, containers, and microservices where persistent local state isn't available or desired

- **Memory-optimized design** - Streaming packfile operations and configurable writing modes minimize memory usage, crucial for bulk operations and memory-constrained environments

- **Flexible storage architecture** - Pluggable object storage and configurable writing modes allow optimization for different deployment patterns, from high-performance in-memory operations to memory-efficient disk-based processing

- **Cloud-native authentication** - Built-in support for Basic Auth and API tokens, designed for automated workflows and CI/CD systems without interactive authentication

- **Essential Git operations** - Focused on core functionality (read/write objects, commit operations, diffing) without the complexity of full Git implementations, reducing attack surface and resource requirements

- **High performance** - Significantly faster than traditional Git implementations for common cloud operations, with up to 300x speed improvements for certain scenarios

## Non-Goals

The following features are explicitly not supported:

- `git://` and Git-over-SSH protocols
- File protocol (local Git operations)
- Commit signing and signature verification
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
| Storage        | Stateless, configurable object storage + writing modes | Local disk operations  |
| Cloning        | Path filtering with glob patterns, shallow clones      | Full repository clones |
| Scope          | Essential operations only                              | Full Git functionality |
| Use Case       | Cloud services, multitenant                            | General purpose        |
| Resource Usage | Minimal footprint                                      | Full Git features      |

Choose nanogit for lightweight cloud services requiring stateless operations and minimal resources. Use go-git when you need full Git functionality, local operations, or advanced features.

These are some of the performance differences between nanogit and go-git in some of the measured scenarios:

| Scenario                                  | Speed       | Memory Usage |
| ----------------------------------------- | ----------- | ------------ |
| CreateFile (XL repo)                      | 306x faster | 186x less    |
| UpdateFile (XL repo)                      | 291x faster | 178x less    |
| DeleteFile (XL repo)                      | 302x faster | 175x less    |
| BulkCreateFiles (1000 files, medium repo) | 607x faster | 11x less     |
| CompareCommits (XL repo)                  | 60x faster  | 96x less     |
| GetFlatTree (XL repo)                     | 258x faster | 160x less    |

For detailed performance metrics, see the [performance analysis](architecture/performance.md).

## Getting Started

Ready to use nanogit? Check out our guides:

- **[Installation](getting-started/installation.md)** - Install nanogit in your project
- **[Quick Start](getting-started/quick-start.md)** - Basic usage examples and common patterns
- **[API Reference](https://pkg.go.dev/github.com/grafana/nanogit)** - Complete API documentation on GoDoc

## Architecture

Learn about nanogit's design and internals:

- **[Architecture Overview](architecture/overview.md)** - Core design principles and components
- **[Storage Backend](architecture/storage.md)** - Pluggable storage and writing modes
- **[Delta Resolution](architecture/delta-resolution.md)** - Git delta handling implementation
- **[Performance](architecture/performance.md)** - Performance characteristics and benchmarks

## Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/grafana/nanogit/blob/main/CONTRIBUTING.md) for details on how to submit pull requests, report issues, and set up your development environment.

## Code of Conduct

This project follows the [Grafana Code of Conduct](https://github.com/grafana/nanogit/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## License

This project is licensed under the [Apache License 2.0](https://github.com/grafana/nanogit/blob/main/LICENSE.md) - see the LICENSE file for details.

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

If you find a security vulnerability, please report it to <security@grafana.com>. For more information, see our [Security Policy](https://github.com/grafana/nanogit/blob/main/SECURITY.md).

## Support

- GitHub Issues: [Create an issue](https://github.com/grafana/nanogit/issues)
- Community: [Grafana Community Forums](https://community.grafana.com)

## Acknowledgments

- The Grafana team for their support and guidance
- The open source community for their valuable feedback and contributions
