# 🏗️ Fish-Speech-Go Architecture

> Technical architecture documentation for the Fish-Speech-Go high-performance API server. 

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Design Principles](#design-principles)
3. [Component Architecture](#component-architecture)
4. [Upstream Compatibility](#upstream-compatibility)
5. [Data Flow](#data-flow)
6. [Streaming Architecture](#streaming-architecture)
7. [Concurrency Model](#concurrency-model)
8. [Backend Communication](#backend-communication)
9. [Configuration System](#configuration-system)
10. [Error Handling Strategy](#error-handling-strategy)
11.  [Deployment Architecture](#deployment-architecture)
12. [Security Considerations](#security-considerations)
13.  [Performance Characteristics](#performance-characteristics)

---

## System Overview

### What is Fish-Speech-Go?

Fish-Speech-Go is a high-performance Go API server that wraps the [Fish-Speech](https://github.com/fishaudio/fish-speech) Python inference backend. It acts as a reverse proxy with enhanced capabilities for streaming, concurrency, and deployment.

### Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              CLIENTS                                      │
│         (Web Apps, Mobile Apps, CLI Tools, Other Services)               │
└─────────────────────────────────┬────────────────────────────────────────┘
                                  │
                                  │ HTTP/HTTPS
                                  │ (JSON or MessagePack)
                                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                         GO API SERVER                                     │
│                       (fish-speech-go)                                    │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                        HTTP Layer                                   │  │
│  │   • Request parsing (JSON, MessagePack, multipart/form-data)       │  │
│  │   • Authentication (Bearer token)                                   │  │
│  │   • Request validation (matching upstream Pydantic schemas)        │  │
│  │   • CORS handling                                                   │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                     Streaming Layer                                 │  │
│  │   • True chunked transfer encoding                                  │  │
│  │   • WAV header generation for streaming                            │  │
│  │   • Audio chunk buffering and delivery                             │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                   Backend Client Layer                              │  │
│  │   • Connection pooling to Python backend                           │  │
│  │   • MessagePack serialization (ormsgpack compatible)               │  │
│  │   • Timeout and retry handling                                      │  │
│  └────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────┬────────────────────────────────────────┘
                                  │
                                  │ HTTP (localhost)
                                  │ MessagePack serialization
                                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      PYTHON INFERENCE BACKEND                             │
│                    (Upstream Fish-Speech Server)                          │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │   tools/api_server.py (Kui/ASGI framework)                         │  │
│  │   • ServeTTSRequest handling                                        │  │
│  │   • TTSInferenceEngine orchestration                               │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │   fish_speech/inference_engine/                                     │  │
│  │   • LLaMA text-to-semantic generation                              │  │
│  │   • DAC/VQGAN audio decoding                                       │  │
│  │   • Reference voice loading                                         │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │   GPU (CUDA)                                                        │  │
│  │   • Neural network inference                                        │  │
│  │   • ~1-2 seconds per request                                       │  │
│  └────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Design Principles

### 1. 100% API Compatibility

The Go server MUST be a drop-in replacement for the Python server. Any client that works with the upstream Fish-Speech API must work with our Go server without modification.

### 2. Preserve Upstream Behavior

- Same request/response schemas
- Same validation rules and error messages
- Same authentication mechanism
- Same endpoint paths

### 3.  Enhance, Don't Replace

- Python handles ML inference (proven, stable)
- Go handles HTTP serving (faster, more concurrent)
- Clear separation of responsibilities

### 4.  Spec-Driven Development

All code must be derived from the documented specifications, not ad-hoc implementation. 

---

## Component Architecture

### Package Structure

```
go/
├── cmd/
│   ├── fish-server/           # API server binary
│   │   └── main. go
│   ├── fish-tts/              # CLI tool
│   │   └── main.go
│   └── fish-ctl/              # Management CLI
│       └── main.go
│
├── internal/
│   ├── api/
│   │   ├── handlers. go        # HTTP request handlers
│   │   ├── middleware.go      # Auth, logging, CORS
│   │   ├── routes.go          # Route definitions
│   │   └── validation.go      # Request validation
│   │
│   ├── schema/                # ⚠️ CRITICAL: Must match upstream exactly
│   │   ├── tts.go             # ServeTTSRequest, ServeReferenceAudio
│   │   ├── vqgan.go           # VQGAN encode/decode schemas
│   │   ├── references.go      # Reference management schemas
│   │   └── responses.go       # Response schemas
│   │
│   ├── streaming/
│   │   ├── chunker.go         # Audio chunk management
│   │   ├── wav. go             # WAV header generation
│   │   └── buffer.go          # Ring buffer implementation
│   │
│   ├── backend/
│   │   ├── client.go          # Python backend HTTP client
│   │   ├── pool.go            # Connection pooling
│   │   └── msgpack.go         # MessagePack encoding
│   │
│   ├── config/
│   │   └── config.go          # Configuration management
│   │
│   └── observability/
│       ├── logging.go         # Structured logging
│       └── metrics.go         # Prometheus metrics
│
└── pkg/
    └── client/                # Public Go SDK
        ├── client.go
        └── types. go
```

---

## Upstream Compatibility

### Source of Truth

All schemas are derived from the upstream Fish-Speech repository:
- **Repository:** https://github.com/fishaudio/fish-speech
- **Commit:** 80db25e72628067aa23f35161f1bf6bffbc2e554
- **Key Files:**
  - `fish_speech/utils/schema.py` - Pydantic request/response models
  - `tools/server/views.py` - API endpoint definitions
  - `tools/server/api_utils.py` - Content-type handling
  - `tools/api_server.py` - Server configuration

### Schema Mapping

#### ServeTTSRequest (Primary)

| Python Field | Python Type | Go Field | Go Type | Default | Validation |
|--------------|-------------|----------|---------|---------|------------|
| `text` | `str` | `Text` | `string` | required | non-empty |
| `chunk_length` | `int` | `ChunkLength` | `int` | `200` | 100-300 |
| `format` | `Literal["wav","pcm","mp3"]` | `Format` | `string` | `"wav"` | enum |
| `references` | `list[ServeReferenceAudio]` | `References` | `[]ServeReferenceAudio` | `[]` | - |
| `reference_id` | `str \| None` | `ReferenceID` | `*string` | `nil` | - |
| `seed` | `int \| None` | `Seed` | `*int` | `nil` | - |
| `use_memory_cache` | `Literal["on","off"]` | `UseMemoryCache` | `string` | `"off"` | enum |
| `normalize` | `bool` | `Normalize` | `bool` | `true` | - |
| `streaming` | `bool` | `Streaming` | `bool` | `false` | - |
| `max_new_tokens` | `int` | `MaxNewTokens` | `int` | `1024` | - |
| `top_p` | `float` | `TopP` | `float64` | `0. 8` | 0.1-1.0 |
| `repetition_penalty` | `float` | `RepetitionPenalty` | `float64` | `1.1` | 0.9-2. 0 |
| `temperature` | `float` | `Temperature` | `float64` | `0. 8` | 0.1-1. 0 |

#### ServeReferenceAudio

| Python Field | Python Type | Go Field | Go Type | Notes |
|--------------|-------------|----------|---------|-------|
| `audio` | `bytes` | `Audio` | `[]byte` | Base64 decoded if string |
| `text` | `str` | `Text` | `string` | Transcript of audio |

### Content-Type Handling

The Go server MUST accept all content types that upstream accepts:

```go
func parseRequestBody(r *http.Request, v interface{}) error {
    contentType := r.Header.Get("Content-Type")
    
    switch {
    case strings.HasPrefix(contentType, "application/msgpack"):
        // Decode MessagePack (ormsgpack compatible)
        return msgpack.NewDecoder(r. Body).Decode(v)
        
    case strings. HasPrefix(contentType, "application/json"):
        // Decode JSON
        return json.NewDecoder(r.Body). Decode(v)
        
    case strings.HasPrefix(contentType, "multipart/form-data"):
        // Handle multipart form data
        return parseMultipartForm(r, v)
        
    default:
        return ErrUnsupportedMediaType
    }
}
```

### Authentication Compatibility

Upstream uses Bearer token authentication:

```python
# From tools/api_server.py
async def verify(token: Annotated[str, Depends(bearer_auth)]):
    if token != self.args.api_key:
        raise HTTPException(401, None, "Invalid token")
```

Go implementation must match:

```go
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if apiKey == "" {
                next.ServeHTTP(w, r)
                return
            }
            
            auth := r.Header. Get("Authorization")
            if ! strings.HasPrefix(auth, "Bearer ") {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            token := strings.TrimPrefix(auth, "Bearer ")
            if token != apiKey {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Data Flow

### TTS Request Flow (Non-Streaming)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        NON-STREAMING REQUEST                             │
└─────────────────────────────────────────────────────────────────────────┘

Client                    Go Server                   Python Backend
   │                          │                             │
   │  POST /v1/tts            │                             │
   │  Content-Type: json      │                             │
   │  {"text": "Hello"}       │                             │
   │─────────────────────────►│                             │
   │                          │                             │
   │                          │  1. Parse request           │
   │                          │  2.  Validate (schema. go)    │
   │                          │  3. Apply defaults          │
   │                          │                             │
   │                          │  POST /v1/tts               │
   │                          │  Content-Type: msgpack      │
   │                          │─────────────────────────────►
   │                          │                             │
   │                          │                             │ LLaMA inference
   │                          │                             │ DAC decode
   │                          │                             │ (~1-2 seconds)
   │                          │                             │
   │                          │◄─────────────────────────────
   │                          │  Complete WAV audio         │
   │                          │                             │
   │◄─────────────────────────│                             │
   │  Content-Type: audio/wav │                             │
   │  <binary audio data>     │                             │
```

### TTS Request Flow (Streaming)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         STREAMING REQUEST                                │
└─────────────────────────────────────────────────────────────────────────┘

Client                    Go Server                   Python Backend
   │                          │                             │
   │  POST /v1/tts            │                             │
   │  {"streaming": true}     │                             │
   │─────────────────────────►│                             │
   │                          │                             │
   │                          │  POST /v1/tts               │
   │                          │  streaming=true             │
   │                          │─────────────────────────────►
   │                          │                             │
   │◄─────────────────────────│                             │
   │  HTTP 200                │                             │
   │  Transfer-Encoding:      │                             │
   │    chunked               │                             │
   │                          │                             │
   │                          │◄─────────────────────────────
   │◄─────────────────────────│  WAV header (44 bytes)     │
   │  [WAV header chunk]      │                             │
   │                          │                             │
   │                          │◄─────────────────────────────
   │◄─────────────────────────│  Audio segment 1           │
   │  [audio chunk 1]         │                             │
   │                          │                             │
   │                          │◄─────────────────────────────
   │◄─────────────────────────│  Audio segment 2           │
   │  [audio chunk 2]         │                             │
   │                          │                             │
   │         ...              │         ...                 │
   │                          │                             │
   │◄─────────────────────────│◄─────────────────────────────
   │  [final chunk + EOF]     │  Final audio               │
```

---

## Streaming Architecture

### Upstream Streaming Implementation

From `fish_speech/inference_engine/__init__.py`:

```python
# Upstream yields InferenceResult objects with codes:
# - "header": WAV header bytes
# - "segment": Audio chunk during generation
# - "final": Complete concatenated audio
# - "error": Error occurred

if req.streaming:
    yield InferenceResult(
        code="header",
        audio=(sample_rate, np.array(wav_chunk_header(sample_rate=sample_rate))),
    )

# During generation:
yield InferenceResult(
    code="segment",
    audio=(sample_rate, segment),
)

# At the end:
yield InferenceResult(
    code="final",
    audio=(sample_rate, audio),
)
```

### Go Streaming Implementation

```go
func (h *Handler) handleStreamingTTS(w http. ResponseWriter, req *ServeTTSRequest) error {
    // Set streaming headers
    w.Header().Set("Content-Type", "audio/wav")
    w.Header().Set("Transfer-Encoding", "chunked")
    w.Header().Set("Content-Disposition", "attachment; filename=audio.wav")
    
    flusher, ok := w.(http. Flusher)
    if !ok {
        return errors.New("streaming not supported")
    }
    
    // Forward request to Python backend with streaming enabled
    resp, err := h. backend.TTSStream(r.Context(), req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Stream chunks as they arrive
    buf := make([]byte, 4096)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            w.Write(buf[:n])
            flusher.Flush() // Critical: force send to client
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### WAV Header for Streaming

When streaming, the file size is unknown.  Use maximum value placeholder:

```go
func generateStreamingWAVHeader(sampleRate, bitsPerSample, channels int) []byte {
    header := make([]byte, 44)
    
    // RIFF header
    copy(header[0:4], "RIFF")
    binary.LittleEndian.PutUint32(header[4:8], 0xFFFFFFFF) // Unknown size
    copy(header[8:12], "WAVE")
    
    // fmt subchunk
    copy(header[12:16], "fmt ")
    binary.LittleEndian. PutUint32(header[16:20], 16) // Subchunk1Size (PCM)
    binary.LittleEndian.PutUint16(header[20:22], 1)  // AudioFormat (PCM)
    binary.LittleEndian. PutUint16(header[22:24], uint16(channels))
    binary. LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
    
    byteRate := sampleRate * channels * bitsPerSample / 8
    binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
    
    blockAlign := channels * bitsPerSample / 8
    binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
    binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
    
    // data subchunk
    copy(header[36:40], "data")
    binary. LittleEndian.PutUint32(header[40:44], 0xFFFFFFFF) // Unknown size
    
    return header
}
```

---

## Backend Communication

### MessagePack Compatibility

Upstream uses `ormsgpack` for Python.  We must ensure our Go msgpack encoding is compatible:

```go
import "github.com/vmihailenco/msgpack/v5"

// Configure msgpack to match ormsgpack behavior
func newMsgpackEncoder(w io.Writer) *msgpack. Encoder {
    enc := msgpack.NewEncoder(w)
    enc.SetCustomStructTag("msgpack")
    return enc
}

func newMsgpackDecoder(r io. Reader) *msgpack.Decoder {
    dec := msgpack.NewDecoder(r)
    dec.SetCustomStructTag("msgpack")
    return dec
}
```

### Struct Tags

All schema structs MUST have both `json` and `msgpack` tags:

```go
type ServeTTSRequest struct {
    Text              string               `json:"text" msgpack:"text"`
    ChunkLength       int                  `json:"chunk_length" msgpack:"chunk_length"`
    Format            string               `json:"format" msgpack:"format"`
    References        []ServeReferenceAudio `json:"references" msgpack:"references"`
    ReferenceID       *string              `json:"reference_id,omitempty" msgpack:"reference_id,omitempty"`
    Seed              *int                 `json:"seed,omitempty" msgpack:"seed,omitempty"`
    UseMemoryCache    string               `json:"use_memory_cache" msgpack:"use_memory_cache"`
    Normalize         bool                 `json:"normalize" msgpack:"normalize"`
    Streaming         bool                 `json:"streaming" msgpack:"streaming"`
    MaxNewTokens      int                  `json:"max_new_tokens" msgpack:"max_new_tokens"`
    TopP              float64              `json:"top_p" msgpack:"top_p"`
    RepetitionPenalty float64              `json:"repetition_penalty" msgpack:"repetition_penalty"`
    Temperature       float64              `json:"temperature" msgpack:"temperature"`
}
```

### Connection Pool Configuration

```go
type BackendClient struct {
    client   *http.Client
    endpoint string
}

func NewBackendClient(endpoint string) *BackendClient {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  true, // Audio is already compressed
    }
    
    client := &http.Client{
        Transport: transport,
        Timeout:   120 * time.Second, // Long timeout for inference
    }
    
    return &BackendClient{
        client:   client,
        endpoint: endpoint,
    }
}
```

---

## Configuration System

### Configuration Sources (Priority Order)

1.  CLI flags (highest)
2. Environment variables
3. Configuration file
4. Default values (lowest)

### Configuration Mapping

| CLI Flag | Environment Variable | Config Key | Default |
|----------|---------------------|------------|---------|
| `--listen` | `FISH_LISTEN` | `server.listen` | `0.0.0. 0:8080` |
| `--backend` | `FISH_BACKEND` | `backend. url` | `http://127.0.0. 1:8081` |
| `--workers` | `FISH_WORKERS` | `queue.workers` | `4` |
| `--max-text-length` | `FISH_MAX_TEXT_LENGTH` | `limits.max_text_length` | `0` |
| `--api-key` | `FISH_API_KEY` | `auth.api_key` | `""` |
| `--log-level` | `FISH_LOG_LEVEL` | `logging.level` | `info` |
| `--config` | - | - | `""` |

### Configuration File Format

```yaml
# config.yaml
server:
  listen: "0.0.0.0:8080"
  read_timeout: 30s
  write_timeout: 120s

backend:
  url: "http://127.0.0. 1:8081"
  timeout: 60s
  max_connections: 100

queue:
  workers: 4
  max_pending: 100

limits:
  max_text_length: 0  # 0 = unlimited

auth:
  api_key: ""  # Empty = no auth

logging:
  level: info
  format: json
```

---

## Error Handling Strategy

### Error Response Format

Match upstream error format:

```go
type ErrorResponse struct {
    Detail string `json:"detail"`
}

// HTTP 400 Bad Request
{"detail": "Text is too long, max length is 10000"}

// HTTP 401 Unauthorized
{"detail": "Invalid token"}

// HTTP 415 Unsupported Media Type
{"detail": "Unsupported content type"}

// HTTP 500 Internal Server Error
{"detail": "Failed to generate speech"}
```

### Error Codes

| HTTP Status | Condition | Message |
|-------------|-----------|---------|
| 400 | Text too long | `Text is too long, max length is {n}` |
| 400 | Streaming with non-WAV | `Streaming only supports WAV format` |
| 400 | Invalid parameter range | `{param} must be between {min} and {max}` |
| 401 | Missing/invalid token | `Invalid token` |
| 415 | Bad content-type | `Unsupported content type` |
| 500 | Backend error | `Failed to generate speech` |
| 502 | Backend unavailable | `Backend service unavailable` |
| 504 | Backend timeout | `Request timeout` |

---

## Deployment Architecture

### Docker Compose Deployment

```yaml
# docker-compose.yml
version: '3.8'

services:
  server:
    build:
      context: ../../../Downloads
      dockerfile: docker/Dockerfile. server
    ports:
      - "8080:8080"
    environment:
      - FISH_BACKEND=http://inference:8081
      - FISH_LOG_LEVEL=info
    depends_on:
      inference:
        condition: service_healthy

  inference:
    build:
      context: ../../../Downloads
      dockerfile: docker/Dockerfile.inference
    environment:
      - CUDA_VISIBLE_DEVICES=0
    volumes:
      - models:/app/checkpoints
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [ gpu ]
    healthcheck:
      test: [ "CMD", "curl", "-f", "http://localhost:8081/v1/health" ]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  models:
```

### Port Allocation

| Service | Port | Purpose |
|---------|------|---------|
| Go Server | 8080 | Public API |
| Python Backend | 8081 | Internal inference |

---

## Performance Characteristics

### Expected Performance Improvements

| Metric | Python Server | Go Server | Improvement |
|--------|--------------|-----------|-------------|
| Startup time | 3-5s | <100ms | 30-50x |
| Memory (idle) | ~200MB | ~15MB | 13x |
| Concurrent connections | ~100-500 | ~10,000+ | 20-100x |
| Time to first byte (streaming) | ~1-2s | ~300-500ms | 2-4x |

### Latency Budget

```
Total request time: ~1500-2000ms

┌─────────────────────────────────────────────────────────────────┐
│ Go Server overhead        │  ~1-5ms                             │
├─────────────────────────────────────────────────────────────────┤
│ Network to backend        │  ~1-5ms                             │
├─────────────────────────────────────────────────────────────────┤
│ Python inference          │  ~1500-2000ms  ← Dominant factor    │
├─────────────────────────────────────────────────────────────────┤
│ Network from backend      │  ~1-5ms                             │
├─────────────────────────────────────────────────────────────────┤
│ Go Server response        │  ~1-5ms                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Appendix: Upstream Code References

### Key Files in fishaudio/fish-speech

| File | Purpose | Our Concern |
|------|---------|-------------|
| `fish_speech/utils/schema. py` | Pydantic models | Schema definitions |
| `tools/server/views. py` | API endpoints | Route paths, behavior |
| `tools/server/api_utils.py` | Request parsing | Content-type handling |
| `tools/api_server.py` | Server setup | Auth, config |
| `fish_speech/inference_engine/__init__.py` | Inference | Streaming protocol |
| `tools/api_client.py` | Reference client | Request format |

### Commit Reference

All specifications derived from:
- **Repository:** fishaudio/fish-speech
- **Commit:** 80db25e72628067aa23f35161f1bf6bffbc2e554

---

*Document Version: 2.0*
*Last Updated: 2024-01*