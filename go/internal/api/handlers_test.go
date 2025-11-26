package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/fish-speech-go/fish-speech-go/internal/backend"
	"github.com/fish-speech-go/fish-speech-go/internal/config"
	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

type mockBackend struct {
	ttsFunc       func(ctx context.Context, req *schema.ServeTTSRequest) ([]byte, string, error)
	ttsStreamFunc func(ctx context.Context, req *schema.ServeTTSRequest) (io.ReadCloser, error)
	healthFunc    func(ctx context.Context) error
	vqganEncodeFn func(ctx context.Context, req *schema.ServeVQGANEncodeRequest) (*schema.ServeVQGANEncodeResponse, error)
	vqganDecodeFn func(ctx context.Context, req *schema.ServeVQGANDecodeRequest) (*schema.ServeVQGANDecodeResponse, error)
}

func (m *mockBackend) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func (m *mockBackend) TTS(ctx context.Context, req *schema.ServeTTSRequest) ([]byte, string, error) {
	if m.ttsFunc != nil {
		return m.ttsFunc(ctx, req)
	}
	return nil, "", backend.ErrBackendUnavailable
}

func (m *mockBackend) TTSStream(ctx context.Context, req *schema.ServeTTSRequest) (io.ReadCloser, error) {
	if m.ttsStreamFunc != nil {
		return m.ttsStreamFunc(ctx, req)
	}
	return nil, backend.ErrBackendUnavailable
}

func (m *mockBackend) VQGANEncode(ctx context.Context, req *schema.ServeVQGANEncodeRequest) (*schema.ServeVQGANEncodeResponse, error) {
	if m.vqganEncodeFn != nil {
		return m.vqganEncodeFn(ctx, req)
	}
	return nil, backend.ErrBackendUnavailable
}

func (m *mockBackend) VQGANDecode(ctx context.Context, req *schema.ServeVQGANDecodeRequest) (*schema.ServeVQGANDecodeResponse, error) {
	if m.vqganDecodeFn != nil {
		return m.vqganDecodeFn(ctx, req)
	}
	return nil, backend.ErrBackendUnavailable
}

func newTestRouter(b backend.Client) http.Handler {
	cfg, _ := config.LoadWithDefaults(nil)
	logger := zerolog.New(io.Discard)
	return NewRouter(cfg, b, logger)
}

func TestHealthGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()

	newTestRouter(&mockBackend{}).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "{\"status\":\"ok\"}\n", rr.Body.String())
}

func TestHealthPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
	rr := httptest.NewRecorder()

	newTestRouter(&mockBackend{}).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "{\"status\":\"ok\"}\n", rr.Body.String())
}

func TestParseTTSRequest_JSON(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	parsed, err := ParseTTSRequest(req)
	require.NoError(t, err)

	require.Equal(t, 200, parsed.ChunkLength)
	require.Equal(t, "wav", parsed.Format)
}

func TestParseTTSRequest_MessagePack(t *testing.T) {
	payload := map[string]interface{}{"text": "hello"}
	encoded, _ := msgpack.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/msgpack")

	parsed, err := ParseTTSRequest(req)
	require.NoError(t, err)
	assert.Equal(t, "hello", parsed.Text)
}

func TestParseTTSRequest_UnsupportedContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewBufferString("hello"))
	req.Header.Set("Content-Type", "text/plain")

	_, err := ParseTTSRequest(req)
	require.Error(t, err)

	httpErr, ok := IsHTTPError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnsupportedMediaType, httpErr.Status)
}

func TestTTSHandler_ValidationErrors(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello","chunk_length":50}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	newTestRouter(&mockBackend{}).ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "{\"detail\":\"chunk_length must be between 100 and 300\"}\n", rr.Body.String())
}

func TestTTSHandler_StreamingNonWavError(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello","streaming":true,"format":"mp3"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	newTestRouter(&mockBackend{}).ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "{\"detail\":\"Streaming only supports WAV format\"}\n", rr.Body.String())
}

func TestTTSHandler_BackendSuccess(t *testing.T) {
	body := bytes.NewBufferString(`{"text":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	backend := &mockBackend{
		ttsFunc: func(ctx context.Context, req *schema.ServeTTSRequest) ([]byte, string, error) {
			return []byte("audio"), "wav", nil
		},
	}

	newTestRouter(backend).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []byte("audio"), rr.Body.Bytes())
	assert.Equal(t, "audio/wav", rr.Header().Get("Content-Type"))
}

func TestAuthMiddleware_NoKeyConfigured(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := AuthMiddleware("")(next)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
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

	require.Equal(t, http.StatusOK, rr.Code)
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

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "{\"detail\":\"Invalid token\"}\n", rr.Body.String())
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := AuthMiddleware("secret")(next)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "{\"detail\":\"Invalid token\"}\n", rr.Body.String())
}

func TestWriteError_MatchesUpstreamFormat(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusBadRequest, "something went wrong")

	assert.Equal(t, "{\"detail\":\"something went wrong\"}\n", rr.Body.String())
}
