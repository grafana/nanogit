.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports@v0.27.0 -w .

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run

.PHONY: test-unit
test-unit:
	go test -coverprofile=coverage.txt -covermode=atomic -race -parallel 6 ./...

.PHONY: test-integration
test-integration:
	go test -tags=integration -cover -race -parallel 6 ./client/integration/...

test: test-unit test-integration
