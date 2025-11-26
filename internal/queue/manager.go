package queue

import (
    "context"
    "fmt"
)

// Manager provides a simple bounded queue modeled as a semaphore.
type Manager struct {
    sem chan struct{}
}

// NewManager creates a queue with the given capacity.
func NewManager(maxPending int) (*Manager, error) {
    if maxPending <= 0 {
        return nil, fmt.Errorf("maxPending must be greater than zero")
    }
    return &Manager{sem: make(chan struct{}, maxPending)}, nil
}

// Acquire reserves a slot for work, respecting context cancellation.
func (m *Manager) Acquire(ctx context.Context) (func(), error) {
    if m == nil {
        return nil, fmt.Errorf("queue manager not configured")
    }
    if ctx == nil {
        ctx = context.Background()
    }

    select {
    case m.sem <- struct{}{}:
        released := false
        release := func() {
            if released {
                return
            }
            released = true
            <-m.sem
        }
        return release, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
