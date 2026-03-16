# nanogit CLI

A command-line interface for [nanogit](https://github.com/grafana/nanogit), a lightweight, HTTPS-only Git implementation designed for cloud-native environments.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/grafana/nanogit.git
cd nanogit

# Build the CLI
make cli-build

# The binary will be available at bin/nanogit
./bin/nanogit --version
```

### Using Go Install

```bash
go install github.com/grafana/nanogit/cli/cmd/nanogit@latest
```

## Usage

```bash
nanogit [flags]
```

### Flags

- `-h, --help` - Show help information
- `-v, --version` - Show version information

## Development

The CLI is a separate Go module located in the `cli/` directory. This keeps CLI-specific dependencies (like cobra) isolated from the main nanogit library.

### Project Structure

```
cli/
├── cmd/
│   └── nanogit/
│       └── main.go # CLI entry point
├── go.mod          # Separate module for CLI dependencies
└── README.md       # This file
```

### Building

```bash
make cli-build
```

### Code Quality

```bash
# Format code
make cli-fmt

# Run linters
make cli-lint
```

## Architecture

The CLI is designed as a separate Go module to avoid polluting the main library with CLI-specific dependencies. It uses:

- **cobra** - Command-line interface framework
- **go.work** - Workspace configuration for local development (automatically resolves the main library)

## Contributing

Contributions are welcome! Please ensure:

1. Code is properly formatted (`make cli-fmt`)
2. All linters pass (`make cli-lint`)
3. The binary builds successfully (`make cli-build`)

## License

Same as the main nanogit project. See the root [LICENSE](../LICENSE) file for details.
