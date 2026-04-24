# Command-Line Interface

nanogit provides a command-line interface for interacting with Git repositories from the terminal. The CLI is designed for cloud-native environments and supports essential Git operations over HTTPS.

**Purpose**: This CLI is primarily a **testing and demonstration tool** for the nanogit library. It provides a git-compatible command-line interface that can serve as a swap-in replacement for basic git operations, making it useful for testing the library and demonstrating its capabilities.

## Installation

### Download Pre-built Binary (Recommended)

Download the latest release for your platform from the [releases page](https://github.com/grafana/nanogit/releases/latest).

**Linux**:
```bash
# AMD64
wget https://github.com/grafana/nanogit/releases/latest/download/nanogit_Linux_x86_64.tar.gz
tar -xzf nanogit_Linux_x86_64.tar.gz
sudo mv nanogit /usr/local/bin/

# ARM64
wget https://github.com/grafana/nanogit/releases/latest/download/nanogit_Linux_arm64.tar.gz
tar -xzf nanogit_Linux_arm64.tar.gz
sudo mv nanogit /usr/local/bin/
```

**macOS**:
```bash
# Apple Silicon (M1/M2/M3)
wget https://github.com/grafana/nanogit/releases/latest/download/nanogit_Darwin_arm64.tar.gz
tar -xzf nanogit_Darwin_arm64.tar.gz
sudo mv nanogit /usr/local/bin/

# Intel
wget https://github.com/grafana/nanogit/releases/latest/download/nanogit_Darwin_x86_64.tar.gz
tar -xzf nanogit_Darwin_x86_64.tar.gz
sudo mv nanogit /usr/local/bin/
```

**Windows**:
```powershell
# Download and extract (PowerShell)
Invoke-WebRequest -Uri "https://github.com/grafana/nanogit/releases/latest/download/nanogit_Windows_x86_64.zip" -OutFile "nanogit.zip"
Expand-Archive nanogit.zip -DestinationPath .
Move-Item nanogit.exe C:\Windows\System32\
```

### Using Go Install

If you have Go installed:

```bash
go install github.com/grafana/nanogit/cli/cmd/nanogit@latest
```

### Build from Source

For development:

```bash
git clone https://github.com/grafana/nanogit.git
cd nanogit
make cli-build
# Binary will be at bin/nanogit
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
  -h, --help              help for nanogit
      --version           version for nanogit
      --username string   Authentication username (can also use NANOGIT_USERNAME env var, defaults to 'git')
      --token string      Authentication token (can also use NANOGIT_TOKEN env var)
      --json              Output results in JSON format
  -v, --verbose           Be verbose (emit Info-level logs to stderr; set NANOGIT_TRACE=1 for Debug/wire detail)
```

### Global Flags

The following flags are available for all commands:

- `--username` - Authentication username (defaults to 'git' if not specified)
- `--token` - Authentication token for private repositories
- `--json` - Output results in JSON format (where applicable)
- `-v`, `--verbose` - Emit Info-level logs to stderr (see [Verbose mode](#verbose-mode))

These flags can also be set via environment variables:
- `NANOGIT_USERNAME` - Authentication username
- `NANOGIT_TOKEN` - Authentication token
- `NANOGIT_TRACE` - When set (to any non-empty value), emits Debug-level wire/protocol logs to stderr

## Verbose mode

nanogit follows git's verbosity conventions. By default, only Warn and Error messages are emitted on stderr. Two levels of extra output are available:

- **`-v` / `--verbose`** — enables Info-level progress messages on stderr. Behaves like `git push -v`.
- **`NANOGIT_TRACE=1`** — enables Debug-level wire/protocol detail on stderr. Mirrors git's `GIT_TRACE=1` / `GIT_CURL_VERBOSE=1` environment variables. Works independently of `-v`.

All log output is written to **stderr**, so `stdout` stays clean for piping commit hashes, JSON output, or blob contents.

```bash
# Info-level progress
nanogit -v clone https://github.com/grafana/nanogit.git ./tmp

# Full wire trace (Debug-level)
NANOGIT_TRACE=1 nanogit clone https://github.com/grafana/nanogit.git ./tmp

# Combine with --json: stdout stays valid JSON
nanogit -v --json ls-tree https://github.com/grafana/nanogit.git main
```

## Authentication

For private repositories, use the `--token` global flag or set the `NANOGIT_TOKEN` environment variable:

```bash
# Using global flag (can be placed before or after command)
nanogit --token YOUR_TOKEN <command> [args...]

# Using environment variable (recommended for repeated use)
export NANOGIT_TOKEN=YOUR_TOKEN
nanogit <command> [args...]

# Custom username (defaults to 'git')
nanogit --username myuser --token YOUR_TOKEN <command> [args...]
```

### Check Version

```bash
nanogit --version
```

## Requirements

nanogit requires **Git Smart HTTP Protocol v2**. Most modern Git hosting providers support protocol v2, but some older servers or certain cloud providers may only support protocol v1.

Use the `check` command to verify if your Git server is compatible before attempting other operations. For a complete round-trip that also exercises the read and write paths, see [Server Compatibility](server-compatibility.md).

## Commands

### check

Check if a Git server is compatible with nanogit by verifying protocol v2 support.

**Usage**:
```bash
nanogit check <repository> [flags]
```

No command-specific flags (uses global flags only).

**Examples**:

Check repository compatibility:
```bash
nanogit check https://example.com/repo.git
```

Output as JSON:
```bash
nanogit --json check https://example.com/repo.git
```

Check with authentication:
```bash
nanogit check https://example.com/private-repo.git --token <token>
```

### ls-remote

List references (branches and tags) from a remote Git repository.

**Usage**:
```bash
nanogit ls-remote <repository> [flags]
```

**Flags**:
- `--heads` - Show only branch references (refs/heads/*)
- `--tags` - Show only tag references (refs/tags/*)

**Examples**:

List all references:
```bash
nanogit ls-remote https://github.com/grafana/nanogit.git
```

List only branches:
```bash
nanogit ls-remote https://github.com/grafana/nanogit.git --heads
```

List only tags:
```bash
nanogit ls-remote https://github.com/grafana/nanogit.git --tags
```

Output as JSON:
```bash
nanogit --json ls-remote https://github.com/grafana/nanogit.git
```

### ls-tree

List the contents of a tree object at a specific reference.

**Usage**:
```bash
nanogit ls-tree <repository> <ref> [flags]
```

**Flags**:
- `-r, --recursive` - List tree contents recursively
- `--name-only` - Show only file names (default shows mode, type, hash, and name)
- `--path` - Path within the tree to list (defaults to root)

**Examples**:

List files at root (shows mode, type, hash, and name by default):
```bash
nanogit ls-tree https://github.com/grafana/nanogit.git main
```

List files in a specific directory:
```bash
nanogit ls-tree https://github.com/grafana/nanogit.git main --path docs
```

List all files recursively:
```bash
nanogit ls-tree https://github.com/grafana/nanogit.git main --recursive
```

Show only file names:
```bash
nanogit ls-tree https://github.com/grafana/nanogit.git main --name-only
```

Output as JSON:
```bash
nanogit --json ls-tree https://github.com/grafana/nanogit.git main
```

### cat-file

Display the contents of a file from a repository.

**Usage**:
```bash
nanogit cat-file <repository> <ref> <path> [flags]
```

No command-specific flags (uses global flags only).

**Examples**:

Display file contents:
```bash
nanogit cat-file https://github.com/grafana/nanogit.git main README.md
```

Display file from a specific tag:
```bash
nanogit cat-file https://github.com/grafana/nanogit.git v1.0.0 docs/api.md
```

Display file from a commit hash:
```bash
nanogit cat-file https://github.com/grafana/nanogit.git abc123def456 src/main.go
```

Output with metadata in JSON format:
```bash
nanogit --json cat-file https://github.com/grafana/nanogit.git main README.md
```

### clone

Clone a repository to a local directory with optional path filtering.

**Usage**:
```bash
nanogit clone <repository> <ref> <destination> [flags]
```

**Flags**:
- `--include` - Include paths (glob patterns, can be specified multiple times)
- `--exclude` - Exclude paths (glob patterns, can be specified multiple times)
- `--batch-size` - Number of blobs to fetch per request (default: 50)
- `--concurrency` - Number of parallel blob fetches (default: 10)

**Examples**:

Clone default branch (HEAD) to current directory:
```bash
nanogit clone https://github.com/grafana/nanogit.git
```

Clone to specific directory:
```bash
nanogit clone https://github.com/grafana/nanogit.git ./my-repo
```

Clone specific branch:
```bash
nanogit clone https://github.com/grafana/nanogit.git --ref main ./my-repo
```

Clone specific tag:
```bash
nanogit clone https://github.com/grafana/nanogit.git --ref v1.0.0 ./my-repo
```

Clone only specific directories:
```bash
nanogit clone https://github.com/grafana/nanogit.git ./my-repo --include 'src/**' --include 'docs/**'
```

Clone excluding certain paths:
```bash
nanogit clone https://github.com/grafana/nanogit.git ./my-repo --exclude 'node_modules/**'
```

Adjust performance settings (defaults are batch-size=50, concurrency=10):
```bash
# Increase for better performance with large repositories
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --batch-size 100 --concurrency 20

# Sequential mode for constrained environments
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --batch-size 1 --concurrency 1
```

**Path Filtering**:

Path filtering uses glob patterns to include or exclude specific files and directories:
- `**` matches any number of directories
- `*` matches any characters within a directory
- `?` matches a single character
- Exclude patterns take precedence over include patterns

**Performance Tuning**:

The clone command uses batching and concurrency to optimize performance:
- **Batch size** (default: 50) - Number of blobs to fetch in a single request. Higher values reduce network round-trips for repositories with many small files.
- **Concurrency** (default: 10) - Number of parallel fetch operations. Higher values improve throughput by utilizing network bandwidth more effectively.

Recommended settings:
- Large repositories: `--batch-size 100 --concurrency 20`
- Default (balanced): `--batch-size 50 --concurrency 10` (automatic)
- Constrained environments: `--batch-size 1 --concurrency 1` (sequential)

### put-file

Create or update a file on a branch in a single commit. The command stages the blob, commits, and pushes in one step — there is no separate staging area. The ref argument must resolve to a **branch**; tags and raw commit hashes are rejected because staged writes target branch tips.

Content is read from **stdin** by default, or from a local file with `--from-file`. Author identity must be supplied explicitly (via `--author` or the `NANOGIT_AUTHOR_NAME` / `NANOGIT_AUTHOR_EMAIL` environment variables) — nanogit does not fabricate a default identity. The committer defaults to the author when not overridden.

**Usage**:
```bash
nanogit put-file <repository> <ref> <path> [-] [flags]
```

The optional trailing `-` indicates stdin explicitly; it can also be omitted. When `--from-file` is used, stdin is ignored.

**Flags**:
- `-m, --message <string>` — commit message (required)
- `--from-file <path>` — read content from a local file instead of stdin
- `--author "Name <email>"` — commit author; falls back to `NANOGIT_AUTHOR_NAME` and `NANOGIT_AUTHOR_EMAIL`. Errors out if unresolved (no silent default)
- `--committer "Name <email>"` — commit committer; falls back to `NANOGIT_COMMITTER_NAME` / `NANOGIT_COMMITTER_EMAIL`, and finally to the author

**Output**: the new commit hash is printed to stdout. With `--json`, a `{"commit": "...", "path": "..."}` object is emitted instead. All log output (including `-v` / `NANOGIT_TRACE`) goes to stderr, so the commit hash is safe to pipe or capture.

**Examples**:

Pipe content on stdin:
```bash
echo "hello" | nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" \
  --author "Jane Doe <jane@example.com>"
```

Read content from a local file:
```bash
nanogit put-file https://github.com/user/repo.git main docs/note.md \
  --from-file ./local.md \
  -m "add note" \
  --author "Jane Doe <jane@example.com>"
```

Use environment variables for the identity (recommended for scripts/CI):
```bash
export NANOGIT_AUTHOR_NAME="Jane Doe"
export NANOGIT_AUTHOR_EMAIL="jane@example.com"
echo "hello" | nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note"
```

Distinct committer identity:
```bash
nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" \
  --author "Jane Doe <jane@example.com>" \
  --committer "CI Bot <ci@example.com>" \
  --from-file ./local.md
```

Observe the protocol layer with verbose/trace output:
```bash
# Info-level progress on stderr, commit hash on stdout
echo "hello" | nanogit -v put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" --author "Jane Doe <jane@example.com>"

# Full wire-level Debug output
echo "hello" | NANOGIT_TRACE=1 nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" --author "Jane Doe <jane@example.com>"
```

Capture the new commit hash in a script:
```bash
commit=$(echo "hello" | nanogit put-file https://github.com/user/repo.git main docs/note.md \
  -m "add note" --author "Jane Doe <jane@example.com>")
echo "Pushed commit: $commit"
```

**Notes**:
- The command is idempotent in the sense that it will `UpdateBlob` if the path already exists on the branch and `CreateBlob` otherwise — callers don't need to know which case applies.
- If the push fails, the in-memory staged writer is cleaned up automatically. No local state is left behind (nanogit is stateless).
- `put-file` is a porcelain convenience for testing and demonstrating the nanogit write path. For batched writes across multiple files, use the `StagedWriter` API from the library directly.
