.PHONY: generate
generate:
	COUNTERFEITER_NO_GENERATE_WARNING=true go generate ./...

.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports@v0.27.0 -w .
	go fmt ./...

.PHONY: lint

lint-staticcheck: 
	go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6 run

.PHONY: test-unit
test-unit:
	go test -race -parallel 6 --short ./...

.PHONY: test-integration
test-integration:
	go run github.com/onsi/ginkgo/v2/ginkgo --race --randomize-all --randomize-suites --fail-on-pending -p -focus "Integration" ./tests

.PHONY: test-providers
test-providers:
	go test -race -run TestProviders ./tests

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
		. ./log ./mocks ./options ./protocol ./storage \
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

	@echo "Running CLI unit tests with coverage..."
	cd cli && GOWORK=off go test -race -coverprofile=../cli-unit.cov -covermode=atomic ./internal/...

	@echo "Running CLI integration tests with coverage..."
	cd cli && GOWORK=off go test -race -coverprofile=../cli-integration.cov -covermode=atomic -run TestCLIIntegration .

	@echo "Merging coverage profiles..."
	@echo "mode: set" > coverage.txt
	@tail -n +2 unit.cov >> coverage.txt || true
	@tail -n +2 integration.cov >> coverage.txt || true
	@tail -n +2 cli-unit.cov >> coverage.txt || true
	@tail -n +2 cli-integration.cov >> coverage.txt || true
	@echo "Combined coverage written to coverage.txt"

test-coverage-html:
	go tool cover -html=coverage.txt

# Performance Testing
# For performance tests, use the dedicated Makefile in perf/
# Example: cd perf && make test-perf-all
.PHONY: test-perf
test-perf:
	@echo "Performance tests have been moved to perf/Makefile"
	@echo "Run: cd perf && make test-perf-all"
	@echo "Or see: cd perf && make help"

# Clone Performance Testing
# Separate from other perf tests - tests real-world clone performance
.PHONY: test-clone-perf
test-clone-perf:
	@echo "Running clone performance tests..."
	cd perf && make test-clone-perf

# Documentation
.PHONY: docs-install
docs-install:
	@echo "Installing documentation dependencies..."
	@command -v npm >/dev/null 2>&1 || (echo "Error: npm is required but not installed" && exit 1)
	npm install

.PHONY: docs-prepare
docs-prepare:
	@echo "Preparing documentation files..."
	./scripts/prepare-docs.sh

.PHONY: docs-serve
docs-serve: docs-prepare
	@echo "Serving documentation at http://localhost:5173"
	npm run docs:dev

.PHONY: docs-build
docs-build: docs-prepare
	@echo "Building documentation..."
	npm run docs:build

.PHONY: docs-preview
docs-preview: docs-build
	@echo "Previewing built documentation..."
	npm run docs:preview

.PHONY: docs
docs: docs-serve

# CLI targets
.PHONY: cli-build
cli-build: ## Build the nanogit CLI
	@echo "Building nanogit CLI..."
	cd cli && GOWORK=off go build -o ../bin/nanogit .

.PHONY: cli-install
cli-install: ## Install the nanogit CLI
	@echo "Installing nanogit CLI..."
	cd cli && GOWORK=off go install .

.PHONY: cli-test
cli-test: ## Run CLI unit tests
	@echo "Testing nanogit CLI..."
	cd cli && GOWORK=off go test -race -v ./...

.PHONY: cli-lint
cli-lint: ## Lint CLI code
	@echo "Linting CLI code..."
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint not found. Install it from https://golangci-lint.run/usage/install/" && exit 1)
	cd cli && golangci-lint run

.PHONY: cli-fmt
cli-fmt: ## Format CLI code
	@echo "Formatting CLI code..."
	@which goimports > /dev/null || (echo "Error: goimports not found. Install it with: go install golang.org/x/tools/cmd/goimports@latest" && exit 1)
	cd cli && goimports -w .
	cd cli && go fmt ./...

.PHONY: cli
cli: cli-build ## Alias for cli-build
