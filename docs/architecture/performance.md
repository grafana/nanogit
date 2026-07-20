# Performance

nanogit is substantially faster and lighter than a general-purpose Git library for the server-side operations it implements. On the XL benchmark tier (15,000 files, 3,000 commits), measured operations run **260–300x faster than [go-git](https://github.com/go-git/go-git) while allocating 150–200x less memory**. This page explains where that difference comes from, shows the measured results, and documents how to run and profile the benchmarks yourself.

Every number here comes from the benchmark suite in [`perf/`](https://github.com/grafana/nanogit/tree/main/perf); the raw output lives in [`perf/LAST_REPORT.md`](https://github.com/grafana/nanogit/blob/main/perf/LAST_REPORT.md).

## Why nanogit is fast

### 1. Stateless operations

Traditional Git implementations operate on a local `.git` directory: every operation reads and writes loose objects, index files, and packfiles on disk. nanogit keeps no local repository state at all — it fetches exactly the objects an operation needs over HTTPS and holds operation state in memory:

- No `.git` directory, index, or object-database I/O
- No file locking; operations on different repositories share nothing
- No per-repository state to create or clean up, which matters for cold starts in containers and serverless environments

### 2. Streaming-first data pipeline

Writes stream directly from packfile creation to the network (`io.Pipe`), with no intermediate files:

- Data begins transmitting while the packfile is still being built
- Memory stays bounded regardless of repository size, because objects are processed as streams rather than materialized

### 3. Fetch only what the operation needs

Because nanogit speaks the [Smart HTTP protocol v2](protocol-v2.md) directly, it can be precise about what it asks the server for: shallow fetches (`deepen 1`), `filter blob:none` when only tree structure is needed, and early termination of packfile reads once every wanted object has arrived. Cost scales with the size of the change, not the size of the repository.

### 4. Configurable storage

Two layers can be tuned per workload (see [Storage Backend](storage.md)):

- **Write-time storage**: `PackfileStorageAuto` (in memory below 10 staged objects and 5 MB, on disk beyond), `PackfileStorageMemory` (fastest), or `PackfileStorageDisk` (minimal memory for bulk operations)
- **Object caching**: a context-injected, pluggable object cache so repeated reads across operations can share fetched objects

## Benchmark results

July 2025 run, nanogit vs go-git on the XL repository tier (15,000 files, 3,000 commits). One run per client per scenario; memory is the change in Go heap allocation measured around each operation.

| Operation (XL tier) | nanogit | go-git | Speed         | Memory      |
| ------------------- | ------- | ------ | ------------- | ----------- |
| CreateFile          | 79.4ms  | 22.34s | 281.6x faster | 198.4x less |
| UpdateFile          | 75.1ms  | 22.33s | 297.3x faster | 189.2x less |
| DeleteFile          | 79.6ms  | 22.34s | 280.5x faster | 200.5x less |
| GetFlatTree         | 76.1ms  | 19.84s | 260.8x faster | 154.3x less |

The gap narrows on smaller repositories, where go-git has less state to materialize: the same four operations are 8.4–10.4x faster on the medium tier (750 files) and 1.4–2.1x faster on the small tier (100 files).

The suite also benchmarks the `git` CLI, which nanogit beats by 85.8–92.8x on wall-clock time in the four scenarios above. Memory comparisons against a subprocess are not meaningful — the CLI's working set lives in the OS page cache rather than in process memory — so the library-to-library comparison is the one that matters for embedding.

Reproduce with `cd perf && make test-perf-setup && make test-perf-all` (requires Docker).

## The trade-off: tailored vs. generic

go-git is a comprehensive Git implementation: all transports, all storage backends, the full feature surface. That breadth has a cost — abstraction layers, full-repository data structures, and local-disk semantics — that it pays on every operation.

nanogit makes the opposite trade. It implements one transport (Smart HTTP v2), one workload shape (server-side reads and writes), and only the object plumbing those need. The performance gap in the table above is mostly this: **go-git materializes and maintains a repository; nanogit fetches, transforms, and streams exactly the objects one operation requires.**

That trade is only a win when the constraints fit. If you need SSH, local worktrees, merges, or the rest of full Git, go-git's breadth is worth its cost — see [When should I not use it?](../index.md#when-should-i-not-use-it).

## Profiling

The `perf/Makefile` carries the profiling targets used to validate optimizations and catch regressions:

```bash
make profile-baseline    # Capture baseline CPU + memory profiles
make profile-cpu         # CPU profile
make profile-mem         # Memory profile
make profile-all         # Both
make profile-all-tree    # Tree operations only
make profile-all-commit  # Commit operations only
make profile-compare     # Current profiles vs. baseline
```

Analyze the resulting profiles with `pprof`:

```bash
go tool pprof profiles/cpu.prof                                  # interactive
go tool pprof -http=:8080 profiles/cpu.prof                      # web UI
go tool pprof -diff_base=profiles/baseline_cpu.prof profiles/cpu.prof  # differential
go tool pprof -alloc_space profiles/mem.prof                     # allocation hotspots
```

Profile before and after every change against a maintained baseline, measure both CPU and memory, and check the result across repository sizes — a win on the XL tier can be a regression on the small one.
