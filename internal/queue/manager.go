package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrQueueFull = errors.New("queue: full")
	ErrShutdown  = errors.New("queue: shutdown")
)

type Config struct {
	Workers  int
	MaxQueue int
}

type Manager struct {
	jobs     chan job
	wg       sync.WaitGroup
	inflight sync.WaitGroup

	closeOnce sync.Once
	closed    chan struct{}

	workers int32
	active  atomic.Int32
}

type job struct {
	ctx    context.Context
	fn     func(context.Context) error
	result chan error
}

func NewManager(cfg Config) *Manager {
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}
	if cfg.MaxQueue < 0 {
		cfg.MaxQueue = 0
	}

	m := &Manager{
		jobs:    make(chan job, cfg.MaxQueue),
		closed:  make(chan struct{}),
		workers: int32(cfg.Workers),
	}

	for i := 0; i < cfg.Workers; i++ {
		m.wg.Add(1)
		go m.worker()
	}

	return m
}

func (m *Manager) Submit(ctx context.Context, fn func(context.Context) error) error {
	select {
	case <-m.closed:
		return ErrShutdown
	default:
	}

	j := job{ctx: ctx, fn: fn, result: make(chan error, 1)}

	if cap(m.jobs) == 0 {
		if m.active.Load() >= m.workers {
			return ErrQueueFull
		}

		select {
		case m.jobs <- j:
		case <-m.closed:
			return ErrShutdown
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case m.jobs <- j:
		case <-m.closed:
			return ErrShutdown
		default:
			return ErrQueueFull
		}
	}

	select {
	case err := <-j.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-m.closed:
		// allow in-flight job to finish if already running
		select {
		case err := <-j.result:
			return err
		default:
			return ErrShutdown
		}
	}
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.closeOnce.Do(func() {
		close(m.closed)
		close(m.jobs)
	})

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		m.inflight.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) worker() {
	defer m.wg.Done()

	for j := range m.jobs {
		m.inflight.Add(1)
		m.active.Add(1)
		j.result <- j.fn(j.ctx)
		m.active.Add(-1)
		m.inflight.Done()
	}
}
