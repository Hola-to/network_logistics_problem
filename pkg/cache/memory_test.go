package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	cache := NewMemoryCache(&Options{
		DefaultTTL: 1 * time.Minute,
		MaxEntries: 100,
	})
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set
	err := cache.Set(ctx, key, value, 0)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Get
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("expected %s, got %s", value, got)
	}
}

func TestMemoryCache_GetNotFound(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	_, err := cache.Get(ctx, "nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"

	cache.Set(ctx, key, []byte("value"), 0)

	err := cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = cache.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after delete, got %v", err)
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"

	// Not exists
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}

	// Set and check
	cache.Set(ctx, key, []byte("value"), 0)
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	cache := NewMemoryCache(&Options{
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	})
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"

	cache.Set(ctx, key, []byte("value"), 100*time.Millisecond)

	// Should exist initially
	_, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("expected key to exist: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist
	_, err = cache.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after TTL, got %v", err)
	}
}

func TestMemoryCache_GetWithTTL(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	ttl := 5 * time.Minute

	cache.Set(ctx, key, []byte("value"), ttl)

	value, remainingTTL, err := cache.GetWithTTL(ctx, key)
	if err != nil {
		t.Fatalf("failed to get with TTL: %v", err)
	}

	if string(value) != "value" {
		t.Errorf("expected 'value', got %s", value)
	}

	// TTL should be close to original (within a few seconds)
	if remainingTTL < 4*time.Minute || remainingTTL > ttl {
		t.Errorf("unexpected remaining TTL: %v", remainingTTL)
	}
}

func TestMemoryCache_MGet(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), 0)
	cache.Set(ctx, "key2", []byte("value2"), 0)

	results, err := cache.MGet(ctx, []string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("failed to mget: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if string(results["key1"]) != "value1" {
		t.Errorf("expected 'value1', got %s", results["key1"])
	}
	if string(results["key2"]) != "value2" {
		t.Errorf("expected 'value2', got %s", results["key2"])
	}
}

func TestMemoryCache_MSet(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	entries := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	err := cache.MSet(ctx, entries, 0)
	if err != nil {
		t.Fatalf("failed to mset: %v", err)
	}

	val1, _ := cache.Get(ctx, "key1")
	val2, _ := cache.Get(ctx, "key2")

	if string(val1) != "value1" {
		t.Errorf("expected 'value1', got %s", val1)
	}
	if string(val2) != "value2" {
		t.Errorf("expected 'value2', got %s", val2)
	}
}

func TestMemoryCache_MDelete(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), 0)
	cache.Set(ctx, "key2", []byte("value2"), 0)
	cache.Set(ctx, "key3", []byte("value3"), 0)

	count, err := cache.MDelete(ctx, []string{"key1", "key2", "nonexistent"})
	if err != nil {
		t.Fatalf("failed to mdelete: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	exists, _ := cache.Exists(ctx, "key3")
	if !exists {
		t.Error("key3 should still exist")
	}
}

func TestMemoryCache_Keys(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "prefix:key1", []byte("value1"), 0)
	cache.Set(ctx, "prefix:key2", []byte("value2"), 0)
	cache.Set(ctx, "other:key3", []byte("value3"), 0)

	keys, err := cache.Keys(ctx, "prefix:*")
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestMemoryCache_DeleteByPattern(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "prefix:key1", []byte("value1"), 0)
	cache.Set(ctx, "prefix:key2", []byte("value2"), 0)
	cache.Set(ctx, "other:key3", []byte("value3"), 0)

	count, err := cache.DeleteByPattern(ctx, "prefix:*")
	if err != nil {
		t.Fatalf("failed to delete by pattern: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	exists, _ := cache.Exists(ctx, "other:key3")
	if !exists {
		t.Error("other:key3 should still exist")
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), 0)
	cache.Set(ctx, "key2", []byte("value2"), 0)

	// Generate some hits and misses
	cache.Get(ctx, "key1")
	cache.Get(ctx, "key1")
	cache.Get(ctx, "nonexistent")

	stats, err := cache.Stats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalKeys != 2 {
		t.Errorf("expected 2 total keys, got %d", stats.TotalKeys)
	}
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.Backend != "memory" {
		t.Errorf("expected backend 'memory', got %s", stats.Backend)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", []byte("value1"), 0)
	cache.Set(ctx, "key2", []byte("value2"), 0)

	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	stats, _ := cache.Stats(ctx)
	if stats.TotalKeys != 0 {
		t.Errorf("expected 0 keys after clear, got %d", stats.TotalKeys)
	}
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxEntries: 3,
		DefaultTTL: time.Minute,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "key1", []byte("value1"), 0)
	time.Sleep(10 * time.Millisecond)
	cache.Set(ctx, "key2", []byte("value2"), 0)
	time.Sleep(10 * time.Millisecond)
	cache.Set(ctx, "key3", []byte("value3"), 0)

	// Access key1 to make it recently used
	cache.Get(ctx, "key1")

	// Add new key, should evict key2 (least recently used)
	cache.Set(ctx, "key4", []byte("value4"), 0)

	// key2 should be evicted
	_, err := cache.Get(ctx, "key2")
	if err != ErrKeyNotFound {
		t.Error("expected key2 to be evicted")
	}

	// key1 should still exist
	_, err = cache.Get(ctx, "key1")
	if err != nil {
		t.Error("expected key1 to still exist")
	}
}

func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache(nil)

	ctx := context.Background()
	cache.Set(ctx, "key", []byte("value"), 0)

	err := cache.Close()
	if err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	// Operations after close should return error
	_, err = cache.Get(ctx, "key")
	if err != ErrCacheClosed {
		t.Errorf("expected ErrCacheClosed, got %v", err)
	}

	// Double close should be safe
	err = cache.Close()
	if err != nil {
		t.Errorf("double close should not error: %v", err)
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		want    bool
	}{
		{"* anything", "*", "anything", true},
		{"prefix:* prefix:key", "prefix:*", "prefix:key", true},
		{"prefix:* other:key", "prefix:*", "other:key", false},
		{"*:suffix prefix:suffix", "*:suffix", "prefix:suffix", true},
		{"*:suffix prefix:other", "*:suffix", "prefix:other", false},
		{"exact exact", "exact", "exact", true},
		{"exact other", "exact", "other", false},
		{"middle wildcard match", "solve:*:abc123", "solve:DINIC:abc123", true},
		{"middle wildcard match 2", "solve:*:abc123", "solve:EDMONDS_KARP:abc123", true},
		{"middle wildcard no match prefix", "solve:*:abc123", "other:DINIC:abc123", false},
		{"middle wildcard no match suffix", "solve:*:abc123", "solve:DINIC:xyz789", false},
		{"middle wildcard empty middle", "solve:*:abc123", "solve::abc123", true},
		{"key too short", "prefix*suffix", "presuf", false},
		{"exact length match", "a*b", "ab", true},
		{"exact length match content", "a*b", "axb", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPattern(tt.pattern, tt.key); got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
			}
		})
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"prefix:key", "prefix"},
		{"key", "other"},
		{"a:b:c", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := extractPrefix(tt.key); got != tt.want {
				t.Errorf("extractPrefix(%s) = %s, want %s", tt.key, got, tt.want)
			}
		})
	}
}
