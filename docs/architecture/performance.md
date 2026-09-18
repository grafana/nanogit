# Performance

nanogit is substantially faster and lighter than a general-purpose Git library for the server-side operations it implements. On the XL benchmark tier (15,000 files, 3,000 commits), measured file operations run **~280–300x faster than [go-git](https://github.com/go-git/go-git) while allocating ~150–200x less memory**. This page explains where that difference comes from, shows the measured results, and documents how to run and profile the benchmarks yourself.

All performance claims on this page come from the benchmark suite in [`perf/`](https://github.com/grafana/nanogit/tree/main/perf); the full raw output lives in [`perf/LAST_REPORT.md`](https://github.com/grafana/nanogit/blob/main/perf/LAST_REPORT.md).

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

Because nanogit speaks the [Smart HTTP protocol v2](protocol-v2.md) directly, it can be precise about what it asks the server for: shallow fetches, path-filtered clones, and batched object requests. Most operations on a large repository touch a small subpath — nanogit's cost scales with the size of the change, not the size of the repository.

### 4. Configurable storage

Two layers can be tuned per workload (see [Storage Backend](storage.md)):

- **Write-time storage**: `PackfileStorageAuto` (≤10 staged objects in memory, more on disk), `PackfileStorageMemory` (fastest), or `PackfileStorageDisk` (minimal memory for bulk operations)
- **Object caching**: a context-injected, pluggable object cache so repeated reads across operations can share fetched objects

## Benchmark results

Numbers below are from the July 2025 run of the [benchmark suite](https://github.com/grafana/nanogit/tree/main/perf) ([full report](https://github.com/grafana/nanogit/blob/main/perf/LAST_REPORT.md)), comparing nanogit against go-git on the XL repository tier (15,000 files, 3,000 commits). Durations are averages over the run; memory multipliers compare peak allocations.

| Operation (XL tier) | nanogit | go-git | Speed        | Memory      |
| ------------------- | ------- | ------ | ------------ | ----------- |
| CreateFile          | 79.4ms  | 22.3s  | 281.6x faster | 198.4x less |
| UpdateFile          | 75.1ms  | 22.3s  | 297.3x faster | 189.2x less |
| DeleteFile          | 79.6ms  | 22.3s  | 280.5x faster | 200.5x less |
| GetFlatTree         | 76.1ms  | 19.8s  | 260.8x faster | 154.3x less |

Two caveats, in the interest of honest numbers:

- **go-git did not complete the `BulkCreateFiles` and `CompareCommits` scenarios** in that run (0% success in the report), so no comparison multipliers are quoted for them. For scale, nanogit completed a 1,000-file bulk create against the medium repository in 102.8ms and a full-history XL commit comparison in 333.5ms.
- The report also benchmarks the **`git` CLI**, which nanogit outperforms by ~86–93x on wall-clock time in the same four XL scenarios (and by 2.6–119x across all scenarios and sizes). Memory comparisons against a subprocess are less meaningful (the CLI's working set lives in the OS page cache rather than process memory), so the library-to-library comparison above is the one that matters for embedding.

The report predates the v1.0 release. Re-running the suite against a current release is tracked as follow-up work; the suite itself is reproducible with `cd perf && make test-perf-setup && make test-perf-all` (requires Docker).

## The trade-off: tailored vs. generic

go-git is a comprehensive Git implementation: all transports, all storage backends, the full feature surface. That breadth has a cost — abstraction layers, full-repository data structures, and local-disk semantics — that it pays on every operation.

nanogit makes the opposite trade. It implements one transport (Smart HTTP v2), one workload shape (server-side reads and writes), and only the object plumbing those need. The performance gap in the table above is mostly this: **go-git materializes and maintains a repository; nanogit fetches, transforms, and streams exactly the objects one operation requires.**

That trade is only a win when the constraints fit. If you need SSH, local worktrees, merges, or the rest of full Git, go-git's breadth is worth its cost — see [When should I not use it?](../index.md#when-should-i-not-use-it).

## Profiling infrastructure

nanogit includes profiling tools for performance analysis and optimization validation, located in `perf/Makefile`:

```bash
# Establish performance baseline
make profile-baseline

# Profile CPU hotspots
make profile-cpu

# Profile memory allocations
make profile-mem

# Profile both CPU and memory
make profile-all

# Compare with baseline
make profile-compare
```

### Optimization methodology

**1. Baseline establishment**
```bash
cd perf
make profile-baseline  # Creates baseline profiles
```
This captures performance characteristics before optimization attempts.

**2. Targeted profiling**
```bash
# Profile specific operations
make profile-all-tree    # Tree operations
make profile-all-commit  # Commit operations
make profile-cpu         # File operations
```

**3. Performance analysis**
```bash
# Interactive analysis
go tool pprof profiles/cpu.prof
go tool pprof profiles/mem.prof

# Web-based analysis
go tool pprof -http=:8080 profiles/cpu.prof

# Comparative analysis
go tool pprof -diff_base=profiles/baseline_cpu.prof profiles/cpu.prof
```

**4. Optimization validation**
```bash
make profile-compare  # Shows improvement metrics
```

### Real optimization examples

**Tree comparison optimization**:
- **Before**: 53.15MB memory usage
- **After**: 11.44MB memory usage
- **Method**: Pre-allocation and capacity estimation
- **Result**: 78% memory reduction

**Packet length reading optimization**:
- **Issue**: 19% CPU regression from frequent allocations
- **Solution**: Buffer pooling with `sync.Pool`
- **Method**: Replace per-call allocations with pooled buffers
- **Result**: Restored CPU performance to baseline

### Performance regression prevention

- Profile before and after every optimization, against a maintained baseline
- Focus on functions consuming >1% of total resources, and on allocations in hot paths
- Measure both CPU and memory impact, and validate across repository sizes

**Differential profiling**:
```bash
go tool pprof -diff_base=baseline.prof current.prof -top
```

**Flame graph generation**:
```bash
go tool pprof -png profiles/cpu.prof > cpu_flame.png
```

**Memory allocation tracking**:
```bash
go tool pprof -alloc_space profiles/mem.prof
```
