.PHONY: help build test lint fmt clean

help:
	@echo "UCP Server Makefile targets:"
	@echo "  make build      - Compile the binary"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run go vet"
	@echo "  make fmt       - Format code (goimports, go fmt)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make run        - Build and run the server"

build:
	go build -o bin/ucp-server ./cmd/ucp-server

test:
	go test -v ./...

lint:
	go vet ./...

fmt:
	goimports -w .
	go fmt ./...

clean:
	rm -rf bin/

run: build
	./bin/ucp-server

.DEFAULT_GOAL := help
