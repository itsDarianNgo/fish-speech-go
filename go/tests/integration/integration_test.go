//go:build integration
// +build integration

// Integration tests require a running Fish-Speech backend.
// Run with: go test -tags=integration ./tests/integration/...
//
// Environment variables:
//   FISH_SERVER_URL - Go server URL (default: http://localhost:8080)
//   FISH_BACKEND_URL - Python backend URL for direct testing (default: http://localhost:8081)

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

var (
	serverURL  string
	backendURL string
	httpClient *http.Client
)

func TestMain(m *testing.M) {
	serverURL = os.Getenv("FISH_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	backendURL = os.Getenv("FISH_BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8081"
	}

	httpClient = &http.Client{
		Timeout: 120 * time.Second,
	}

	if !waitForServer(serverURL, 30*time.Second) {
		fmt.Fprintf(os.Stderr, "Server at %s not ready\n", serverURL)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/v1/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

// =============================================================================
// Health Endpoint Tests (TC-001, TC-002)
// =============================================================================

func TestHealthGet(t *testing.T) {
	resp, err := httpClient.Get(serverURL + "/v1/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]string
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "ok", health["status"])
}

func TestHealthPost(t *testing.T) {
	resp, err := httpClient.Post(serverURL+"/v1/health", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]string
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "ok", health["status"])
}

func TestHealthDetailed(t *testing.T) {
	resp, err := httpClient.Get(serverURL + "/v1/health?detailed=true")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "ok", health["status"])

	backend, ok := health["backend"].(map[string]interface{})
	require.True(t, ok, "backend field should be present")
	assert.Equal(t, "healthy", backend["status"])
}

// =============================================================================
// TTS Endpoint Tests (TC-003 through TC-008)
// =============================================================================

func TestTTSBasicJSON(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:   "Hello, this is a test.",
		Format: "wav",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "audio/wav", resp.Header.Get("Content-Type"))

	audio, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.True(t, len(audio) > 44, "Audio should be longer than WAV header")
	assert.Equal(t, "RIFF", string(audio[0:4]))
	assert.Equal(t, "WAVE", string(audio[8:12]))

	t.Logf("Generated %d bytes of audio", len(audio))
}

func TestTTSWithParameters(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:              "Testing with custom parameters.",
		Format:            "wav",
		ChunkLength:       200,
		Temperature:       0.7,
		TopP:              0.9,
		RepetitionPenalty: 1.2,
		MaxNewTokens:      512,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	audio, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.True(t, len(audio) > 1000, "Audio should be generated")
}

func TestTTSMP3Format(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:   "Testing MP3 format.",
		Format: "mp3",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "audio/mpeg")
}

func TestTTSStreaming(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:      "Testing streaming output with a longer sentence.",
		Format:    "wav",
		Streaming: true,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "chunked", resp.Header.Get("Transfer-Encoding"))

	audio, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.True(t, len(audio) > 44, "Should receive audio data")

	t.Logf("Streamed %d bytes of audio", len(audio))
}

// =============================================================================
// Validation Error Tests (TC-009 through TC-015)
// =============================================================================

func TestTTSStreamingNonWavError(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:      "Test",
		Format:    "mp3",
		Streaming: true,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errResp)
	assert.Equal(t, "Streaming only supports WAV format", errResp["detail"])
}

func TestTTSInvalidChunkLength(t *testing.T) {
	testCases := []struct {
		name        string
		chunkLength int
	}{
		{name: "too low", chunkLength: 50},
		{name: "too high", chunkLength: 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"text":         "Test",
				"chunk_length": tc.chunkLength,
			}
			body, _ := json.Marshal(reqBody)

			resp, err := httpClient.Post(
				serverURL+"/v1/tts",
				"application/json",
				bytes.NewReader(body),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestTTSInvalidTopP(t *testing.T) {
	reqBody := map[string]interface{}{
		"text":  "Test",
		"top_p": 1.5,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTTSInvalidTemperature(t *testing.T) {
	reqBody := map[string]interface{}{
		"text":        "Test",
		"temperature": 0.05,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// =============================================================================
// Content-Type Tests (TC-019)
// =============================================================================

func TestTTSUnsupportedContentType(t *testing.T) {
	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"text/plain",
		bytes.NewReader([]byte("Hello world")),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
}

// =============================================================================
// Reference Management Tests (TC-022 through TC-024)
// =============================================================================

func TestReferencesList(t *testing.T) {
	resp, err := httpClient.Get(serverURL + "/v1/references")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var refs schema.ListReferencesResponse
	err = json.NewDecoder(resp.Body).Decode(&refs)
	require.NoError(t, err)

	assert.True(t, refs.Success)
	t.Logf("Found %d references", len(refs.ReferenceIDs))
}

// =============================================================================
// Comparison Tests - Go Server vs Python Server
// =============================================================================

func TestComparisonHealthEndpoint(t *testing.T) {
	goResp, err := httpClient.Get(serverURL + "/v1/health")
	require.NoError(t, err)
	defer goResp.Body.Close()

	pyResp, err := httpClient.Get(backendURL + "/v1/health")
	require.NoError(t, err)
	defer pyResp.Body.Close()

	var goHealth, pyHealth map[string]string
	json.NewDecoder(goResp.Body).Decode(&goHealth)
	json.NewDecoder(pyResp.Body).Decode(&pyHealth)

	assert.Equal(t, goHealth["status"], pyHealth["status"])
}

func TestComparisonTTSProducesAudio(t *testing.T) {
	seed := 42
	reqBody := schema.ServeTTSRequest{
		Text:        "Comparison test.",
		Format:      "wav",
		Temperature: 0.7,
		TopP:        0.8,
		Seed:        &seed,
	}
	body, _ := json.Marshal(reqBody)

	goResp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer goResp.Body.Close()

	goAudio, _ := io.ReadAll(goResp.Body)

	pyResp, err := httpClient.Post(
		backendURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer pyResp.Body.Close()

	pyAudio, _ := io.ReadAll(pyResp.Body)

	assert.True(t, len(goAudio) > 44, "Go server should produce audio")
	assert.True(t, len(pyAudio) > 44, "Python server should produce audio")

	assert.Equal(t, "RIFF", string(goAudio[0:4]))
	assert.Equal(t, "RIFF", string(pyAudio[0:4]))

	t.Logf("Go server: %d bytes, Python server: %d bytes", len(goAudio), len(pyAudio))
}

// =============================================================================
// Performance Tests
// =============================================================================

func TestPerformanceTTSLatency(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:   "Short test.",
		Format: "wav",
	}
	body, _ := json.Marshal(reqBody)

	start := time.Now()

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	io.ReadAll(resp.Body)

	elapsed := time.Since(start)
	t.Logf("TTS request completed in %v", elapsed)

	assert.Less(t, elapsed, 30*time.Second, "TTS should complete within 30 seconds")
}

func TestPerformanceStreamingFirstByte(t *testing.T) {
	reqBody := schema.ServeTTSRequest{
		Text:      "Testing time to first byte with streaming.",
		Format:    "wav",
		Streaming: true,
	}
	body, _ := json.Marshal(reqBody)

	start := time.Now()

	resp, err := httpClient.Post(
		serverURL+"/v1/tts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	firstChunk := make([]byte, 1024)
	n, err := resp.Body.Read(firstChunk)
	require.NoError(t, err)

	timeToFirstByte := time.Since(start)
	t.Logf("Time to first byte: %v (read %d bytes)", timeToFirstByte, n)

	io.ReadAll(resp.Body)

	assert.Less(t, timeToFirstByte, 5*time.Second, "First byte should arrive within 5 seconds")
}
