# Performance Testing Suite

This package provides comprehensive performance benchmarking for comparing nanogit, go-git, and git CLI across various Git operations using containerized Gitea servers for self-contained testing.

## Architecture

### Core Components

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

## Test Scenarios

### 1. File Operations

- **CreateFile**: Create new files at various path depths
- **UpdateFile**: Modify existing file content
- **DeleteFile**: Remove files from repository

### 2. CompareCommits

- Adjacent commits (single file changes)
- Distant commits (multiple commits apart)
- Large diffs (many file changes)

### 3. GetFlatTree

- Small trees (50 files)
- Medium trees (500 files)
- Large trees (2000+ files)

### 4. Bulk Operations

- Bulk file creation (10, 100, 1000 files)
- Mixed operations (create/update/delete)

## Usage

### Running Benchmarks

**Important**: These tests require Docker and are disabled by default.

```bash
# First-time setup: Generate test repository archives
cd tests/performance
go run ./cmd/generate_repo

# Enable performance tests (required)
export RUN_PERFORMANCE_TESTS=true

# Optional: Add network latency simulation (in milliseconds)
export PERF_TEST_LATENCY_MS=50

# Run all performance tests
go test -v ./tests/performance

# Run specific test suite
go test -v ./tests/performance -run TestFileOperationsPerformance

# Run Go benchmarks for detailed profiling (no latency)
go test -bench=. ./tests/performance

# Generate CPU and memory profiles
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof ./tests/performance
```

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

- **nanogit**: Uses native client and StagedWriter interfaces
- **go-git**: Memory-only storage for consistent performance testing
- **git-cli**: Temporary local repositories with shell command execution

### Performance Considerations

- Tests run in isolated Docker containers with controlled environments
- Memory measurements include GC cycles for accuracy
- Multiple iterations provide statistical significance
- Network latency can be simulated for realistic testing
- Results are cached and aggregated for reporting

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

