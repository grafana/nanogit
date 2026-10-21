# Error handling

nanogit returns wrapped, typed errors designed for `errors.Is` and `errors.As`. Every error keeps its full context chain, so you can branch on the category while still logging the complete story.

## Sentinel errors (`errors.Is`)

The root package exports sentinels for the common categories:

| Sentinel | Meaning |
| -------- | ------- |
| `ErrObjectNotFound` | An object (blob, tree, commit, **or ref/path** — see below) doesn't exist |
| `ErrObjectAlreadyExists` | Creating something that already exists (e.g. `CreateBlob` on an existing path, `CreateRef` on an existing ref) |
| `ErrUnauthorized` | Missing or invalid credentials (HTTP 401) |
| `ErrPermissionDenied` | Valid credentials, insufficient rights (HTTP 403) |
| `ErrRepositoryNotFound` | The repository itself doesn't exist (HTTP 404) |
| `ErrServerUnavailable` | The server failed or is unreachable (5xx, network) |
| `ErrNothingToCommit` / `ErrNothingToPush` | Writer misuse: nothing staged / nothing committed |
| `ErrWriterCleanedUp` | Using a `StagedWriter` after `Cleanup` |
| `ErrUnexpectedObjectType` / `ErrUnexpectedObjectCount` | Protocol-level surprises in the server's response |
| `ErrEmptyPath` / `ErrEmptyRefName` / `ErrEmptyCommitMessage` / `ErrInvalidAuthor` | Input validation |

```go
blob, err := client.GetBlobByPath(ctx, commit.Tree, "config/app.yaml")
switch {
case errors.Is(err, nanogit.ErrObjectNotFound):
    // File isn't there — create it instead.
case errors.Is(err, nanogit.ErrUnauthorized), errors.Is(err, nanogit.ErrPermissionDenied):
    // Credentials problem — surface to the tenant.
case errors.Is(err, nanogit.ErrServerUnavailable):
    // Transient — retry or back off.
case err != nil:
    return fmt.Errorf("read config: %w", err)
}
```

Not-found errors for refs and paths **unwrap to `ErrObjectNotFound`**, so a single `errors.Is(err, nanogit.ErrObjectNotFound)` covers missing files, trees, commits, and refs.

## Typed errors (`errors.As`)

When you need the detail — which path, which ref, which hash — match the typed errors:

| Type | Context fields | Returned by |
| ---- | -------------- | ----------- |
| `*PathNotFoundError` | `Path` | `GetBlobByPath`, `GetTreeByPath`, writer path operations |
| `*RefNotFoundError` | `RefName` | `GetRef`, `UpdateRef`, `DeleteRef`, `NewStagedWriter` |
| `*RefAlreadyExistsError` | `RefName` | `CreateRef` |
| `*ObjectNotFoundError` | `ObjectID` | object fetches by hash |
| `*ObjectAlreadyExistsError` | `ObjectID` | staging duplicates |
| `*AuthorError` | `Field`, `Reason` | `Commit` with invalid author/committer |

```go
_, err := client.GetRef(ctx, "refs/heads/feature-x")
var refErr *nanogit.RefNotFoundError
if errors.As(err, &refErr) {
    fmt.Printf("ref %q does not exist; creating it\n", refErr.RefName)
}
```

## Response-limit errors

If you cap response sizes with `options.WithLimits`, operations that exceed a cap fail with `*client.ErrResponseTooLarge` (from `protocol/client`), carrying the exceeded `Limit` and the operation class in `Op`. See [Response limits](response-limits.md).

## Practical guidance

- **Wrap, don't replace**: `fmt.Errorf("sync dashboards for %s: %w", tenant, err)` keeps `errors.Is`/`errors.As` working up the stack.
- **Retryability**: `ErrServerUnavailable` and timeouts are the transient category — the [retry mechanism](../architecture/retry.md) already handles them for GET/DELETE and 429s if you enable it; treat 4xx categories as permanent.
- **Writer flows**: after a failed `Push`, the writer keeps its staged state — you can retry `Push` directly (see [Writing](writing.md#errors-retries-and-cleanup)).
