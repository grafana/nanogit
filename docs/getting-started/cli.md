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
  -v, --version           version for nanogit
      --username string   Authentication username (can also use NANOGIT_USERNAME env var, defaults to 'git')
      --token string      Authentication token (can also use NANOGIT_TOKEN env var)
      --json              Output results in JSON format
```

### Global Flags

The following flags are available for all commands:

- `--username` - Authentication username (defaults to 'git' if not specified)
- `--token` - Authentication token for private repositories
- `--json` - Output results in JSON format (where applicable)

These flags can also be set via environment variables:
- `NANOGIT_USERNAME` - Authentication username
- `NANOGIT_TOKEN` - Authentication token

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

Use the `check` command to verify if your Git server is compatible before attempting other operations.

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
