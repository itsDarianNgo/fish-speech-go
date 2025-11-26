# 📡 Fish-Speech-Go API Reference

> Complete API specification for the Fish-Speech-Go server, ensuring 100% compatibility with upstream Fish-Speech. 

---

## Table of Contents

1.  [Overview](#overview)
2. [Authentication](#authentication)
3. [Content Types](#content-types)
4. [Endpoints](#endpoints)
   - [Health Check](#health-check)
   - [Text-to-Speech](#text-to-speech)
   - [VQGAN Encode](#vqgan-encode)
   - [VQGAN Decode](#vqgan-decode)
   - [References Management](#references-management)
5. [Request Schemas](#request-schemas)
6. [Response Formats](#response-formats)
7. [Error Handling](#error-handling)
8.  [Examples](#examples)

---

## Overview

### Base URL

```
http://localhost:8080
```

### API Version

All endpoints are prefixed with `/v1/`. 

### Compatibility

This API is 100% compatible with the upstream Fish-Speech Python server.  Any client that works with the original server will work with Fish-Speech-Go. 

---

## Authentication

### Bearer Token Authentication

If the server is started with an API key, all requests must include an `Authorization` header:

```
Authorization: Bearer <api_key>
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/tts \
  -H "Authorization: Bearer sk_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world"}'
```

**Error Response (401 Unauthorized):**
```json
{"detail": "Invalid token"}
```

---

## Content Types

### Supported Request Content Types

| Content-Type | Description | Usage |
|--------------|-------------|-------|
| `application/json` | JSON encoding | Default, human-readable |
| `application/msgpack` | MessagePack encoding | Efficient binary format |
| `multipart/form-data` | Form data with files | File uploads |

### Response Content Types

| Endpoint | Success Response | Error Response |
|----------|------------------|----------------|
| `/v1/tts` | `audio/wav`, `audio/mpeg`, `audio/pcm` | `application/json` |
| `/v1/health` | `application/json` | `application/json` |
| `/v1/vqgan/*` | `application/msgpack` | `application/json` |
| `/v1/references/*` | `application/json` or `application/msgpack` | `application/json` |

---

## Endpoints

### Health Check

Check if the server is running and healthy.

#### `GET /v1/health`

**Request:**
```bash
curl http://localhost:8080/v1/health
```

**Response (200 OK):**
```json
{"status": "ok"}
```

#### `POST /v1/health`

Same as GET, provided for compatibility.

---

### Text-to-Speech

Generate speech audio from text.

#### `POST /v1/tts`

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer <api_key>  (if authentication enabled)
```

**Request Body:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `text` | string | ✅ Yes | - | Text to synthesize |
| `chunk_length` | integer | No | `200` | Chunk length (100-300) |
| `format` | string | No | `"wav"` | Output format: `wav`, `pcm`, `mp3` |
| `references` | array | No | `[]` | Reference audio for voice cloning |
| `reference_id` | string | No | `null` | ID of saved reference voice |
| `seed` | integer | No | `null` | Random seed for reproducibility |
| `use_memory_cache` | string | No | `"off"` | Memory cache: `on` or `off` |
| `normalize` | boolean | No | `true` | Normalize text for stability |
| `streaming` | boolean | No | `false` | Enable streaming response |
| `max_new_tokens` | integer | No | `1024` | Maximum tokens to generate |
| `top_p` | float | No | `0.8` | Top-p sampling (0.1-1.0) |
| `repetition_penalty` | float | No | `1.1` | Repetition penalty (0.9-2.0) |
| `temperature` | float | No | `0.8` | Sampling temperature (0. 1-1.0) |

**Example Request (JSON):**
```json
{
  "text": "Hello, this is a test of Fish Speech.",
  "format": "wav",
  "temperature": 0.7,
  "top_p": 0.8
}
```

**Example Request with Voice Cloning:**
```json
{
  "text": "Hello in a cloned voice.",
  "references": [
    {
      "audio": "<base64-encoded-audio-bytes>",
      "text": "This is the transcript of the reference audio."
    }
  ]
}
```

**Response (Non-Streaming):**
```
HTTP/1.1 200 OK
Content-Type: audio/wav
Content-Disposition: attachment; filename=audio.wav

<binary audio data>
```

**Response (Streaming):**
```
HTTP/1. 1 200 OK
Content-Type: audio/wav
Transfer-Encoding: chunked
Content-Disposition: attachment; filename=audio.wav

<chunked binary audio data>
```

**Validation Errors (400 Bad Request):**
```json
{"detail": "Text is too long, max length is 10000"}
```
```json
{"detail": "Streaming only supports WAV format"}
```
```json
{"detail": "chunk_length must be between 100 and 300"}
```

---

### VQGAN Encode

Encode audio to semantic tokens. 

#### `POST /v1/vqgan/encode`

**Request Body:**
```json
{
  "audios": ["<base64-encoded-audio-bytes>", ...]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `audios` | array[bytes] | ✅ Yes | List of audio files to encode |

**Response (200 OK):**
```
Content-Type: application/msgpack

{
  "tokens": [[[1, 2, 3, ... ], ... ], ...]
}
```

---

### VQGAN Decode

Decode semantic tokens to audio.

#### `POST /v1/vqgan/decode`

**Request Body:**
```json
{
  "tokens": [[[1, 2, 3, ...], ...], ...]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tokens` | array[array[array[int]]] | ✅ Yes | Semantic tokens to decode |

**Response (200 OK):**
```
Content-Type: application/msgpack

{
  "audios": ["<pcm-float16-bytes>", ...]
}
```

---

### References Management

Manage saved voice references.

#### `POST /v1/references/add`

Add a new reference voice. 

**Request Body:**
```json
{
  "id": "my-voice",
  "audio": "<base64-encoded-audio-bytes>",
  "text": "Transcript of the audio"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `id` | string | ✅ Yes | 1-255 chars, alphanumeric + `-_` |
| `audio` | bytes | ✅ Yes | Audio file bytes |
| `text` | string | ✅ Yes | Non-empty transcript |

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Reference added successfully",
  "reference_id": "my-voice"
}
```

#### `GET /v1/references`

List all saved references. 

**Response (200 OK):**
```json
{
  "success": true,
  "reference_ids": ["voice-1", "voice-2"],
  "message": "Success"
}
```

#### `DELETE /v1/references/{id}`

Delete a reference voice.

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Reference deleted successfully",
  "reference_id": "my-voice"
}
```

---

## Request Schemas

### ServeTTSRequest

```go
type ServeTTSRequest struct {
    // Required
    Text string `json:"text" msgpack:"text" validate:"required"`
    
    // Audio generation parameters
    ChunkLength       int     `json:"chunk_length" msgpack:"chunk_length" validate:"min=100,max=300"`
    Format            string  `json:"format" msgpack:"format" validate:"oneof=wav pcm mp3"`
    MaxNewTokens      int     `json:"max_new_tokens" msgpack:"max_new_tokens"`
    TopP              float64 `json:"top_p" msgpack:"top_p" validate:"min=0.1,max=1.0"`
    RepetitionPenalty float64 `json:"repetition_penalty" msgpack:"repetition_penalty" validate:"min=0.9,max=2. 0"`
    Temperature       float64 `json:"temperature" msgpack:"temperature" validate:"min=0. 1,max=1.0"`
    
    // Voice cloning
    References   []ServeReferenceAudio `json:"references" msgpack:"references"`
    ReferenceID  *string               `json:"reference_id,omitempty" msgpack:"reference_id,omitempty"`
    
    // Options
    Seed           *int   `json:"seed,omitempty" msgpack:"seed,omitempty"`
    UseMemoryCache string `json:"use_memory_cache" msgpack:"use_memory_cache" validate:"oneof=on off"`
    Normalize      bool   `json:"normalize" msgpack:"normalize"`
    Streaming      bool   `json:"streaming" msgpack:"streaming"`
}
```

### ServeReferenceAudio

```go
type ServeReferenceAudio struct {
    Audio []byte `json:"audio" msgpack:"audio"` // Raw bytes or base64
    Text  string `json:"text" msgpack:"text"`   // Transcript
}
```

### ServeVQGANEncodeRequest

```go
type ServeVQGANEncodeRequest struct {
    Audios [][]byte `json:"audios" msgpack:"audios"`
}
```

### ServeVQGANDecodeRequest

```go
type ServeVQGANDecodeRequest struct {
    Tokens [][][]int `json:"tokens" msgpack:"tokens"`
}
```

### AddReferenceRequest

```go
type AddReferenceRequest struct {
    ID    string `json:"id" msgpack:"id" validate:"min=1,max=255,alphanumdash"`
    Audio []byte `json:"audio" msgpack:"audio" validate:"required"`
    Text  string `json:"text" msgpack:"text" validate:"required,min=1"`
}
```

---

## Response Formats

### Success Response (TTS)

For TTS requests, the response is binary audio data:

```
HTTP/1.1 200 OK
Content-Type: audio/wav
Content-Disposition: attachment; filename=audio.wav
Content-Length: 123456

<binary audio data>
```

### Success Response (JSON)

For non-audio endpoints:

```json
{
  "success": true,
  "message": "Operation completed",
  "data": { ... }
}
```

### Success Response (MessagePack)

For VQGAN endpoints, response is MessagePack encoded:

```
HTTP/1.1 200 OK
Content-Type: application/msgpack

<msgpack binary data>
```

---

## Error Handling

### Error Response Format

All errors return JSON:

```json
{
  "detail": "Human-readable error message"
}
```

### HTTP Status Codes

| Status | Meaning | When Used |
|--------|---------|-----------|
| 200 | OK | Successful request |
| 400 | Bad Request | Invalid input, validation failure |
| 401 | Unauthorized | Missing or invalid API key |
| 404 | Not Found | Resource not found |
| 415 | Unsupported Media Type | Invalid Content-Type |
| 500 | Internal Server Error | Server-side error |
| 502 | Bad Gateway | Backend unavailable |
| 504 | Gateway Timeout | Backend timeout |

### Common Error Messages

| Error | HTTP Status | Message |
|-------|-------------|---------|
| Text too long | 400 | `Text is too long, max length is {n}` |
| Invalid streaming format | 400 | `Streaming only supports WAV format` |
| Invalid chunk_length | 400 | `chunk_length must be between 100 and 300` |
| Invalid top_p | 400 | `top_p must be between 0.1 and 1.0` |
| Invalid temperature | 400 | `temperature must be between 0.1 and 1.0` |
| Invalid repetition_penalty | 400 | `repetition_penalty must be between 0. 9 and 2.0` |
| Auth failure | 401 | `Invalid token` |
| Unsupported content type | 415 | `Unsupported content type` |
| Generation failed | 500 | `Failed to generate speech` |
| Backend error | 502 | `Backend service unavailable` |
| Timeout | 504 | `Request timeout` |

---

## Examples

### cURL Examples

#### Basic TTS (JSON)

```bash
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world! "}' \
  -o output.wav
```

#### TTS with Parameters

```bash
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello with custom settings.",
    "temperature": 0.7,
    "top_p": 0.9,
    "format": "mp3"
  }' \
  -o output.mp3
```

#### Streaming TTS

```bash
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Streaming audio output.", "streaming": true}' \
  --no-buffer \
  -o output.wav
```

#### With Authentication

```bash
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_your_api_key" \
  -d '{"text": "Authenticated request. "}' \
  -o output.wav
```

#### Voice Cloning

```bash
# First, base64 encode your reference audio
AUDIO_B64=$(base64 -w 0 reference.wav)

curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d "{
    \"text\": \"Hello in cloned voice.\",
    \"references\": [{
      \"audio\": \"$AUDIO_B64\",
      \"text\": \"This is what I said in the reference audio.\"
    }]
  }" \
  -o cloned.wav
```

### Python Examples

#### Using requests

```python
import requests

# Basic TTS
response = requests.post(
    "http://localhost:8080/v1/tts",
    json={"text": "Hello from Python!"},
)
with open("output.wav", "wb") as f:
    f.write(response. content)
```

#### Using MessagePack

```python
import requests
import ormsgpack

data = {
    "text": "Hello with MessagePack!",
    "temperature": 0.7,
}

response = requests. post(
    "http://localhost:8080/v1/tts",
    data=ormsgpack.packb(data),
    headers={"Content-Type": "application/msgpack"},
)
with open("output.wav", "wb") as f:
    f.write(response. content)
```

#### Streaming

```python
import requests

response = requests.post(
    "http://localhost:8080/v1/tts",
    json={"text": "Streaming from Python!", "streaming": True},
    stream=True,
)

with open("output.wav", "wb") as f:
    for chunk in response.iter_content(chunk_size=4096):
        f.write(chunk)
```

### Go Examples

#### Using the SDK

```go
package main

import (
    "context"
    "os"
    
    fishspeech "github.com/yourusername/fish-speech-go/go/pkg/client"
)

func main() {
    client, _ := fishspeech.NewClient("http://localhost:8080")
    
    audio, _ := client.Generate(context.Background(), &fishspeech. TTSRequest{
        Text:        "Hello from Go!",
        Temperature: 0.7,
    })
    
    os.WriteFile("output.wav", audio, 0644)
}
```

---

## Appendix: Upstream Compatibility Reference

This API specification is derived from:

- **Repository:** https://github.com/fishaudio/fish-speech
- **Commit:** 80db25e72628067aa23f35161f1bf6bffbc2e554
- **Files:**
  - `fish_speech/utils/schema.py`
  - `tools/server/views. py`
  - `tools/api_client.py`

---

*Document Version: 1.0*
*Last Updated: 2024-01*