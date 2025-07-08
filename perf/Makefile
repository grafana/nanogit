# Performance Testing Makefile
# Run from tests/performance directory

.PHONY: help test-perf test-perf-consistency test-perf-simple test-perf-nanogit test-perf-gogit test-perf-cli
.PHONY: test-perf-file-ops test-perf-compare test-perf-tree test-perf-bulk test-perf-all test-perf-setup
.PHONY: test-perf-small test-perf-medium test-perf-large test-perf-xlarge
.PHONY: test-perf-tree-small test-perf-tree-medium test-perf-tree-large test-perf-tree-xlarge
.PHONY: test-perf-bulk-small test-perf-bulk-medium test-perf-bulk-large test-perf-bulk-xlarge
.PHONY: test-perf-compare-small test-perf-compare-medium test-perf-compare-large test-perf-compare-xlarge
.PHONY: profile-cpu profile-mem profile-all profile-compare profile-clean profile-baseline

# Default target
help:
	@echo "Performance Testing Targets:"
	@echo ""
	@echo "Setup:"
	@echo "  test-perf-setup     - Generate test repository data (one-time setup)"
	@echo ""
	@echo "Main Test Suites:"
	@echo "  test-perf-all       - Run all performance tests"
	@echo "  test-perf           - Run core performance tests (consistency + file ops)"
	@echo "  test-perf-consistency - Client consistency tests"
	@echo "  test-perf-simple    - Simple consistency tests only"
	@echo ""
	@echo "Individual Test Types:"
	@echo "  test-perf-file-ops  - File operations performance"
	@echo "  test-perf-compare   - Commit comparison performance"
	@echo "  test-perf-tree      - Tree listing performance"
	@echo "  test-perf-bulk      - Bulk operations performance"
	@echo ""
	@echo "Client-Specific:"
	@echo "  test-perf-nanogit   - Test nanogit client only"
	@echo "  test-perf-gogit     - Test go-git client only"
	@echo "  test-perf-cli       - Test git-cli client only"
	@echo ""
	@echo "Repository Size-Specific:"
	@echo "  test-perf-small     - All tests on small repos"
	@echo "  test-perf-medium    - All tests on medium repos"
	@echo "  test-perf-large     - All tests on large repos"
	@echo "  test-perf-xlarge    - All tests on xlarge repos"
	@echo ""
	@echo "Test Type + Size Combinations:"
	@echo "  test-perf-tree-{small,medium,large,xlarge}     - FlatTree by size"
	@echo "  test-perf-bulk-{small,medium}                  - Bulk ops (small/medium only)"
	@echo "  test-perf-compare-{small,medium}               - Compare commits (small/medium only)"
	@echo ""
	@echo "Profiling Targets (nanogit-only, no consistency tests):"
	@echo "  profile-baseline    - Create baseline profiles (run before optimizations)"
	@echo "  profile-cpu         - Generate CPU profile for file operations"
	@echo "  profile-mem         - Generate memory profile for file operations"
	@echo "  profile-all         - Generate both CPU and memory profiles"
	@echo "  profile-all-tree    - Generate profiles for tree operations"
	@echo "  profile-all-commit  - Generate profiles for commit operations"
	@echo "  profile-bulk        - Generate profiles for bulk operations"
	@echo "  profile-nanogit-only - Generate profiles for all nanogit tests"
	@echo "  profile-compare     - Compare profiles with baseline"
	@echo "  profile-clean       - Clean up profile files"
	@echo ""
	@echo "Note: All profiling targets use NANOGIT_PROFILE_MODE=true to run nanogit-only"

# Setup performance test data (one-time setup)
test-perf-setup:
	@echo "Setting up performance test data..."
	go run ./cmd/generate_repo

# Run all performance tests with all clients
test-perf-all:
	@echo "Running all performance tests (this may take a while)..."
	RUN_PERFORMANCE_TESTS=true go test -v -timeout 30m -run "Performance" .

# Consistency tests - verify all clients produce identical results
test-perf-consistency:
	@echo "Running client consistency tests..."
	RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run "TestClientConsistency|TestSimpleClientConsistency" .

# Simple consistency tests only
test-perf-simple:
	@echo "Running simple client consistency tests..."
	RUN_PERFORMANCE_TESTS=true go test -v -timeout 5m -run TestSimpleClientConsistency .

# Individual performance test suites
test-perf-file-ops:
	@echo "Running file operations performance tests..."
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run TestFileOperationsPerformance .

