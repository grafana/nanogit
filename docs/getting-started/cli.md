# Command-Line Interface

nanogit provides a command-line interface for interacting with Git repositories from the terminal. The CLI is designed for cloud-native environments and supports essential Git operations over HTTPS.

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
  -h, --help      help for nanogit
  -v, --version   version for nanogit
```

### Check Version

```bash
nanogit --version
```

## Commands

### ls-remote

List references (branches and tags) from a remote Git repository.

**Usage**:
```bash
nanogit ls-remote <repository> [flags]
```

**Flags**:
- `--heads` - Show only branch references (refs/heads/*)
- `--tags` - Show only tag references (refs/tags/*)
- `--json` - Output results in JSON format
- `--username` - Authentication username (defaults to 'git')
- `--token` - Authentication token

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
nanogit ls-remote https://github.com/grafana/nanogit.git --json
```

With authentication (for private repositories):
```bash
# Using token (username defaults to 'git')
nanogit ls-remote https://github.com/user/private-repo.git --token YOUR_TOKEN

# Using environment variable
NANOGIT_TOKEN=YOUR_TOKEN nanogit ls-remote https://github.com/user/private-repo.git

# With custom username
nanogit ls-remote https://github.com/user/private-repo.git --username myuser --token YOUR_TOKEN

# Using environment variables for both
NANOGIT_USERNAME=myuser NANOGIT_TOKEN=YOUR_TOKEN nanogit ls-remote https://github.com/user/private-repo.git
```

**Environment Variables**:
- `NANOGIT_TOKEN` - Authentication token
- `NANOGIT_USERNAME` - Authentication username (defaults to 'git' if not set)

### ls-tree

List the contents of a tree object at a specific reference.

**Usage**:
```bash
nanogit ls-tree <repository> <ref> [flags]
```

**Flags**:
- `-r, --recursive` - List tree contents recursively
- `-l, --long` - Show detailed information (mode, type, hash)
- `--json` - Output results in JSON format
- `--path` - Path within the tree to list (defaults to root)
- `--username` - Authentication username (defaults to 'git')
- `--token` - Authentication token

**Examples**:

List files at root:
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

Show detailed information:
```bash
nanogit ls-tree https://github.com/grafana/nanogit.git v1.0.0 --long
```

Output as JSON:
```bash
nanogit ls-tree https://github.com/grafana/nanogit.git main --json
```

With authentication:
```bash
# Using token (username defaults to 'git')
nanogit ls-tree https://github.com/user/private-repo.git main --token YOUR_TOKEN

# Using environment variables
NANOGIT_TOKEN=YOUR_TOKEN nanogit ls-tree https://github.com/user/private-repo.git main
```

### cat-file

Display the contents of a file from a repository.

**Usage**:
```bash
nanogit cat-file <repository> <ref> <path> [flags]
```

**Flags**:
- `--json` - Output file metadata in JSON format
- `--username` - Authentication username (defaults to 'git')
- `--token` - Authentication token

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
nanogit cat-file https://github.com/grafana/nanogit.git main README.md --json
```

With authentication:
```bash
# Using token (username defaults to 'git')
nanogit cat-file https://github.com/user/private-repo.git main file.txt --token YOUR_TOKEN

# Using environment variables
NANOGIT_TOKEN=YOUR_TOKEN nanogit cat-file https://github.com/user/private-repo.git main file.txt
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
- `--username` - Authentication username (defaults to 'git')
- `--token` - Authentication token

**Examples**:

Clone entire repository:
```bash
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo
```

Clone only specific directories:
```bash
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --include 'src/**' --include 'docs/**'
```

Clone excluding certain paths:
```bash
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --exclude 'node_modules/**' --exclude '*.tmp'
```

Clone with both include and exclude patterns:
```bash
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --include 'src/**' --exclude '*.test.go'
```

Clone from a specific tag:
```bash
nanogit clone https://github.com/grafana/nanogit.git v1.0.0 ./my-repo
```

Adjust performance settings (defaults are batch-size=50, concurrency=10):
```bash
# Increase for better performance with large repositories
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --batch-size 100 --concurrency 20

# Sequential mode for constrained environments
nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --batch-size 1 --concurrency 1
```

With authentication:
```bash
# Using token (username defaults to 'git')
nanogit clone https://github.com/user/private-repo.git main ./my-repo --token YOUR_TOKEN

# Using environment variables
NANOGIT_TOKEN=YOUR_TOKEN nanogit clone https://github.com/user/private-repo.git main ./my-repo
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
