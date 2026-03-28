build:
    go build ./...

test:
    go test ./...

lint:
    golangci-lint run

format:
    gofmt -w .

check: format lint test
