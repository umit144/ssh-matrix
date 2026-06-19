VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean

build:
	go build -ldflags="$(LDFLAGS)" -o ssh-matrix .

test:
	go test ./... -race
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out | grep total

lint:
	golangci-lint run ./...

clean:
	rm -f ssh-matrix ssh-matrix-* coverage.out
