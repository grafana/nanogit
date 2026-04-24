# nanogit CLI

A command-line interface for [nanogit](https://github.com/grafana/nanogit), a lightweight, HTTPS-only Git implementation designed for cloud-native environments.

**Purpose**: This CLI is primarily a **testing and demonstration tool** for the nanogit library. It provides a git-compatible command-line interface that can serve as a swap-in replacement for basic git operations, making it useful for testing the library and demonstrating its capabilities.

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
nanogit [global flags] <command> [command flags] [arguments]
```

### Global Flags

Available for all commands:

- `-h, --help` - Show help information
- `--version` - Show version information
- `--username <string>` - Authentication username (defaults to 'git', can also use `NANOGIT_USERNAME` env var)
- `--token <string>` - Authentication token (can also use `NANOGIT_TOKEN` env var)
- `--json` - Output results in JSON format (where applicable)
- `-v, --verbose` - Emit Info-level logs to stderr (see [Verbose mode](#verbose-mode))

### Verbose mode

nanogit follows git's verbosity conventions. By default only warnings and errors are emitted. Two levels of extra output are available:

- **`-v` / `--verbose`** — enables Info-level progress messages on stderr, like `git push -v`.
- **`NANOGIT_TRACE=1`** environment variable — enables Debug-level wire/protocol detail, like `GIT_TRACE=1` / `GIT_CURL_VERBOSE=1`. The env var works on its own — you don't also need `-v`.

All log output goes to stderr, so `stdout` stays pipeable even when verbose mode is on.

```bash
# Info-level progress
nanogit -v clone https://github.com/grafana/nanogit.git ./tmp

# Full wire trace (Debug)
NANOGIT_TRACE=1 nanogit clone https://github.com/grafana/nanogit.git ./tmp

# Combine with --json: stdout stays valid JSON, logs go to stderr
nanogit -v --json ls-tree https://github.com/grafana/nanogit.git main
```

### Default repository

Set `NANOGIT_REPO` once to avoid repeating the repository URL on every command. When it is set, the `<repository>` positional argument becomes optional; pass an explicit URL to override.

```bash
export NANOGIT_REPO=https://github.com/user/repo.git
nanogit check
nanogit ls-tree main
echo "hi" | nanogit put-file main note.md -m "add note"
nanogit cat-file main note.md
```

For `clone`, a single positional argument is treated as the destination path unless it looks like a URL (contains `://`).

### Authentication

For private repositories, use the `--token` flag or set the `NANOGIT_TOKEN` environment variable:

```bash
# Using global flag
nanogit --token YOUR_TOKEN <command> [args...]

# Using environment variable (recommended)
export NANOGIT_TOKEN=YOUR_TOKEN
nanogit <command> [args...]

# Custom username (defaults to 'git')
nanogit --username myuser --token YOUR_TOKEN <command> [args...]
```

### Requirements

nanogit requires **Git Smart HTTP Protocol v2**. Most modern Git hosting providers support protocol v2, but some older servers or certain cloud providers may only support protocol v1.

Use `nanogit check <repository>` to verify compatibility.

### Commands

#### check

Check if a Git server supports protocol v2 and is compatible with nanogit.

```bash
# Check compatibility
nanogit check https://example.com/repo.git

# Check with authentication
nanogit check https://example.com/private-repo.git --token <token>

# Output as JSON
nanogit --json check https://example.com/repo.git
```

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
nanogit --json ls-remote https://github.com/grafana/nanogit.git
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

# Show only file names
nanogit ls-tree https://github.com/grafana/nanogit.git main --name-only

# Output as JSON
nanogit --json ls-tree https://github.com/grafana/nanogit.git main
```

#### cat-file

Display the contents of a file.

```bash
# Display file contents
nanogit cat-file https://github.com/grafana/nanogit.git main README.md

# Display file from a tag
nanogit cat-file https://github.com/grafana/nanogit.git v1.0.0 docs/api.md

# Output as JSON with metadata
nanogit --json cat-file https://github.com/grafana/nanogit.git main README.md
```

#### clone

Clone a repository to a local directory with optional path filtering.

```bash
# Clone default branch (HEAD) to current directory
nanogit clone https://github.com/grafana/nanogit.git

# Clone to specific directory
nanogit clone https://github.com/grafana/nanogit.git ./my-repo

# Clone specific branch
nanogit clone https://github.com/grafana/nanogit.git --ref main ./my-repo

# Clone specific tag
nanogit clone https://github.com/grafana/nanogit.git --ref v1.0.0 ./my-repo

# Clone only specific directories
nanogit clone https://github.com/grafana/nanogit.git ./my-repo --include 'src/**' --include 'docs/**'

# Clone excluding certain paths
nanogit clone https://github.com/grafana/nanogit.git ./my-repo --exclude 'node_modules/**'

# Adjust performance (defaults: batch-size=50, concurrency=10)
nanogit clone https://github.com/grafana/nanogit.git ./my-repo --batch-size 100 --concurrency 20
```

#### put-file

Create or update a file on a branch in a single commit. The change is staged, committed, and pushed in one step. The ref must be a branch — tags and raw commit hashes are rejected because staged writes target branch tips.

Content is read from **stdin** by default, or from a local file with `--from-file`. Author identity must be supplied explicitly (via `--author` or the `NANOGIT_AUTHOR_NAME` / `NANOGIT_AUTHOR_EMAIL` environment variables) — nanogit does not fabricate a default. Committer defaults to the author unless overridden.

```bash
# Pipe content on stdin
echo "hello" | nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" \
  --author "Jane Doe <jane@example.com>"

# Read content from a local file
nanogit put-file https://github.com/user/repo.git main docs/note.md \
  --from-file ./local.md \
  -m "add note" \
  --author "Jane Doe <jane@example.com>"

# Author via environment (recommended for scripts)
export NANOGIT_AUTHOR_NAME="Jane Doe"
export NANOGIT_AUTHOR_EMAIL="jane@example.com"
echo "hello" | nanogit put-file https://github.com/user/repo.git main docs/note.md -m "add note"

# Distinct committer
nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" \
  --author "Jane Doe <jane@example.com>" \
  --committer "CI Bot <ci@example.com>" \
  --from-file ./local.md

# With verbose/trace output to see what the protocol layer is doing
echo "hello" | nanogit -v put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" --author "Jane Doe <jane@example.com>"

echo "hello" | NANOGIT_TRACE=1 nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" --author "Jane Doe <jane@example.com>"

# JSON output ({commit, path})
echo "hello" | nanogit --json put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" --author "Jane Doe <jane@example.com>"
```

**Flags:**

- `-m, --message <string>` — commit message (required)
- `--from-file <path>` — read content from a local file (mutually exclusive with stdin marker `-`)
- `--author "Name <email>"` — commit author; falls back to `NANOGIT_AUTHOR_NAME` and `NANOGIT_AUTHOR_EMAIL`, errors if unresolved
- `--committer "Name <email>"` — commit committer; falls back to `NANOGIT_COMMITTER_NAME`/`NANOGIT_COMMITTER_EMAIL`, then to the author

**Non-default output** is printed to stdout as the new commit hash (or as `{"commit": "...", "path": "..."}` when `--json` is set). All log output (including `-v` / `NANOGIT_TRACE`) goes to stderr, so the commit hash is safe to pipe or capture.

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
