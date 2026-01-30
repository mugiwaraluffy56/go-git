.PHONY: build run clean test install lint help

# Binary name
BINARY_NAME=gogit
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-w -s"

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gogit

## run: Run the application
run:
	$(GORUN) ./cmd/gogit $(ARGS)

## test: Run tests
test:
	$(GOTEST) -v -race ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## clean: Clean build files
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## install: Install binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/gogit

## fmt: Format code
fmt:
	$(GOFMT) ./...

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

## tidy: Tidy dependencies
tidy:
	$(GOMOD) tidy

## deps: Download dependencies
deps:
	$(GOMOD) download

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Demo targets for testing gogit
.PHONY: demo demo-clean

## demo: Run a demo of gogit
demo: build
	@echo "=== GoGit Demo ==="
	@mkdir -p /tmp/gogit-demo
	@cd /tmp/gogit-demo && rm -rf .gogit
	@echo "1. Initializing repository..."
	@cd /tmp/gogit-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) init
	@echo ""
	@echo "2. Creating test file..."
	@echo "Hello, GoGit!" > /tmp/gogit-demo/hello.txt
	@echo ""
	@echo "3. Adding file..."
	@cd /tmp/gogit-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) add hello.txt
	@echo ""
	@echo "4. Checking status..."
	@cd /tmp/gogit-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) status
	@echo ""
	@echo "5. Committing..."
	@cd /tmp/gogit-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) commit -m "Initial commit"
	@echo ""
	@echo "6. Viewing log..."
	@cd /tmp/gogit-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) log
	@echo ""
	@echo "=== Demo Complete ==="

## demo-clean: Clean demo directory
demo-clean:
	rm -rf /tmp/gogit-demo
