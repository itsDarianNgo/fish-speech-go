package streaming

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAcquireTimeout indicates the chunker could not provide a slot within the configured timeout.
	ErrAcquireTimeout = errors.New("chunker: acquire timeout")
)

// Chunker limits concurrent streaming operations using a semaphore-style slot pool.
type Chunker struct {
	slots          chan struct{}
	acquireTimeout time.Duration
}

// ChunkerConfig controls how the Chunker gates concurrent access.
type ChunkerConfig struct {
	MaxConcurrent  int
	AcquireTimeout time.Duration
}

// NewChunker constructs a Chunker with the provided configuration.
func NewChunker(cfg ChunkerConfig) *Chunker {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	return &Chunker{
		slots:          make(chan struct{}, cfg.MaxConcurrent),
		acquireTimeout: cfg.AcquireTimeout,
	}
}

// Acquire reserves a slot for work. The returned release function must be called to free the slot.
func (c *Chunker) Acquire(ctx context.Context) (func(), error) {
	// Fast path when a slot is immediately available.
	select {
	case c.slots <- struct{}{}:
		return c.releaseFn(), nil
	default:
	}

	// Slow path waits based on the configured timeout or context cancellation.
	if c.acquireTimeout <= 0 {
		select {
		case c.slots <- struct{}{}:
			return c.releaseFn(), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	timer := time.NewTimer(c.acquireTimeout)
	defer timer.Stop()

	select {
	case c.slots <- struct{}{}:
		return c.releaseFn(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, ErrAcquireTimeout
	}
}

func (c *Chunker) releaseFn() func() {
	return func() {
		select {
		case <-c.slots:
		default:
		}
	}
}

// Stream executes the provided function while holding a slot. The slot is released when the
// function returns, allowing callers to guard streaming workloads without manual bookkeeping.
func (c *Chunker) Stream(ctx context.Context, streamFn func(context.Context) error) error {
	release, err := c.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	return streamFn(ctx)
}
