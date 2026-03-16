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

## Future Commands

The CLI is currently in its initial state. Future versions will include:

- List remote references (branches, tags)
- Browse repository tree contents
- Read file contents from repositories
- Clone repositories with path filtering
- Authentication support
- JSON output for automation
