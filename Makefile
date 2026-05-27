# ============================================================
# issue2md - Makefile
# ============================================================

# Binary output directory
BIN_DIR := bin

# Binary names
CLI_BINARY := $(BIN_DIR)/issue2md
WEB_BINARY := $(BIN_DIR)/issue2md-web

# Docker
DOCKER_IMAGE := issue2md
DOCKER_TAG := latest

# ============================================================
# Targets
# ============================================================

.PHONY: build build-cli build-web test test-integration lint docker-build clean help

## build: Compile both CLI and Web binaries
build: build-cli build-web

## build-cli: Compile the CLI binary
build-cli:
	go build -o $(CLI_BINARY) ./cmd/issue2md/

## build-web: Compile the Web binary (requires cmd/issue2mdweb/main.go)
build-web:
	@if [ -z "$$(ls cmd/issue2mdweb/*.go 2>/dev/null)" ]; then \
		echo "SKIP: cmd/issue2mdweb/ has no Go files yet"; \
	else \
		go build -o $(WEB_BINARY) ./cmd/issue2mdweb/; \
	fi

## test: Run all unit tests
test:
	go test ./... -v -count=1

## test-integration: Run integration tests (requires GITHUB_TOKEN)
test-integration:
	go test ./... -v -count=1 -tags=integration

## lint: Run golangci-lint static analysis
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, falling back to go vet"; \
		go vet ./...; \
	fi

## docker-build: Build container image using Dockerfile
docker-build:
	DOCKER_BUILDKIT=0 docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## clean: Remove all build artifacts
clean:
	rm -rf $(BIN_DIR)

## help: Show this help message
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
