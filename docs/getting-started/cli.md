# CLI Tool

The nanogit CLI provides command-line access to nanogit's core Git operations. It's designed for debugging, scripting, and manual repository operations.

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

### ls-remote - List Remote References

List all references (branches and tags) in a remote repository.

```bash
# List all references
nanogit ls-remote https://github.com/grafana/nanogit

# List only branches
nanogit ls-remote https://github.com/grafana/nanogit --heads

# List only tags
nanogit ls-remote https://github.com/grafana/nanogit --tags

# JSON output
nanogit ls-remote https://github.com/grafana/nanogit --json
```

**Options:**
- `--heads` - Show only branches (refs/heads/*)
- `--tags` - Show only tags (refs/tags/*)
- `--json` - Output in JSON format

### ls-tree - List Tree Contents

List the contents of a tree object at a specific reference or commit.

```bash
# List root directory (short branch name)
nanogit ls-tree https://github.com/grafana/nanogit main

# List with commit hash
nanogit ls-tree https://github.com/grafana/nanogit abc123def456...

# List specific directory
nanogit ls-tree https://github.com/grafana/nanogit main src/

# Recursive listing
nanogit ls-tree https://github.com/grafana/nanogit main --recursive

# Show detailed output (mode, type, hash)
nanogit ls-tree https://github.com/grafana/nanogit main --long

# JSON output
nanogit ls-tree https://github.com/grafana/nanogit main --json
```

**Options:**
- `-r, --recursive` - List all files recursively
- `-l, --long` - Show detailed information (mode, type, hash)
- `--json` - Output in JSON format

**Reference formats accepted:**
- Short branch name: `main` → tries `refs/heads/main`
- Short tag name: `v1.0.0` → tries `refs/tags/v1.0.0`
- Full reference: `refs/heads/main`
- Commit hash: `66130f71cf74d30fa73de8d4abe6fa587bce97b6`

### cat-file - Show File Contents

Output the contents of a file from a repository.

```bash
# Output file content (short branch name)
nanogit cat-file https://github.com/grafana/nanogit main README.md

# Output with commit hash
nanogit cat-file https://github.com/grafana/nanogit abc123... README.md

# Show file type and size
nanogit cat-file https://github.com/grafana/nanogit main README.md --show-type --show-size

# JSON output
nanogit cat-file https://github.com/grafana/nanogit main README.md --json
```

**Options:**
- `--show-type` - Display the object type (blob)
- `--show-size` - Display the object size in bytes
- `--json` - Output in JSON format

### clone - Clone Repository

Clone a repository to the local filesystem with optional path filtering.

```bash
# Basic clone (defaults to main branch)
nanogit clone https://github.com/grafana/nanogit /tmp/repo

# Clone specific branch (short name)
nanogit clone https://github.com/grafana/nanogit /tmp/repo --ref develop

# Clone at specific commit
nanogit clone https://github.com/grafana/nanogit /tmp/repo --ref abc123...

# Clone with path filtering
nanogit clone https://github.com/grafana/nanogit /tmp/repo \
  --include-paths "*.go,*.md" --exclude-paths "**/*_test.go"

# Performance tuning for large repos
nanogit clone https://github.com/grafana/nanogit /tmp/repo \
  --batch-size 50 --concurrency 8

# JSON output
nanogit clone https://github.com/grafana/nanogit /tmp/repo --json
```

**Options:**
- `--ref <name>` - Branch, tag, or commit to clone (default: `main`)
- `--include-paths <patterns>` - Comma-separated glob patterns to include
- `--exclude-paths <patterns>` - Comma-separated glob patterns to exclude
- `--batch-size <n>` - Blob fetch batch size (0=sequential)
- `--concurrency <n>` - Parallel fetch workers (0=sequential)
- `--json` - Output in JSON format

**Path filtering examples:**
```bash
# Clone only documentation
nanogit clone <url> /tmp/repo --include-paths "docs/**,*.md"

# Clone Go code, exclude tests
nanogit clone <url> /tmp/repo --include-paths "**/*.go" --exclude-paths "**/*_test.go"

# Clone specific directories
nanogit clone <url> /tmp/repo --include-paths "src/**,internal/**"
```

## Authentication

### Environment Variables

Authentication tokens can be provided via environment variables:

```bash
export NANOGIT_TOKEN="your-token"
# or
export GITHUB_TOKEN="your-github-token"
# or
export GITLAB_TOKEN="your-gitlab-token"

nanogit clone https://github.com/user/private-repo /tmp/repo
```

**Precedence:** `NANOGIT_TOKEN` > `GITHUB_TOKEN` > `GITLAB_TOKEN`

### Command-Line Flags

Authentication can also be provided via command-line flags:

```bash
# Token authentication
nanogit clone https://github.com/user/private-repo /tmp/repo --token "your-token"

# Basic authentication
nanogit clone https://github.com/user/repo /tmp/repo \
  --username "user" --password "pass"
```

**Flag precedence:** Command-line flags override environment variables.

## Output Formats

### Human-Readable (Default)

The default output format uses colors and emojis for better readability:

```bash
$ nanogit ls-remote https://github.com/grafana/nanogit --heads
de1efa70...  refs/heads/bugfix/clone-issues
66130f71...  refs/heads/main
```

Features:
- Colored output using ANSI colors
- Status indicators with emojis (✓, ✗, ⚠)
- Human-readable sizes and counts
- Formatted tables for structured data

### JSON

Use the `--json` flag for machine-parsable output:

```bash
$ nanogit ls-remote https://github.com/grafana/nanogit --heads --json
{
  "refs": [
    {
      "name": "refs/heads/bugfix/clone-issues",
      "hash": "de1efa702957c32f308c786107b19c373e0a4502"
    },
    {
      "name": "refs/heads/main",
      "hash": "66130f71cf74d30fa73de8d4abe6fa587bce97b6"
    }
  ]
}
```

Features:
- Valid JSON output to stdout
- Snake_case field names
- Errors included in JSON format
- Suitable for parsing with `jq`, `python`, etc.

## Global Flags

These flags are available on all commands:

- `--token <token>` - Authentication token
- `--username <user>` - Username for basic auth
- `--password <pass>` - Password for basic auth
- `--json` - Output in JSON format
- `--debug` - Enable debug logging
- `-h, --help` - Show help

## Examples

### Debugging Repository Issues

```bash
# Check if a reference exists
nanogit ls-remote https://github.com/user/repo | grep main

# Verify a commit exists
nanogit ls-tree https://github.com/user/repo abc123def456...

# Inspect file at specific commit
nanogit cat-file https://github.com/user/repo abc123... package.json
```

### Scripting

```bash
#!/bin/bash
# Clone and extract specific files

REPO_URL="https://github.com/grafana/nanogit"
COMMIT="66130f71cf74d30fa73de8d4abe6fa587bce97b6"
OUTPUT_DIR="/tmp/nanogit-docs"

# Clone only documentation
nanogit clone "$REPO_URL" "$OUTPUT_DIR" \
  --ref "$COMMIT" \
  --include-paths "docs/**,*.md" \
  --json | jq '.filtered_files'
```

### JSON Processing

```bash
# Get all branch names
nanogit ls-remote https://github.com/grafana/nanogit --heads --json | \
  jq -r '.refs[].name'

# Count files in a directory
nanogit ls-tree https://github.com/grafana/nanogit main --recursive --json | \
  jq '.entries | length'

# Extract clone statistics
nanogit clone https://github.com/grafana/nanogit /tmp/repo --json | \
  jq '{total: .total_files, filtered: .filtered_files, commit: .commit}'
```

## Architecture

The CLI is implemented as a separate Go module (`cli/`) to avoid adding dependencies (cobra, fatih/color) to the main nanogit library. Key design decisions:

- **Separate Module**: Uses its own `go.mod` with replace directive
- **Workspace Integration**: Part of the main go.work workspace
- **No Dependency Pollution**: Main library remains dependency-free
- **Consistent Interface**: Uses the same nanogit.Client interface

## Troubleshooting

### Authentication Errors

```bash
Error: authentication required
```

**Solution:** Provide authentication via environment variable or flag:
```bash
export GITHUB_TOKEN="your-token"
# or
nanogit <command> --token "your-token"
```

### Reference Not Found

```bash
Error: resolving ref main: reference not found: main
```

**Solution:** Try with full reference path:
```bash
nanogit ls-tree <url> refs/heads/main
```

Or check available references:
```bash
nanogit ls-remote <url> --heads
```

### Connection Errors

```bash
Error: Get "https://...": dial tcp: lookup ...: no such host
```

**Solution:** Verify the repository URL is correct and accessible.

## Contributing

The CLI code is located in the `cli/` directory. To contribute:

1. Make changes in `cli/`
2. Run tests: `cd cli && go test -v ./...`
3. Run linter: `cd cli && golangci-lint run`
4. Format code: `make cli-fmt`
5. Build: `make cli-build`

See [CONTRIBUTING.md](https://github.com/grafana/nanogit/blob/main/CONTRIBUTING.md) for more details.
