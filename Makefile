.PHONY: build test lint install clean run help

# Variables
BINARY_NAME=project-manager
GO=go
GOLANGCI_LINT=golangci-lint
GOFLAGS=-v

# Default target
all: test build

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

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