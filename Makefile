.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports@v0.27.0 -w .

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test: lint
	go test -race ./...