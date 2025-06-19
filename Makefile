.PHONY: generate
generate:
	COUNTERFEITER_NO_GENERATE_WARNING=true go generate ./...

.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports@v0.27.0 -w .

.PHONY: lint

lint-staticcheck: 
	go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run

.PHONY: test-unit
test-unit:
	go test -race -parallel 6 --short ./...

.PHONY: test-integration
test-integration:
	go run github.com/onsi/ginkgo/v2/ginkgo --race --randomize-all --randomize-suites --fail-on-pending -p -focus "Integration" ./tests

.PHONY: test-providers
test-providers:
	go test -race -run TestProviders 

test: test-unit test-integration

test-coverage:
	@echo "Running unit tests with coverage..."
	go run github.com/onsi/ginkgo/v2/ginkgo \
		--p \
		--race \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--cover \
		--coverpkg=./... \
		--coverprofile=unit.cov \
		./... \
		-- -test.short

	@echo "Running integration tests with coverage..."
	go run github.com/onsi/ginkgo/v2/ginkgo \
		--p \
		--race \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--cover \
		--coverpkg=./... \
		--coverprofile=integration.cov \
		./tests

	@echo "Merging coverage profiles..."
	@echo "mode: set" > coverage.txt
	@tail -n +2 unit.cov >> coverage.txt || true
	@tail -n +2 integration.cov >> coverage.txt || true
	@echo "Combined coverage written to coverage.txt"

test-coverage-html:
	go tool cover -html=coverage.txt

# Performance Testing Targets
.PHONY: test-perf test-perf-consistency test-perf-simple test-perf-nanogit test-perf-gogit test-perf-cli
.PHONY: test-perf-file-ops test-perf-compare test-perf-tree test-perf-bulk test-perf-all test-perf-setup
.PHONY: test-perf-small test-perf-medium test-perf-large test-perf-xlarge

# Setup performance test data (one-time setup)
test-perf-setup:
	@echo "Setting up performance test data..."
	cd tests/performance && go run ./cmd/generate_repo

# Run all performance tests with all clients
test-perf-all:
	@echo "Running all performance tests (this may take a while)..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 30m .

# Consistency tests - verify all clients produce identical results
test-perf-consistency:
	@echo "Running client consistency tests..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run "TestClientConsistency|TestSimpleClientConsistency" .

# Simple consistency tests only
test-perf-simple:
	@echo "Running simple client consistency tests..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 5m -run TestSimpleClientConsistency .

# Individual performance test suites
test-perf-file-ops:
	@echo "Running file operations performance tests..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 15m -run TestFileOperationsPerformance .

test-perf-compare:
	@echo "Running commit comparison performance tests..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestCompareCommitsPerformance .

test-perf-tree:
	@echo "Running tree listing performance tests..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestGetFlatTreePerformance .

test-perf-bulk:
	@echo "Running bulk operations performance tests..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 15m -run TestBulkOperationsPerformance .

# Client-specific tests (run consistency tests with specific client filtering)
test-perf-nanogit:
	@echo "Running performance tests for nanogit client only..."
	@echo "Note: This runs consistency tests which include nanogit comparisons"
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestSimpleClientConsistency . | grep -E "(nanogit|PASS|FAIL|RUN)"

test-perf-gogit:
	@echo "Running performance tests for go-git client only..."
	@echo "Note: This runs consistency tests which include go-git comparisons"
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestSimpleClientConsistency . | grep -E "(go-git|PASS|FAIL|RUN)"

test-perf-cli:
	@echo "Running performance tests for git-cli client only..."
	@echo "Note: This runs consistency tests which include git-cli comparisons"
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 10m -run TestSimpleClientConsistency . | grep -E "(git-cli|PASS|FAIL|RUN)"

# Repository size-specific tests (run all test types for specific repo sizes)
test-perf-small:
	@echo "Running performance tests for small repositories only..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 15m -run "TestFileOperationsPerformance.*small|TestCompareCommitsPerformance.*small|TestGetFlatTreePerformance.*small|TestBulkOperationsPerformance.*small" .

test-perf-medium:
	@echo "Running performance tests for medium repositories only..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 20m -run "TestFileOperationsPerformance.*medium|TestCompareCommitsPerformance.*medium|TestGetFlatTreePerformance.*medium|TestBulkOperationsPerformance.*medium" .

test-perf-large:
	@echo "Running performance tests for large repositories only..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 25m -run "TestFileOperationsPerformance.*large|TestCompareCommitsPerformance.*large|TestGetFlatTreePerformance.*large|TestBulkOperationsPerformance.*large" .

test-perf-xlarge:
	@echo "Running performance tests for xlarge repositories only..."
	cd tests/performance && RUN_PERFORMANCE_TESTS=true go test -v -timeout 30m -run "TestFileOperationsPerformance.*xlarge|TestCompareCommitsPerformance.*xlarge|TestGetFlatTreePerformance.*xlarge|TestBulkOperationsPerformance.*xlarge" .

# Full performance benchmark suite (combines all test types)
test-perf: test-perf-consistency test-perf-file-ops
	@echo "Core performance testing completed"