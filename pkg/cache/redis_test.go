package cache

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

func TestNewRedisCache(t *testing.T) {
	skipIfNoRedis(t)

	opts := &Options{
		Backend:       "redis",
		RedisAddr:     os.Getenv("REDIS_TEST_ADDR"),
		RedisPassword: os.Getenv("REDIS_TEST_PASSWORD"),
		RedisDB:       0,
		DefaultTTL:    time.Minute,
	}

	cache, err := NewRedisCache(opts)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test Set/Get
	err = cache.Set(ctx, "test-key", []byte("test-value"), time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	val, err := cache.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(val) != "test-value" {
		t.Errorf("Get() = %s, want test-value", string(val))
	}

	// Cleanup
	cache.Delete(ctx, "test-key")
}

func TestRedisCache_NotFound(t *testing.T) {
	skipIfNoRedis(t)

	opts := &Options{
		Backend:   "redis",
		RedisAddr: os.Getenv("REDIS_TEST_ADDR"),
	}

	cache, err := NewRedisCache(opts)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	_, err = cache.Get(context.Background(), "nonexistent-key")
	if err != ErrKeyNotFound {
		t.Errorf("Get() error = %v, want ErrKeyNotFound", err)
	}
}
