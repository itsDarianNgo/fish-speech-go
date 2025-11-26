package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"fish-speech-go/internal/backend"
	"fish-speech-go/internal/streaming"
)

const (
	maxRequestBodyBytes  int64 = 1 << 20 // 1 MiB
	maxTextLength              = 2048
	maxReferenceIDLength       = 128
	defaultAudioFormat         = "wav"
)

type ttsRequest struct {
	Text        string   `json:"text"`
	ReferenceID string   `json:"reference_id"`
	Streaming   *bool    `json:"streaming"`
	Format      string   `json:"format"`
	TopP        *float64 `json:"top_p,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type ttsBackend interface {
	StreamTTS(ctx context.Context, payload backend.TTSRequest) (*http.Response, error)
}

// TTSHandler handles /v1/tts requests by validating input and streaming responses from the backend.
type TTSHandler struct {
	chunker *streaming.Chunker
	backend ttsBackend
}

// NewTTSHandler constructs a new handler guarded by the provided chunker and backed by the backend client.
func NewTTSHandler(chunker *streaming.Chunker, backend ttsBackend) *TTSHandler {
	return &TTSHandler{chunker: chunker, backend: backend}
}

// ServeHTTP validates the request and proxies it to the backend using the chunker's Stream guard.
func (h *TTSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only POST is supported")
		return
	}

	payload, err := h.parseRequest(w, r)
	if err != nil {
		return
	}

	streamErr := h.chunker.Stream(r.Context(), func(ctx context.Context) error {
		resp, err := h.backend.StreamTTS(ctx, payload)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if ct := resp.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.WriteHeader(resp.StatusCode)
		_, copyErr := io.Copy(w, resp.Body)
		return copyErr
	})

	if streamErr == nil {
		return
	}

	h.handleStreamError(w, streamErr)
}

func (h *TTSHandler) parseRequest(w http.ResponseWriter, r *http.Request) (backend.TTSRequest, error) {
	var payload ttsRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			h.writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds limit")
			return backend.TTSRequest{}, err
		}
		h.writeError(w, http.StatusBadRequest, "invalid_request", "failed to decode request payload")
		return backend.TTSRequest{}, err
	}

	if err := decoder.Decode(new(struct{})); err != io.EOF {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain a single JSON object")
		return backend.TTSRequest{}, fmt.Errorf("extra data after JSON payload")
	}

	if strings.TrimSpace(payload.Text) == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "text is required")
		return backend.TTSRequest{}, fmt.Errorf("text missing")
	}
	payload.Text = strings.TrimSpace(payload.Text)
	if len(payload.Text) > maxTextLength {
		h.writeError(w, http.StatusBadRequest, "limit_exceeded", fmt.Sprintf("text exceeds max length of %d", maxTextLength))
		return backend.TTSRequest{}, fmt.Errorf("text too long")
	}

	payload.ReferenceID = strings.TrimSpace(payload.ReferenceID)
	if len(payload.ReferenceID) > maxReferenceIDLength {
		h.writeError(w, http.StatusBadRequest, "limit_exceeded", fmt.Sprintf("reference_id exceeds max length of %d", maxReferenceIDLength))
		return backend.TTSRequest{}, fmt.Errorf("reference id too long")
	}

	if payload.Streaming == nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "streaming flag is required")
		return backend.TTSRequest{}, fmt.Errorf("streaming flag missing")
	}
	if !*payload.Streaming {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "streaming must be enabled")
		return backend.TTSRequest{}, fmt.Errorf("streaming disabled")
	}

	if payload.Format == "" {
		payload.Format = defaultAudioFormat
	}
	if payload.Format != "wav" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "unsupported audio format")
		return backend.TTSRequest{}, fmt.Errorf("unsupported format")
	}

	if payload.TopP != nil {
		if *payload.TopP <= 0 || *payload.TopP > 1 {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "top_p must be in (0, 1]")
			return backend.TTSRequest{}, fmt.Errorf("invalid top_p")
		}
	}

	if payload.Temperature != nil {
		if *payload.Temperature < 0 || *payload.Temperature > 2 {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "temperature must be between 0 and 2")
			return backend.TTSRequest{}, fmt.Errorf("invalid temperature")
		}
	}

	return backend.TTSRequest{
		Text:        payload.Text,
		ReferenceID: payload.ReferenceID,
		Streaming:   *payload.Streaming,
		Format:      payload.Format,
		TopP:        payload.TopP,
		Temperature: payload.Temperature,
	}, nil
}

func (h *TTSHandler) handleStreamError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, streaming.ErrAcquireTimeout):
		h.writeError(w, http.StatusGatewayTimeout, "acquire_timeout", "concurrent request limit reached")
	case errors.Is(err, context.DeadlineExceeded):
		h.writeError(w, http.StatusGatewayTimeout, "timeout", "request timed out")
	default:
		var httpErr backend.HTTPError
		if errors.As(err, &httpErr) {
			h.writeError(w, http.StatusBadGateway, "backend_error", httpErr.Error())
			return
		}

		h.writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

func (h *TTSHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
