# Response limits

When a service talks to Git servers it doesn't control — the multitenant situation nanogit was built for — an oversized (or malicious) response can exhaust memory or disk. `options.WithLimits` caps how many bytes nanogit will read from the server per HTTP response, classified by operation, so a single misbehaving repository can't take out the process.

By default there are **no limits** (a zero `Limits` preserves historic behavior).

```go
client, err := nanogit.NewHTTPClient(repoURL,
    options.WithBasicAuth("git", token),
    options.WithLimits(options.Limits{
        SingleObjectFetchMaxBytes:   64 << 20,   // 64 MiB: GetBlob, GetTree, GetCommit
        MultiObjectFetchMaxBytes:    1 << 30,    // 1 GiB: GetFlatTree, ListCommits, CompareCommits, Clone
        RefsMetadataMaxBytes:        8 << 20,    // 8 MiB: ListRefs, GetRef, protocol detection
        ReceivePackResponseMaxBytes: 1 << 20,    // 1 MiB: server replies to pushes
    }),
)
```

## The four classes

| Field | Covers | Sizing intuition |
| ----- | ------ | ---------------- |
| `SingleObjectFetchMaxBytes` | fetches that target one object (`GetBlob`, `GetTree`, `GetCommit`) | a bit above your largest expected file |
| `MultiObjectFetchMaxBytes` | fetches that may return many objects (`GetFlatTree`, `ListCommits`, `CompareCommits`, `Clone`) | scales with repository size — orders of magnitude above the single-object cap |
| `RefsMetadataMaxBytes` | ref listings and protocol detection | small; grows with ref count (a 1 MiB floor always applies to the protocol-detection path) |
| `ReceivePackResponseMaxBytes` | the server's reply to a push | small; it's a status report, not content |

A zero value for any field means "no limit" for that class. Negative values are rejected when the option is applied.

## When a cap is hit

The operation fails with a `*client.ErrResponseTooLarge` (from `github.com/grafana/nanogit/protocol/client`) that records which `Limit` was exceeded and the operation class in `Op`:

```go
_, err := client.GetFlatTree(ctx, commitHash)
var tooLarge *protoclient.ErrResponseTooLarge
if errors.As(err, &tooLarge) {
    log.Printf("tree listing exceeded %d bytes (%s) — raise MultiObjectFetchMaxBytes or filter paths",
        tooLarge.Limit, tooLarge.Op)
}
```

Hitting a cap mid-stream is not retried by the [retry mechanism](../architecture/retry.md) — it is a deterministic outcome, not a transient failure. Raise the cap, or narrow the operation (path-filtered [clone](../getting-started/quick-start.md#cloning-a-repository), `ListCommitsOptions.PerPage`, subtree reads).

## On the CLI

The same four caps are exposed as global flags: `--max-bytes-single-object`, `--max-bytes-multi-object`, `--max-bytes-refs`, and `--max-bytes-receive-pack` — see the [CLI docs](../getting-started/cli.md#global-flags).
