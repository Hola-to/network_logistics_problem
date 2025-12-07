package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"
)

func skipIfNoRedis(t *testing.T) {
	if os.Getenv("REDIS_TEST_ADDR") == "" {
		t.Skip("REDIS_TEST_ADDR not set, skipping Redis tests")
	}
}

func TestNewRedisLimiter(t *testing.T) {
	skipIfNoRedis(t)

	cfg := &Config{
		Requests:      10,
		Window:        time.Minute,
		Strategy:      "sliding_window",
		Backend:       "redis",
		RedisAddr:     os.Getenv("REDIS_TEST_ADDR"),
		RedisPassword: os.Getenv("REDIS_TEST_PASSWORD"),
	}

	limiter, err := NewRedisLimiter(cfg)
	if err != nil {
		t.Fatalf("NewRedisLimiter() error = %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "test-ratelimit-key"

	// Reset first
	limiter.Reset(ctx, key)

	// Should allow
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if !allowed {
		t.Error("first request should be allowed")
	}

	// Cleanup
	limiter.Reset(ctx, key)
}

func TestRedisLimiter_GetInfo(t *testing.T) {
	skipIfNoRedis(t)

	cfg := &Config{
		Requests:  5,
		Window:    time.Minute,
		RedisAddr: os.Getenv("REDIS_TEST_ADDR"),
	}

	limiter, err := NewRedisLimiter(cfg)
	if err != nil {
		t.Fatalf("NewRedisLimiter() error = %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "test-info-key"

	limiter.Reset(ctx, key)
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	info, err := limiter.GetInfo(ctx, key)
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}

	if info.Limit != 5 {
		t.Errorf("Limit = %d, want 5", info.Limit)
	}
	if info.Remaining != 3 {
		t.Errorf("Remaining = %d, want 3", info.Remaining)
	}

	limiter.Reset(ctx, key)
}
