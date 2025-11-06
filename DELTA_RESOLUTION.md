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

nanogit uses Git protocol v2 (Smart HTTP protocol), which provides various fetch capabilities including thin-pack, no-progress, include-tag, and object filtering.

**Critical limitation**: There is **NO capability to disable delta objects** in Git protocol v2.

### Why "thin-pack" Doesn't Help

The `thin-pack` capability controls whether the server can send "thin" packfiles that reference objects the client already has, but it does **NOT** control deltification. This is a common misunderstanding when working with Git protocols.

### Attempted Workarounds (All Failed)

We investigated several approaches to avoid deltas, all unsuccessful:

1. **Request full objects via protocol**: No protocol capability exists
2. **Use `no-thin` pack**: Only affects base object inclusion, not deltification
3. **Fetch individual objects**: Server still sends deltas for efficiency
4. **Use different protocol version**: v0, v1, v2 all support deltas without disable option

### Server-Side Behavior

The server's pack generation decides deltification based on:
- Repository pack structure
- Recent `git repack` operations
- Server-side optimization settings

Even when requesting individual objects, the server may still send them as deltas if they were deltified in the repository's pack files.

**Conclusion**: Delta handling is mandatory for any Git client implementation.

## nanogit's Delta Resolution Approach

Given the impossibility of disabling deltas, nanogit implements **stateless in-memory delta resolution**.

### Design Constraints

1. **Stateless architecture**: No persistent local .git directory or object cache
2. **No thin-pack support**: All base objects must be in the same fetch response
3. **In-memory processing**: Deltas resolved during fetch operation
4. **Single-fetch scope**: Base objects only available within the same fetch

### Implementation Strategy

The delta resolution process follows these stages:

1. **Fetch Request**: Client requests objects from server using Git protocol v2
2. **Packfile Response**: Server returns packfile containing regular objects and deltas
3. **Object Collection**: Regular objects stored immediately, deltas collected separately
4. **Delta Resolution**: Iteratively resolve deltas by finding bases and applying instructions
5. **Return Complete Objects**: All objects fully resolved and available to caller

### Handling Delta Chains

Delta chains occur when a delta's base object is itself a delta (e.g., Object C depends on Object B which depends on Object A). nanogit resolves these chains iteratively:

- **First iteration**: Resolve deltas whose bases are available
- **Subsequent iterations**: Resolve deltas whose bases were just resolved
- **Maximum iterations**: Limited to prevent infinite loops in case of errors

### Delta Application Process

For each delta object, nanogit:

1. Validates the base object size matches the delta's expectation
2. Processes each delta instruction in sequence:
   - **Copy instructions**: Extract bytes from base object
   - **Insert instructions**: Add new literal bytes
3. Reconstructs the complete target object
4. Calculates the hash of the resolved object
5. Parses structured data for trees and commits

## Storage Integration

Delta resolution integrates with nanogit's pluggable storage system:

- Base objects can be retrieved from storage if previously fetched
- Resolved objects are added to storage for use in resolving subsequent deltas
- Storage contents remain available for the session duration

### Storage Lifecycle

1. Fetch starts with storage context injected
2. Regular objects added to storage immediately
3. Delta resolution checks storage for base objects
4. Resolved objects added to storage
5. All objects available post-fetch

## Limitations and Edge Cases

### Current Limitations

1. **OFS_DELTA not supported**: Only REF_DELTA (type 7) is handled
   - Impact: May fail with packfiles using offset deltas
   - Mitigation: Most servers prefer REF_DELTA for network protocol

2. **Thin-pack not supported**:
   - Ensures all base objects are in response
   - Trade-off: Slightly larger packfile transfers

3. **No persistent cache**:
   - Stateless operation by design
   - Cannot use previously fetched objects as bases across sessions

4. **Delta chain depth limit**:
   - Max iterations prevents infinite loops
   - Git servers typically limit chain depth to 50

### Error Scenarios

**Missing base object**: Occurs when server sends thin-pack or incomplete packfile. Verify thin-pack is not requested.

**Base size mismatch**: Base object corrupted or wrong base selected. Check object integrity.

**Delta chain too deep**: Circular reference or extremely deep chain. May require investigating server packfile generation.

## Performance Considerations

### Memory Usage

Delta resolution operates entirely in-memory, requiring space for:
- Regular objects received from server
- Delta objects awaiting resolution
- Resolved objects after delta application

For typical fetches with moderate deltification, memory usage remains reasonable for cloud environments.

### CPU Cost

Delta application is CPU-intensive as it involves:
- Memory copy operations for copy instructions
- Minimal overhead for insert operations
- Object parsing for trees and commits after resolution

The network bandwidth savings from deltas typically outweigh the CPU cost.

### Network Efficiency

Deltas significantly reduce network transfer size:
- Without deltas: 100% of object data
- With deltas: 30-50% of object data (typical savings)

This makes delta support essential for acceptable fetch/clone performance.

## Comparison with Other Implementations

### go-git

Uses similar iterative delta resolution approach in-memory. Supports both OFS_DELTA and REF_DELTA types.

### libgit2

Resolves deltas during index writing phase. Can leverage on-disk object database for finding base objects. Supports streaming delta application.

### Git CLI

Resolves deltas while writing packfile to disk. Uses .git/objects as base object source. Supports thin-pack with local object database.

### nanogit Unique Characteristics

1. **Stateless**: No .git directory, everything in-memory
2. **No thin-pack**: Self-contained packfiles only
3. **Storage-agnostic**: Pluggable storage backend
4. **Cloud-focused**: Optimized for serverless/container environments

## Testing Strategy

nanogit includes comprehensive testing for delta resolution:

### Unit Tests

Cover delta application logic including:
- Basic copy and insert operations
- Mixed operation sequences
- Error conditions (size mismatch, bounds checking)
- Binary data handling

### Integration Tests

Verify end-to-end delta handling with real Git servers:
- Deltified blobs from modified files
- Deltified tree objects
- Deltified commit objects
- Delta chains (deltas of deltas)
- Complex repository structures
- Edge cases (empty files, large files)

Test repositories are created with explicit deltification using `git repack` to ensure deltas are present.

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

For implementation details, see:
- `protocol/client/fetch.go` - Main delta resolution logic
- `protocol/delta.go` - Delta structure and parsing
- `protocol/delta_apply.go` - Delta application algorithm
- `protocol/packfile.go` - Packfile object processing
- `tests/delta_integration_test.go` - Integration test suite

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
