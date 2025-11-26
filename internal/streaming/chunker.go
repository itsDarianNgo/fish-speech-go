package streaming

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAcquireTimeout indicates the chunker could not provide a slot within the configured timeout.
	ErrAcquireTimeout = errors.New("chunker: acquire timeout")
	// ErrLimitExceeded indicates the chunker refused a slot because concurrency is exhausted.
	ErrLimitExceeded = errors.New("chunker: limit exceeded")
)

// Chunker limits concurrent streaming operations using a semaphore-style slot pool.
type Chunker struct {
	slots          chan struct{}
	acquireTimeout time.Duration
	metrics        *Metrics
}

// ChunkerConfig controls how the Chunker gates concurrent access.
type ChunkerConfig struct {
	MaxConcurrent  int
	AcquireTimeout time.Duration
	Metrics        *Metrics
}

// NewChunker constructs a Chunker with the provided configuration.
func NewChunker(cfg ChunkerConfig) *Chunker {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	return &Chunker{
		slots:          make(chan struct{}, cfg.MaxConcurrent),
		acquireTimeout: cfg.AcquireTimeout,
		metrics:        cfg.Metrics,
	}
}

// Acquire reserves a slot for work. The returned release function must be called to free the slot.
func (c *Chunker) Acquire(ctx context.Context) (func(), error) {
	// Fast path when a slot is immediately available.
	select {
	case c.slots <- struct{}{}:
		return c.onAcquire(), nil
	default:
	}

	// Slow path waits based on the configured timeout or context cancellation.
	if c.acquireTimeout <= 0 {
		select {
		case c.slots <- struct{}{}:
			return c.onAcquire(), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if c.metrics != nil {
				c.metrics.IncLimitExceeded()
			}
			return nil, ErrLimitExceeded
		}
	}

	timer := time.NewTimer(c.acquireTimeout)
	defer timer.Stop()

	select {
	case c.slots <- struct{}{}:
		return c.onAcquire(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		if c.metrics != nil {
			c.metrics.IncAcquireTimeouts()
		}
		return nil, ErrAcquireTimeout
	}
}

func (c *Chunker) onAcquire() func() {
	if c.metrics != nil {
		c.metrics.IncActiveStreams()
	}

	return func() {
		select {
		case <-c.slots:
		default:
		}

		if c.metrics != nil {
			c.metrics.DecActiveStreams()
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
