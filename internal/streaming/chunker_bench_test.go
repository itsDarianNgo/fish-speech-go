package streaming

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkChunkerAcquire(b *testing.B) {
	cases := []struct {
		name           string
		maxConcurrent  int
		acquireTimeout time.Duration
		parallelism    int
	}{
		{name: "max1-timeout1ms", maxConcurrent: 1, acquireTimeout: time.Millisecond, parallelism: 4},
		{name: "max4-timeout1ms", maxConcurrent: 4, acquireTimeout: time.Millisecond, parallelism: 4},
		{name: "max16-timeout5ms", maxConcurrent: 16, acquireTimeout: 5 * time.Millisecond, parallelism: 8},
		{name: "max32-no-timeout", maxConcurrent: 32, acquireTimeout: 0, parallelism: 8},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			chunker := NewChunker(ChunkerConfig{MaxConcurrent: tc.maxConcurrent, AcquireTimeout: tc.acquireTimeout})
			b.ReportAllocs()
			if tc.parallelism > 0 {
				b.SetParallelism(tc.parallelism)
			}
			b.ResetTimer()

			ctx := context.Background()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					release, err := chunker.Acquire(ctx)
					if err != nil {
						b.Fatalf("unexpected acquire error: %v", err)
					}
					release()
				}
			})
		})
	}
}

func BenchmarkChunkerStreamContention(b *testing.B) {
	cases := []struct {
		name           string
		maxConcurrent  int
		workers        int
		acquireTimeout time.Duration
		holdOps        int
	}{
		{name: "2slots-8workers-short-timeout", maxConcurrent: 2, workers: 8, acquireTimeout: 500 * time.Microsecond, holdOps: 64},
		{name: "4slots-16workers-short-timeout", maxConcurrent: 4, workers: 16, acquireTimeout: time.Millisecond, holdOps: 64},
		{name: "8slots-32workers-no-timeout", maxConcurrent: 8, workers: 32, acquireTimeout: 0, holdOps: 32},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			chunker := NewChunker(ChunkerConfig{MaxConcurrent: tc.maxConcurrent, AcquireTimeout: tc.acquireTimeout})
			var timeouts int64

			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				// RunParallel spawns GOMAXPROCS*parallelism goroutines; give each a small work section.
				for pb.Next() {
					release, err := chunker.Acquire(ctx)
					if err != nil {
						if err == ErrAcquireTimeout {
							atomic.AddInt64(&timeouts, 1)
							continue
						}
						b.Fatalf("unexpected acquire error: %v", err)
					}

					for i := 0; i < tc.holdOps; i++ {
						runtime.Gosched()
					}
					release()
				}
			})

			b.ReportMetric(float64(timeouts), "timeouts")
			b.ReportMetric(float64(timeouts)/float64(b.N), "timeouts/op")
		})
	}
}
