# Contributing to Fish-Speech-Go

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## 🐛 Reporting Bugs

1. Check existing issues to avoid duplicates
2. Use the bug report template
3. Include:
   - OS and Docker version
   - GPU model and CUDA version
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs

## 💡 Feature Requests

1. Check existing issues/discussions
2. Describe the use case
3. Explain why it benefits others

## 🔧 Development Setup

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- NVIDIA GPU with CUDA (for testing)

### Local Development

```bash
# Clone the repo
git clone https://github.com/fish-speech-go/fish-speech-go.git
cd fish-speech-go

# Run Go tests
cd go
go test ./...

# Build locally
go build -o bin/server ./cmd/server

# Run with Docker
cd ../docker
cp .env.example .env
# Edit .env with your HF_TOKEN
docker compose up -d
```

## 📝 Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `cd go && go test ./...`
5. Commit with clear messages
6. Push to your fork
7. Open a Pull Request

### PR Guidelines

- One feature/fix per PR
- Update documentation if needed
- Add tests for new features
- Follow existing code style
- Keep commits atomic and well-described

## 🧪 Testing

### Go Tests

```bash
cd go
go test ./...           # Run all tests
go test ./... -v        # Verbose output
go test ./... -cover    # With coverage
```

### Integration Tests

```bash
cd docker
docker compose up -d
curl http://localhost:8080/v1/health
```

## 📁 Code Structure

```
go/
├── cmd/server/         # Application entrypoint
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── config/         # Configuration loading
│   └── tts/            # TTS backend client
└── go.mod
```

## 🎨 Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` before committing
- Keep functions small and focused
- Add comments for exported functions

## 📖 Documentation

- Update README.md for user-facing changes
- Add godoc comments for public APIs
- Include examples for new features

## 🏷️ Versioning

We use [Semantic Versioning](https://semver.org/):
- MAJOR: Breaking API changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

## 💬 Questions?

- Open a GitHub Discussion
- Check existing issues

Thank you for contributing! 🎉
