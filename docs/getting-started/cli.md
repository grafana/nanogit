# Command-Line Interface

nanogit provides a command-line interface for interacting with Git repositories from the terminal. The CLI is designed for cloud-native environments and supports essential Git operations over HTTPS.

## Installation

### From Source

Clone the repository and build the CLI:

```bash
git clone https://github.com/grafana/nanogit.git
cd nanogit
make cli-build
```

The binary will be available at `bin/nanogit`.

### Using Go Install

Install directly using Go:

```bash
go install github.com/grafana/nanogit/cli/cmd/nanogit@latest
```

## Basic Usage

Run `nanogit` to see the help information:

```bash
nanogit --help
```

Output:
```
nanogit is a lightweight, HTTPS-only Git implementation written in Go,
designed for cloud-native environments. It provides essential Git operations
optimized for server-side usage with pluggable storage backends.

For more information, visit: https://github.com/grafana/nanogit

Usage:
  nanogit [flags]

Flags:
  -h, --help      help for nanogit
  -v, --version   version for nanogit
```

### Check Version

```bash
nanogit --version
```

## Architecture

The CLI is implemented as a separate Go module in the `cli/` directory. This design keeps CLI-specific dependencies (like cobra) isolated from the main nanogit library, ensuring the library remains lightweight and focused.

### Module Structure

```
cli/
├── cmd/
│   └── nanogit/
│       └── main.go # CLI entry point
├── go.mod          # Separate module for CLI dependencies
└── README.md       # CLI-specific documentation
```

For local development, the workspace (`go.work`) automatically resolves the main nanogit library from the repository root.

## Development

### Building

Build the CLI binary:

```bash
make cli-build
```

### Code Quality

Format the code:

```bash
make cli-fmt
```

Run linters:

```bash
make cli-lint
```

## Future Enhancements

The CLI is currently in its initial state, providing basic help and version information. Future versions will include commands for:

- **Repository inspection**: List remote references, browse tree contents
- **File operations**: Read file contents from repositories
- **Cloning**: Clone repositories with path filtering support
- **Advanced features**: Authentication, JSON output, performance tuning

For the latest updates and planned features, see the [project roadmap](https://github.com/grafana/nanogit/issues).

## Contributing

Contributions to the CLI are welcome! When adding new commands or features:

1. Follow the existing code structure
2. Ensure all code is properly formatted (`make cli-fmt`)
3. Run linters and fix any issues (`make cli-lint`)
4. Test the binary builds successfully (`make cli-build`)

See the main [Contributing Guide](https://github.com/grafana/nanogit/blob/main/CONTRIBUTING.md) for more details.
