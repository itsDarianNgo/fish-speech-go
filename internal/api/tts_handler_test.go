package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"fish-speech-go/internal/backend"
	"fish-speech-go/internal/streaming"
)

type stubBackend struct {
	stream func(context.Context, backend.TTSRequest) (*http.Response, error)
}

func (s *stubBackend) StreamTTS(ctx context.Context, req backend.TTSRequest) (*http.Response, error) {
	return s.stream(ctx, req)
}

func TestTTSHandlerValidatesRequest(t *testing.T) {
	handler := NewTTSHandler(streaming.NewChunker(streaming.ChunkerConfig{MaxConcurrent: 2}), &stubBackend{})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	resp, err := http.Post(server.URL, "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}

	if payload.Error.Code != "invalid_request" {
		t.Fatalf("unexpected error code: %s", payload.Error.Code)
	}
}

func TestTTSHandlerStreamsSuccess(t *testing.T) {
	backendCalled := make(chan backend.TTSRequest, 1)
	handler := NewTTSHandler(streaming.NewChunker(streaming.ChunkerConfig{MaxConcurrent: 2}), &stubBackend{
		stream: func(_ context.Context, req backend.TTSRequest) (*http.Response, error) {
			backendCalled <- req
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"audio/wav"}},
				Body:       io.NopCloser(strings.NewReader("audio-bytes")),
			}, nil
		},
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	resp, err := http.Post(server.URL, "application/json", bytes.NewBufferString(`{"text":"hello","format":"wav","streaming":true}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "audio/wav" {
		t.Fatalf("unexpected content type: %s", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != "audio-bytes" {
		t.Fatalf("unexpected body: %s", string(body))
	}

	select {
	case req := <-backendCalled:
		if req.Text != "hello" || req.Format != "wav" || !req.Streaming {
			t.Fatalf("unexpected backend request: %+v", req)
		}
	case <-time.After(time.Second):
		t.Fatalf("backend was not called")
	}
}

func TestTTSHandlerAcquireTimeout(t *testing.T) {
	chunker := streaming.NewChunker(streaming.ChunkerConfig{MaxConcurrent: 1, AcquireTimeout: 50 * time.Millisecond})
	release := make(chan struct{})
	started := make(chan struct{})

	backendStub := &stubBackend{stream: func(_ context.Context, _ backend.TTSRequest) (*http.Response, error) {
		pr, pw := io.Pipe()
		go func() {
			close(started)
			_, _ = pw.Write([]byte("chunk"))
			<-release
			pw.Close()
		}()

		return &http.Response{StatusCode: http.StatusOK, Body: pr}, nil
	}}

	handler := NewTTSHandler(chunker, backendStub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := http.Post(server.URL, "application/json", bytes.NewBufferString(`{"text":"first","format":"wav"}`))
		if err != nil {
			t.Errorf("first request failed: %v", err)
			return
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	<-started

	resp, err := http.Post(server.URL, "application/json", bytes.NewBufferString(`{"text":"second","format":"wav"}`))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if payload.Error.Code != "acquire_timeout" {
		t.Fatalf("unexpected error code: %s", payload.Error.Code)
	}

	close(release)
	wg.Wait()
}
