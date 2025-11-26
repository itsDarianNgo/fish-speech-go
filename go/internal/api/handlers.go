package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/fish-speech-go/fish-speech-go/internal/backend"
	"github.com/fish-speech-go/fish-speech-go/internal/config"
	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// HealthResponse represents the health payload including optional backend status.
type HealthResponse struct {
	Status  string         `json:"status"`
	Backend *BackendHealth `json:"backend,omitempty"`
}

// BackendHealth captures backend health diagnostics.
type BackendHealth struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Handler encapsulates dependencies for HTTP handlers.
type Handler struct {
	backend backend.Backend
	config  *config.Config
	logger  zerolog.Logger
}

// NewHandler constructs a Handler.
func NewHandler(backend backend.Backend, cfg *config.Config, logger zerolog.Logger) *Handler {
	return &Handler{backend: backend, config: cfg, logger: logger}
}

// Health Handlers
func (h *Handler) HandleHealthGet(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "ok"}

	if r.URL.Query().Get("detailed") == "true" {
		start := time.Now()
		err := h.backend.Health(r.Context())
		latency := time.Since(start).Milliseconds()

		if err != nil {
			response.Backend = &BackendHealth{Status: "unhealthy", LatencyMs: latency, Error: err.Error()}
		} else {
			response.Backend = &BackendHealth{Status: "healthy", LatencyMs: latency}
		}
	}

	WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) HandleHealthPost(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TTS Handler
func (h *Handler) HandleTTS(w http.ResponseWriter, r *http.Request) {
	req, err := ParseTTSRequest(r)
	if err != nil {
		h.handleParseError(w, err)
		return
	}

	if h.config.Limits.MaxTextLength > 0 && len(req.Text) > h.config.Limits.MaxTextLength {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("Text is too long, max length is %d", h.config.Limits.MaxTextLength))
		return
	}

	if req.Streaming && req.Format != "wav" {
		WriteError(w, http.StatusBadRequest, "Streaming only supports WAV format")
		return
	}

	if req.Streaming {
		h.handleStreamingTTS(w, r, req)
		return
	}

	h.handleNonStreamingTTS(w, r, req)
}

func (h *Handler) handleNonStreamingTTS(w http.ResponseWriter, r *http.Request, req *schema.ServeTTSRequest) {
	audioData, format, err := h.backend.TTS(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("TTS backend error")
		h.handleBackendError(w, err)
		return
	}

	WriteAudio(w, format, audioData)
}

func (h *Handler) handleStreamingTTS(w http.ResponseWriter, r *http.Request, req *schema.ServeTTSRequest) {
	stream, err := h.backend.TTSStream(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("TTS streaming backend error")
		h.handleBackendError(w, err)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Disposition", "attachment; filename=audio.wav")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr == nil {
				flusher.Flush()
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			h.logger.Error().Err(err).Msg("Error streaming audio")
			break
		}
	}
}

// VQGAN handlers
func (h *Handler) HandleVQGANEncode(w http.ResponseWriter, r *http.Request) {
	var req schema.ServeVQGANEncodeRequest
	if err := ParseRequestBody(r, &req); err != nil {
		h.handleParseError(w, err)
		return
	}

	if len(req.Audios) == 0 {
		WriteError(w, http.StatusBadRequest, "No audio provided")
		return
	}

	resp, err := h.backend.VQGANEncode(r.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("VQGAN encode error")
		h.handleBackendError(w, err)
		return
	}

	WriteMsgpack(w, http.StatusOK, resp)
}

func (h *Handler) HandleVQGANDecode(w http.ResponseWriter, r *http.Request) {
	var req schema.ServeVQGANDecodeRequest
	if err := ParseRequestBody(r, &req); err != nil {
		h.handleParseError(w, err)
		return
	}

	if len(req.Tokens) == 0 {
		WriteError(w, http.StatusBadRequest, "No tokens provided")
		return
	}

	resp, err := h.backend.VQGANDecode(r.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("VQGAN decode error")
		h.handleBackendError(w, err)
		return
	}

	WriteMsgpack(w, http.StatusOK, resp)
}

// Reference handlers
func (h *Handler) HandleAddReference(w http.ResponseWriter, r *http.Request) {
	var req schema.AddReferenceRequest

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			WriteError(w, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		req.ID = r.FormValue("id")
		req.Text = r.FormValue("text")

		file, _, err := r.FormFile("audio")
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Audio file required")
			return
		}
		defer file.Close()

		audioBytes, err := io.ReadAll(file)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Failed to read audio file")
			return
		}
		req.Audio = audioBytes
	} else {
		if err := ParseRequestBody(r, &req); err != nil {
			h.handleParseError(w, err)
			return
		}
	}

	if err := validateAddReferenceRequest(&req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.backend.AddReference(r.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Add reference error")
		h.handleBackendError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleListReferences(w http.ResponseWriter, r *http.Request) {
	resp, err := h.backend.ListReferences(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("List references error")
		h.handleBackendError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleDeleteReference(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "Reference ID required")
		return
	}

	resp, err := h.backend.DeleteReference(r.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Delete reference error")
		h.handleBackendError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func validateAddReferenceRequest(req *schema.AddReferenceRequest) error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if len(req.ID) > 255 {
		return errors.New("id must be 255 characters or less")
	}

	validID := regexp.MustCompile(`^[a-zA-Z0-9\-_ ]+$`)
	if !validID.MatchString(req.ID) {
		return errors.New("id must contain only alphanumeric characters, dashes, underscores, and spaces")
	}

	if len(req.Audio) == 0 {
		return errors.New("audio is required")
	}

	if req.Text == "" {
		return errors.New("text is required")
	}

	return nil
}

func (h *Handler) handleBackendError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		WriteError(w, http.StatusGatewayTimeout, "Request timeout")
		return
	}
	if errors.Is(err, context.Canceled) {
		WriteError(w, http.StatusBadRequest, "Request cancelled")
		return
	}

	if errors.Is(err, backend.ErrBackendTimeout) {
		WriteError(w, http.StatusGatewayTimeout, "Request timeout")
		return
	}

	var backendErr *backend.BackendError
	if errors.As(err, &backendErr) {
		switch backendErr.StatusCode {
		case http.StatusBadRequest:
			WriteError(w, http.StatusBadRequest, backendErr.Message)
		case http.StatusNotFound:
			WriteError(w, http.StatusNotFound, backendErr.Message)
		default:
			WriteError(w, http.StatusBadGateway, "Backend error")
		}
		return
	}

	WriteError(w, http.StatusBadGateway, "Backend service unavailable")
}

func (h *Handler) handleParseError(w http.ResponseWriter, err error) {
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		WriteError(w, parseErr.Status, parseErr.Message)
		return
	}

	WriteError(w, http.StatusBadRequest, "Invalid request body")
}
