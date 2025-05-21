# nanogit

A limited, cloud-ready Git implementation for use in Grafana.

[![License](https://img.shields.io/github/license/grafana/nanogit)](LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/grafana/nanogit)](https://goreportcard.com/report/github.com/grafana/nanogit)
[![GoDoc](https://godoc.org/github.com/grafana/nanogit?status.svg)](https://godoc.org/github.com/grafana/nanogit)
[![codecov](https://codecov.io/gh/grafana/nanogit/branch/main/graph/badge.svg)](https://codecov.io/gh/grafana/nanogit)

## Overview

nanogit is a lightweight Git implementation designed for cloud environments, with a focus on HTTPS-based operations. It provides a subset of Git functionality optimized for reading and writing Git objects over HTTPS, particularly targeting GitHub.com integration.

## Features

* Read Git files over HTTPS on github.com
* Read Git trees over HTTPS on github.com
* Write new Git objects over HTTPS on github.com
* Write Git object deltas over HTTPS on github.com
* Support for SHA-1 hashing in repositories

## Future Goals

* Support any HTTPS Git service that supports `git-upload-pack` on `Git-Protocol: version=2` (e.g., GitLab)
* Support SHA-256 repositories on top of SHA-1 repositories

## Non-Goals

The following features are explicitly not supported:

* `git://` and Git-over-SSH protocols
* File protocol (local Git operations)
* Commit signing and signature verification
* Full Git clones
* Tag creation (reading is supported)
* Git hooks
* Git configuration management
* Direct .git directory access
* Git deltas (`git diff`) in outputs or API
* "Dumb" servers
* Branch renames
* Complex permissions (all objects use mode 0644)

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
import "github.com/grafana/nanogit"

// Example usage will be added as the project matures
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to submit pull requests, report issues, and set up your development environment.

## Code of Conduct

This project follows the [Grafana Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## License

This project is licensed under the [Apache License 2.0](LICENSE.md) - see the LICENSE file for details.

## Project Status

This project is currently in active development. While it's open source, it's important to note that it was initially created as part of a hackathon. We're working to make it production-ready, but please use it with appropriate caution.

## Documentation

* [API Documentation](https://godoc.org/github.com/grafana/nanogit)
* [Contributing Guide](CONTRIBUTING.md)
* [Code of Conduct](CODE_OF_CONDUCT.md)

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

## Why nanogit?

While [go-git](https://github.com/go-git/go-git) is a mature Git implementation, we created nanogit for cloud-native, multitenant environments where a minimal, stateless approach is essential:

### Key Differences from go-git

1. **Cloud-Native Design**
   - HTTPS-only
   - No local disk operations or full clones
   - Stateless by design for multitenant environments
   - Minimal memory and network footprint per operation

2. **Focused Scope**
   - Essential Git operations only
   - No hooks, signing, or configuration management
   - Clear boundaries on supported features
   - Smaller security surface area

### When to Use nanogit

Choose nanogit when you need:
- A lightweight Git client for cloud services
- Stateless, multitenant Git operations
- Minimal resource usage

### When to Use go-git

Consider using go-git when you need:
- Full Git functionality
- Local disk operations
- All Git protocols (git://, ssh://, file://)
- Advanced Git features
