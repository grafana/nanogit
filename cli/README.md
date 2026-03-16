# nanogit CLI

A command-line interface for [nanogit](https://github.com/grafana/nanogit), a lightweight, HTTPS-only Git implementation designed for cloud-native environments.

## Installation

### Download Pre-built Binary (Recommended)

Download the latest release for your platform:

**Linux / macOS**:
```bash
# Visit https://github.com/grafana/nanogit/releases/latest
# Download the appropriate binary for your platform
# Example for Linux AMD64:
wget https://github.com/grafana/nanogit/releases/latest/download/nanogit_Linux_x86_64.tar.gz
tar -xzf nanogit_Linux_x86_64.tar.gz
sudo mv nanogit /usr/local/bin/
```

**Windows**:
```powershell
# Download from https://github.com/grafana/nanogit/releases/latest
# Extract nanogit.exe and add to PATH
```

See the [releases page](https://github.com/grafana/nanogit/releases/latest) for all available platforms.

### Using Go Install

```bash
go install github.com/grafana/nanogit/cli/cmd/nanogit@latest
```

### Build from Source

```bash
git clone https://github.com/grafana/nanogit.git
cd nanogit
make cli-build
# Binary will be at bin/nanogit
```

## Usage

```bash
nanogit [flags]
nanogit [command]
```

### Global Flags

- `-h, --help` - Show help information
- `-v, --version` - Show version information

### Commands

#### ls-remote

List references from a remote Git repository.

```bash
# List all references
nanogit ls-remote https://github.com/grafana/nanogit.git

# List only branches
nanogit ls-remote https://github.com/grafana/nanogit.git --heads

# List only tags
nanogit ls-remote https://github.com/grafana/nanogit.git --tags

# Output as JSON
nanogit ls-remote https://github.com/grafana/nanogit.git --json

# With authentication (username defaults to 'git')
NANOGIT_TOKEN=token nanogit ls-remote https://github.com/user/private-repo.git

# With custom username
nanogit ls-remote https://github.com/user/private-repo.git --username myuser --token token
```

#### ls-tree

List the contents of a tree object.

```bash
# List files at root
nanogit ls-tree https://github.com/grafana/nanogit.git main

# List files in a directory
nanogit ls-tree https://github.com/grafana/nanogit.git main --path docs

# List all files recursively
nanogit ls-tree https://github.com/grafana/nanogit.git main --recursive

# Show detailed information
nanogit ls-tree https://github.com/grafana/nanogit.git main --long

# Output as JSON
nanogit ls-tree https://github.com/grafana/nanogit.git main --json

# With authentication
NANOGIT_TOKEN=token nanogit ls-tree https://github.com/user/private-repo.git main
```

#### cat-file

Display the contents of a file.

```bash
# Display file contents
nanogit cat-file https://github.com/grafana/nanogit.git main README.md

# Display file from a tag
nanogit cat-file https://github.com/grafana/nanogit.git v1.0.0 docs/api.md

# Output as JSON with metadata
nanogit cat-file https://github.com/grafana/nanogit.git main README.md --json

# With authentication
NANOGIT_TOKEN=token nanogit cat-file https://github.com/user/private-repo.git main file.txt
```

#### clone

Clone a repository to a local directory with optional path filtering.

```bash
# Clone entire repository
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo

# Clone only specific directories
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --include 'src/**' --include 'docs/**'

# Clone excluding certain paths
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --exclude 'node_modules/**' --exclude '*.tmp'

# Clone with include and exclude patterns
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --include 'src/**' --exclude '*.test.go'

# Clone from a tag
nanogit clone https://github.com/grafana/nanogit.git v1.0.0 ./my-repo

# Adjust performance (defaults: batch-size=50, concurrency=10)
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --batch-size 100 --concurrency 20

# With authentication
NANOGIT_TOKEN=token nanogit clone https://github.com/user/private-repo.git main ./my-repo
```

For more details, see the [CLI documentation](https://grafana.github.io/nanogit/getting-started/cli/).

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
