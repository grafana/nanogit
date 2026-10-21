# History and diffs

nanogit reads commit history and computes file-level diffs without cloning: [`ListCommits`](https://pkg.go.dev/github.com/grafana/nanogit#Client.ListCommits) walks history backwards from a commit, and [`CompareCommits`](https://pkg.go.dev/github.com/grafana/nanogit#Client.CompareCommits) diffs two commits' trees.

## Listing commits

`ListCommits` starts at a commit (usually a branch tip) and walks parent links, with GitHub-style pagination and filtering via `ListCommitsOptions`:

```go
ref, err := client.GetRef(ctx, "refs/heads/main")
if err != nil {
    return err
}

commits, err := client.ListCommits(ctx, ref.Hash, nanogit.ListCommitsOptions{
    PerPage: 50, // default 30, max 100
    Page:    1,  // 1-based
})
if err != nil {
    return err
}
for _, c := range commits {
    fmt.Printf("%s %s <%s> %s\n", c.Hash, c.Author.Name, c.Author.Email, c.Message)
}
```

Filter to a path and a time window — e.g. "who touched the dashboards directory this month":

```go
commits, err := client.ListCommits(ctx, ref.Hash, nanogit.ListCommitsOptions{
    Path:  "dashboards/",                        // only commits affecting this path
    Since: time.Now().AddDate(0, -1, 0),         // created after
    Until: time.Now(),                           // created before
})
```

## Comparing commits

`CompareCommits` returns the file-level changes between a base and a head commit, sorted by path:

```go
base, err := client.GetRef(ctx, "refs/tags/v1.0.0")
if err != nil {
    return err
}
head, err := client.GetRef(ctx, "refs/heads/main")
if err != nil {
    return err
}

changes, err := client.CompareCommits(ctx, base.Hash, head.Hash)
if err != nil {
    return err
}
for _, change := range changes {
    fmt.Printf("%s %s\n", change.Status, change.Path)
}
```

Each `CommitFile` carries the path, mode, hash, and object type on **both** sides (`Path`/`OldPath`, `Mode`/`OldMode`, `Hash`/`OldHash`, `Type`/`OldType`) plus a `Status` from `protocol`:

| Status | Meaning |
| ------ | ------- |
| `FileStatusAdded` (`A`) | present in head only |
| `FileStatusModified` (`M`) | content or mode differs |
| `FileStatusDeleted` (`D`) | present in base only |
| `FileStatusTypeChanged` (`T`) | blob ↔ tree change at the same path |
| `FileStatusRenamed` (`R`) | delete/add pair with identical content (opt-in, below) |

### Rename detection

Off by default; enable it per call:

```go
changes, err := client.CompareCommits(ctx, base.Hash, head.Hash, nanogit.WithRenameDetection())
if err != nil {
    return err
}
for _, change := range changes {
    if change.Status == protocol.FileStatusRenamed {
        fmt.Printf("renamed: %s -> %s\n", change.OldPath, change.Path)
    }
}
```

Detection matches deleted and added files with identical content hashes — exact-content renames, not similarity-based heuristics like `git diff -M`.

## Reading a single commit

`GetCommit` fetches one commit's metadata (author, committer, message, parent, root tree). The `Tree` hash is the usual entry point for reads — pass it to `GetBlobByPath` or `GetFlatTree`:

```go
commit, err := client.GetCommit(ctx, ref.Hash)
if err != nil {
    return err
}
fmt.Printf("%s by %s at %s\n", commit.Hash, commit.Author.Name, commit.Time())
```

## Cost model

Both operations fetch commit and tree objects on demand over HTTPS. On large repositories:

- Prefer `Path`-filtered `ListCommits` over walking everything client-side
- Diffs fetch both commits' trees (concurrently); repeated comparisons benefit from a shared [object cache](../architecture/storage.md) via `storage.ToContext`
- Cap worst-case response sizes with [response limits](response-limits.md) (`MultiObjectFetchMaxBytes` covers both operations)
