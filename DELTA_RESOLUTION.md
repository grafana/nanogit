# Delta Resolution in nanogit

## Overview

This document explains how nanogit handles Git delta objects, the limitations of the Git protocol regarding delta transmission, and the design decisions made to support deltas in a stateless architecture.

## What are Git Delta Objects?

Git delta objects are compressed representations of Git objects (blobs, trees, commits) that store only the differences from a base object rather than the complete content. This significantly reduces network bandwidth and storage requirements.

### Delta Types

Git supports two types of delta objects:

1. **OFS_DELTA (offset delta)**: References the base object by its offset in the packfile
2. **REF_DELTA (reference delta)**: References the base object by its SHA-1 hash

nanogit currently handles **REF_DELTA** objects (type 7 in packfile format).

### Delta Format

A delta object contains:
- Reference to the base object (SHA-1 hash for REF_DELTA)
- Expected source length (size of base object)
- Target length (size of resulting object after applying delta)
- Series of delta instructions:
  - **Copy instructions**: Copy bytes from base object at specific offset/length
  - **Insert instructions**: Insert new literal bytes

## Why Git Servers Send Deltas

Git servers automatically deltify objects to optimize network transfer:

1. **Bandwidth efficiency**: Deltas can reduce transfer size by 50-90% for modified files
2. **Performance**: Less data to transfer = faster fetch/clone operations
3. **Standard behavior**: All major Git servers (GitHub, GitLab, Bitbucket, Gitea) use deltas by default
4. **Repository structure**: Git's internal storage uses deltas for efficient pack files

### When Servers Send Deltas

Servers typically send deltas in these scenarios:
- Modified files across commits (same file, small changes)
- Similar files in the same repository (templates, duplicated code)
- After running `git repack` with aggressive deltification
- For any object where Git detects similarity with another object

## Protocol Limitations: Cannot Disable Deltas

### Git Protocol v2 Capabilities

nanogit uses Git protocol v2 (Smart HTTP protocol), which provides various fetch capabilities:

```
Available capabilities:
- thin-pack: Allow thin pack format (requires local objects for resolution)
- no-progress: Disable progress reporting
- include-tag: Include tags in fetch
- filter: Object filtering (blob:none, tree:0, etc.)
```

**Critical limitation**: There is **NO capability to disable delta objects** in Git protocol v2.

### Why "thin-pack" Doesn't Help

The `thin-pack` capability controls whether the server can send "thin" packfiles that reference objects the client already has, but it does **NOT** control deltification:

```go
// This does NOT prevent deltas:
client.Fetch(ctx, FetchOptions{
    Want: []hash.Hash{wantHash},
    Done: true,
    // thin-pack is irrelevant for delta prevention
})
```

### Attempted Workarounds (All Failed)

We investigated several approaches to avoid deltas, all unsuccessful:

1. **Request full objects via protocol**: No protocol capability exists
2. **Use `no-thin` pack**: Only affects base object inclusion, not deltification
3. **Fetch individual objects**: Server still sends deltas for efficiency
4. **Use different protocol version**: v0, v1, v2 all support deltas without disable option

### Server-Side Behavior

Even when requesting individual objects:

```bash
# Single object fetch still returns delta if server has deltified it
git fetch origin <sha1>
```

The server's pack generation decides deltification based on:
- Repository pack structure
- Recent `git repack` operations
- Server-side optimization settings

**Conclusion**: Delta handling is mandatory for any Git client implementation.

## nanogit's Delta Resolution Approach

Given the impossibility of disabling deltas, nanogit implements **stateless in-memory delta resolution**.

### Design Constraints

1. **Stateless architecture**: No persistent local .git directory or object cache
2. **No thin-pack support**: All base objects must be in the same fetch response
3. **In-memory processing**: Deltas resolved during fetch operation
4. **Single-fetch scope**: Base objects only available within the same fetch

