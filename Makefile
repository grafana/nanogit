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

# Performance Testing 
# For performance tests, use the dedicated Makefile in tests/performance/
# Example: cd tests/performance && make test-perf-all
.PHONY: test-perf
test-perf:
	@echo "Performance tests have been moved to tests/performance/Makefile"
	@echo "Run: cd tests/performance && make test-perf-all"
	@echo "Or see: cd tests/performance && make help"
