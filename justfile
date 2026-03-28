build:
    go build ./...

start *args: build
    ./nexus {{args}}

test:
    go test ./...

lint:
    golangci-lint run

format:
    gofmt -w .

check: format lint test
