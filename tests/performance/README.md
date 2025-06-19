# Performance Testing Suite

This package provides comprehensive performance benchmarking for comparing nanogit, go-git, and git CLI across various Git operations using containerized Gitea servers for self-contained testing.

## Architecture

### Module Structure

The performance tests are in a **separate Go module** (`tests/performance/go.mod`) to:
- Avoid dependency conflicts with the main nanogit module
- Include heavy dependencies (testcontainers, go-git) only for performance testing
- Allow independent versioning and dependency management

### Core Components

- **go.mod**: Separate module with performance testing dependencies
- **types.go**: Common interfaces and data structures
- **metrics.go**: Performance metrics collection and reporting
- **gitserver.go**: Containerized Gitea server with pre-created repository mounting
- **clients/**: Implementation wrappers for each Git client
- **cmd/generate_repo/**: Script to generate test repository archives
- **testdata/**: Pre-created repository archives and documentation
- **benchmark_test.go**: Main benchmark test suites

### Supported Clients

1. **nanogit**: Native nanogit client implementation
2. **go-git**: go-git library with memory storage
3. **git-cli**: Standard git command-line interface

### Self-Contained Testing

- Uses testcontainers to spin up Gitea servers automatically
- Generates test repositories with realistic data patterns
- Simulates network latency for more realistic benchmarks
- No external dependencies on GitHub or other services

## Test Types

### 1. Consistency Tests
Verify that all three Git clients (nanogit, go-git, git-cli) produce identical results:

- **Simple Consistency** (`test-perf-simple`): Basic functionality verification
  - File creation, updates, and deletions
  - Bulk operations with multiple files
  - Repository integrity preservation
  - ~5 minutes runtime

- **Full Consistency** (`test-perf-consistency`): Comprehensive cross-client comparison
  - Complex scenarios with sequential operations
  - Git CLI verification of all operations
  - File content and commit message validation
  - ~10 minutes runtime

### 2. Performance Benchmarks
Measure and compare performance across different repository sizes and operations:

- **File Operations** (`test-perf-file-ops`): Core Git file operations
  - Create, update, delete files
  - Tested across small/medium/large/xlarge repositories
  - Multiple iterations for statistical significance

- **Commit Comparison** (`test-perf-compare`): Diff and comparison operations
  - Adjacent commits vs distant commits
  - Small vs large changesets
  - Performance scaling with repository size

- **Tree Listing** (`test-perf-tree`): Repository browsing operations
  - Flat tree traversal
  - Performance with varying file counts
  - Directory depth impact

- **Bulk Operations** (`test-perf-bulk`): Multi-file operations
  - Single commit with multiple files (10, 100, 1000 files)
  - Efficiency of batched operations
  - Memory usage patterns

### 3. Client-Specific Testing
Focus testing on individual Git client implementations:

- **Nanogit** (`test-perf-nanogit`): Native nanogit client performance
- **Go-Git** (`test-perf-gogit`): go-git library with shallow clone optimizations  
- **Git CLI** (`test-perf-cli`): Standard git command-line interface

## Test Scenarios

### Repository Sizes
- **Small**: 50 files, 32 commits
- **Medium**: 500 files, 165 commits  
- **Large**: 2000 files, 692 commits
- **XLarge**: 10000 files, 2602 commits

### Operation Patterns
- **Sequential**: One operation at a time
- **Bulk**: Multiple files in single commit
- **Mixed**: Combination of creates, updates, deletes

## Quick Reference

| Make Target | Purpose | Runtime | Description |
|-------------|---------|---------|-------------|
| `test-perf-setup` | Setup | 1-2 min | Generate test repository archives (one-time) |
| `test-perf-simple` | Consistency | ~5 min | Basic client functionality verification |
| `test-perf-consistency` | Consistency | ~10 min | Full cross-client comparison |
| `test-perf-nanogit` | Client-specific | ~5 min | Focus on nanogit client testing |
| `test-perf-gogit` | Client-specific | ~5 min | Focus on go-git client testing |
| `test-perf-cli` | Client-specific | ~5 min | Focus on git-cli client testing |
| `test-perf-file-ops` | Performance | ~15 min | File operations benchmarks |
| `test-perf-compare` | Performance | ~10 min | Commit comparison benchmarks |
| `test-perf-tree` | Performance | ~10 min | Tree listing benchmarks |
| `test-perf-bulk` | Performance | ~15 min | Bulk operations benchmarks |
| `test-perf` | Combined | ~25 min | Core tests (consistency + file ops) |
| `test-perf-all` | Complete | ~30+ min | All performance tests |

## Usage

### Make Targets (Recommended)

**Important**: These tests require Docker and are disabled by default.

```bash
# First-time setup: Generate test repository archives (required)
make test-perf-setup

# Quick consistency tests (recommended for development)
make test-perf-simple           # Simple client consistency tests
make test-perf-consistency      # Full client consistency tests

# Individual performance test suites
make test-perf-file-ops         # File operations (create/update/delete)
make test-perf-compare          # Commit comparison operations
make test-perf-tree             # Tree listing operations  
make test-perf-bulk             # Bulk file operations

# Client-specific tests (filters output for specific client)
make test-perf-nanogit          # Tests focusing on nanogit client
make test-perf-gogit            # Tests focusing on go-git client
make test-perf-cli              # Tests focusing on git-cli client

# Complete test suites
make test-perf                  # Core performance tests (consistency + file ops)
make test-perf-all              # All performance tests (may take 30+ minutes)
```

### Manual Test Execution

For advanced usage and custom configurations:

```bash
# Performance tests are in a separate Go module, so you need to cd first
cd tests/performance

# Enable performance tests (required for manual execution)
export RUN_PERFORMANCE_TESTS=true

# Optional: Add network latency simulation (in milliseconds)
export PERF_TEST_LATENCY_MS=50

# Run all performance tests
go test -v .

# Run specific test suite
go test -v -run TestFileOperationsPerformance .

# Run Go benchmarks for detailed profiling (no latency)
go test -bench=. .

# Generate CPU and memory profiles
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof .

# Return to root directory
cd ../..
```

**Note**: The performance tests are in a separate Go module (`tests/performance/go.mod`) to avoid dependency conflicts with the main nanogit module. The make targets automatically handle the directory change.

### Environment Setup

#### Prerequisites

- **Docker**: Required for testcontainers
- **Git CLI**: Required for git-cli client testing

#### Configuration

```bash
# Enable performance tests (required)
export RUN_PERFORMANCE_TESTS=true

# Optional: Network latency simulation (milliseconds)
export PERF_TEST_LATENCY_MS=100  # Simulates 100ms network latency

# Tests are self-contained - no external repos needed!
```

### Network Latency Simulation

The framework can simulate network latency to test performance under realistic network conditions:

- **0ms** (default): No latency, maximum speed
- **50ms**: Typical cloud-to-cloud latency
- **100ms**: Typical internet latency
- **250ms**: High latency scenarios

### Test Data Generation

Test repositories use pre-created archives for fast, consistent testing:

#### First-time Setup

```bash
# Generate test repository archives (one-time setup)
cd tests/performance
go run ./cmd/generate_repo
```

This creates four repository archives in `testdata/`:
- `small-repo.tar.gz` - Small repository (100 files, 50 commits, 4 directory levels)
- `medium-repo.tar.gz` - Medium repository (750 files, 200 commits, 5 directory levels)  
- `large-repo.tar.gz` - Large repository (3000 files, 800 commits, 6 directory levels)
- `xlarge-repo.tar.gz` - Extra-large repository (15000 files, 3000 commits, 8 directory levels)

#### How It Works

```go
// Repositories are extracted and mounted from pre-created archives
server, err := NewGitServer(ctx, networkLatency)
repositories, err := server.ProvisionTestRepositories(ctx) // Extracts archives

// Benefits:
// - Fast test startup (no file/commit generation time)
// - Consistent test data across runs
// - Better performance isolation
// - Reproducible results
```

## Metrics and Reporting

### Collected Metrics

- **Duration**: Execution time (average, median, P95)
- **Memory Usage**: Peak and average memory consumption
- **Success Rate**: Percentage of successful operations
- **Operation Count**: Number of files/changes processed

### Report Formats

- **JSON**: Machine-readable detailed results
- **Text**: Human-readable summary statistics

### Sample Output

```
=== FileOperations_small_repo ===

nanogit:
  Runs: 3
  Success Rate: 100.00%
  Duration - Avg: 1.2s, Median: 1.1s, P95: 1.5s
  Memory - Avg: 2.1MB, Median: 2.0MB

go-git:
  Runs: 3
  Success Rate: 100.00%
  Duration - Avg: 0.8s, Median: 0.7s, P95: 1.0s
  Memory - Avg: 15.3MB, Median: 14.8MB
```

## Configuration

### BenchmarkConfig

```go
type BenchmarkConfig struct {
    RepoURL     string        // Target repository
    RepoSize    string        // "small", "medium", "large"
    FileCount   int           // Number of files in test
    Iterations  int           // Repetitions per test
    Timeout     time.Duration // Maximum test duration
}
```

### Repository Specifications

```go
type RepoSpec struct {
    Name          string        // Repository identifier
    FileCount     int           // Total number of files
    CommitCount   int           // Number of commits to generate
    MaxDepth      int           // Maximum directory depth
    FileSizes     []int         // Distribution of file sizes
    BinaryFiles   int           // Number of binary files
    Branches      int           // Number of branches
}
```

## Implementation Notes

### Client Implementations

- **nanogit**: Uses native client and StagedWriter interfaces with HTTP-based operations
- **go-git**: Optimized implementation with shallow clones, single-branch fetching, and no repository caching for consistent testing
- **git-cli**: Temporary local repositories with shell command execution, fresh clone per operation

### Performance Considerations

- Tests run in isolated Docker containers with controlled environments
- Memory measurements include GC cycles for accuracy
- Multiple iterations provide statistical significance
- Network latency can be simulated for realistic testing
- Results are cached and aggregated for reporting

### Go-Git Optimizations

The go-git client has been optimized for better performance:

- **Shallow Clones**: Uses `--depth=1` for file operations to minimize data transfer
- **Single Branch**: Only clones the default branch with `SingleBranch: true`
- **No Repository Caching**: Fresh clone for each operation ensures consistent test conditions
- **Optimized Object Cache**: Smaller LRU cache (256 objects) for better memory usage
- **Smart Clone Selection**: Shallow clones for file operations, full clones for commit comparisons

### Limitations

- **Docker Required**: Tests require Docker for testcontainers
- **git-cli**: Requires git binary installation
- **Resource Intensive**: Spins up Gitea containers for each test run
- **Network Latency**: Simulated latency may not perfectly match real-world conditions
- **No CI/CD**: These tests are designed for manual/on-demand execution only

## Future Enhancements

- **CI/CD Integration**: Lightweight mode for automated regression detection
- **Additional Operations**: Support for merge, rebase, and branch operations
- **Performance Baselines**: Establish and track performance regression thresholds
- **Historical Tracking**: Store and compare performance trends over time
- **Resource Monitoring**: Track CPU, disk I/O, and network usage
- **Parallel Testing**: Run multiple client tests concurrently