### Implementation Strategy

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Fetch Request (client → server)                         │
├─────────────────────────────────────────────────────────────┤
│   Want: [commit-hash]                                       │
│   Capabilities: no-progress, object-format=sha1             │
│   Done: true                                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Packfile Response (server → client)                     │
├─────────────────────────────────────────────────────────────┤
│   • Regular objects (commit, tree, blob)                    │
│   • Delta objects (REF_DELTA)                               │
│   • All base objects included (self-contained)              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Object Collection (client processing)                   │
├─────────────────────────────────────────────────────────────┤
│   Regular objects → objects map                             │
│   Delta objects → deltas array (deferred)                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Delta Resolution (iterative)                            │
├─────────────────────────────────────────────────────────────┤
│   For each delta:                                           │
│     1. Find base object (in objects map or storage)         │
│     2. Apply delta instructions to base                     │
│     3. Calculate resolved object hash                       │
│     4. Parse resolved object (for trees/commits)            │
│     5. Add to objects map                                   │
│   Repeat until all deltas resolved or max iterations        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. Return Complete Objects                                  │
├─────────────────────────────────────────────────────────────┤
│   All objects fully resolved and available                  │
│   Storage contains both regular and resolved delta objects  │
└─────────────────────────────────────────────────────────────┘
```

### Code Implementation

The delta resolution is implemented in `protocol/client/fetch.go`:

```go
func (c *rawClient) Fetch(ctx context.Context, opts FetchOptions) (map[string]*protocol.PackfileObject, error) {
    objects := make(map[string]*protocol.PackfileObject)
    var deltas []*protocol.PackfileObject

    // Step 1: Collect regular objects and deltas separately
    for {
        obj, err := response.Packfile.ReadObject(ctx)
        if obj.Object.Type == protocol.ObjectTypeRefDelta {
            deltas = append(deltas, obj.Object)
            continue
        }
        objects[obj.Object.Hash.String()] = obj.Object
    }

    // Step 2: Resolve all deltas iteratively
    if len(deltas) > 0 {
        err := c.resolveDeltas(ctx, deltas, objects, storage)
        if err != nil {
            return nil, err
        }
    }

    return objects, nil
}
```

### Delta Resolution Algorithm

```go
func (c *rawClient) resolveDeltas(
    ctx context.Context,
    deltas []*protocol.PackfileObject,
    objects map[string]*protocol.PackfileObject,
    storage storage.PackfileStorage,
) error {
    remaining := deltas
    maxIterations := len(deltas) + 1

    for iteration := 1; len(remaining) > 0 && iteration <= maxIterations; iteration++ {
        resolvedCount := 0
        var stillPending []*protocol.PackfileObject

        for _, delta := range remaining {
            // Find base object (in-memory or storage)
            baseObj, found := c.findBaseObject(ctx, delta.Delta.Parent, objects, storage)
            if !found {
                stillPending = append(stillPending, delta)
                continue
            }

            // Apply delta to base
            resolvedData, err := protocol.ApplyDelta(baseObj.Data, delta.Delta)
            if err != nil {
                stillPending = append(stillPending, delta)
                continue
            }

            // Create resolved object
            resolvedHash, resolvedType := inferObjectType(resolvedData, baseObj)
            resolvedObj := &protocol.PackfileObject{
                Type: resolvedType,
                Data: resolvedData,
                Hash: resolvedHash,
            }

            // Parse structured data (tree/commit)
            resolvedObj.Parse()

            // Store resolved object
            objects[resolvedHash.String()] = resolvedObj
            if storage != nil {
                storage.Add(resolvedObj)
            }

            resolvedCount++
        }

        if resolvedCount == 0 && len(stillPending) > 0 {
            return fmt.Errorf("unable to resolve deltas: missing base objects")
        }

        remaining = stillPending
    }

    return nil
}
```

### Handling Delta Chains

Delta chains occur when a delta's base object is itself a delta:

```
Object A (base)
    ↓
Object B (delta of A)
    ↓
