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
- **Clone Performance Tests**: Real-world clone operations against live repositories (e.g., grafana/grafana)
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
| `test-perf-clone` | Clone performance | ~5 min |
| `test-perf-small` | Small repos only | ~3 min |
| `test-perf-all` | Everything | ~25 min |

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

## Clone Performance Tests

The clone performance tests validate real-world clone operations against live repositories:

- **TestClonePerformanceSmall**: Filtered clone of grafana/grafana (~150 files)
- **TestClonePerformanceLarge**: Larger filtered clone (~1000+ files)  
- **TestCloneConsistency**: Multiple runs to verify consistent behavior

These tests:
- Use live GitHub repositories (no Docker required)
- Test the clone fix for missing tree objects
- Measure throughput, success rates, and reliability
- Validate file filtering and progress tracking

```bash
cd perf
make test-perf-clone
```

**Note**: Clone tests require internet connectivity and may be affected by GitHub rate limiting.

## Manual Execution

```bash
cd perf
export RUN_PERFORMANCE_TESTS=true

# Run specific tests
go test -v -run TestFileOperationsPerformance .
go test -v -run TestClonePerformance .

# Run benchmarks
go test -bench=. .
```

See `make help` for complete target list.