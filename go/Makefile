# Fish-Speech-Go Makefile

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: all build build-server build-tts build-ctl test clean install run run-dev docker-build help test-coverage

all: build

build: build-server build-tts build-ctl

build-server:
go build $(LDFLAGS) -o bin/fish-server ./cmd/fish-server

build-tts:
go build $(LDFLAGS) -o bin/fish-tts ./cmd/fish-tts

build-ctl:
go build $(LDFLAGS) -o bin/fish-ctl ./cmd/fish-ctl

test:
go test -v ./...

test-coverage:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

clean:
rm -rf bin/
rm -f coverage.out coverage.html

install: build
cp bin/fish-server $(GOPATH)/bin/
cp bin/fish-tts $(GOPATH)/bin/
cp bin/fish-ctl $(GOPATH)/bin/

# Development helpers
run:
go run ./cmd/fish-server

run-dev:
go run ./cmd/fish-server --log-format text --log-level debug

# Docker targets (for Phase 6)
docker-build:
docker build -f ../docker/Dockerfile.server -t fish-speech-go-server ..

# Help
help:
@echo "Available targets:"
@echo "  build        - Build all binaries"
@echo "  build-server - Build fish-server only"
@echo "  build-tts    - Build fish-tts only"
@echo "  build-ctl    - Build fish-ctl only"
@echo "  test         - Run tests"
@echo "  clean        - Remove build artifacts"
@echo "  install      - Install binaries to GOPATH/bin"
@echo "  run          - Run server with default settings"
@echo "  run-dev      - Run server with dev settings (text logs, debug level)"
