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

## Future Commands

Future versions will include:

- Clone repositories with path filtering
