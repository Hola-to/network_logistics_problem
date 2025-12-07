//go:build integration

package pkg_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/pkg/cache"
	"logistics/tests/integration/testutil"
)

func TestRedisCache_SetGetDelete(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:    "redis",
		RedisAddr:  addr,
		DefaultTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	key := testutil.UniqueKey(t, "cache")

	// Set
	err = c.Set(ctx, key, []byte("test-value"), time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	val, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(val) != "test-value" {
		t.Errorf("value = %s, want test-value", string(val))
	}

	// Delete
	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = c.Get(ctx, key)
	if err != cache.ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after delete, got %v", err)
	}
}

func TestRedisCache_Exists(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	key := testutil.UniqueKey(t, "exists")

	// Should not exist
	exists, err := c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("key should not exist initially")
	}

	// Set
	c.Set(ctx, key, []byte("value"), time.Minute)

	// Should exist
	exists, err = c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("key should exist after set")
	}

	// Cleanup
	c.Delete(ctx, key)
}

func TestRedisCache_TTLExpiry(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	key := testutil.UniqueKey(t, "ttl")

	// Set with short TTL
	err = c.Set(ctx, key, []byte("value"), 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	_, err = c.Get(ctx, key)
	if err != nil {
		t.Fatalf("should exist immediately: %v", err)
	}

	// Wait for expiry
	time.Sleep(300 * time.Millisecond)

	// Should be expired
	_, err = c.Get(ctx, key)
	if err != cache.ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after TTL, got %v", err)
	}
}

func TestRedisCache_GetWithTTL(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	key := testutil.UniqueKey(t, "getttl")

	// Set with 1 minute TTL
	c.Set(ctx, key, []byte("value"), time.Minute)

	// Get with TTL
	val, ttl, err := c.GetWithTTL(ctx, key)
	if err != nil {
		t.Fatalf("GetWithTTL failed: %v", err)
	}
	if string(val) != "value" {
		t.Errorf("value = %s, want value", string(val))
	}
	if ttl < 50*time.Second || ttl > time.Minute {
		t.Errorf("ttl = %v, expected ~1 minute", ttl)
	}

	c.Delete(ctx, key)
}

func TestRedisCache_MOperations(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	prefix := testutil.UniqueKey(t, "mops")

	// MSet
	entries := map[string][]byte{
		prefix + ":1": []byte("v1"),
		prefix + ":2": []byte("v2"),
		prefix + ":3": []byte("v3"),
	}
	err = c.MSet(ctx, entries, time.Minute)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// MGet
	keys := []string{prefix + ":1", prefix + ":2", prefix + ":3", prefix + ":missing"}
	result, err := c.MGet(ctx, keys)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("MGet returned %d keys, want 3", len(result))
	}
	if string(result[prefix+":1"]) != "v1" {
		t.Errorf("result[:1] = %s, want v1", string(result[prefix+":1"]))
	}

	// MDelete
	count, err := c.MDelete(ctx, []string{prefix + ":1", prefix + ":2", prefix + ":3"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}
	if count != 3 {
		t.Errorf("MDelete count = %d, want 3", count)
	}
}

func TestRedisCache_Keys_DeleteByPattern(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	prefix := testutil.UniqueKey(t, "pattern")

	// Setup
	c.Set(ctx, prefix+":a:1", []byte("1"), time.Minute)
	c.Set(ctx, prefix+":a:2", []byte("2"), time.Minute)
	c.Set(ctx, prefix+":b:1", []byte("3"), time.Minute)

	// Keys with pattern
	keys, err := c.Keys(ctx, prefix+":a:*")
	if err != nil {
		t.Fatalf("Keys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("Keys returned %d, want 2", len(keys))
	}

	// DeleteByPattern - all with prefix
	count, err := c.DeleteByPattern(ctx, prefix+":*")
	if err != nil {
		t.Fatalf("DeleteByPattern failed: %v", err)
	}
	if count != 3 {
		t.Errorf("DeleteByPattern count = %d, want 3", count)
	}

	// Verify all deleted
	keys, _ = c.Keys(ctx, prefix+":*")
	if len(keys) != 0 {
		t.Errorf("should have 0 keys after delete, got %d", len(keys))
	}
}

func TestRedisCache_Stats(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	stats, err := c.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.Backend != "redis" {
		t.Errorf("Backend = %s, want redis", stats.Backend)
	}
	if stats.TotalKeys < 0 {
		t.Error("TotalKeys should not be negative")
	}
}

func TestRedisCache_Clear(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
		RedisDB:   15, // Use separate DB for this test
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	// Add some data
	for i := 0; i < 10; i++ {
		c.Set(ctx, fmt.Sprintf("clear:key:%d", i), []byte("value"), time.Minute)
	}

	// Clear
	err = c.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify empty
	stats, _ := c.Stats(ctx)
	if stats.TotalKeys != 0 {
		t.Errorf("TotalKeys = %d after clear, want 0", stats.TotalKeys)
	}
}

func TestRedisCache_Concurrent(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:       "redis",
		RedisAddr:     addr,
		RedisPoolSize: 20,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	prefix := testutil.UniqueKey(t, "concurrent")

	var wg sync.WaitGroup
	errors := make(chan error, 200)

	// 100 concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("%s:%d", prefix, id)
			if err := c.Set(ctx, key, []byte(fmt.Sprintf("value-%d", id)), time.Minute); err != nil {
				errors <- fmt.Errorf("set %d: %w", id, err)
			}
		}(i)
	}

	wg.Wait()

	// 100 concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("%s:%d", prefix, id)
			val, err := c.Get(ctx, key)
			if err != nil {
				errors <- fmt.Errorf("get %d: %w", id, err)
				return
			}
			expected := fmt.Sprintf("value-%d", id)
			if string(val) != expected {
				errors <- fmt.Errorf("value mismatch for %d: got %s, want %s", id, string(val), expected)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check errors
	for err := range errors {
		t.Error(err)
	}

	// Cleanup
	c.DeleteByPattern(ctx, prefix+":*")
}

func TestRedisCache_SolverCache(t *testing.T) {
	addr := testutil.RequireRedis(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	c, err := cache.NewRedisCache(&cache.Options{
		Backend:   "redis",
		RedisAddr: addr,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	testutil.Cleanup(t, func() { c.Close() })

	solverCache := cache.NewSolverCache(c, 5*time.Minute)

	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 1, To: 3, Capacity: 15, Cost: 2},
			{From: 2, To: 4, Capacity: 10, Cost: 1},
			{From: 3, To: 4, Capacity: 15, Cost: 1},
		},
	}

	result := &cache.CachedSolveResult{
		MaxFlow:           25,
		TotalCost:         50,
		Status:            "FLOW_STATUS_OPTIMAL",
		Iterations:        10,
		ComputationTimeMs: 5.5,
		FlowEdges: []*cache.FlowEdgeCache{
			{From: 1, To: 2, Flow: 10, Capacity: 10, Utilization: 1.0},
			{From: 1, To: 3, Flow: 15, Capacity: 15, Utilization: 1.0},
		},
	}

	// Set
	err = solverCache.Set(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, result, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	got, found, err := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("expected to find cached result")
	}
	if got.MaxFlow != 25 {
		t.Errorf("MaxFlow = %f, want 25", got.MaxFlow)
	}

	// Different algorithm should not be found
	_, found, _ = solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
	if found {
		t.Error("should not find result for different algorithm")
	}

	// Invalidate
	err = solverCache.Invalidate(ctx, graph)
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// Should be gone
	_, found, _ = solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC)
	if found {
		t.Error("should not find result after invalidate")
	}
}
