package benchmark

import (
	"fmt"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/pkg/cache"
)

func BenchmarkGraphHash(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		graph := createGraphForBenchmark(size)
		b.Run(fmt.Sprintf("nodes_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cache.GraphHash(graph)
			}
		})
	}
}

func BenchmarkGraphHash_DenseGraph(b *testing.B) {
	sizes := []int{50, 100, 200}

	for _, size := range sizes {
		graph := createDenseGraphForBenchmark(size)
		b.Run(fmt.Sprintf("nodes_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cache.GraphHash(graph)
			}
		})
	}
}

func BenchmarkQuickHash(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cache.QuickHash(data)
			}
		})
	}
}

func BenchmarkShortHash(b *testing.B) {
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.ShortHash(data)
	}
}

func BenchmarkBuildSolveKey(b *testing.B) {
	graphHash := "abc123def456"
	algorithm := "ALGORITHM_DINIC"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.BuildSolveKey(graphHash, algorithm)
	}
}

func BenchmarkBuildSolveKeyWithOptions(b *testing.B) {
	graphHash := "abc123def456"
	algorithm := "ALGORITHM_DINIC"
	optionsHash := "opts789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.BuildSolveKeyWithOptions(graphHash, algorithm, optionsHash)
	}
}

func createGraphForBenchmark(nodes int) *commonv1.Graph {
	g := &commonv1.Graph{
		SourceId: 1,
		SinkId:   int64(nodes),
		Nodes:    make([]*commonv1.Node, nodes),
		Edges:    make([]*commonv1.Edge, nodes-1),
	}
	for i := 0; i < nodes; i++ {
		g.Nodes[i] = &commonv1.Node{
			Id:   int64(i + 1),
			Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE,
		}
		if i > 0 {
			g.Edges[i-1] = &commonv1.Edge{
				From:     int64(i),
				To:       int64(i + 1),
				Capacity: 10,
				Cost:     float64(i),
			}
		}
	}
	return g
}

func createDenseGraphForBenchmark(nodes int) *commonv1.Graph {
	edgeCount := nodes * 5 // Примерно 5 рёбер на узел

	g := &commonv1.Graph{
		SourceId: 1,
		SinkId:   int64(nodes),
		Nodes:    make([]*commonv1.Node, nodes),
		Edges:    make([]*commonv1.Edge, 0, edgeCount),
	}

	for i := 0; i < nodes; i++ {
		g.Nodes[i] = &commonv1.Node{Id: int64(i + 1)}
	}

	for i := 0; i < nodes; i++ {
		for j := i + 1; j < nodes && j <= i+5; j++ {
			g.Edges = append(g.Edges, &commonv1.Edge{
				From:     int64(i + 1),
				To:       int64(j + 1),
				Capacity: 10,
			})
		}
	}

	return g
}
