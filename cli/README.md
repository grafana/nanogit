# nanogit CLI

A command-line tool for interacting with Git repositories using nanogit.

## Installation

### From Source

```bash
git clone https://github.com/grafana/nanogit
cd nanogit
make cli-build
# Binary will be at bin/nanogit
```

### Using Go Install

```bash
go install github.com/grafana/nanogit/cli@latest
```

## Commands

### ls-remote - List remote references

```bash
# List all references
nanogit ls-remote https://github.com/grafana/nanogit

# List only branches
nanogit ls-remote https://github.com/grafana/nanogit --heads

# List only tags
nanogit ls-remote https://github.com/grafana/nanogit --tags
```

### ls-tree - List tree contents

```bash
# List root directory (short branch name)
nanogit ls-tree https://github.com/grafana/nanogit main

# List with commit hash
nanogit ls-tree https://github.com/grafana/nanogit abc123def456...

# Recursive listing with details
nanogit ls-tree https://github.com/grafana/nanogit main --recursive --long
```

**Note:** Commands accept branch names ("main"), tag names ("v1.0.0"), full refs ("refs/heads/main"), or commit hashes (40 hex characters).

### cat-file - Show file contents

```bash
# Output file content (short branch name)
nanogit cat-file https://github.com/grafana/nanogit main README.md

# Output with commit hash
nanogit cat-file https://github.com/grafana/nanogit abc123def456... README.md

# Show with type and size
nanogit cat-file https://github.com/grafana/nanogit main README.md --show-type --show-size
```

### clone - Clone repository

```bash
# Basic clone (defaults to main branch)
nanogit clone https://github.com/grafana/nanogit /tmp/repo

# Clone specific branch (short name)
nanogit clone https://github.com/grafana/nanogit /tmp/repo --ref develop

# Clone at specific commit
nanogit clone https://github.com/grafana/nanogit /tmp/repo --ref abc123def456...

# Clone with path filtering
nanogit clone https://github.com/grafana/nanogit /tmp/repo \
  --include-paths "docs/**" --exclude-paths "**/*.test.go"
```

## Authentication

### Environment Variables

```bash
export NANOGIT_TOKEN="your-token"
# or
export GITHUB_TOKEN="your-github-token"
# or
export GITLAB_TOKEN="your-gitlab-token"
```

### Command-Line Flags

```bash
nanogit clone https://github.com/user/private-repo /tmp/repo --token "your-token"

# Or with basic auth
nanogit clone https://github.com/user/repo /tmp/repo \
  --username "user" --password "pass"
```

## Output Formats

### Human-Readable (Default)

Colored output with emojis for better readability.

### JSON

```bash
nanogit ls-remote https://github.com/grafana/nanogit --json
```

### Debug Mode

```bash
nanogit ls-tree https://github.com/grafana/nanogit main --debug
```

## Development

### Build

```bash
make cli-build
```

### Test

```bash
make cli-test
```

### Format

```bash
make cli-fmt
```

### Lint

```bash
make cli-lint
```

## Architecture

The CLI is a separate Go module to avoid adding dependencies (cobra, fatih/color) to the main nanogit library. It uses:

- **cobra** - CLI framework
- **fatih/color** - Colored output
- **nanogit** - Core Git operations

## License

Apache License 2.0 - See LICENSE file for details.
