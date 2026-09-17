# Response limits

When a service talks to Git servers it doesn't control — the multitenant situation nanogit was built for — an oversized (or malicious) response can exhaust memory or disk. [`options.WithLimits`](https://pkg.go.dev/github.com/grafana/nanogit/options#WithLimits) caps how many bytes nanogit will read from the server per HTTP response, classified by operation, so a single misbehaving repository can't take out the process.

By default the four **wire** caps are off (a zero `Limits` preserves historic behavior). The one exception is the decoded-object cap (`MaxObjectDecodedBytes`), which always enforces a built-in default — see [Decompression bombs](#decompression-bombs) below.

```go
client, err := nanogit.NewHTTPClient(repoURL,
    options.WithBasicAuth("git", token),
    options.WithLimits(options.Limits{
        SingleObjectFetchMaxBytes:   64 << 20,   // 64 MiB: GetBlob, GetTree, GetCommit
        MultiObjectFetchMaxBytes:    1 << 30,    // 1 GiB: GetFlatTree, ListCommits, CompareCommits, Clone
        RefsMetadataMaxBytes:        8 << 20,    // 8 MiB: ListRefs, GetRef, protocol detection
        ReceivePackResponseMaxBytes: 1 << 20,    // 1 MiB: server replies to pushes
        MaxObjectDecodedBytes:       64 << 20,   // 64 MiB: inflated size of any single object
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

A zero value for any of these four fields means "no limit" for that class. Negative values are rejected when the option is applied.

## Decompression bombs

The four wire caps bound the **compressed** bytes read off the network. They do not, on their own, stop a *decompression bomb*: a server can return a small compressed object — comfortably under a wire cap — that inflates to gigabytes and exhausts memory as it is decoded.

`MaxObjectDecodedBytes` closes that gap. It caps the **decoded (inflated)** size of any single object read from a packfile. An object's declared decoded size is checked *before* it is allocated, so an oversized object is rejected up front rather than after it has inflated in memory. The same ceiling is applied to objects reconstructed from deltas.

Unlike the four wire caps, this field is **never fully disabled**: a zero value leaves nanogit's built-in default (`protocol.MaxUnpackedObjectSize`, 10 MiB) in force. Set a positive value to raise or lower that ceiling to match your largest legitimate object.

An object over the cap fails with a `*protocol.ObjectTooLargeError` (which wraps the `protocol.ErrObjectTooLarge` sentinel), recording the declared `Size` and the `Limit`:

```go
_, err := client.GetBlob(ctx, blobHash)
var tooLarge *protocol.ObjectTooLargeError
if errors.As(err, &tooLarge) {
    log.Printf("object inflates to %d bytes, over the %d-byte cap — raise MaxObjectDecodedBytes",
        tooLarge.Size, tooLarge.Limit)
}
// errors.Is(err, protocol.ErrObjectTooLarge) also matches.
```

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

The same caps are exposed as global flags: `--max-bytes-single-object`, `--max-bytes-multi-object`, `--max-bytes-refs`, `--max-bytes-receive-pack`, and `--max-object-decoded-bytes` — see the [CLI docs](../getting-started/cli.md#global-flags).
