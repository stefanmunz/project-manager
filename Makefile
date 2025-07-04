.PHONY: build test lint install clean run help install-tools

# Variables
BINARY_NAME=project-manager
GO=go
GOLANGCI_LINT=$(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)
GOFLAGS=-v
GOLANGCI_LINT_VERSION=v1.61.0

# Default target
all: test build

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "Tools installed successfully!"
	@echo ""
	@echo "Note: Make sure $(shell go env GOPATH)/bin is in your PATH"
	@echo "You can add it with: export PATH=$$PATH:$(shell go env GOPATH)/bin"

## build: Build the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

## test: Run tests
test:
	$(GO) test $(GOFLAGS) ./...

## test-race: Run tests with race detector
test-race:
	$(GO) test -race $(GOFLAGS) ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GO) test -cover $(GOFLAGS) ./...

## lint: Run linters
lint:
	$(GOLANGCI_LINT) run

## install: Install the binary
install:
	$(GO) install $(GOFLAGS) .

## clean: Clean build artifacts
clean:
	$(GO) clean
	rm -f $(BINARY_NAME)

## run: Run the application
run:
	$(GO) run .

## deps: Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

## fmt: Format code
fmt:
	$(GO) fmt ./...

## vet: Run go vet
vet:
	$(GO) vet ./...