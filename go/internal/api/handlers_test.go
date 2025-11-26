package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fish-speech-go/fish-speech-go/internal/config"
	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// Mock backend for testing
type mockBackend struct {
	healthErr       error
	ttsResponse     []byte
	ttsErr          error
	vqganEncodeResp *schema.ServeVQGANEncodeResponse
	vqganEncodeErr  error
	vqganDecodeResp *schema.ServeVQGANDecodeResponse
	vqganDecodeErr  error
	addRefResp      *schema.AddReferenceResponse
	addRefErr       error
	listRefResp     *schema.ListReferencesResponse
	listRefErr      error
	deleteRefResp   *schema.DeleteReferenceResponse
	deleteRefErr    error
}

func (m *mockBackend) Health(ctx context.Context) error {
	return m.healthErr
}

func (m *mockBackend) TTS(ctx context.Context, req *schema.ServeTTSRequest) ([]byte, string, error) {
	if m.ttsErr != nil {
		return nil, "", m.ttsErr
	}
	return m.ttsResponse, "wav", nil
}

func (m *mockBackend) TTSStream(ctx context.Context, req *schema.ServeTTSRequest) (io.ReadCloser, error) {
	if m.ttsErr != nil {
		return nil, m.ttsErr
	}
	return io.NopCloser(bytes.NewReader(m.ttsResponse)), nil
}

func (m *mockBackend) VQGANEncode(ctx context.Context, req *schema.ServeVQGANEncodeRequest) (*schema.ServeVQGANEncodeResponse, error) {
	return m.vqganEncodeResp, m.vqganEncodeErr
}

func (m *mockBackend) VQGANDecode(ctx context.Context, req *schema.ServeVQGANDecodeRequest) (*schema.ServeVQGANDecodeResponse, error) {
	return m.vqganDecodeResp, m.vqganDecodeErr
}

func (m *mockBackend) AddReference(ctx context.Context, req *schema.AddReferenceRequest) (*schema.AddReferenceResponse, error) {
	return m.addRefResp, m.addRefErr
}

func (m *mockBackend) ListReferences(ctx context.Context) (*schema.ListReferencesResponse, error) {
	return m.listRefResp, m.listRefErr
}

func (m *mockBackend) DeleteReference(ctx context.Context, id string) (*schema.DeleteReferenceResponse, error) {
	return m.deleteRefResp, m.deleteRefErr
}

// Health tests
func TestHealthGet_Basic(t *testing.T) {
	h := NewHandler(&mockBackend{}, testConfig(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()

	h.HandleHealthGet(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp["status"])
}

func TestHealthGet_Detailed(t *testing.T) {
	h := NewHandler(&mockBackend{}, testConfig(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/health?detailed=true", nil)
	w := httptest.NewRecorder()

	h.HandleHealthGet(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp.Status)
	assert.NotNil(t, resp.Backend)
	assert.Equal(t, "healthy", resp.Backend.Status)
}

func TestHealthGet_Detailed_BackendUnhealthy(t *testing.T) {
	mock := &mockBackend{healthErr: errors.New("connection refused")}
	h := NewHandler(mock, testConfig(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/health?detailed=true", nil)
	w := httptest.NewRecorder()

	h.HandleHealthGet(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "unhealthy", resp.Backend.Status)
}

// VQGAN tests
func TestVQGANEncode_Success(t *testing.T) {
	mock := &mockBackend{vqganEncodeResp: &schema.ServeVQGANEncodeResponse{Tokens: [][][]int{{{1, 2, 3}}}}}
	h := NewHandler(mock, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.ServeVQGANEncodeRequest{Audios: [][]byte{[]byte("fake audio")}})

	req := httptest.NewRequest(http.MethodPost, "/v1/vqgan/encode", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleVQGANEncode(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "msgpack")
}

func TestVQGANEncode_NoAudio(t *testing.T) {
	h := NewHandler(&mockBackend{}, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.ServeVQGANEncodeRequest{Audios: [][]byte{}})

	req := httptest.NewRequest(http.MethodPost, "/v1/vqgan/encode", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleVQGANEncode(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVQGANDecode_Success(t *testing.T) {
	mock := &mockBackend{vqganDecodeResp: &schema.ServeVQGANDecodeResponse{Audios: [][]byte{[]byte("audio")}}}
	h := NewHandler(mock, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.ServeVQGANDecodeRequest{Tokens: [][][]int{{{1, 2}}}})
	req := httptest.NewRequest(http.MethodPost, "/v1/vqgan/decode", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleVQGANDecode(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "msgpack")
}

func TestVQGANDecode_NoTokens(t *testing.T) {
	h := NewHandler(&mockBackend{}, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.ServeVQGANDecodeRequest{})
	req := httptest.NewRequest(http.MethodPost, "/v1/vqgan/decode", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleVQGANDecode(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Reference tests
func TestAddReference_Success(t *testing.T) {
	mock := &mockBackend{addRefResp: &schema.AddReferenceResponse{Success: true, Message: "Reference added successfully", ReferenceID: "test-voice"}}
	h := NewHandler(mock, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.AddReferenceRequest{ID: "test-voice", Audio: []byte("fake audio data"), Text: "This is a test transcript"})

	req := httptest.NewRequest(http.MethodPost, "/v1/references/add", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleAddReference(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp schema.AddReferenceResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "test-voice", resp.ReferenceID)
}

func TestAddReference_InvalidID(t *testing.T) {
	h := NewHandler(&mockBackend{}, testConfig(), testLogger())

	testCases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"invalid chars", "test@voice!"},
		{"too long", string(make([]byte, 300))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(schema.AddReferenceRequest{ID: tc.id, Audio: []byte("fake audio"), Text: "transcript"})

			req := httptest.NewRequest(http.MethodPost, "/v1/references/add", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleAddReference(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestListReferences_Success(t *testing.T) {
	mock := &mockBackend{listRefResp: &schema.ListReferencesResponse{Success: true, ReferenceIDs: []string{"voice-1", "voice-2"}, Message: "Success"}}
	h := NewHandler(mock, testConfig(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/references", nil)
	w := httptest.NewRecorder()

	h.HandleListReferences(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp schema.ListReferencesResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.ReferenceIDs, 2)
}

func TestDeleteReference_Success(t *testing.T) {
	mock := &mockBackend{deleteRefResp: &schema.DeleteReferenceResponse{Success: true, Message: "Reference deleted successfully", ReferenceID: "test-voice"}}
	h := NewHandler(mock, testConfig(), testLogger())

	req := httptest.NewRequest(http.MethodDelete, "/v1/references/test-voice", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-voice")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.HandleDeleteReference(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Backend error handling tests
func TestTTS_BackendTimeout(t *testing.T) {
	mock := &mockBackend{ttsErr: context.DeadlineExceeded}
	h := NewHandler(mock, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.ServeTTSRequest{Text: "Hello"})
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTTS(w, req)

	assert.Equal(t, http.StatusGatewayTimeout, w.Code)

	var resp schema.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Request timeout", resp.Detail)
}

func TestTTS_BackendUnavailable(t *testing.T) {
	mock := &mockBackend{ttsErr: errors.New("connection refused")}
	h := NewHandler(mock, testConfig(), testLogger())

	reqBody, _ := json.Marshal(schema.ServeTTSRequest{Text: "Hello"})
	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTTS(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

// Authentication middleware tests
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

// Helper functions
func testConfig() *config.Config {
	return &config.Config{Limits: config.LimitsConfig{MaxTextLength: 10000}}
}

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}
