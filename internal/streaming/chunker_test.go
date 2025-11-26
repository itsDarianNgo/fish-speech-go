package streaming

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestChunkerLimitsConcurrency(t *testing.T) {
	chunker, err := NewChunker(Options{MaxConcurrent: 2, AcquireTimeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("unexpected error creating chunker: %v", err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	hold := make(chan struct{})
	acquired := make(chan struct{}, 2)

	// Consume both slots.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, acquireErr := chunker.Acquire(ctx)
			if acquireErr != nil {
				t.Fatalf("expected acquisition to succeed: %v", acquireErr)
			}
			acquired <- struct{}{}
			defer release()
			<-hold
		}()
	}

	// Ensure both goroutines acquired their slots before testing the limit.
	for i := 0; i < 2; i++ {
		select {
		case <-acquired:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for goroutines to acquire slots")
		}
	}

	// Attempt to acquire a third slot with a short timeout.
	_, acquireErr := chunker.Acquire(ctx)
	if !errors.Is(acquireErr, ErrAcquireTimeout) {
		t.Fatalf("expected ErrAcquireTimeout, got %v", acquireErr)
	}

	close(hold)
	wg.Wait()
}

func TestStreamReleasesOnError(t *testing.T) {
	chunker, err := NewChunker(Options{MaxConcurrent: 1, AcquireTimeout: 0})
	if err != nil {
		t.Fatalf("unexpected error creating chunker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstDone := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		errCh <- chunker.Stream(ctx, func(context.Context) error {
			<-firstDone
			return errors.New("fail")
		})
	}()

	// Allow the first stream to start and hold the slot, then let it finish.
	time.Sleep(10 * time.Millisecond)
	close(firstDone)

	if err := <-errCh; err == nil {
		t.Fatalf("expected first stream to fail")
	}

	acquireErr := chunker.Stream(ctx, func(context.Context) error { return nil })
	if acquireErr != nil {
		t.Fatalf("expected slot to be released after error; got %v", acquireErr)
	}
}

func TestInvalidOptions(t *testing.T) {
	if _, err := NewChunker(Options{}); err == nil {
		t.Fatalf("expected error for missing MaxConcurrent")
	}
}
