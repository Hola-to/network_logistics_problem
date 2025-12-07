package benchmark

import (
	"fmt"
	"testing"

	"logistics/pkg/domain"
)

func BenchmarkBFS(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes_%d", size), func(b *testing.B) {
			g := generateLinearGraph(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				domain.BFS(g, g.SourceID, g.SinkID)
			}
		})
	}
}

func BenchmarkBFS_Dense(b *testing.B) {
	sizes := []int{50, 100, 200}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes_%d", size), func(b *testing.B) {
			g := generateDenseGraph(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				domain.BFS(g, g.SourceID, g.SinkID)
			}
		})
	}
}

func BenchmarkBFSLevel(b *testing.B) {
	sizes := []int{100, 500, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes_%d", size), func(b *testing.B) {
			g := generateLinearGraph(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				domain.BFSLevel(g, g.SourceID)
			}
		})
	}
}

func BenchmarkBFSReachable(b *testing.B) {
	g := generateLinearGraph(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.BFSReachable(g, g.SourceID)
	}
}

func BenchmarkFindConnectedComponents(b *testing.B) {
	g := generateDisconnectedGraph(1000, 10) // 10 components of 100 nodes each

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.FindConnectedComponents(g)
	}
}

func BenchmarkGraph_Clone(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes_%d", size), func(b *testing.B) {
			g := generateLinearGraph(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				g.Clone()
			}
		})
	}
}

func BenchmarkGraph_Validate(b *testing.B) {
	g := generateLinearGraph(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Validate()
	}
}

func BenchmarkCalculateGraphStatistics(b *testing.B) {
	g := generateLinearGraph(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.CalculateGraphStatistics(g)
	}
}

func BenchmarkCalculateFlowStatistics(b *testing.B) {
	g := generateGraphWithFlow(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.CalculateFlowStatistics(g)
	}
}

func BenchmarkFindBottlenecks(b *testing.B) {
	g := generateGraphWithFlow(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.FindBottlenecks(g, 0.9)
	}
}

func BenchmarkReconstructPath(b *testing.B) {
	// Simulate BFS result
	parent := make(map[int64]int64)
	for i := int64(1); i < 1000; i++ {
		parent[i+1] = i
	}
	parent[1] = -1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.ReconstructPath(parent, 1, 1000)
	}
}

func BenchmarkAugmentPath(b *testing.B) {
	g := generateLinearGraph(100)
	path := make([]int64, 100)
	for i := 0; i < 100; i++ {
		path[i] = int64(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.AugmentPath(g, path, 1.0)
		// Reset flow for next iteration
		for _, edge := range g.Edges {
			edge.CurrentFlow = 0
		}
	}
}

// Helper functions

func generateLinearGraph(nodes int) *domain.Graph {
	g := domain.NewGraph()
	g.SourceID = 1
	g.SinkID = int64(nodes)

	for i := 1; i <= nodes; i++ {
		g.AddNode(&domain.Node{ID: int64(i)})
	}
	for i := 1; i < nodes; i++ {
		g.AddEdge(&domain.Edge{
			From:     int64(i),
			To:       int64(i + 1),
			Capacity: 100,
		})
	}
	return g
}

func generateDenseGraph(nodes int) *domain.Graph {
	g := domain.NewGraph()
	g.SourceID = 1
	g.SinkID = int64(nodes)

	for i := 1; i <= nodes; i++ {
		g.AddNode(&domain.Node{ID: int64(i)})
	}

	// Add edges to create dense graph
	for i := 1; i <= nodes; i++ {
		for j := i + 1; j <= nodes && j <= i+10; j++ {
			g.AddEdge(&domain.Edge{
				From:     int64(i),
				To:       int64(j),
				Capacity: 100,
			})
		}
	}
	return g
}

func generateDisconnectedGraph(totalNodes, components int) *domain.Graph {
	g := domain.NewGraph()
	nodesPerComponent := totalNodes / components

	nodeID := int64(1)
	for c := 0; c < components; c++ {
		startID := nodeID
		for i := 0; i < nodesPerComponent; i++ {
			g.AddNode(&domain.Node{ID: nodeID})
			if i > 0 {
				g.AddEdge(&domain.Edge{
					From:     nodeID - 1,
					To:       nodeID,
					Capacity: 100,
				})
			}
			nodeID++
		}
		if c == 0 {
			g.SourceID = startID
			g.SinkID = nodeID - 1
		}
	}
	return g
}

func generateGraphWithFlow(nodes int) *domain.Graph {
	g := generateLinearGraph(nodes)

	// Add flow
	for _, edge := range g.Edges {
		edge.CurrentFlow = edge.Capacity * 0.7
	}

	return g
}
