package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Requests <= 0 {
		t.Error("Requests should be positive")
	}
	if cfg.Window <= 0 {
		t.Error("Window should be positive")
	}
	if cfg.Strategy == "" {
		t.Error("Strategy should not be empty")
	}
}

func TestNewMemoryLimiter(t *testing.T) {
	limiter := NewMemoryLimiter(nil)
	defer limiter.Close()

	if limiter == nil {
		t.Fatal("NewMemoryLimiter returned nil")
	}
}

func TestMemoryLimiter_Allow(t *testing.T) {
	cfg := &Config{
		Requests:        5,
		Window:          time.Second,
		Strategy:        "sliding_window",
		CleanupInterval: time.Minute,
	}
	limiter := NewMemoryLimiter(cfg)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if allowed {
		t.Error("6th request should be denied")
	}
}

func TestMemoryLimiter_AllowN(t *testing.T) {
	cfg := &Config{
		Requests:        10,
		Window:          time.Second,
		Strategy:        "sliding_window",
		CleanupInterval: time.Minute,
	}
	limiter := NewMemoryLimiter(cfg)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Allow 5 requests at once
	allowed, err := limiter.AllowN(ctx, key, 5)
	if err != nil {
		t.Fatalf("AllowN() error = %v", err)
	}
	if !allowed {
		t.Error("5 requests should be allowed")
	}

	// Allow another 5
	allowed, err = limiter.AllowN(ctx, key, 5)
	if err != nil {
		t.Fatalf("AllowN() error = %v", err)
	}
	if !allowed {
		t.Error("another 5 requests should be allowed")
	}

	// 11th request should be denied
	allowed, err = limiter.AllowN(ctx, key, 1)
	if err != nil {
		t.Fatalf("AllowN() error = %v", err)
	}
	if allowed {
		t.Error("11th request should be denied")
	}
}

func TestMemoryLimiter_Reset(t *testing.T) {
	cfg := &Config{
		Requests:        2,
		Window:          time.Second,
		Strategy:        "sliding_window",
		CleanupInterval: time.Minute,
	}
	limiter := NewMemoryLimiter(cfg)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Use up the limit
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("should be rate limited")
	}

	// Reset
	limiter.Reset(ctx, key)

	// Should be allowed again
	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Error("should be allowed after reset")
	}
}

func TestMemoryLimiter_GetInfo(t *testing.T) {
	cfg := &Config{
		Requests:        10,
		Window:          time.Minute,
		Strategy:        "sliding_window",
		CleanupInterval: time.Minute,
	}
	limiter := NewMemoryLimiter(cfg)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Initial state
	info, err := limiter.GetInfo(ctx, key)
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}
	if info.Limit != 10 {
		t.Errorf("Limit = %d, want 10", info.Limit)
	}
	if info.Remaining != 10 {
		t.Errorf("Remaining = %d, want 10", info.Remaining)
	}

	// After some requests
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	info, _ = limiter.GetInfo(ctx, key)
	if info.Remaining != 8 {
		t.Errorf("Remaining = %d, want 8", info.Remaining)
	}
}

func TestMemoryLimiter_TokenBucket(t *testing.T) {
	cfg := &Config{
		Requests:        5,
		Window:          time.Second,
		Strategy:        "token_bucket",
		BurstSize:       2,
		CleanupInterval: time.Minute,
	}
	limiter := NewMemoryLimiter(cfg)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Should allow up to Requests + BurstSize
	for i := 0; i < 7; i++ {
		allowed, _ := limiter.Allow(ctx, key)
		if !allowed {
			t.Errorf("Request %d should be allowed with burst", i+1)
		}
	}
}

func TestMemoryLimiter_Close(t *testing.T) {
	limiter := NewMemoryLimiter(nil)

	err := limiter.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Double close should not error
	err = limiter.Close()
	if err != nil {
		t.Errorf("Double Close() error = %v", err)
	}

	// Operations after close should fail
	ctx := context.Background()
	_, err = limiter.Allow(ctx, "key")
	if err != ErrLimiterClosed {
		t.Errorf("Allow after close should return ErrLimiterClosed, got %v", err)
	}
}

