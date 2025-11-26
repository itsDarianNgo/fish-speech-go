package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type blockingProducer struct {
	started chan struct{}
	release chan struct{}
}

type responseResult struct {
	resp *http.Response
	err  error
}

func newBlockingProducer(buffer int) *blockingProducer {
	return &blockingProducer{
		started: make(chan struct{}, buffer),
		release: make(chan struct{}),
	}
}

func (p *blockingProducer) Produce(_ context.Context) error {
	p.started <- struct{}{}
	<-p.release
	return nil
}

func TestChunkHandlerRespectsMaxConcurrent(t *testing.T) {
	chunker := NewSemaphoreChunker(2, 0)
	producer := newBlockingProducer(3)
	metrics := NewMetrics()
	logBuf := &bytes.Buffer{}
	handler := NewChunkRequestHandler(chunker, producer, WithMetrics(metrics), WithLogger(log.New(logBuf, "", 0)))

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	var wg sync.WaitGroup
	startRequest := func() chan responseResult {
		return launchRequest(t, server.URL, &wg)
	}

	// Occupy both slots.
	resp1Ch := startRequest()
	resp2Ch := startRequest()

	waitForStarts(t, producer, 2)

	if metrics.ActiveStreams() != 2 {
		t.Fatalf("expected active streams to reflect two in-flight requests, got %d", metrics.ActiveStreams())
	}

	// Third request should immediately receive limit exceeded.
	resp3Ch := startRequest()
	resp3 := waitForResponse(t, resp3Ch)
	defer resp3.Body.Close()

	assertErrorResponse(t, resp3, http.StatusServiceUnavailable, ChunkerErrorLimitExceeded)

	if metrics.LimitExceededResponses() != 1 {
		t.Fatalf("expected limit exceeded counter to be 1, got %d", metrics.LimitExceededResponses())
	}

	// Unblock the first two requests and assert they complete.
	close(producer.release)

	resp1 := waitForResponse(t, resp1Ch)
	resp2 := waitForResponse(t, resp2Ch)
	defer resp1.Body.Close()
	defer resp2.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected first request 200 OK, got %d", resp1.StatusCode)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected second request 200 OK, got %d", resp2.StatusCode)
	}

	wg.Wait()

	if metrics.ActiveStreams() != 0 {
		t.Fatalf("expected active streams to return to zero, got %d", metrics.ActiveStreams())
	}

	if !strings.Contains(logBuf.String(), `"chunker_error_code":"limit_exceeded"`) {
		t.Fatalf("expected structured log to include chunker error code, got: %s", logBuf.String())
	}
}

func TestChunkHandlerRespectsAcquireTimeout(t *testing.T) {
	chunker := NewSemaphoreChunker(1, 50*time.Millisecond)
	producer := newBlockingProducer(2)
	metrics := NewMetrics()
	handler := NewChunkRequestHandler(chunker, producer, WithMetrics(metrics))

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	var wg sync.WaitGroup

	firstRespCh := launchRequest(t, server.URL, &wg)

	waitForStarts(t, producer, 1)

	resp2 := waitForResponse(t, launchRequest(t, server.URL, &wg))
	defer resp2.Body.Close()

	assertErrorResponse(t, resp2, http.StatusGatewayTimeout, ChunkerErrorTimeout)

	if metrics.AcquireTimeouts() != 1 {
		t.Fatalf("expected acquire timeout counter to be 1, got %d", metrics.AcquireTimeouts())
	}

	close(producer.release)

	resp1 := waitForResponse(t, firstRespCh)
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected first request 200 OK, got %d", resp1.StatusCode)
	}

	wg.Wait()
}

func waitForStarts(t *testing.T, producer *blockingProducer, expected int) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	count := 0
	for count < expected {
		select {
		case <-producer.started:
			count++
		case <-timeout:
			t.Fatalf("timed out waiting for %d producer starts, got %d", expected, count)
		}
	}
}

func launchRequest(t *testing.T, url string, wg *sync.WaitGroup) chan responseResult {
	wg.Add(1)
	respCh := make(chan responseResult, 1)
	go func() {
		defer wg.Done()
		resp, err := http.Post(url, "application/json", bytes.NewBufferString("{}"))
		respCh <- responseResult{resp: resp, err: err}
	}()
	return respCh
}

func waitForResponse(t *testing.T, ch <-chan responseResult) *http.Response {
	t.Helper()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("request failed: %v", res.err)
		}
		if res.resp == nil {
			t.Fatalf("expected response to be non-nil")
		}
		return res.resp
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for response")
	}
	return nil
}

func assertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedCode ChunkerErrorCode) {
	t.Helper()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    ChunkerErrorCode `json:"code"`
			Message string           `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error.Code != expectedCode {
		t.Fatalf("expected error code %q, got %q", expectedCode, payload.Error.Code)
	}
	if payload.Error.Message == "" {
		t.Fatalf("expected error message to be populated")
	}
}
