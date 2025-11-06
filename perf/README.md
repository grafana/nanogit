# Performance Testing Suite

Comprehensive performance benchmarking comparing nanogit, go-git, and git CLI using containerized Gitea servers.

## Quick Start

```bash
cd perf

# One-time setup
make test-perf-setup

# Quick tests (recommended)
make test-perf-simple      # Basic consistency tests (~3 min)
make help                  # See all available targets
```

## Test Types

- **Consistency Tests**: Verify all clients produce identical results
- **Performance Benchmarks**: Measure duration and memory across repository sizes
- **Client-Specific**: Focus on individual Git implementations

### Repository Sizes
- **Small**: 50 files, 32 commits
- **Medium**: 500 files, 165 commits  
- **Large**: 2000 files, 692 commits
- **XLarge**: 10000 files, 2602 commits

## Common Targets

| Target | Purpose | Runtime |
|--------|---------|---------|
| `test-perf-setup` | Generate test data | 1-2 min |
| `test-perf-simple` | Basic consistency | ~3 min |
| `test-perf-consistency` | Full consistency | ~5 min |
| `test-perf-file-ops` | File operations | ~8 min |
| `test-perf-tree` | Tree listing | ~4 min |
| `test-perf-bulk` | Bulk operations | ~7 min |
| `test-perf-small` | Small repos only | ~3 min |
| `test-perf-all` | Everything | ~20 min |

## Requirements

- **Docker**: For testcontainers
- **Git CLI**: For git-cli client testing
- **Separate Go module**: `perf/go.mod`

## Configuration

```bash
# Enable tests (required)
export RUN_PERFORMANCE_TESTS=true

# Optional: Network latency simulation
export PERF_TEST_LATENCY_MS=100

# Optional: Specific repositories only
export PERF_TEST_REPOS=small,medium
```

## Architecture

- **Self-contained**: Uses pre-created repository archives
- **Containerized**: Gitea servers with Docker
- **Multi-client**: nanogit, go-git, git CLI comparison
- **Metrics**: Duration, memory, success rates with JSON/text reports

## Test Data Generation

Pre-generated Git repository archives provide fast, consistent testing:

```bash
cd perf
go run ./cmd/generate_repo
```

Creates four archives in `testdata/`:
- `small-repo.tar.gz` - 100 files, 50 commits
- `medium-repo.tar.gz` - 750 files, 200 commits  
- `large-repo.tar.gz` - 3000 files, 800 commits
- `xlarge-repo.tar.gz` - 15000 files, 3000 commits

Each archive contains a complete Git repository with realistic file structure, various file types, and full commit history. Benefits: fast startup, consistent data, reproducible test conditions.

## Manual Execution

```bash
cd perf
export RUN_PERFORMANCE_TESTS=true

# Run specific tests
go test -v -run TestFileOperationsPerformance .

# Run benchmarks
go test -bench=. .
```

See `make help` for complete target list.

## Clone Performance Testing

nanogit includes specialized performance tests for the `Clone` operation that measure real-world cloning performance against production repositories. These tests are separate from the standard performance suite because they interact with external repositories and take longer to execute.

### Real-World Clone Testing

The clone performance tests target the `grafana/grafana` repository on GitHub, providing realistic performance measurements against a large, production repository with thousands of files and commits.

**Test Scenarios**:

```bash
# Run clone performance tests with subpath filtering
make test-clone-perf

# Run full repository clone test (optional)
make test-clone-perf-full
```

**Tested Scenarios**:
1. **Subpath Filtering**: Clone specific directories (e.g., `pkg/api/**`)
2. **File Type Filtering**: Clone only specific file types (e.g., `*.go`, `*.md`)
3. **Multi-Path Filtering**: Clone multiple directories in a single operation
4. **Full Repository Clone**: Clone entire repository without filtering (optional, requires explicit opt-in)

### Performance Metrics

Clone performance tests measure and report:

- **Duration**: Total time to complete clone operation
- **File Counts**: Total files in repository vs. filtered files cloned
- **Filter Efficiency**: Percentage of files included after filtering
- **Throughput**: Files per second cloning rate
- **Verification**: Disk write validation and directory structure checks

Example output:
```
Clone completed successfully
Duration: 12.5s
Total files in repository: 15,234
Filtered files cloned: 127
Clone path: /tmp/nanogit_clone
Performance metrics:
  - Files/second: 10.16
  - Filter ratio: 0.83% of files cloned
```

### Why Separate from Standard Performance Tests

Clone performance tests are isolated from the main performance suite for several reasons:

1. **External Dependencies**: Requires network access to GitHub
2. **Execution Time**: Takes 30-60 minutes to run all scenarios
3. **Resource Usage**: Downloads real repository data (hundreds of MB)
4. **Environment Variables**: Separate opt-in via `RUN_CLONE_PERF_TESTS=true`
5. **CI Integration**: Runs independently from unit/integration tests

### CI Integration

The clone performance tests run automatically in CI as a separate job:

```yaml
test-clone-perf:
  name: Clone Performance Tests
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - name: Run clone performance tests
      run: cd perf && make test-clone-perf
```

This provides continuous monitoring of clone performance against real-world repositories and detects performance regressions in the Clone functionality.

### Make Targets

```bash
# From root directory
make test-clone-perf              # Run clone tests with subpath filtering

# From perf directory
make test-clone-perf              # Run clone tests with subpath filtering
make test-clone-perf-subpath      # Alias for test-clone-perf
make test-clone-perf-full         # Run full repository clone (requires RUN_FULL_CLONE_TEST=true)
```

### Configuration

```bash
# Enable clone performance tests
export RUN_CLONE_PERF_TESTS=true

# Enable full repository clone test (optional)
export RUN_FULL_CLONE_TEST=true
```

### Optimization Opportunities

Clone performance testing enables identification of optimization opportunities:

- **Network Efficiency**: Measure impact of packfile fetching strategies
- **Filter Performance**: Validate path filtering implementation efficiency
- **Memory Usage**: Monitor memory consumption during large repository clones
- **Concurrent Operations**: Test performance of parallel blob fetching
- **Incremental Improvements**: Track performance trends over time

These real-world tests complement the synthetic benchmarks by providing practical performance data against production repositories, ensuring nanogit's Clone functionality meets real-world performance requirements.