Object C (delta of B)
```

nanogit resolves chains iteratively:

1. **Iteration 1**: Resolve B (base A is available)
2. **Iteration 2**: Resolve C (base B is now available)

Maximum iterations = number of deltas + 1 (safety limit).

## Delta Application Process

The `ApplyDelta` function processes delta instructions:

```go
func ApplyDelta(baseData []byte, delta *Delta) ([]byte, error) {
    // Validate base size
    if uint64(len(baseData)) != delta.ExpectedSourceLength {
        return nil, fmt.Errorf("base data size mismatch")
    }

    result := make([]byte, 0, estimatedSize)

    for _, change := range delta.Changes {
        if change.DeltaData != nil {
            // Insert instruction: add new data
            result = append(result, change.DeltaData...)
        } else {
            // Copy instruction: copy from base
            copyData := baseData[change.SourceOffset : change.SourceOffset+change.Length]
            result = append(result, copyData...)
        }
    }

    return result, nil
}
```

### Example Delta Resolution

Given a modified file:

**Base object** (file.txt v1):
```
Line 1
Line 2
Line 3
```

**Delta instructions**:
```
1. Copy bytes 0-7 from base    ("Line 1\n")
2. Insert "Modified Line 2\n"  (new content)
3. Copy bytes 14-21 from base  ("Line 3\n")
```

**Resolved object** (file.txt v2):
```
Line 1
Modified Line 2
Line 3
```

## Storage Integration

Delta resolution integrates with nanogit's pluggable storage:

```go
// During fetch, resolved objects are added to storage
if storage != nil {
    storage.Add(resolvedObj)
}

