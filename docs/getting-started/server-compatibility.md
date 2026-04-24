# Server Compatibility Check

nanogit requires **Git Smart HTTP Protocol v2**. Before integrating it into your service, confirm that your Git server supports the operations you need. This page walks through a short CLI-based round trip that exercises the protocol handshake, a read, a write, and a read-after-write verification.

If any step fails, nanogit is not the right fit for that server and you'll want to fall back to the standard `git` CLI or a provider that supports protocol v2.

## Prerequisites

- A scratch repository you can write to (GitHub, GitLab, Bitbucket, or a self-hosted server)
- A personal access token with push access (for private repos or write operations)
- The `nanogit` CLI — grab a [pre-built binary from the latest release](https://github.com/grafana/nanogit/releases/latest), `go install github.com/grafana/nanogit/cli/cmd/nanogit@latest`, or build from source:

```bash
go build -o nanogit ./cli/cmd/nanogit
```

See [CLI Installation](cli.md#installation) for platform-specific binary downloads.

## Configure credentials

Export credentials once so the commands below stay readable. `NANOGIT_USERNAME` defaults to `git`, which is what most providers expect when authenticating with a token.

```bash
export NANOGIT_USERNAME=git                       # default; override for providers that require a specific user
export NANOGIT_TOKEN=ghp_xxx                      # required for private repos or write operations
export NANOGIT_AUTHOR_NAME="You"
export NANOGIT_AUTHOR_EMAIL="you@example.com"

REPO=https://github.com/<you>/<scratch>.git
```

## 1. Verify protocol v2

```bash
./nanogit check "$REPO"
```

Expected output on a compatible server:

```
Checking compatibility for: https://github.com/<you>/<scratch>.git

✅ Compatible - Server supports Git protocol v2

This server is compatible with nanogit. You can use:
  • nanogit ls-remote
  • nanogit ls-tree
  • nanogit cat-file
  • nanogit clone
```

If you see `❌ Not Compatible`, stop here — the server only speaks protocol v1 and the remaining commands will not work. For JSON output suitable for scripting, pass `--json`.

## 2. List files (read path)

```bash
./nanogit ls-tree "$REPO" main
```

This confirms authentication, branch resolution, and tree fetching all work against the server.

## 3. Write a file (write path)

```bash
echo "hi" | ./nanogit put-file "$REPO" main note.md -m "add note"
```

`put-file` stages the blob, creates a commit, and pushes in a single step. It prints the new commit hash to stdout. If the push fails, the server has rejected the write path — check branch protection rules, token scopes, and server-side hooks.

## 4. Verify (read-after-write)

```bash
./nanogit cat-file "$REPO" main note.md
```

You should see `hi`. This closes the loop: if all four commands succeed, nanogit can read from and write to this server.

## Troubleshooting

Add `-v` for progress on stderr, or `NANOGIT_TRACE=1` for full Git wire-level detail. Both leave stdout clean so commit hashes and file contents stay pipeable.

```bash
./nanogit -v check "$REPO"
NANOGIT_TRACE=1 ./nanogit ls-tree "$REPO" main
```

See [Verbose mode](cli.md#verbose-mode) for details on the logging levels.

## Next steps

- [Quick Start](quick-start.md) — use nanogit as a Go library
- [CLI Reference](cli.md) — full command documentation