func TestMemoryLimiter_Wait(t *testing.T) {
	cfg := &Config{
		Requests:        1,
		Window:          100 * time.Millisecond,
		Strategy:        "sliding_window",
		CleanupInterval: time.Minute,
	}
	limiter := NewMemoryLimiter(cfg)
	defer limiter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Use up the limit
	limiter.Allow(ctx, "key")

	// Wait should timeout
	err := limiter.Wait(ctx, "key")
	if err != context.DeadlineExceeded {
		t.Errorf("Wait() should timeout, got %v", err)
	}
}

func TestNew(t *testing.T) {
	t.Run("memory backend", func(t *testing.T) {
		limiter, err := New(&Config{
			Backend:         "memory",
			Requests:        10,
			Window:          time.Second,
			CleanupInterval: time.Minute,
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer limiter.Close()
	})

	t.Run("default backend", func(t *testing.T) {
		limiter, err := New(&Config{
			Backend:         "",
			Requests:        10,
			Window:          time.Second,
			CleanupInterval: time.Minute,
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer limiter.Close()
	})

	t.Run("nil config", func(t *testing.T) {
		limiter, err := New(nil)
		if err != nil {
			t.Fatalf("New(nil) error = %v", err)
		}
		defer limiter.Close()
	})
}

func TestKeyExtractors(t *testing.T) {
	ctx := context.Background()
	method := "/test.Service/Method"

	t.Run("DefaultKeyExtractor with x-forwarded-for", func(t *testing.T) {
		metadata := map[string]string{"x-forwarded-for": "192.168.1.1"}
		key := DefaultKeyExtractor(ctx, method, metadata)
		if key != "192.168.1.1" {
			t.Errorf("key = %v, want 192.168.1.1", key)
		}
	})

	t.Run("DefaultKeyExtractor with x-real-ip", func(t *testing.T) {
		metadata := map[string]string{"x-real-ip": "10.0.0.1"}
		key := DefaultKeyExtractor(ctx, method, metadata)
		if key != "10.0.0.1" {
			t.Errorf("key = %v, want 10.0.0.1", key)
		}
	})

	t.Run("DefaultKeyExtractor fallback", func(t *testing.T) {
		metadata := map[string]string{}
		key := DefaultKeyExtractor(ctx, method, metadata)
		if key != "unknown" {
			t.Errorf("key = %v, want unknown", key)
		}
	})

	t.Run("MethodKeyExtractor", func(t *testing.T) {
		key := MethodKeyExtractor(ctx, method, nil)
		if key != method {
			t.Errorf("key = %v, want %v", key, method)
		}
	})

	t.Run("UserKeyExtractor with user", func(t *testing.T) {
		metadata := map[string]string{"x-user-id": "user123"}
		key := UserKeyExtractor(ctx, method, metadata)
		if key != "user123" {
			t.Errorf("key = %v, want user123", key)
		}
	})

	t.Run("UserKeyExtractor fallback", func(t *testing.T) {
		metadata := map[string]string{"x-forwarded-for": "1.2.3.4"}
		key := UserKeyExtractor(ctx, method, metadata)
		if key != "1.2.3.4" {
			t.Errorf("key = %v, want 1.2.3.4", key)
		}
	})

	t.Run("CompositeKeyExtractor", func(t *testing.T) {
		extractor := CompositeKeyExtractor(MethodKeyExtractor, UserKeyExtractor)
		metadata := map[string]string{"x-user-id": "user1"}
		key := extractor(ctx, method, metadata)
		expected := method + ":user1:"
		if key != expected {
			t.Errorf("key = %v, want %v", key, expected)
		}
	})
}

func TestRateLimitedMethods(t *testing.T) {
	defaultCfg := &Config{Requests: 100}
	methods := NewRateLimitedMethods(defaultCfg)

	// Get default
	cfg := methods.Get("/unknown/method")
	if cfg.Requests != 100 {
		t.Errorf("default config Requests = %d, want 100", cfg.Requests)
	}

	// Set specific
	methods.Set("/specific/method", &Config{Requests: 10})
	cfg = methods.Get("/specific/method")
	if cfg.Requests != 10 {
		t.Errorf("specific config Requests = %d, want 10", cfg.Requests)
	}
}
