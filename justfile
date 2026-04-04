generate:
    sqlc generate

build:
    go build -o nexus ./cmd/nexus/

start *args:
    go run ./cmd/nexus/main.go {{args}}

test:
    go test ./...

lint:
    golangci-lint run

format:
    gofmt -w .

check: format lint test
