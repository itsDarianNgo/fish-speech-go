package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/fish-speech-go/fish-speech-go/internal/config"
)

func newTestRouter() http.Handler {
	cfg, _ := config.LoadWithDefaults(nil)
	logger := zerolog.New(io.Discard)
	return NewRouter(cfg, logger)
}

func TestHealthGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()

	newTestRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	expected := "{\"status\":\"ok\"}\n"
	if rr.Body.String() != expected {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestHealthPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
	rr := httptest.NewRecorder()

	newTestRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	expected := "{\"status\":\"ok\"}\n"
	if rr.Body.String() != expected {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestParseTTSRequest_JSON(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	parsed, err := ParseTTSRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if parsed.ChunkLength != 200 || parsed.Format != "wav" {
		t.Fatalf("defaults not applied: %+v", parsed)
	}
}

func TestParseTTSRequest_MessagePack(t *testing.T) {
	payload := map[string]interface{}{"text": "hello"}
	encoded, _ := msgpack.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/msgpack")

	parsed, err := ParseTTSRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if parsed.Text != "hello" {
		t.Fatalf("expected text to be parsed, got %s", parsed.Text)
	}
}

func TestParseTTSRequest_UnsupportedContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewBufferString("hello"))
	req.Header.Set("Content-Type", "text/plain")

	_, err := ParseTTSRequest(req)
	if err == nil {
		t.Fatalf("expected error for unsupported content type")
	}
	if httpErr, ok := IsHTTPError(err); !ok || httpErr.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("expected HTTP 415 error, got %v", err)
	}
}

func TestTTSHandler_ValidationErrors(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello","chunk_length":50}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if rr.Body.String() != "{\"detail\":\"chunk_length must be between 100 and 300\"}\n" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestTTSHandler_StreamingNonWavError(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello","streaming":true,"format":"mp3"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if rr.Body.String() != "{\"detail\":\"Streaming only supports WAV format\"}\n" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_NoKeyConfigured(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := AuthMiddleware("")(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected pass through, got %d", rr.Code)
	}
}

func TestAuthMiddleware_ValidKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")

	handler := AuthMiddleware("secret")(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected pass through, got %d", rr.Code)
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")

	handler := AuthMiddleware("secret")(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if rr.Body.String() != "{\"detail\":\"Invalid token\"}\n" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := AuthMiddleware("secret")(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if rr.Body.String() != "{\"detail\":\"Invalid token\"}\n" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestWriteError_MatchesUpstreamFormat(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusBadRequest, "something went wrong")

	if rr.Body.String() != "{\"detail\":\"something went wrong\"}\n" {
		t.Fatalf("unexpected error body: %s", rr.Body.String())
	}
}
