//go:build integration

package pkg_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"logistics/pkg/ratelimit"
	"logistics/tests/integration/testutil"
)

func TestRedisLimiter_Basic(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  5,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "ratelimit")
	limiter.Reset(ctx, key)

	// Should allow 5 requests
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th should be denied
	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("6th request should be denied")
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_AllowN(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  10,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "allowN")
	limiter.Reset(ctx, key)

	// Allow 5 at once
	allowed, _ := limiter.AllowN(ctx, key, 5)
	if !allowed {
		t.Error("5 requests should be allowed")
	}

	// Allow another 5
	allowed, _ = limiter.AllowN(ctx, key, 5)
	if !allowed {
		t.Error("another 5 requests should be allowed")
	}

	// 11th should fail
	allowed, _ = limiter.AllowN(ctx, key, 1)
	if allowed {
		t.Error("11th request should be denied")
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_GetInfo(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  10,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "info")
	limiter.Reset(ctx, key)

	// Use some quota
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	info, err := limiter.GetInfo(ctx, key)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Limit != 10 {
		t.Errorf("Limit = %d, want 10", info.Limit)
	}
	if info.Remaining != 7 {
		t.Errorf("Remaining = %d, want 7", info.Remaining)
	}
	if info.ResetAt.IsZero() {
		t.Error("ResetAt should not be zero")
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_Reset(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  2,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "reset")
	limiter.Reset(ctx, key)

	// Use up limit
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	// Should be denied
	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("should be rate limited")
	}

	// Reset
	err = limiter.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be allowed again
	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Error("should be allowed after reset")
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_WindowReset(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  3,
		Window:    500 * time.Millisecond,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "window")
	limiter.Reset(ctx, key)

	// Use up limit
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, key)
	}

	// Should be denied
	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("should be denied before window reset")
	}

	// Wait for window to reset
	time.Sleep(600 * time.Millisecond)

	// Should be allowed again
	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Error("should be allowed after window reset")
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_Concurrent(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  100,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "concurrent")
	limiter.Reset(ctx, key)

	var wg sync.WaitGroup
	var allowed, denied int64

	// 200 concurrent requests, only 100 should pass
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := limiter.Allow(ctx, key)
			if ok {
				atomic.AddInt64(&allowed, 1)
			} else {
				atomic.AddInt64(&denied, 1)
			}
		}()
	}

	wg.Wait()

	if allowed != 100 {
		t.Errorf("allowed = %d, want 100", allowed)
	}
	if denied != 100 {
		t.Errorf("denied = %d, want 100", denied)
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_Wait(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.ContextWithDuration(t, 2*time.Second)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  1,
		Window:    500 * time.Millisecond,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "wait")
	limiter.Reset(ctx, key)

	// Use up limit
	limiter.Allow(ctx, key)

	// Wait should block and then succeed after window reset
	start := time.Now()
	err = limiter.Wait(ctx, key)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if elapsed < 400*time.Millisecond {
		t.Errorf("Wait returned too quickly: %v", elapsed)
	}

	limiter.Reset(ctx, key)
}

func TestRedisLimiter_Wait_Timeout(t *testing.T) {
	addr := testutil.RequireRedis(t)

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  1,
		Window:    time.Hour, // Very long window
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "wait_timeout")
	ctx, cancel := testutil.ContextWithDuration(t, 200*time.Millisecond)
	defer cancel()

	limiter.Reset(ctx, key)

	// Use up limit
	limiter.Allow(ctx, key)

	// Wait should timeout
	err = limiter.Wait(ctx, key)
	if err == nil {
		t.Error("Wait should have timed out")
	}

	limiter.Reset(context.Background(), key)
}

func TestRedisLimiter_MultipleKeys(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  2,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key1 := testutil.UniqueKey(t, "multi1")
	key2 := testutil.UniqueKey(t, "multi2")

	limiter.Reset(ctx, key1)
	limiter.Reset(ctx, key2)

	// Exhaust key1
	limiter.Allow(ctx, key1)
	limiter.Allow(ctx, key1)

	// key1 should be denied
	allowed, _ := limiter.Allow(ctx, key1)
	if allowed {
		t.Error("key1 should be rate limited")
	}

	// key2 should still work
	allowed, _ = limiter.Allow(ctx, key2)
	if !allowed {
		t.Error("key2 should be allowed")
	}

	limiter.Reset(ctx, key1)
	limiter.Reset(ctx, key2)
}

func TestRedisLimiter_Close(t *testing.T) {
	addr := testutil.RequireRedis(t)

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  10,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	// Close should not error
	err = limiter.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestRedisLimiter_HighThroughput(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	limiter, err := ratelimit.NewRedisLimiter(&ratelimit.Config{
		Requests:  10000,
		Window:    time.Minute,
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}
	testutil.Cleanup(t, func() { limiter.Close() })

	key := testutil.UniqueKey(t, "throughput")
	limiter.Reset(ctx, key)

	start := time.Now()
	var wg sync.WaitGroup
	var count int64

	// 1000 goroutines, 10 requests each
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if ok, _ := limiter.Allow(ctx, key); ok {
					atomic.AddInt64(&count, 1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Processed %d requests in %v (%.0f req/s)", count, elapsed, float64(count)/elapsed.Seconds())

	if count != 10000 {
		t.Errorf("allowed = %d, want 10000", count)
	}

	limiter.Reset(ctx, key)
}
