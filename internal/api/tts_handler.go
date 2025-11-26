package api

import (
    "context"
"encoding/json"
"errors"
"log"
"net/http"
"time"

    "github.com/username/fish-speech-go/internal/backend"
    "github.com/username/fish-speech-go/internal/queue"
    "github.com/username/fish-speech-go/internal/streaming"
)

// TTSRequest captures the public API payload for /v1/tts.
type TTSRequest struct {
    Text              string              `json:"text"`
    ReferenceID       *string             `json:"reference_id,omitempty"`
    References        []backend.Reference `json:"references,omitempty"`
    MaxNewTokens      int                 `json:"max_new_tokens,omitempty"`
    ChunkLength       int                 `json:"chunk_length,omitempty"`
    TopP              float64             `json:"top_p,omitempty"`
    Temperature       float64             `json:"temperature,omitempty"`
    RepetitionPenalty float64             `json:"repetition_penalty,omitempty"`
    Format            string              `json:"format,omitempty"`
    Streaming         bool                `json:"streaming,omitempty"`
    Seed              *int                `json:"seed,omitempty"`
}

// ErrorResponse mirrors the structured error shape used by the API.
type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

// ErrorDetail provides machine-readable error codes and a human-friendly message.
type ErrorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// TTSBackend streams synthesized audio to the HTTP response.
type TTSBackend interface {
    StreamTTS(ctx context.Context, req *backend.Request, w http.ResponseWriter) error
}

// TTSHandler wraps the TTS streaming path with concurrency limits and optional queueing.
type TTSHandler struct {
    chunker *streaming.Chunker
    backend TTSBackend
    queue   *queue.Manager
    logger  *log.Logger
    limits  Limits
}

// Limits defines validation constraints for the handler.
type Limits struct {
    MaxTextLength int
}

// NewTTSHandler constructs a TTSHandler guarded by the provided components.
func NewTTSHandler(chunker *streaming.Chunker, backend TTSBackend, queueMgr *queue.Manager) *TTSHandler {
    return &TTSHandler{
        chunker: chunker,
        backend: backend,
        queue:   queueMgr,
        logger:  log.Default(),
        limits: Limits{
            MaxTextLength: 10000,
        },
    }
}

// ServeHTTP handles POST /v1/tts requests and ensures streaming respects guardrails.
func (h *TTSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    if h.chunker == nil {
        h.writeError(w, http.StatusInternalServerError, "missing chunker")
        return
    }

    var payload TTSRequest
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid JSON payload")
        return
    }

    if err := h.validate(payload); err != nil {
        h.writeError(w, http.StatusBadRequest, err.Error())
        return
    }

    backendReq := &backend.Request{
        Text:              payload.Text,
        ReferenceID:       payload.ReferenceID,
        References:        payload.References,
        MaxNewTokens:      payload.MaxNewTokens,
        ChunkLength:       payload.ChunkLength,
        TopP:              payload.TopP,
        Temperature:       payload.Temperature,
        RepetitionPenalty: payload.RepetitionPenalty,
        Format:            payload.Format,
        Streaming:         true,
        Seed:              payload.Seed,
    }

    ctx := r.Context()

releaseQueue := func() {}
if h.queue != nil {
queueCtx, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
var err error
releaseQueue, err = h.queue.Acquire(queueCtx)
cancel()
if err != nil {
h.writeError(w, http.StatusServiceUnavailable, "queue is full")
return
}
}
defer releaseQueue()

    err := h.chunker.Stream(ctx, func(ctx context.Context) error {
        if h.backend == nil {
            return errors.New("backend not configured")
        }
        h.writeStreamingHeaders(w)
        return h.backend.StreamTTS(ctx, backendReq, w)
    })

    if err != nil {
        h.writeChunkerError(w, err)
        return
    }
}

func (h *TTSHandler) validate(req TTSRequest) error {
    if req.Text == "" {
        return errors.New("text is required")
    }
    if h.limits.MaxTextLength > 0 && len(req.Text) > h.limits.MaxTextLength {
        return errors.New("text exceeds maximum length")
    }
    if req.Format == "" {
        req.Format = "wav"
    }
    return nil
}

func (h *TTSHandler) writeStreamingHeaders(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "audio/wav")
    w.Header().Set("Transfer-Encoding", "chunked")

    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }
}

func (h *TTSHandler) writeChunkerError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, streaming.ErrLimitExceeded):
        h.writeError(w, http.StatusServiceUnavailable, streaming.ErrLimitExceeded.Message)
    case errors.Is(err, streaming.ErrAcquireTimeout):
        h.writeError(w, http.StatusGatewayTimeout, streaming.ErrAcquireTimeout.Message)
    default:
        h.writeError(w, http.StatusBadGateway, err.Error())
    }
}

func (h *TTSHandler) writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    _ = json.NewEncoder(w).Encode(ErrorResponse{
        Error: ErrorDetail{
            Code:    h.errorCodeFromStatus(status),
            Message: message,
        },
    })
}

func (h *TTSHandler) errorCodeFromStatus(status int) string {
    switch status {
    case http.StatusServiceUnavailable:
        return string(streaming.ChunkerLimitExceeded)
    case http.StatusGatewayTimeout:
        return string(streaming.ChunkerAcquireTimeout)
    default:
        return http.StatusText(status)
    }
}
