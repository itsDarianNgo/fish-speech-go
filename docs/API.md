# API Reference

Complete API documentation for Fish-Speech-Go.

## Base URL

```
http://localhost:8080/v1
```

## Authentication

If `API_KEY` is configured, include the header:

```
Authorization: Bearer <your-api-key>
```

---

## Endpoints

### Health Check

Check if the service is running.

```
GET /v1/health
```

#### Response

```json
{
  "status": "ok"
}
```

---

### Generate Speech (OpenAI-Compatible)

Generate audio from text using OpenAI-compatible format.

```
POST /v1/audio/speech
```

#### Request Body

```json
{
  "model": "fish-speech",
  "voice": "default",
  "input": "Hello, world!",
  "response_format": "wav"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | No | Model name (default: `fish-speech`) |
| `voice` | string | No | Voice ID (default: `default`) |
| `input` | string | **Yes** | Text to synthesize |
| `response_format` | string | No | Output format: `wav`, `mp3` (default: `wav`) |

#### Response

Audio file with appropriate Content-Type header.

#### Example

```bash
curl -X POST http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{
    "model": "fish-speech",
    "voice": "default",
    "input": "Hello, world!"
  }' \
  --output speech.wav
```

---

### Generate Speech (Simple)

Simplified TTS endpoint.

```
POST /v1/tts
```

#### Request Body

```json
{
  "text": "Hello, world!",
  "voice": "default"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | **Yes** | Text to synthesize |
| `voice` | string | No | Voice ID (default: `default`) |

#### Response

Audio file (WAV format).

#### Example

```bash
curl -X POST http://localhost:8080/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!"}' \
  --output speech.wav
```

---

### List Voices

Get available voices.

```
GET /v1/audio/voices
GET /v1/voices
```

#### Response

```json
{
  "voices": [
    {
      "id": "default",
      "name": "Default",
      "description": "Default Fish-Speech voice"
    }
  ]
}
```

---

## Error Responses

Errors return JSON with appropriate HTTP status codes.

```json
{
  "error": {
    "message": "Text is required",
    "type": "invalid_request_error",
    "code": "missing_required_field"
  }
}
```

### Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Invalid or missing API key |
| 500 | Internal Server Error |
| 503 | Service Unavailable - Inference backend down |

---

## Rate Limits

No rate limits when self-hosted. You're limited only by your hardware.

---

## SDKs

Fish-Speech-Go is compatible with OpenAI SDKs:

- [Python](https://github.com/openai/openai-python): `pip install openai`
- [Node.js](https://github.com/openai/openai-node): `npm install openai`
- [Go](https://github.com/sashabaranov/go-openai): `go get github.com/sashabaranov/go-openai`

Just change the base URL:

```python
client = OpenAI(base_url="http://localhost:8080/v1", api_key="unused")
```
