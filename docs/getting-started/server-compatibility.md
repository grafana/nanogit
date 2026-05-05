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

Export credentials once so the commands below stay readable. `NANOGIT_USERNAME` defaults to `git`, which is what most providers expect when authenticating with a token. `NANOGIT_REPO` lets every command below omit the repository URL argument.

```bash
export NANOGIT_USERNAME=git                       # default; override for providers that require a specific user
export NANOGIT_TOKEN=ghp_xxx                      # required for private repos or write operations
export NANOGIT_AUTHOR_NAME="You"
export NANOGIT_AUTHOR_EMAIL="you@example.com"
export NANOGIT_REPO=https://github.com/<you>/<scratch>.git
```

## 1. Verify protocol v2

```bash
./nanogit check
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
./nanogit ls-tree main
```

This confirms authentication, branch resolution, and tree fetching all work against the server.

## 3. Write a file (write path)

```bash
echo "hi" | ./nanogit put-file main note.md -m "add note"
```

`put-file` stages the blob, creates a commit, and pushes in a single step. It prints the new commit hash to stdout. If the push fails, the server has rejected the write path — check branch protection rules, token scopes, and server-side hooks.

## 4. Verify (read-after-write)

```bash
./nanogit cat-file main note.md
```

You should see `hi`. This closes the loop: if all four commands succeed, nanogit can read from and write to this server.

## Receive-pack capabilities

When step 3 succeeds but the pushed commit does not appear on the branch (or the branch ends up empty), the server is almost certainly negotiating a capability set that nanogit cannot parse correctly. This most commonly happens when a server wraps `report-status` inside side-band channel 1 — a configuration seen on some GitLab deployments — which hides the actual push outcome from nanogit's parser.

### Defaults advertised by nanogit

Unless overridden, nanogit advertises the following capabilities on every receive-pack push:

| Capability          | Purpose                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| `report-status-v2`  | Ask the server for a structured report describing the push outcome.    |
| `side-band-64k`     | Allow the server to multiplex data/progress/error on side-band channels. |
| `quiet`             | Suppress non-error progress output.                                     |
| `object-format=sha1`| Declare SHA-1 as the object hash algorithm.                             |
| `agent=nanogit`     | Identify the client for server-side logging.                            |

The authoritative list lives in `protocol.DefaultReceivePackCapabilities()`.

### Overriding the set

Both the library and the CLI let you replace the advertised set when the default negotiation breaks against a particular server. There is no merge: whatever you pass becomes the complete advertisement.

From the CLI, pass `--receive-pack-capability` once per token (the flag is repeatable):

```bash
echo "hi" | ./nanogit put-file main note.md -m "add note" \
  --receive-pack-capability=report-status-v2 \
  --receive-pack-capability=quiet \
  --receive-pack-capability=object-format=sha1 \
  --receive-pack-capability=agent=nanogit
```

That example drops `side-band-64k` while keeping the rest of the default set — the recommended workaround for the GitLab side-band-wrapping case above.

From the library, use `options.WithReceivePackCapabilities`:

```go
import (
    "github.com/grafana/nanogit"
    "github.com/grafana/nanogit/options"
    "github.com/grafana/nanogit/protocol"
)

caps := []protocol.Capability{
    protocol.CapReportStatusV2,
    protocol.CapQuiet,
    protocol.CapObjectFormatSHA1,
    protocol.CapAgent("nanogit"),
}
client, err := nanogit.NewHTTPClient(repoURL,
    options.WithBasicAuth("git", token),
    options.WithReceivePackCapabilities(caps...),
)
```

Typed helpers (`CapReportStatusV2`, `CapSideBand64k`, `CapQuiet`, `CapObjectFormatSHA1`, `CapAgent(name)`) are exposed under `protocol`. Any other string literal can be passed through as `protocol.Capability("foo")` when you need to negotiate something the helpers don't cover.

### Programmatic negotiation

Knowing the right subset to advertise requires knowing what the server supports. `options.WithCapabilityNegotiation()` flips that around: nanogit fetches `GET info/refs?service=git-receive-pack` once per client lifetime, parses the v1-style capability advertisement, and intersects it with the desired set on every subsequent ref update. The fetch result is cached behind `sync.Once`, so writer resets after `Push` and `Cleanup` reuse the same negotiated set without extra round-trips.

```go
client, err := nanogit.NewHTTPClient(repoURL,
    options.WithBasicAuth("git", token),
    options.WithCapabilityNegotiation(),
)
```

The option composes with `WithReceivePackCapabilities`: the explicit set becomes the desired list, and negotiation filters it down to what the server actually advertises. This is the right pattern when you want the safety of a hand-curated list *and* the defensiveness of dropping anything the server omits.

`report-status-v2` and `agent=` are always retained on the client side regardless of the server's advertisement. Dropping `report-status-v2` would re-introduce silent push success on rejection (nanogit's response parser depends on it), and `agent=` is a pure client self-identifier with no server-side feature semantics.

The CLI exposes the same option via `--negotiate-capabilities`:

```bash
NANOGIT_TRACE=1 ./nanogit -v put-file main note.md -m "add note" \
  --negotiate-capabilities < note.md
```

The trace output will show one extra `GET info/refs?service=git-receive-pack` before the push and a receive-pack POST that advertises only the intersected set.

Negotiation is opt-in. When it fails (network error, 4xx/5xx, malformed advertisement) the push aborts; silent fallback to the static set would hide server misconfiguration and contradict the explicit opt-in. This option is "defensive correctness for strict servers and cleaner traces," not a fix for protocol bugs — for example, it does not bypass servers that wrap `report-status` in side-band channel 1, because they *do* advertise `side-band-64k`. For that case, drop `side-band-64k` explicitly with `WithReceivePackCapabilities`.

### When to override

- Pushes appear to succeed but the branch is unchanged or empty — retry with `side-band-64k` removed.
- A server log shows it expects a specific capability (for example a hosted provider documenting a required `agent=` prefix or a non-default object format) — build the minimal set it accepts.
- You are reproducing a bug against a specific server — override explicitly so the capability set is deterministic across runs.

Do not override preemptively: the defaults are tuned for the common case and removing `side-band-64k` against a compliant server loses useful progress reporting.

## Troubleshooting

Add `-v` for progress on stderr, or `NANOGIT_TRACE=1` for full Git wire-level detail. Both leave stdout clean so commit hashes and file contents stay pipeable.

```bash
./nanogit -v check
NANOGIT_TRACE=1 ./nanogit ls-tree main
```

Each command still accepts the repository URL as its first positional argument, so you can override `NANOGIT_REPO` inline — for example `./nanogit check https://other.example.com/repo.git`.

See [Verbose mode](cli.md#verbose-mode) for details on the logging levels.

## Next steps

- [Quick Start](quick-start.md) — use nanogit as a Go library
- [CLI Reference](cli.md) — full command documentation
