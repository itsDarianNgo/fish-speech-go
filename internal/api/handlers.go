package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// ChunkerErrorCode identifies the category of chunker failure.
type ChunkerErrorCode string

const (
	ChunkerErrorLimitExceeded ChunkerErrorCode = "limit_exceeded"
	ChunkerErrorTimeout       ChunkerErrorCode = "acquire_timeout"
)

// ChunkerError wraps errors returned by the chunker so they can be mapped to HTTP responses.
type ChunkerError struct {
	Code ChunkerErrorCode
	Err  error
}

func (e ChunkerError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e ChunkerError) Unwrap() error {
	return e.Err
}

var (
	// ErrLimitExceeded indicates the chunker refused a request because capacity is exhausted.
	ErrLimitExceeded = errors.New("max concurrent limit exceeded")
	// ErrAcquireTimeout indicates the chunker could not acquire capacity before the timeout.
	ErrAcquireTimeout = errors.New("acquire timeout exceeded")
)

// Chunker reserves capacity for handling requests.
type Chunker interface {
	Acquire(ctx context.Context) (release func(), err error)
}

// Producer performs the work once a chunk has been acquired.
type Producer interface {
	Produce(ctx context.Context) error
}

// ChunkRequestHandler wraps an HTTP handler with chunking limits.
type ChunkRequestHandler struct {
	chunker  Chunker
	producer Producer
}

// NewChunkRequestHandler constructs a handler that enforces chunker limits before invoking the producer.
func NewChunkRequestHandler(chunker Chunker, producer Producer) *ChunkRequestHandler {
	return &ChunkRequestHandler{chunker: chunker, producer: producer}
}

// ServeHTTP enforces chunker limits and delegates to the producer.
func (h *ChunkRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	release, err := h.chunker.Acquire(r.Context())
	if err != nil {
		h.writeChunkerError(w, err)
		return
	}
	defer release()

	if err := h.producer.Produce(r.Context()); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SemaphoreChunker implements Chunker with a buffered channel.
type SemaphoreChunker struct {
	sem            chan struct{}
	acquireTimeout time.Duration
}

// NewSemaphoreChunker constructs a chunker with the provided capacity and optional acquire timeout.
func NewSemaphoreChunker(maxConcurrent int, acquireTimeout time.Duration) *SemaphoreChunker {
	return &SemaphoreChunker{
		sem:            make(chan struct{}, maxConcurrent),
		acquireTimeout: acquireTimeout,
	}
}

// Acquire reserves a slot or returns a ChunkerError if limits are exceeded.
func (c *SemaphoreChunker) Acquire(ctx context.Context) (func(), error) {
	select {
	case c.sem <- struct{}{}:
		return func() { <-c.sem }, nil
	default:
		// No immediate capacity, try waiting if a timeout is configured.
	}

	if c.acquireTimeout == 0 {
		return nil, ChunkerError{Code: ChunkerErrorLimitExceeded, Err: ErrLimitExceeded}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.acquireTimeout)
	defer cancel()

	select {
	case c.sem <- struct{}{}:
		return func() { <-c.sem }, nil
	case <-timeoutCtx.Done():
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			return nil, ChunkerError{Code: ChunkerErrorTimeout, Err: ErrAcquireTimeout}
		}
		return nil, timeoutCtx.Err()
	}
}

func (h *ChunkRequestHandler) writeChunkerError(w http.ResponseWriter, err error) {
	var chunkErr ChunkerError
	if errors.As(err, &chunkErr) {
		switch chunkErr.Code {
		case ChunkerErrorLimitExceeded:
			h.writeJSON(w, http.StatusServiceUnavailable, errorPayload(chunkErr.Code, chunkErr.Error()))
			return
		case ChunkerErrorTimeout:
			h.writeJSON(w, http.StatusGatewayTimeout, errorPayload(chunkErr.Code, chunkErr.Error()))
			return
		}
	}

	h.writeError(w, http.StatusInternalServerError, err.Error())
}

func (h *ChunkRequestHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorPayload("internal_error", message))
}

func errorPayload(code ChunkerErrorCode, message string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

func (h *ChunkRequestHandler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
