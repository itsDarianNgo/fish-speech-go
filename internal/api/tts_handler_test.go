package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/username/fish-speech-go/internal/backend"
	"github.com/username/fish-speech-go/internal/queue"
	"github.com/username/fish-speech-go/internal/streaming"
)

func TestTTSHandler_LimitExceeded(t *testing.T) {
	chunker, err := streaming.NewChunker(streaming.Options{MaxConcurrent: 2})
	if err != nil {
		t.Fatalf("failed to create chunker: %v", err)
	}

	q, err := queue.NewManager(4)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	block := make(chan struct{})
	started := make(chan struct{}, 2)
	var wg sync.WaitGroup

	backend := &blockingBackend{started: started, block: block, done: &wg}
	handler := NewTTSHandler(chunker, backend, q)

	wg.Add(2)
	payload := []byte(`{"text":"hello"}`)

	for i := 0; i < 2; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(payload))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}()
	}

	for i := 0; i < 2; i++ {
		<-started
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	time.AfterFunc(30*time.Millisecond, cancel)

	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(payload)).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for limit exceeded, got %d", rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if resp.Error.Code != string(streaming.ChunkerLimitExceeded) {
		t.Fatalf("expected error code %q, got %q", streaming.ChunkerLimitExceeded, resp.Error.Code)
	}

	close(block)
	wg.Wait()
}

func TestTTSHandler_AcquireTimeout(t *testing.T) {
	chunker, err := streaming.NewChunker(streaming.Options{MaxConcurrent: 2, AcquireTimeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatalf("failed to create chunker: %v", err)
	}

	q, err := queue.NewManager(4)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	block := make(chan struct{})
	started := make(chan struct{}, 2)
	var wg sync.WaitGroup

	backend := &blockingBackend{started: started, block: block, done: &wg}
	handler := NewTTSHandler(chunker, backend, q)

	wg.Add(2)
	payload := []byte(`{"text":"hello"}`)

	for i := 0; i < 2; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(payload))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}()
	}

	for i := 0; i < 2; i++ {
		<-started
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 for acquire timeout, got %d", rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if resp.Error.Code != string(streaming.ChunkerAcquireTimeout) {
		t.Fatalf("expected error code %q, got %q", streaming.ChunkerAcquireTimeout, resp.Error.Code)
	}

	close(block)
	wg.Wait()
}

func TestTTSHandler_QueueFull(t *testing.T) {
	chunker, err := streaming.NewChunker(streaming.Options{MaxConcurrent: 1})
	if err != nil {
		t.Fatalf("failed to create chunker: %v", err)
	}

	q, err := queue.NewManager(1)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	// Fill queue slot.
	release, err := q.Acquire(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire queue slot: %v", err)
	}
	defer release()

	handler := NewTTSHandler(chunker, &noopBackend{}, q)

	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader([]byte(`{"text":"hello"}`)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when queue is full, got %d", rr.Code)
	}
}

func TestTTSHandler_InvalidPayload(t *testing.T) {
	chunker, _ := streaming.NewChunker(streaming.Options{MaxConcurrent: 1})
	q, _ := queue.NewManager(1)
	handler := NewTTSHandler(chunker, &noopBackend{}, q)

	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader([]byte(`{"text":""}`)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rr.Code)
	}
}

func TestTTSHandler_HappyPath(t *testing.T) {
	chunker, _ := streaming.NewChunker(streaming.Options{MaxConcurrent: 1})
	q, _ := queue.NewManager(1)

	backend := &recordingBackend{}
	handler := NewTTSHandler(chunker, backend, q)

	req := httptest.NewRequest(http.MethodPost, "/v1/tts", bytes.NewReader([]byte(`{"text":"hello"}`)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if !backend.called {
		t.Fatalf("expected backend to be invoked")
	}
}

type blockingBackend struct {
	started chan struct{}
	block   chan struct{}
	done    *sync.WaitGroup
}

func (b *blockingBackend) StreamTTS(ctx context.Context, req *backend.Request, w http.ResponseWriter) error {
	if b.started != nil {
		b.started <- struct{}{}
	}

	<-b.block

	if b.done != nil {
		b.done.Done()
	}

	return nil
}

type noopBackend struct{}

func (n *noopBackend) StreamTTS(ctx context.Context, req *backend.Request, w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

type recordingBackend struct {
	called bool
}

func (r *recordingBackend) StreamTTS(ctx context.Context, req *backend.Request, w http.ResponseWriter) error {
	r.called = true
	w.WriteHeader(http.StatusOK)
	return nil
}
