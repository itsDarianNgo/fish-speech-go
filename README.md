# 🐟 Fish-Speech-Go

A high-performance, OpenAI-compatible API server for [Fish-Speech](https://github.com/fishaudio/fish-speech) text-to-speech.

**Run state-of-the-art TTS locally with a familiar API.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](docker/)

## ✨ Why Fish-Speech-Go?

| Feature | OpenAI TTS | Fish-Speech-Go |
|---------|-----------|----------------|
| **Cost** | $15/1M characters | Free (self-hosted) |
| **Privacy** | Data sent to cloud | 100% local |
| **Rate Limits** | Yes | No |
| **Offline** | No | Yes |
| **Voice Cloning** | No | Yes |
| **API Compatibility** | - | OpenAI-compatible |

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- NVIDIA GPU with CUDA support
- [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
- Hugging Face account (free)

### 1. Clone & Configure

```bash
git clone https://github.com/fish-speech-go/fish-speech-go.git
cd fish-speech-go/docker
cp .env.example .env
```

Edit `.env` and add your Hugging Face token:
```env
HF_TOKEN=hf_your_token_here
```

> 📝 **Get your token:** https://huggingface.co/settings/tokens
>
> ⚠️ **Accept the license:** https://huggingface.co/fishaudio/openaudio-s1-mini

### 2. Start Services

```bash
docker compose up -d
```

First run downloads models (~2GB) - takes a few minutes.

### 3. Verify

```bash
curl http://localhost:8080/v1/health
```

## 📖 API Reference

### OpenAI-Compatible Endpoints

#### Generate Speech

```bash
# POST /v1/audio/speech
curl -X POST http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{
    "model": "fish-speech",
    "voice": "default",
    "input": "Hello, world!"
  }' \
  --output speech.wav
```

#### Alternative TTS Endpoint

```bash
# POST /v1/tts
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!"}' \
  --output speech.wav
```

#### Health Check

```bash
# GET /v1/health
curl http://localhost:8080/v1/health
# Response: {"status": "ok"}
```

#### List Voices

```bash
# GET /v1/audio/voices
curl http://localhost:8080/v1/audio/voices
```

### Request Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `input` / `text` | string | Text to convert to speech | required |
| `model` | string | Model name | `fish-speech` |
| `voice` | string | Voice ID | `default` |
| `response_format` | string | Output format (wav, mp3) | `wav` |

## 🔧 Integration Examples

### Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"  # Required by SDK but not validated
)

response = client.audio.speech.create(
    model="fish-speech",
    voice="default",
    input="Hello from Python!"
)

response.stream_to_file("output.mp3")
```

### JavaScript/TypeScript

```typescript
import OpenAI from 'openai';
import fs from 'fs';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'not-needed',
});

const response = await client.audio.speech.create({
  model: 'fish-speech',
  voice: 'default',
  input: 'Hello from JavaScript!',
});

const buffer = Buffer.from(await response.arrayBuffer());
fs.writeFileSync('output.mp3', buffer);
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "os"
)

func main() {
    payload := map[string]string{
        "input": "Hello from Go!",
        "model": "fish-speech",
        "voice": "default",
    }
    body, _ := json.Marshal(payload)

    resp, _ := http.Post(
        "http://localhost:8080/v1/audio/speech",
        "application/json",
        bytes.NewReader(body),
    )
    defer resp.Body.Close()

    out, _ := os.Create("output.wav")
    defer out.Close()
    io.Copy(out, resp.Body)
}
```

### cURL

```bash
curl -X POST http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{
    "model": "fish-speech",
    "voice": "default",
    "input": "Hello from the command line!"
  }' \
  --output speech.mp3
```

## ⚙️ Configuration

Configure via environment variables in `docker/.env`:

| Variable | Description | Default |
|----------|-------------|---------|
| `HF_TOKEN` | Hugging Face API token | **required** |
| `SERVER_PORT` | Go server port | `8080` |
| `API_KEY` | Optional API key for authentication | (none) |
| `LOG_LEVEL` | Log level: debug, info, warn, error | `info` |
| `LOG_FORMAT` | Log format: json, text | `json` |
| `MAX_TEXT_LENGTH` | Max input length (0 = unlimited) | `0` |

## 🏗️ Architecture

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Your App       │────▶│   Go Server      │────▶│   Fish-Speech    │
│   (Client)       │◀────│   (Port 8080)    │◀────│   (Port 8081)    │
└──────────────────┘     └──────────────────┘     └──────────────────┘
      HTTP/JSON            Fast API Layer           ML Inference
                           OpenAI-Compatible        GPU (CUDA)
```

**Why this design?**
- **Go** handles HTTP, routing, validation, auth, logging (what Go does best)
- **Python** handles ML inference, GPU operations (what Python does best)
- **Result:** Fast, scalable, production-ready TTS

## 📁 Project Structure

```
fish-speech-go/
├── go/                      # Go API server
│   ├── cmd/fish-server/     # Main entrypoint
│   ├── cmd/fish-tts/        # CLI client for TTS
│   ├── cmd/fish-ctl/        # Management CLI
│   ├── internal/            # Core packages (api, backend, config, schema)
│   └── go.mod
├── docker/
│   ├── Dockerfile.server    # Go server image
│   ├── Dockerfile.inference # Fish-Speech image
│   ├── docker-compose.yml
│   ├── .env.example         # Example configuration
│   └── .env                 # Your local config (gitignored)
├── docs/                    # Additional documentation
├── scripts/                 # Helper scripts
├── LICENSE
└── README.md
```

## 🛠️ Development

### Run Go Tests

```bash
cd go
go test ./...
```

### Build Go Binary

```bash
cd go
go build -o bin/server ./cmd/fish-server
```

### Build Docker Images

```bash
cd docker
docker compose build
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📜 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- [Fish-Speech](https://github.com/fishaudio/fish-speech) - The amazing TTS model
- [fishaudio](https://github.com/fishaudio) - Model creators

**⭐ Star this repo if you find it useful!**
