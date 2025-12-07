package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"logistics/pkg/ratelimit"
)

func BenchmarkMemoryLimiter_Allow(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000000,
		Window:          time.Minute,
		Strategy:        "sliding_window",
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "benchmark-key")
	}
}

func BenchmarkMemoryLimiter_Allow_Parallel(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000000,
		Window:          time.Minute,
		Strategy:        "sliding_window",
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow(ctx, "benchmark-key")
		}
	})
}

func BenchmarkMemoryLimiter_Allow_MultipleKeys(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000,
		Window:          time.Minute,
		Strategy:        "sliding_window",
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, fmt.Sprintf("key-%d", i%1000))
	}
}

func BenchmarkMemoryLimiter_TokenBucket(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000000,
		Window:          time.Minute,
		Strategy:        "token_bucket",
		BurstSize:       100,
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "benchmark-key")
	}
}

func BenchmarkMemoryLimiter_GetInfo(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000,
		Window:          time.Minute,
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		limiter.Allow(ctx, "info-key")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.GetInfo(ctx, "info-key")
	}
}

func BenchmarkMemoryLimiter_Reset(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000,
		Window:          time.Minute,
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("reset-key-%d", i)
		limiter.Allow(ctx, key)
		limiter.Reset(ctx, key)
	}
}

func BenchmarkMemoryLimiter_HighContention(b *testing.B) {
	limiter := ratelimit.NewMemoryLimiter(&ratelimit.Config{
		Requests:        1000000,
		Window:          time.Minute,
		Strategy:        "sliding_window",
		CleanupInterval: time.Hour,
	})
	defer limiter.Close()

	ctx := context.Background()

	// Single key with high contention
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow(ctx, "contention-key")
		}
	})
}

func BenchmarkKeyExtractors(b *testing.B) {
	ctx := context.Background()
	method := "/test.Service/Method"
	metadata := map[string]string{
		"x-forwarded-for": "192.168.1.1",
		"x-user-id":       "user-123",
	}

	b.Run("DefaultKeyExtractor", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ratelimit.DefaultKeyExtractor(ctx, method, metadata)
		}
	})

	b.Run("MethodKeyExtractor", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ratelimit.MethodKeyExtractor(ctx, method, metadata)
		}
	})

	b.Run("UserKeyExtractor", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ratelimit.UserKeyExtractor(ctx, method, metadata)
		}
	})

	b.Run("CompositeKeyExtractor", func(b *testing.B) {
		extractor := ratelimit.CompositeKeyExtractor(
			ratelimit.MethodKeyExtractor,
			ratelimit.UserKeyExtractor,
		)
		for i := 0; i < b.N; i++ {
			extractor(ctx, method, metadata)
		}
	})
}
