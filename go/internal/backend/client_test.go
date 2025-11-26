package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fish-speech-go/fish-speech-go/internal/config"
	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

func TestEncodeTTSRequest(t *testing.T) {
	req := &schema.ServeTTSRequest{
		Text:              "Hello world",
		ChunkLength:       200,
		Format:            "wav",
		Temperature:       0.8,
		TopP:              0.8,
		MaxNewTokens:      10,
		RepetitionPenalty: 1.1,
	}

	data, err := EncodeTTSRequest(req)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = DecodeMsgpack(data, &decoded)
	require.NoError(t, err)

	assert.Contains(t, decoded, "text")
	assert.Contains(t, decoded, "chunk_length")
	assert.Contains(t, decoded, "max_new_tokens")
	assert.Contains(t, decoded, "top_p")
	assert.Contains(t, decoded, "repetition_penalty")
	assert.Contains(t, decoded, "temperature")
}

func TestTTS_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/tts", r.URL.Path)
		assert.Equal(t, "application/msgpack", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "audio/wav")
		w.Write([]byte("fake audio data"))
	}))
	defer mockServer.Close()

	client := NewBackendClient(&config.BackendConfig{URL: mockServer.URL, Timeout: 10 * time.Second})

	audio, format, err := client.TTS(context.Background(), &schema.ServeTTSRequest{Text: "Hello"})

	require.NoError(t, err)
	assert.Equal(t, "wav", format)
	assert.Equal(t, []byte("fake audio data"), audio)
}

func TestTTS_BackendError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail": "Internal error"}`))
	}))
	defer mockServer.Close()

	client := NewBackendClient(&config.BackendConfig{URL: mockServer.URL, Timeout: 10 * time.Second})

	_, _, err := client.TTS(context.Background(), &schema.ServeTTSRequest{Text: "Hello"})

	require.Error(t, err)
	assert.True(t, IsBackendError(err))
}

func TestTTS_Timeout(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer mockServer.Close()

	client := NewBackendClient(&config.BackendConfig{URL: mockServer.URL, Timeout: 100 * time.Millisecond})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := client.TTS(ctx, &schema.ServeTTSRequest{Text: "Hello"})

	require.Error(t, err)
}

func TestHealth_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer mockServer.Close()

	client := NewBackendClient(&config.BackendConfig{URL: mockServer.URL, Timeout: 10 * time.Second})

	err := client.Health(context.Background())
	require.NoError(t, err)
}

func TestHealth_Failure(t *testing.T) {
	client := NewBackendClient(&config.BackendConfig{URL: "http://localhost:9999", Timeout: 1 * time.Second})

	err := client.Health(context.Background())
	require.Error(t, err)
}