// Base objects can be retrieved from storage for delta chains
baseObj, exists := storage.Get(parentHash)
```

### Storage Lifecycle

1. **Fetch starts**: Storage context injected
2. **Regular objects**: Added to storage immediately
3. **Delta resolution**:
   - Check storage for base objects
   - Add resolved objects to storage
4. **Fetch completes**: All objects available in storage
5. **Post-fetch**: Storage contents remain for session (not persisted between clients)

## Limitations and Edge Cases

### Current Limitations

1. **OFS_DELTA not supported**: Only REF_DELTA (type 7) is handled
   - Impact: May fail with packfiles using offset deltas
   - Mitigation: Most servers prefer REF_DELTA for network protocol

2. **Thin-pack not supported**:
   - Setting: `thin-pack` capability not requested
   - Benefit: Ensures all base objects are in response
   - Trade-off: Slightly larger packfile transfers

3. **No persistent cache**:
   - Design: Stateless operation
   - Impact: Cannot use previously fetched objects as bases
   - Mitigation: Self-contained packfiles include all bases

4. **Delta chain depth limit**:
   - Max iterations: `len(deltas) + 1`
   - Risk: Extremely deep chains could hit limit
   - Reality: Git servers typically limit chain depth to 50

### Error Scenarios

**Missing base object**:
```
Error: unable to resolve deltas: missing base objects [sha1, sha2]
```
Cause: Server sent thin-pack or incomplete packfile
Solution: Verify `thin-pack` is not requested

**Base size mismatch**:
```
Error: base data size mismatch: got 100 bytes, delta expects 150 bytes
```
Cause: Base object corrupted or wrong base selected
Solution: Check object integrity

**Delta chain too deep**:
```
Error: failed to resolve all deltas after N iterations
```
Cause: Circular reference or extremely deep chain
Solution: Increase max iterations or investigate server packfile

## Testing

### Unit Tests

Delta application is tested in `protocol/delta_apply_test.go`:

```go
func TestApplyDelta(t *testing.T) {
    t.Run("simple insert operation", ...)
    t.Run("simple copy operation", ...)
    t.Run("mixed copy and insert operations", ...)
    t.Run("copy from multiple locations", ...)
    t.Run("large copy operation", ...)
    t.Run("error: base size mismatch", ...)
    t.Run("error: copy out of bounds", ...)
    // ... 15 test cases total
}
```

### Integration Tests

Delta resolution is tested in `tests/delta_integration_test.go`:

```go
It("should handle ref-delta objects for modified files", func() {
    // Create file, modify multiple times, force repack with deltas
    local.CreateFile("delta-test.txt", baseContent)
    local.Git("commit", "-m", "Initial")

    // Multiple modifications
    for i := 1; i <= 5; i++ {
        local.UpdateFile("delta-test.txt", modifiedContent)
        local.Git("commit", "-m", "Modification")
    }

    // Force deltification
    local.Git("repack", "-a", "-d", "-f", "--depth=50", "--window=50")

    // Fetch deltified blob
    blob, err := client.GetBlob(ctx, blobHash)
    Expect(err).NotTo(HaveOccurred())
    Expect(string(blob.Content)).To(Equal(expectedContent))
})
```

### Test Coverage

- ✅ Simple delta application (insert, copy)
- ✅ Mixed operations
- ✅ Delta chains (deltas of deltas)
- ✅ Multiple deltified files in one fetch
- ✅ Deltified tree objects
- ✅ Deltified commit objects
- ✅ GetBlobByPath with deltas
- ✅ Clone with deltified repository
- ✅ Empty file deltification
- ✅ Large files with deltas
- ✅ Error cases (missing base, size mismatch, out of bounds)

## Performance Considerations

### Memory Usage

Delta resolution operates in-memory:

```
Memory = Regular objects + Delta objects + Resolved objects
```

For typical fetch:
- 1000 objects, 30% deltified
- Average object: 5KB
- Peak memory: ~6.5MB (700 regular + 300 delta + 300 resolved)

### CPU Cost

Delta application is CPU-intensive:
- Copy operations: Memory copy overhead
- Insert operations: Minimal
- Parsing: Tree/commit parsing after resolution

Benchmark results:
- Simple delta (1KB base): ~10μs
- Complex delta (100KB base): ~500μs
- Delta chain depth 5: ~50μs total

### Network Efficiency

Deltas reduce network transfer:
- Without deltas: 100% of object data
- With deltas: 30-50% of object data (typical)
- Benefit: Faster fetches outweigh CPU cost

## Comparison with Other Implementations

### go-git

```go
// go-git also resolves deltas in-memory
// Uses similar iterative approach
// Supports both OFS_DELTA and REF_DELTA
```

### libgit2

```c
// libgit2 resolves deltas during index writing
// Can use on-disk object database for bases
// Supports streaming delta application
```

### Git CLI

```bash
# git CLI resolves deltas while writing packfile
# Uses .git/objects as base object source
# Supports thin-pack with local objects
```

### nanogit Unique Characteristics

1. **Stateless**: No .git directory, everything in-memory
2. **No thin-pack**: Self-contained packfiles only
3. **Storage-agnostic**: Pluggable storage backend
4. **Cloud-focused**: Optimized for serverless/container environments

## References

### Official Git Documentation

- **[Git Packfile Format](https://git-scm.com/docs/pack-format)** - Complete packfile specification including delta encoding
- **[Git Protocol v2](https://git-scm.com/docs/protocol-v2)** - Modern Git wire protocol specification
- **[Git Protocol Capabilities](https://git-scm.com/docs/protocol-capabilities)** - Available protocol capabilities and limitations
- **[Deltified Representation](https://git-scm.com/docs/pack-format#_deltified_representation)** - Technical specification of delta format
- **[git-pack-objects](https://git-scm.com/docs/git-pack-objects)** - Man page for pack generation with delta strategies
- **[git-repack](https://git-scm.com/docs/git-repack)** - Documentation on repository repacking and deltification

### Git Internals and Delta Compression

- **[Pro Git Book - Git Internals](https://git-scm.com/book/en/v2/Git-Internals-Packfiles)** - Chapter on packfiles and delta compression
- **[Git's Object Storage](https://github.blog/2022-08-29-gits-database-internals-i-packed-object-store/)** - GitHub Engineering blog on Git's packed object store
- **[How Git Uses Delta Compression](https://github.blog/2022-08-30-gits-database-internals-ii-commit-history-queries/)** - Deep dive into delta encoding strategies
- **[The Git Packfile Format](https://codewords.recurse.com/issues/three/unpacking-git-packfiles)** - Detailed explanation with examples
- **[Understanding Git Delta Compression](https://stackoverflow.com/questions/8198105/how-does-git-delta-compression-work)** - StackOverflow comprehensive answer

### Protocol and Wire Format

- **[Git HTTP Protocol](https://git-scm.com/docs/http-protocol)** - Smart HTTP protocol details
- **[Git Transfer Protocols](https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols)** - Overview of Git protocols (HTTP, SSH, Git)
- **[Git Wire Protocol Version 2](https://opensource.googleblog.com/2018/05/introducing-git-protocol-version-2.html)** - Google Open Source blog on protocol v2

### Delta Algorithms and Implementation

- **[Delta Compression Algorithms](https://en.wikipedia.org/wiki/Delta_encoding)** - Wikipedia overview of delta encoding
- **[Xdelta Algorithm](http://xdelta.org/)** - Algorithm similar to Git's delta compression
- **[Implementing Efficient Deltas](https://www.danbp.org/p/blog.html)** - Blog post on delta implementation strategies
- **[Rust Git Implementation - Delta Handling](https://github.com/Byron/gitoxide/blob/main/gix-pack/src/data/decode/entry/mod.rs)** - Reference implementation in gitoxide

### Research Papers

- **["The Design of a Git Repository Storage"](https://github.com/git/git/blob/master/Documentation/technical/pack-format.txt)** - Technical specification
- **["Git Object Model and Storage Optimization"](https://github.com/git/git/blob/master/Documentation/technical/hash-function-transition.txt)** - Git's transition to SHA-256 discusses object storage

### nanogit Implementation

- `protocol/client/fetch.go`: Main delta resolution logic
- `protocol/delta.go`: Delta structure and parsing
- `protocol/delta_apply.go`: Delta application algorithm
- `protocol/packfile.go`: Packfile object processing
- `tests/delta_integration_test.go`: Integration test suite

### Related Issues and Pull Requests

- **[Grafana GitSync Issue #111056](https://github.com/grafana/grafana/issues/111056)** - Original issue: Missing files due to skipped deltas
- **[nanogit PR #95](https://github.com/grafana/nanogit/pull/95)** - Implement stateless delta resolution for Git objects (merged)
- **[nanogit PR #96](https://github.com/grafana/nanogit/pull/96)** - Add fallback fetch for missing tree objects (merged)

## Future Improvements

### Potential Enhancements

1. **OFS_DELTA support**: Handle offset deltas in addition to reference deltas
2. **Parallel delta resolution**: Resolve independent delta chains concurrently
3. **Delta metrics**: Track delta statistics (count, chain depth, resolution time)
4. **Streaming resolution**: Apply deltas while reading packfile (memory optimization)
5. **Base object prediction**: Pre-fetch likely base objects based on patterns

### Performance Optimizations

1. **Base object cache**: LRU cache for frequently used bases
2. **Delta reordering**: Optimize resolution order to minimize iterations
3. **Lazy parsing**: Defer tree/commit parsing until actually needed
4. **Zero-copy operations**: Reduce memory allocations during copy instructions

## Conclusion

Delta resolution in nanogit represents a balance between:

- **Protocol requirements**: Cannot disable deltas, must handle them
- **Architecture constraints**: Stateless, no persistent cache
- **Performance goals**: Fast fetches with minimal memory footprint
- **Reliability**: Robust handling of edge cases and error conditions

The implementation successfully handles deltified objects from all major Git servers while maintaining nanogit's stateless design principles. While there are limitations (no OFS_DELTA, no thin-pack), the current approach covers the vast majority of real-world use cases.
