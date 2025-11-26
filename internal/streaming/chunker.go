package streaming

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ChunkerErrorCode enumerates the categories of errors returned by the chunker.
type ChunkerErrorCode string

const (
	// ChunkerLimitExceeded indicates that no semaphore slot was available
	// before the caller timed out or canceled the operation.
	ChunkerLimitExceeded ChunkerErrorCode = "limit_exceeded"
	// ChunkerAcquireTimeout indicates that the chunker timed out waiting for a slot
	// using its configured AcquireTimeout value.
	ChunkerAcquireTimeout ChunkerErrorCode = "acquire_timeout"
)

// ChunkerError provides structured context about concurrency limiting failures.
type ChunkerError struct {
	Code    ChunkerErrorCode
	Message string
}

// Error implements the error interface.
func (e ChunkerError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

// Is allows errors.Is to match by error code, enabling callers to react to
// concurrency guardrails without depending on the error message.
func (e ChunkerError) Is(target error) bool {
	t, ok := target.(ChunkerError)
	if ok {
		return e.Code == t.Code
	}

	pt, ok := target.(*ChunkerError)
	if ok && pt != nil {
		return e.Code == pt.Code
	}

	return false
}

var (
	// ErrLimitExceeded is returned when the chunker cannot obtain a semaphore slot
	// before the caller cancels the attempt.
	ErrLimitExceeded = ChunkerError{Code: ChunkerLimitExceeded, Message: "chunker concurrency limit reached"}
	// ErrAcquireTimeout is returned when the chunker exceeds its internal AcquireTimeout
	// while waiting for a slot.
	ErrAcquireTimeout = ChunkerError{Code: ChunkerAcquireTimeout, Message: "timed out waiting for chunker slot"}
)

// Options configures how the chunker gates concurrent streaming operations.
type Options struct {
	// MaxConcurrent defines the maximum number of concurrent streaming tasks allowed.
	// This value is required and must be greater than zero.
	MaxConcurrent int
	// AcquireTimeout defines how long to wait for a semaphore slot before failing with
	// ErrAcquireTimeout. A zero value disables the internal timeout, leaving the caller's
	// context to control cancellation.
	AcquireTimeout time.Duration
}

// Chunker limits the number of concurrent streaming operations using a semaphore.
type Chunker struct {
	sem            chan struct{}
	acquireTimeout time.Duration
}

// NewChunker initializes a Chunker with the provided options.
func NewChunker(opts Options) (*Chunker, error) {
	if opts.MaxConcurrent <= 0 {
		return nil, fmt.Errorf("MaxConcurrent must be greater than zero")
	}

	return &Chunker{
		sem:            make(chan struct{}, opts.MaxConcurrent),
		acquireTimeout: opts.AcquireTimeout,
	}, nil
}

// Acquire obtains a semaphore slot respecting context cancellation and the configured
// AcquireTimeout. It returns a release function that must be called to free the slot.
func (c *Chunker) Acquire(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}

	slotCtx := ctx
	var cancel context.CancelFunc
	if c.acquireTimeout > 0 {
		slotCtx, cancel = context.WithTimeout(ctx, c.acquireTimeout)
		defer cancel()
	}

	select {
	case c.sem <- struct{}{}:
		var once sync.Once
		release := func() {
			once.Do(func() {
				<-c.sem
			})
		}
		return release, nil
	case <-slotCtx.Done():
		if errors.Is(slotCtx.Err(), context.DeadlineExceeded) {
			return nil, ErrAcquireTimeout
		}
		return nil, ErrLimitExceeded
	}
}

// Stream wraps Acquire and executes the producer while holding the semaphore slot.
// It ensures the slot is released even when the producer returns an error.
func (c *Chunker) Stream(ctx context.Context, producer func(context.Context) error) error {
	release, err := c.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	if producer == nil {
		return nil
	}

	return producer(ctx)
}
