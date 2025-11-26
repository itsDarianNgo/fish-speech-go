package streaming

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestChunkerTracksActiveStreams(t *testing.T) {
	metrics := NewMetrics()
	chunker := NewChunker(ChunkerConfig{MaxConcurrent: 1, AcquireTimeout: time.Second, Metrics: metrics})

	release, err := chunker.Acquire(context.Background())
	if err != nil {
		t.Fatalf("expected successful acquire: %v", err)
	}

	if metrics.ActiveStreams() != 1 {
		t.Fatalf("expected active streams to be 1 after acquire, got %d", metrics.ActiveStreams())
	}

	release()

	if metrics.ActiveStreams() != 0 {
		t.Fatalf("expected active streams to return to 0 after release, got %d", metrics.ActiveStreams())
	}
}

func TestChunkerAcquireTimeoutMetrics(t *testing.T) {
	metrics := NewMetrics()
	chunker := NewChunker(ChunkerConfig{MaxConcurrent: 1, AcquireTimeout: 5 * time.Millisecond, Metrics: metrics})

	release, err := chunker.Acquire(context.Background())
	if err != nil {
		t.Fatalf("expected initial acquire to succeed: %v", err)
	}
	defer release()

	_, err = chunker.Acquire(context.Background())
	if !errors.Is(err, ErrAcquireTimeout) {
		t.Fatalf("expected acquire timeout, got: %v", err)
	}

	if metrics.AcquireTimeouts() != 1 {
		t.Fatalf("expected one acquire timeout recorded, got %d", metrics.AcquireTimeouts())
	}
}
