package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/pkg/cache"
)

func BenchmarkMemoryCache_Set(b *testing.B) {
	c := cache.NewMemoryCache(nil)
	defer c.Close()

	ctx := context.Background()
	value := make([]byte, 1024) // 1KB value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(ctx, fmt.Sprintf("key-%d", i%10000), value, time.Minute)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	c := cache.NewMemoryCache(nil)
	defer c.Close()

	ctx := context.Background()
	c.Set(ctx, "benchmark-key", []byte("benchmark-value"), time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ctx, "benchmark-key")
	}
}

func BenchmarkMemoryCache_SetGet(b *testing.B) {
	c := cache.NewMemoryCache(nil)
	defer c.Close()

	ctx := context.Background()
	value := []byte("test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		c.Set(ctx, key, value, time.Minute)
		c.Get(ctx, key)
	}
}

func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	c := cache.NewMemoryCache(nil)
	defer c.Close()

	ctx := context.Background()
	value := []byte("test-value")

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			c.Set(ctx, key, value, time.Minute)
			c.Get(ctx, key)
			i++
		}
	})
}

func BenchmarkMemoryCache_MSet(b *testing.B) {
	c := cache.NewMemoryCache(nil)
	defer c.Close()

	ctx := context.Background()
	entries := make(map[string][]byte)
	for i := 0; i < 100; i++ {
		entries[fmt.Sprintf("mset-key-%d", i)] = []byte("value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.MSet(ctx, entries, time.Minute)
	}
}

func BenchmarkMemoryCache_MGet(b *testing.B) {
	c := cache.NewMemoryCache(nil)
	defer c.Close()

	ctx := context.Background()
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("mget-key-%d", i)
		keys[i] = key
		c.Set(ctx, key, []byte("value"), time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.MGet(ctx, keys)
	}
}

func BenchmarkMemoryCache_ValueSizes(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			c := cache.NewMemoryCache(nil)
			defer c.Close()

			ctx := context.Background()
			value := make([]byte, size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				c.Set(ctx, "key", value, time.Minute)
				c.Get(ctx, "key")
			}
		})
	}
}

func BenchmarkMemoryCache_Eviction(b *testing.B) {
	c := cache.NewMemoryCache(&cache.Options{
		MaxEntries: 1000,
		DefaultTTL: time.Minute,
	})
	defer c.Close()

	ctx := context.Background()
	value := []byte("test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(ctx, fmt.Sprintf("evict-key-%d", i), value, time.Minute)
	}
}

func BenchmarkSolverCache_SetGet(b *testing.B) {
	memCache := cache.NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := cache.NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := createBenchmarkGraph(100)
	result := &cache.CachedSolveResult{
		MaxFlow:   100,
		TotalCost: 500,
		FlowEdges: make([]*cache.FlowEdgeCache, 50),
	}
	for i := 0; i < 50; i++ {
		result.FlowEdges[i] = &cache.FlowEdgeCache{
			From: int64(i), To: int64(i + 1), Flow: 10, Capacity: 10,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		solverCache.Set(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, result, 0)
		solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC)
	}
}

func createBenchmarkGraph(nodes int) *commonv1.Graph {
	g := &commonv1.Graph{
		SourceId: 1,
		SinkId:   int64(nodes),
		Nodes:    make([]*commonv1.Node, nodes),
		Edges:    make([]*commonv1.Edge, nodes-1),
	}
	for i := 0; i < nodes; i++ {
		g.Nodes[i] = &commonv1.Node{Id: int64(i + 1)}
		if i > 0 {
			g.Edges[i-1] = &commonv1.Edge{
				From:     int64(i),
				To:       int64(i + 1),
				Capacity: 10,
				Cost:     1,
			}
		}
	}
	return g
}
