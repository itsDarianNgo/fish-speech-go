# Fish-Speech-Go Makefile

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: all build build-server build-tts build-ctl test clean install docker-build docker-up docker-down docker-logs run run-dev help test-coverage integration-test

all: build

# =============================================================================
# Build
# =============================================================================

build: build-server build-tts build-ctl

build-server:
	go build $(LDFLAGS) -o bin/fish-server ./cmd/fish-server

build-tts:
	go build $(LDFLAGS) -o bin/fish-tts ./cmd/fish-tts

build-ctl:
	go build $(LDFLAGS) -o bin/fish-ctl ./cmd/fish-ctl

# =============================================================================
# Test
# =============================================================================

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

integration-test:
	cd ../scripts && ./run-integration-tests.sh

# =============================================================================
# Docker
# =============================================================================

docker-build:
	cd ../docker && docker compose build

docker-up:
	cd ../docker && docker compose up -d

docker-down:
	cd ../docker && docker compose down

docker-logs:
	cd ../docker && docker compose logs -f

# =============================================================================
# Development
# =============================================================================

run:
	go run ./cmd/fish-server

run-dev:
	go run ./cmd/fish-server --log-format text --log-level debug

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

install: build
	cp bin/fish-server $(GOPATH)/bin/
	cp bin/fish-tts $(GOPATH)/bin/
	cp bin/fish-ctl $(GOPATH)/bin/

# =============================================================================
# Help
# =============================================================================

help:
	@echo "Fish-Speech-Go Build System"
	@echo ""
	@echo "Build targets:"
	@echo "  build            Build all binaries"
	@echo "  build-server     Build fish-server"
	@echo "  build-tts        Build fish-tts"
	@echo "  build-ctl        Build fish-ctl"
	@echo ""
	@echo "Test targets:"
	@echo "  test             Run unit tests"
	@echo "  test-coverage    Run tests with coverage"
	@echo "  integration-test Run integration tests (requires Docker)"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build     Build Docker images"
	@echo "  docker-up        Start Docker services"
	@echo "  docker-down      Stop Docker services"
	@echo "  docker-logs      View Docker logs"
	@echo ""
	@echo "Development:"
	@echo "  run              Run server with defaults"
	@echo "  run-dev          Run server with debug logging"
	@echo "  clean            Remove build artifacts"
	@echo "  install          Install binaries to GOPATH"
