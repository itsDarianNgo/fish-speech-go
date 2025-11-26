package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/fish-speech-go/fish-speech-go/internal/backend"
	"github.com/fish-speech-go/fish-speech-go/internal/config"
	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// Handler encapsulates dependencies for HTTP handlers.
type Handler struct {
	backend backend.Client
	config  *config.Config
	logger  zerolog.Logger
}

// NewHandler constructs a Handler.
func NewHandler(backend backend.Client, cfg *config.Config, logger zerolog.Logger) *Handler {
	return &Handler{backend: backend, config: cfg, logger: logger}
}

// Health Handlers
func (h *Handler) HandleHealthGet(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) HandleHealthPost(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TTS Handler
type parseFunc func(r *http.Request) (*schema.ServeTTSRequest, error)

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

		if errors.Is(err, context.DeadlineExceeded) {
			WriteError(w, http.StatusGatewayTimeout, "Request timeout")
			return
		}

		if errors.Is(err, backend.ErrBackendTimeout) {
			WriteError(w, http.StatusGatewayTimeout, "Request timeout")
			return
		}

		WriteError(w, http.StatusBadGateway, "Backend service unavailable")
		return
	}

	WriteAudio(w, format, audioData)
}

func (h *Handler) handleStreamingTTS(w http.ResponseWriter, r *http.Request, req *schema.ServeTTSRequest) {
	stream, err := h.backend.TTSStream(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("TTS streaming backend error")
		WriteError(w, http.StatusBadGateway, "Backend service unavailable")
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

func (h *Handler) handleParseError(w http.ResponseWriter, err error) {
	if httpErr, ok := IsHTTPError(err); ok {
		WriteError(w, httpErr.Status, httpErr.Message)
		return
	}

	WriteError(w, http.StatusBadRequest, "Invalid request")
}

// Stub handlers
func (h *Handler) HandleVQGANEncode(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func (h *Handler) HandleVQGANDecode(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func (h *Handler) HandleAddReference(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func (h *Handler) HandleListReferences(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func (h *Handler) HandleDeleteReference(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}
