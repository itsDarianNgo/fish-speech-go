package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestManagerProcessesJobs(t *testing.T) {
	manager := NewManager(Config{Workers: 2, MaxQueue: 2})
	t.Cleanup(func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	})

	var mu sync.Mutex
	results := make([]int, 0, 3)

	for i := 0; i < 3; i++ {
		i := i
		if err := manager.Submit(context.Background(), func(context.Context) error {
			mu.Lock()
			results = append(results, i)
			mu.Unlock()
			return nil
		}); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestManagerQueueFull(t *testing.T) {
	manager := NewManager(Config{Workers: 1, MaxQueue: 0})
	defer manager.Shutdown(context.Background())

	start := make(chan struct{})
	release := make(chan struct{})

	go func() {
		_ = manager.Submit(context.Background(), func(context.Context) error {
			close(start)
			<-release
			return nil
		})
	}()

	select {
	case <-start:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	if err := manager.Submit(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	close(release)
}

func TestManagerShutdownWaitsForInflight(t *testing.T) {
	manager := NewManager(Config{Workers: 1, MaxQueue: 1})

	start := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		_ = manager.Submit(context.Background(), func(context.Context) error {
			close(start)
			<-release
			close(finished)
			return nil
		})
	}()

	select {
	case <-start:
	case <-time.After(time.Second):
		t.Fatalf("job did not start")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		shutdownDone <- manager.Shutdown(ctx)
	}()

	select {
	case err := <-shutdownDone:
		if err == nil {
			t.Fatal("shutdown returned before job finished")
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("shutdown did not time out")
	}

	close(release)
	<-finished

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown after release failed: %v", err)
	}
}
