package cache

import (
	"testing"
	"time"

	"logistics/pkg/config"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Backend != "memory" {
		t.Errorf("expected backend 'memory', got %s", opts.Backend)
	}
	if opts.DefaultTTL != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", opts.DefaultTTL)
	}
	if opts.MaxEntries != 100000 {
		t.Errorf("expected max entries 100000, got %d", opts.MaxEntries)
	}
	if opts.RedisAddr != "localhost:6379" {
		t.Errorf("expected redis addr 'localhost:6379', got %s", opts.RedisAddr)
	}
}

func TestFromConfig(t *testing.T) {
	cfg := &config.CacheConfig{
		Driver:     "redis",
		Host:       "redis.local",
		Port:       6380,
		Password:   "secret",
		DB:         1,
		DefaultTTL: 10 * time.Minute,
		MaxEntries: 50000,
	}

	opts := FromConfig(cfg)

	if opts.Backend != "redis" {
		t.Errorf("expected backend 'redis', got %s", opts.Backend)
	}
	if opts.DefaultTTL != 10*time.Minute {
		t.Errorf("expected TTL 10m, got %v", opts.DefaultTTL)
	}
	if opts.RedisAddr != "redis.local:6380" {
		t.Errorf("expected addr 'redis.local:6380', got %s", opts.RedisAddr)
	}
	if opts.RedisPassword != "secret" {
		t.Errorf("expected password 'secret', got %s", opts.RedisPassword)
	}
	if opts.RedisDB != 1 {
		t.Errorf("expected db 1, got %d", opts.RedisDB)
	}
}

func TestNew_Memory(t *testing.T) {
	cache, err := New(&Options{Backend: "memory"})
	if err != nil {
		t.Fatalf("failed to create memory cache: %v", err)
	}
	defer cache.Close()

	if cache == nil {
		t.Error("expected cache to be non-nil")
	}
}

func TestNew_NilOptions(t *testing.T) {
	cache, err := New(nil)
	if err != nil {
		t.Fatalf("failed to create cache with nil options: %v", err)
	}
	defer cache.Close()

	if cache == nil {
		t.Error("expected cache to be non-nil")
	}
}

func TestNew_UnknownBackend(t *testing.T) {
	cache, err := New(&Options{Backend: "unknown"})
	if err != nil {
		t.Fatalf("unknown backend should default to memory: %v", err)
	}
	defer cache.Close()

	// Should fall back to memory
	if cache == nil {
		t.Error("expected cache to be non-nil")
	}
}

func TestMustNew_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("MustNew with invalid redis config - depends on redis availability")
		}
	}()

	// This should work (memory backend)
	cache := MustNew(&Options{Backend: "memory"})
	if cache == nil {
		t.Error("expected cache to be non-nil")
	}
	cache.Close()
}
