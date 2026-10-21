# Writing with the StagedWriter

All writes in nanogit go through the `StagedWriter`: a transactional interface that stages any number of changes in memory (or on disk), turns them into commits, and pushes everything to the server as a single atomic ref update. Nothing touches the remote repository until `Push`.

The full lifecycle:

```go
// 1. Resolve the branch you want to write to.
ref, err := client.GetRef(ctx, "refs/heads/main")
if err != nil {
    return err
}

// 2. Create a writer on top of that ref.
writer, err := client.NewStagedWriter(ctx, ref)
if err != nil {
    return err
}

// 3. Stage changes. Every staged operation returns the resulting object hash.
if _, err := writer.CreateBlob(ctx, "configs/app.yaml", []byte("replicas: 3\n")); err != nil {
    return err
}
if _, err := writer.UpdateBlob(ctx, "README.md", []byte("updated\n")); err != nil {
    return err
}
if _, err := writer.DeleteBlob(ctx, "legacy/old.txt"); err != nil {
    return err
}

// 4. Commit the staged changes.
author := nanogit.Author{Name: "Bot", Email: "bot@example.com", Time: time.Now()}
committer := nanogit.Committer{Name: "Bot", Email: "bot@example.com", Time: time.Now()}
commit, err := writer.Commit(ctx, "Update configuration", author, committer)
if err != nil {
    return err
}

// 5. Push. This is the only step that changes the remote: it uploads the
// packfile and moves the ref to the new commit.
if err := writer.Push(ctx); err != nil {
    return err
}
fmt.Println("pushed", commit.Hash)
```

## Staging operations

| Operation | What it stages |
| --------- | -------------- |
| `CreateBlob(ctx, path, content)` | A new file. Fails if the path already exists. |
| `UpdateBlob(ctx, path, content)` | New content for an existing file. Fails if the path doesn't exist. |
| `DeleteBlob(ctx, path)` | Removal of a file. |
| `MoveBlob(ctx, src, dest)` | A file move (copy to `dest` + delete `src`). |
| `DeleteTree(ctx, path)` | Removal of a directory and everything under it. |
| `MoveTree(ctx, src, dest)` | A recursive directory move. |
| `BlobExists(ctx, path)` / `GetTree(ctx, path)` | Read helpers that see the staged state, useful for deciding between create and update. |

Paths are slash-separated and relative to the repository root. All files are written with mode `0644` — fine-grained permissions are a [non-goal](../index.md#when-should-i-not-use-it).

## Several commits, one push

`Commit` can be called multiple times before a single `Push`; each commit chains onto the previous one, and `Push` uploads the whole chain and points the ref at the last commit:

```go
if _, err := writer.CreateBlob(ctx, "file1.txt", []byte("one")); err != nil {
    return err
}
if _, err := writer.Commit(ctx, "Add file1", author, committer); err != nil {
    return err
}

if _, err := writer.CreateBlob(ctx, "file2.txt", []byte("two")); err != nil {
    return err
}
if _, err := writer.Commit(ctx, "Add file2", author, committer); err != nil {
    return err
}

// One ref update covering both commits.
if err := writer.Push(ctx); err != nil {
    return err
}
```

## Errors, retries, and cleanup

- Committing with nothing staged returns `nanogit.ErrNothingToCommit`; pushing with nothing committed returns `nanogit.ErrNothingToPush`.
- **A failed `Push` leaves the writer intact**: the staged objects and commits are retained, so you can call `Push(ctx)` again (for example after a transient network failure — or wire up the [retry mechanism](../architecture/retry.md) to do it for you).
- `Cleanup(ctx)` discards all staged state and releases resources (including any temp files from disk-backed storage). Call it when abandoning a writer; after cleanup the writer returns `nanogit.ErrWriterCleanedUp` for further operations.
- A successful `Push` resets the writer onto the new commit, so a long-lived writer can keep staging follow-up changes.

## Writing modes: memory, disk, auto

Staged objects are buffered according to a writer option:

```go
// Auto (default): small change sets stay in memory, large ones spill to disk.
writer, err := client.NewStagedWriter(ctx, ref)

// Memory: fastest, unbounded memory use — bulk imports beware.
writer, err = client.NewStagedWriter(ctx, ref, nanogit.WithMemoryStorage())

// Disk: minimal memory for bulk operations, at the cost of temp-file I/O.
writer, err = client.NewStagedWriter(ctx, ref, nanogit.WithDiskStorage())
```

See [Storage Backend](../architecture/storage.md) for how the modes work internally.

## Related

- [Commit signing](commit-signing.md) — sign the commits a writer creates
- [Error handling](error-handling.md) — the errors write operations return
- [Quick Start](../getting-started/quick-start.md) — the short version of this page
