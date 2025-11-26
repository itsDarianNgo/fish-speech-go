package streaming

import (
	"context"
	"testing"
	"time"
)

func BenchmarkChunkerAcquireRelease(b *testing.B) {
	chunker, err := NewChunker(Options{MaxConcurrent: 32, AcquireTimeout: time.Second})
	if err != nil {
		b.Fatalf("failed to create chunker: %v", err)
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			release, err := chunker.Acquire(ctx)
			if err != nil {
				b.Fatalf("unexpected acquire error: %v", err)
			}
			release()
		}
	})
}

func BenchmarkChunkerContention(b *testing.B) {
	chunker, err := NewChunker(Options{MaxConcurrent: 8, AcquireTimeout: time.Second})
	if err != nil {
		b.Fatalf("failed to create chunker: %v", err)
	}

	b.ReportAllocs()
	b.SetParallelism(64)
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			_ = chunker.Stream(ctx, func(context.Context) error {
				time.Sleep(100 * time.Microsecond)
				return nil
			})
		}
	})
}