test-perf-compare:
	@echo "Running commit comparison performance tests..."
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run TestCompareCommitsPerformance .

test-perf-tree:
	@echo "Running tree listing performance tests..."
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run TestGetFlatTreePerformance .

test-perf-bulk:
	@echo "Running bulk operations performance tests..."
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run TestBulkOperationsPerformance .

# Client-specific tests (run consistency tests with specific client filtering)
test-perf-nanogit:
	@echo "Running performance tests for nanogit client only..."
	@echo "Note: This runs consistency tests which include nanogit comparisons"
	RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestSimpleClientConsistency . | grep -E "(nanogit|PASS|FAIL|RUN)"

test-perf-gogit:
	@echo "Running performance tests for go-git client only..."
	@echo "Note: This runs consistency tests which include go-git comparisons"
	RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestSimpleClientConsistency . | grep -E "(go-git|PASS|FAIL|RUN)"

test-perf-cli:
	@echo "Running performance tests for git-cli client only..."
	@echo "Note: This runs consistency tests which include git-cli comparisons"
	RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestSimpleClientConsistency . | grep -E "(git-cli|PASS|FAIL|RUN)"

# Repository size-specific tests (run all test types for specific repo sizes)
test-perf-small:
	@echo "Running performance tests for small repositories only..."
	PERF_TEST_REPOS=small RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run "TestFileOperationsPerformance|TestCompareCommitsPerformance|TestGetFlatTreePerformance|TestBulkOperationsPerformance" .

test-perf-medium:
	@echo "Running performance tests for medium repositories only..."
	PERF_TEST_REPOS=medium RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 20m -run "TestFileOperationsPerformance|TestCompareCommitsPerformance|TestGetFlatTreePerformance|TestBulkOperationsPerformance" .

test-perf-large:
	@echo "Running performance tests for large repositories only..."
	PERF_TEST_REPOS=large RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 25m -run "TestFileOperationsPerformance|TestCompareCommitsPerformance|TestGetFlatTreePerformance|TestBulkOperationsPerformance" .

test-perf-xlarge:
	@echo "Running performance tests for xlarge repositories only..."
	PERF_TEST_REPOS=xlarge RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 30m -run "TestFileOperationsPerformance|TestCompareCommitsPerformance|TestGetFlatTreePerformance|TestBulkOperationsPerformance" .

# Repository size-specific FlatTree benchmarks
test-perf-tree-small:
	@echo "Running FlatTree performance tests for small repositories..."
	PERF_TEST_REPOS=small RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestGetFlatTreePerformance/small_tree" .

test-perf-tree-medium:
	@echo "Running FlatTree performance tests for medium repositories..."
	PERF_TEST_REPOS=medium RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestGetFlatTreePerformance/medium_tree" .

test-perf-tree-large:
	@echo "Running FlatTree performance tests for large repositories..."
	PERF_TEST_REPOS=large RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run "TestGetFlatTreePerformance/large_tree" .

test-perf-tree-xlarge:
	@echo "Running FlatTree performance tests for xlarge repositories..."
	PERF_TEST_REPOS=xlarge RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 20m -run "TestGetFlatTreePerformance/xlarge_tree" .

# Repository size-specific Bulk Operations benchmarks
test-perf-bulk-small:
	@echo "Running Bulk Operations performance tests for small repositories..."
	PERF_TEST_REPOS=small RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestBulkOperationsPerformance/bulk_.*_small" .

test-perf-bulk-medium:
	@echo "Running Bulk Operations performance tests for medium repositories..."
	PERF_TEST_REPOS=medium RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run "TestBulkOperationsPerformance/bulk_.*_medium" .

# Note: Bulk operations skip large and xlarge repositories to avoid excessive load
test-perf-bulk-large:
	@echo "Bulk Operations tests skip large repositories - use test-perf-bulk instead for full coverage"
	@echo "Available bulk operations: small, medium only"

test-perf-bulk-xlarge:
	@echo "Bulk Operations tests skip xlarge repositories - use test-perf-bulk instead for full coverage"
	@echo "Available bulk operations: small, medium only"

# Repository size-specific Compare Commits benchmarks
test-perf-compare-small:
	@echo "Running Compare Commits performance tests for small repositories..."
	PERF_TEST_REPOS=small RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestCompareCommitsPerformance/.*_small" .

test-perf-compare-medium:
	@echo "Running Compare Commits performance tests for medium repositories..."
	PERF_TEST_REPOS=medium RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestCompareCommitsPerformance/.*_medium" .

