# 🐟 Fish-Speech-Go

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

High-performance Go wrapper for [Fish-Speech](https://github.com/fishaudio/fish-speech) - the state-of-the-art open-source Text-to-Speech system.

## ✨ Features

- **True Streaming** - Audio starts playing immediately, not after full generation
- **High Concurrency** - Handle 10,000+ concurrent connections
- **Low Memory** - ~15MB vs ~200MB for Python server
- **Fast Startup** - <100ms vs 3-5 seconds
- **Simple Deployment** - Single binary or Docker

## 🚀 Quick Start

### Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/your-username/fish-speech-go.git
cd fish-speech-go

# Start everything (downloads models automatically)
cd docker
docker compose up -d

# Test it
curl http://localhost:8080/v1/health
```

### From Source

```bash
# Build
cd go
go build -o bin/fish-server ./cmd/fish-server

# Run (requires Python backend on port 8081)
./bin/fish-server --backend http://localhost:8081
```

## 📖 Usage

### Generate Speech

```bash
# Using curl
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!"}' \
  -o output.wav

# Using fish-tts CLI
fish-tts "Hello, world!" -o output.wav

# With streaming
fish-tts "Hello, world!" --stream -o output.wav
```

### Voice Cloning

```bash
fish-tts "Hello in cloned voice" \
  --reference voice-sample.wav \
  --reference-text "Text spoken in the sample" \
  -o cloned.wav
```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| FISH_LISTEN | 0.0.0.0:8080 | Server listen address |
| FISH_BACKEND | http://127.0.0.1:8081 | Python backend URL |
| FISH_API_KEY | (none) | API key for authentication |
| FISH_LOG_LEVEL | info | Log level (debug, info, warn, error) |

### CLI Flags

```bash
fish-server \
  --listen 0.0.0.0:8080 \
  --backend http://localhost:8081 \
  --api-key your-secret-key \
  --log-level debug
```

## 🧪 Testing

```bash
# Unit tests
cd go && go test ./...

# Integration tests (requires Docker)
./scripts/run-integration-tests.ps1  # Windows
./scripts/run-integration-tests.sh   # Linux/Mac
```

## 📚 Documentation

- Architecture - System design and components
- API Reference - Complete API specification
- Compatibility Tests - Test cases

## 📄 License

Apache 2.0 - See LICENSE for details.

## 🙏 Acknowledgments

- fishaudio/fish-speech - The amazing TTS engine
