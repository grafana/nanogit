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
	go test -coverprofile=coverage.txt -covermode=atomic -race -parallel 6 $(shell go list ./... | grep -v './test')

.PHONY: test-integration
test-integration:
	go run github.com/onsi/ginkgo/v2/ginkgo -r --race --randomize-all --randomize-suites --fail-on-pending -p ./test

test: test-unit test-integration