# Note: Compare Commits tests skip large and xlarge repositories to avoid excessive load  
test-perf-compare-large:
	@echo "Compare Commits tests skip large repositories - use test-perf-compare instead for full coverage"
	@echo "Available compare commits: small, medium only"

test-perf-compare-xlarge:
	@echo "Compare Commits tests skip xlarge repositories - use test-perf-compare instead for full coverage"
	@echo "Available compare commits: small, medium only"

# Full performance benchmark suite (combines all test types)
test-perf: test-perf-consistency test-perf-file-ops
	@echo "Core performance testing completed"

# Profiling targets (nanogit-only, skip consistency tests)
profile-cpu:
	@echo "Generating CPU profile for file operations (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestFileOperationsPerformance" -cpuprofile=profiles/cpu.prof .
	@echo "CPU profile saved to profiles/cpu.prof"
	@echo "Analyze with: go tool pprof profiles/cpu.prof"

profile-mem:
	@echo "Generating memory profile for file operations (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestFileOperationsPerformance" -memprofile=profiles/mem.prof .
	@echo "Memory profile saved to profiles/mem.prof"
	@echo "Analyze with: go tool pprof profiles/mem.prof"

profile-all:
	@echo "Generating both CPU and memory profiles (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestFileOperationsPerformance" -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof .
	@echo "Profiles saved to profiles/"
	@echo "Analyze CPU: go tool pprof profiles/cpu.prof"
	@echo "Analyze Memory: go tool pprof profiles/mem.prof"

profile-all-tree:
	@echo "Generating both CPU and memory profiles for tree operations (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestGetFlatTreePerformance" -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof .
	@echo "Profiles saved to profiles/"
	@echo "Analyze CPU: go tool pprof profiles/cpu.prof"
	@echo "Analyze Memory: go tool pprof profiles/mem.prof"

profile-all-commit:
	@echo "Generating both CPU and memory profiles for commit operations (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestCompareCommitsPerformance" -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof .
	@echo "Profiles saved to profiles/"
	@echo "Analyze CPU: go tool pprof profiles/cpu.prof"
	@echo "Analyze Memory: go tool pprof profiles/mem.prof"
profile-compare:
	@echo "Comparing profiles with baseline..."
	@if [ ! -f profiles/baseline_cpu.prof ]; then \
		echo "No baseline found. Run 'make profile-baseline' first."; \
		exit 1; \
	fi
	@if [ ! -f profiles/cpu.prof ]; then \
		echo "No current profile found. Run 'make profile-cpu' first."; \
		exit 1; \
	fi
	@echo "CPU comparison (baseline vs current):"
	go tool pprof -base=profiles/baseline_cpu.prof profiles/cpu.prof -top
	@echo ""
	@echo "Memory comparison (baseline vs current):"
	@if [ -f profiles/baseline_mem.prof ] && [ -f profiles/mem.prof ]; then \
		go tool pprof -base=profiles/baseline_mem.prof profiles/mem.prof -top; \
	else \
		echo "Memory profiles not available for comparison"; \
	fi

profile-baseline:
	@echo "Creating baseline profiles from all operations (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run "TestFileOperationsPerformance|TestGetFlatTreePerformance|TestCompareCommitsPerformance|TestBulkOperationsPerformance" -cpuprofile=profiles/baseline_cpu.prof -memprofile=profiles/baseline_mem.prof .
	@echo "Baseline profiles saved to profiles/baseline_*.prof"

profile-clean:
	@echo "Cleaning up profile files..."
	@rm -rf profiles/
	@echo "Profile files cleaned"

# Additional nanogit-only profiling targets
profile-bulk:
	@echo "Generating profiles for bulk operations (nanogit only)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 10m -run "TestBulkOperationsPerformance" -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof .
	@echo "Profiles saved to profiles/"
	@echo "Analyze CPU: go tool pprof profiles/cpu.prof"
	@echo "Analyze Memory: go tool pprof profiles/mem.prof"

profile-nanogit-only:
	@echo "Generating profiles for all nanogit tests (no consistency tests)..."
	@mkdir -p profiles
	RUN_PERFORMANCE_TESTS=true NANOGIT_PROFILE_MODE=true go test -v -timeout 15m -run "TestFileOperationsPerformance|TestGetFlatTreePerformance|TestCompareCommitsPerformance|TestBulkOperationsPerformance" -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof .
	@echo "Profiles saved to profiles/"
	@echo "Analyze CPU: go tool pprof profiles/cpu.prof"
	@echo "Analyze Memory: go tool pprof profiles/mem.prof"