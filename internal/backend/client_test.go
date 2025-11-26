package backend

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientStreamTTSSuccess(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/msgpack" {
			t.Fatalf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		expected := []byte{
			0x83,
			0xa4, 't', 'e', 'x', 't',
			0xa5, 'h', 'e', 'l', 'l', 'o',
			0xa9, 's', 't', 'r', 'e', 'a', 'm', 'i', 'n', 'g', 0xc2,
			0xa6, 'f', 'o', 'r', 'm', 'a', 't',
			0xa3, 'w', 'a', 'v',
		}

		if !bytes.Equal(body, expected) {
			t.Fatalf("unexpected msgpack payload: %v", body)
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio"))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, 5*time.Second, server.Client())
	resp, err := client.StreamTTS(context.Background(), TTSRequest{Text: "hello", Format: "wav"})
	if err != nil {
		t.Fatalf("StreamTTS returned error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(body) != "audio" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestClientStreamTTSErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, 0, server.Client())
	_, err := client.StreamTTS(context.Background(), TTSRequest{Text: "hello", Format: "wav"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", httpErr.StatusCode)
	}
}

func TestClientStreamTTSTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, 50*time.Millisecond, server.Client())
	_, err := client.StreamTTS(context.Background(), TTSRequest{Text: "hello", Format: "wav"})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}
