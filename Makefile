.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports@v0.27.0 -w .

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run

.PHONY: test
test: lint
	go test -cover -race -parallel 4 ./...
