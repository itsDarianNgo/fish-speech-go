package queue

import (
	"context"
	"testing"
)

func TestManager_AcquireAndRelease(t *testing.T) {
	mgr, err := NewManager(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	release, err := mgr.Acquire(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire: %v", err)
	}

	done := make(chan struct{})
	go func() {
		rel, err := mgr.Acquire(context.Background())
		if err != nil {
			t.Fatalf("second acquire failed: %v", err)
		}
		rel()
		close(done)
	}()

	release()

	<-done
}

func TestManager_Cancellation(t *testing.T) {
	mgr, _ := NewManager(1)
	release, _ := mgr.Acquire(context.Background())
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := mgr.Acquire(ctx); err == nil {
		t.Fatalf("expected cancellation error")
	}
}
