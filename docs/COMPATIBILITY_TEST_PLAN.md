# 🧪 Fish-Speech-Go Compatibility Test Plan

> Comprehensive test plan to verify that Fish-Speech-Go is 100% compatible with the upstream Fish-Speech API.

---

## Table of Contents

1.  [Overview](#overview)
2. [Test Environment](#test-environment)
3. [Test Categories](#test-categories)
4. [Test Cases](#test-cases)
5. [Automated Test Suite](#automated-test-suite)
6. [Manual Verification](#manual-verification)
7. [Performance Comparison](#performance-comparison)
8. [Regression Testing](#regression-testing)

---

## Overview

### Objective

Verify that any client working with the upstream Fish-Speech Python server will work identically with Fish-Speech-Go, with no modifications required.

### Success Criteria

1. All API endpoints return identical response structures
2. All request schemas are accepted with same validation rules
3. All error conditions return identical error messages and status codes
4.  Streaming behavior is compatible with existing clients
5. Authentication works identically
6. Content-type handling matches upstream

### Test Approach

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        COMPATIBILITY TESTING                             │
└─────────────────────────────────────────────────────────────────────────┘

                    ┌─────────────────────┐
                    │    Test Client      │
                    │  (Same requests)    │
                    └──────────┬──────────┘
                               │
              ┌────────────────┴────────────────┐
              │                                 │
              ▼                                 ▼
   ┌─────────────────────┐           ┌─────────────────────┐
   │   Python Server     │           │    Go Server        │
   │   (Upstream)        │           │   (Fish-Speech-Go)  │
   │   Port: 8080        │           │   Port: 8090        │
   └──────────┬──────────┘           └──────────┬──────────┘
              │                                 │
              ▼                                 ▼
   ┌─────────────────────┐           ┌─────────────────────┐
   │   Response A        │    ══     │   Response B        │
   │   (Expected)        │  COMPARE  │   (Actual)          │
   └─────────────────────┘           └─────────────────────┘
```

---

## Test Environment

### Required Infrastructure

```yaml
# docker-compose.test.yml
version: '3.8'

services:
  # Upstream Python server (reference)
  python-server:
    image: fishaudio/fish-speech:latest
    ports:
      - "8080:8080"
    command: >
      python -m tools.api_server
      --listen 0.0. 0.0:8080
      --llama-checkpoint-path checkpoints/openaudio-s1-mini
    volumes:
      - models:/app/checkpoints
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [ gpu ]

  # Go server under test
  go-server:
    build:
      context: ../../../Downloads
      dockerfile: docker/Dockerfile.server
    ports:
      - "8090:8080"
    environment:
      - FISH_BACKEND=http://python-backend:8081
    depends_on:
      - python-backend

  # Python backend for Go server
  python-backend:
    image: fishaudio/fish-speech:latest
    ports:
      - "8081:8081"
    command: >
      python -m tools.api_server
      --listen 0.0.0.0:8081
      --llama-checkpoint-path checkpoints/openaudio-s1-mini
    volumes:
      - models:/app/checkpoints
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [ gpu ]

volumes:
  models:
```

### Test Configuration

```yaml
# test-config.yaml
python_server:
  url: "http://localhost:8080"
  
go_server:
  url: "http://localhost:8090"

timeouts:
  health_check: 5s
  tts_request: 120s
  
comparison:
  # Allow small differences in audio (due to floating point)
  audio_tolerance_bytes: 100
  # Timing differences allowed
  timing_tolerance_ms: 100
```

---

## Test Categories

### Category 1: Schema Compatibility

Verify that request/response schemas match exactly. 

### Category 2: Endpoint Compatibility

Verify all endpoints exist and behave identically. 

### Category 3: Content-Type Handling

Verify all content types are handled correctly. 

### Category 4: Validation Compatibility

Verify validation rules and error messages match. 

### Category 5: Authentication Compatibility

Verify authentication behavior matches. 

### Category 6: Streaming Compatibility

Verify streaming responses are compatible.

### Category 7: Error Handling Compatibility

Verify error responses match exactly.

---

## Test Cases

### TC-001: Health Check GET

**Objective:** Verify `/v1/health` GET endpoint compatibility.

**Request:**
```http
GET /v1/health HTTP/1.1
Host: localhost
```

**Expected Response:**
```json
{"status": "ok"}
```

**Verification:**
- [ ] Status code is 200
- [ ] Response body matches exactly
- [ ] Content-Type is `application/json`

---

### TC-002: Health Check POST

**Objective:** Verify `/v1/health` POST endpoint compatibility.

**Request:**
```http
POST /v1/health HTTP/1.1
Host: localhost
```

**Expected Response:**
```json
{"status": "ok"}
```

**Verification:**
- [ ] Status code is 200
- [ ] Response body matches exactly

---

### TC-003: TTS Basic Request (JSON)

**Objective:** Verify basic TTS with JSON content type.

**Request:**
```http
POST /v1/tts HTTP/1. 1
Host: localhost
Content-Type: application/json

{"text": "Hello, world!"}
```

**Verification:**
- [ ] Status code is 200
- [ ] Content-Type is `audio/wav`
- [ ] Response is valid WAV audio
- [ ] Audio duration is reasonable (>0.5s, <30s)

---

### TC-004: TTS Basic Request (MessagePack)

**Objective:** Verify basic TTS with MessagePack content type.

**Request:**
```http
POST /v1/tts HTTP/1. 1
Host: localhost
Content-Type: application/msgpack

<msgpack-encoded: {"text": "Hello, world!"}>
```

**Verification:**
- [ ] Status code is 200
- [ ] Content-Type is `audio/wav`
- [ ] Response is valid WAV audio

---

### TC-005: TTS with All Parameters

**Objective:** Verify TTS with all optional parameters.

**Request:**
```json
{
  "text": "Testing all parameters.",
  "chunk_length": 250,
  "format": "wav",
  "max_new_tokens": 512,
  "top_p": 0.9,
  "repetition_penalty": 1.2,
  "temperature": 0.7,
  "normalize": true,
  "use_memory_cache": "off",
  "seed": 42
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Request accepted with all parameters
- [ ] Audio generated successfully

---

### TC-006: TTS MP3 Format

**Objective:** Verify TTS with MP3 output format.

**Request:**
```json
{
  "text": "Testing MP3 format.",
  "format": "mp3"
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Content-Type is `audio/mpeg`
- [ ] Response is valid MP3 audio

---

### TC-007: TTS PCM Format

**Objective:** Verify TTS with PCM output format.

**Request:**
```json
{
  "text": "Testing PCM format.",
  "format": "pcm"
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Response is valid PCM audio

---

### TC-008: TTS Streaming

**Objective:** Verify streaming response. 

**Request:**
```json
{
  "text": "Testing streaming output.",
  "streaming": true
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Transfer-Encoding is `chunked`
- [ ] Response starts before generation completes
- [ ] Complete audio is valid WAV

---

### TC-009: TTS Streaming Non-WAV Error

**Objective:** Verify streaming only works with WAV format. 

**Request:**
```json
{
  "text": "Testing streaming with MP3.",
  "streaming": true,
  "format": "mp3"
}
```

**Expected Response:**
```
HTTP/1.1 400 Bad Request

{"detail": "Streaming only supports WAV format"}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error message matches exactly

---

### TC-010: TTS Text Too Long

**Objective:** Verify text length validation (when configured).

**Setup:** Start server with `--max-text-length 100`

**Request:**
```json
{
  "text": "<101+ characters>"
}
```

**Expected Response:**
```
HTTP/1.1 400 Bad Request

{"detail": "Text is too long, max length is 100"}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error message includes max length

---

### TC-011: TTS Invalid chunk_length (Too Low)

**Objective:** Verify chunk_length minimum validation.

**Request:**
```json
{
  "text": "Test",
  "chunk_length": 50
}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error indicates chunk_length out of range

---

### TC-012: TTS Invalid chunk_length (Too High)

**Objective:** Verify chunk_length maximum validation. 

**Request:**
```json
{
  "text": "Test",
  "chunk_length": 500
}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error indicates chunk_length out of range

---

### TC-013: TTS Invalid top_p

**Objective:** Verify top_p range validation.

**Request:**
```json
{
  "text": "Test",
  "top_p": 1.5
}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error indicates top_p out of range (0. 1-1.0)

---

### TC-014: TTS Invalid temperature

**Objective:** Verify temperature range validation.

**Request:**
```json
{
  "text": "Test",
  "temperature": 0.05
}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error indicates temperature out of range (0.1-1.0)

---

### TC-015: TTS Invalid repetition_penalty

**Objective:** Verify repetition_penalty range validation.

**Request:**
```json
{
  "text": "Test",
  "repetition_penalty": 3.0
}
```

**Verification:**
- [ ] Status code is 400
- [ ] Error indicates repetition_penalty out of range (0.9-2.0)

---

### TC-016: Authentication Required

**Objective:** Verify authentication when API key is configured.

**Setup:** Start server with `--api-key test_key_123`

**Request (no auth):**
```http
POST /v1/tts HTTP/1.1
Host: localhost
Content-Type: application/json

{"text": "Test"}
```

**Expected Response:**
```
HTTP/1.1 401 Unauthorized

{"detail": "Invalid token"}
```

**Verification:**
- [ ] Status code is 401
- [ ] Error message matches

---

### TC-017: Authentication Valid

**Objective:** Verify valid authentication. 

**Setup:** Start server with `--api-key test_key_123`

**Request:**
```http
POST /v1/tts HTTP/1. 1
Host: localhost
Content-Type: application/json
Authorization: Bearer test_key_123

{"text": "Test"}
```

**Verification:**
- [ ] Status code is 200
- [ ] Request processed successfully

---

### TC-018: Authentication Invalid Token

**Objective:** Verify invalid token rejection.

**Setup:** Start server with `--api-key test_key_123`

**Request:**
```http
POST /v1/tts HTTP/1.1
Host: localhost
Content-Type: application/json
Authorization: Bearer wrong_key

{"text": "Test"}
```

**Expected Response:**
```
HTTP/1. 1 401 Unauthorized

{"detail": "Invalid token"}
```

**Verification:**
- [ ] Status code is 401
- [ ] Error message matches exactly

---

### TC-019: Unsupported Content-Type

**Objective:** Verify unsupported content type rejection.

**Request:**
```http
POST /v1/tts HTTP/1.1
Host: localhost
Content-Type: text/plain

Hello world
```

**Verification:**
- [ ] Status code is 415
- [ ] Error indicates unsupported media type

---

### TC-020: Voice Cloning with Reference

**Objective:** Verify voice cloning with inline reference.

**Request:**
```json
{
  "text": "Testing voice cloning.",
  "references": [
    {
      "audio": "<base64-encoded-audio>",
      "text": "Reference transcript"
    }
  ]
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Audio generated successfully
- [ ] Voice characteristics resemble reference

---

### TC-021: Voice Cloning with Reference ID

**Objective:** Verify voice cloning with saved reference.

**Setup:** First add a reference via `/v1/references/add`

**Request:**
```json
{
  "text": "Testing voice cloning with saved reference.",
  "reference_id": "test-voice"
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Audio generated successfully

---

### TC-022: Add Reference

**Objective:** Verify reference addition.

**Request:**
```http
POST /v1/references/add HTTP/1.1
Content-Type: application/json

{
  "id": "test-voice",
  "audio": "<base64-encoded-audio>",
  "text": "Reference transcript"
}
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Reference added successfully",
  "reference_id": "test-voice"
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Response matches expected structure

---

### TC-023: List References

**Objective:** Verify reference listing.

**Request:**
```http
GET /v1/references HTTP/1.1
```

**Expected Response:**
```json
{
  "success": true,
  "reference_ids": ["test-voice"],
  "message": "Success"
}
```

**Verification:**
- [ ] Status code is 200
- [ ] Response includes previously added reference

---

### TC-024: VQGAN Encode

**Objective:** Verify VQGAN encoding endpoint.

**Request:**
```http
POST /v1/vqgan/encode HTTP/1. 1
Content-Type: application/msgpack

<msgpack