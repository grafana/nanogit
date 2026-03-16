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
- `--token` - Authentication token (or use environment variables)

**Examples**:

List all references:
```bash
nanogit ls-remote https://github.com/grafana/nanogit
```

List only branches:
```bash
nanogit ls-remote https://github.com/grafana/nanogit --heads
```

List only tags:
```bash
nanogit ls-remote https://github.com/grafana/nanogit --tags
```

Output as JSON:
```bash
nanogit ls-remote https://github.com/grafana/nanogit --json
```

With authentication (for private repositories):
```bash
# Using flag
nanogit ls-remote https://github.com/user/private-repo --token YOUR_TOKEN

# Using environment variable
NANOGIT_TOKEN=YOUR_TOKEN nanogit ls-remote https://github.com/user/private-repo
```

## Future Commands

Future versions will include:

- Browse repository tree contents
- Read file contents from repositories
- Clone repositories with path filtering
