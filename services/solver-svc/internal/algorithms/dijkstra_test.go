package algorithms

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"logistics/services/solver-svc/internal/graph"
)

func TestDijkstra_SimpleGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 10, 1.0)
	g.AddEdgeWithReverse(1, 2, 10, 2.0)
	g.AddEdgeWithReverse(0, 2, 10, 5.0)

	result := Dijkstra(g, 0)

	// Shortest path to 2: 0->1->2 with cost 3
	if math.Abs(result.Distances[2]-3.0) > 1e-9 {
		t.Errorf("Expected distance 3, got %f", result.Distances[2])
	}

	if result.Parent[2] != 1 {
		t.Errorf("Expected parent[2] = 1, got %d", result.Parent[2])
	}
}

func TestDijkstra_Unreachable(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddNode(0)
	g.AddNode(1)
	g.AddNode(2)
	g.AddEdgeWithReverse(0, 1, 10, 1.0)
	// No path to 2

	result := Dijkstra(g, 0)

	if result.Distances[2] < graph.Infinity-graph.Epsilon {
		t.Errorf("Expected infinity for unreachable node, got %f", result.Distances[2])
	}
}

func TestDijkstraWithPotentials_ReducedCosts(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 10, 5.0)
	g.AddEdgeWithReverse(1, 2, 10, 3.0)
	g.AddEdgeWithReverse(0, 2, 10, 10.0)

	// Potentials (e.g., from Bellman-Ford)
	potentials := map[int64]float64{
		0: 0,
		1: 5,
		2: 8,
	}

	result := DijkstraWithPotentials(g, 0, potentials)

	// With potentials, reduced costs should be non-negative
	// reduced_cost(0->1) = 5 + 0 - 5 = 0
	// reduced_cost(1->2) = 3 + 5 - 8 = 0
	// reduced_cost(0->2) = 10 + 0 - 8 = 2

	// Shortest path: 0->1->2 with reduced cost 0
	if math.Abs(result.Distances[2]-0.0) > 1e-9 {
		t.Errorf("Expected reduced distance 0, got %f", result.Distances[2])
	}
}

func TestDijkstra_ZeroCostEdges(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 10, 0)
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	result := Dijkstra(g, 0)

	if result.Distances[3] != 0 {
		t.Errorf("Expected distance 0, got %f", result.Distances[3])
	}
}

func TestDijkstra_NegativeEdgeFallback(t *testing.T) {
	g := graph.NewResidualGraph()

	// Graph with negative edge - should fall back to Bellman-Ford
	g.AddEdgeWithReverse(0, 1, 10, 5.0)
	g.AddEdgeWithReverse(1, 2, 10, -2.0) // Negative edge
	g.AddEdgeWithReverse(0, 2, 10, 10.0)

	result := Dijkstra(g, 0)

	// Should have used Bellman-Ford fallback
	assert.True(t, result.UsedBellmanFord)
	// Should still find correct distance
	assert.InDelta(t, 3.0, result.Distances[2], graph.Epsilon) // 5 + (-2) = 3
}

func TestDijkstraWithPotentials_NegativeReducedCostFallback(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 10, 5.0)
	g.AddEdgeWithReverse(1, 2, 10, 3.0)

	// Bad potentials that create significant negative reduced cost
	potentials := map[int64]float64{
		0: 0,
		1: 0,   // reduced_cost(0->1) = 5 + 0 - 0 = 5 (ok)
		2: 100, // reduced_cost(1->2) = 3 + 0 - 100 = -97 (negative!)
	}

	result := DijkstraWithPotentials(g, 0, potentials)

	// Should have fallen back to Bellman-Ford due to negative reduced cost
	assert.True(t, result.UsedBellmanFord)
}

func TestDijkstraWithPotentials_SmallNegativeClamped(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 10, 5.0)
	g.AddEdgeWithReverse(1, 2, 10, 3.0)

	// Potentials with tiny numerical error
	potentials := map[int64]float64{
		0: 0,
		1: 5,
		2: 8.0000000001, // Creates tiny negative reduced cost due to float error
	}

	result := DijkstraWithPotentials(g, 0, potentials)

	// Should NOT fall back for tiny errors, just clamp to 0
	assert.False(t, result.Canceled)
	// Distance should still be approximately correct
	assert.InDelta(t, 0.0, result.Distances[2], 0.001)
}

func TestDijkstraWithContext_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()

	// Create large graph
	for i := int64(0); i < 1000; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1.0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := DijkstraWithContext(ctx, g, 0)

	assert.True(t, result.Canceled)
}

func TestDijkstraWithPotentialsContext_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()

	for i := int64(0); i < 500; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1.0)
	}

	potentials := make(map[int64]float64)
	for i := int64(0); i <= 500; i++ {
		potentials[i] = float64(i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := DijkstraWithPotentialsContext(ctx, g, 0, potentials)

	assert.True(t, result.Canceled)
}

func TestDijkstraWithFallback(t *testing.T) {
	tests := []struct {
		name            string
		buildGraph      func() *graph.ResidualGraph
		wantBellmanFord bool
	}{
		{
			name: "no_negative_edges",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 10, 1.0)
				g.AddEdgeWithReverse(1, 2, 10, 2.0)
				return g
			},
			wantBellmanFord: false,
		},
		{
			name: "with_negative_edges",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 10, 1.0)
				g.AddEdgeWithReverse(1, 2, 10, -1.0)
				return g
			},
			wantBellmanFord: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			result := DijkstraWithFallback(context.Background(), g, 0)

			assert.Equal(t, tt.wantBellmanFord, result.UsedBellmanFord)
			assert.False(t, result.Canceled)
		})
	}
}

func TestDijkstra_DiamondGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	// Diamond: multiple paths to destination
	g.AddEdgeWithReverse(0, 1, 10, 1.0)
	g.AddEdgeWithReverse(0, 2, 10, 4.0)
	g.AddEdgeWithReverse(1, 3, 10, 2.0)
	g.AddEdgeWithReverse(2, 3, 10, 1.0)

	result := Dijkstra(g, 0)

	// Shortest path: 0->1->3 with cost 3
	assert.InDelta(t, 3.0, result.Distances[3], graph.Epsilon)
}

func TestDijkstra_SingleNode(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(0)

	result := Dijkstra(g, 0)

	assert.InDelta(t, 0.0, result.Distances[0], graph.Epsilon)
}

func TestDijkstra_ZeroCapacityEdge(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(0, 1, 0, 1.0) // Zero capacity
	g.AddEdgeWithReverse(1, 2, 10, 1.0)

	result := Dijkstra(g, 0)

	// Node 1 and 2 should be unreachable due to zero capacity
	assert.True(t, result.Distances[1] >= graph.Infinity-graph.Epsilon)
	assert.True(t, result.Distances[2] >= graph.Infinity-graph.Epsilon)
}

func TestDijkstraResult_Interface(t *testing.T) {
	result := &DijkstraResult{
		Distances: map[int64]float64{0: 0, 1: 5},
		Parent:    map[int64]int64{0: -1, 1: 0},
	}

	// Test interface methods
	distances := result.GetDistances()
	parent := result.GetParent()

	assert.Equal(t, 0.0, distances[0])
	assert.Equal(t, 5.0, distances[1])
	assert.Equal(t, int64(-1), parent[0])
	assert.Equal(t, int64(0), parent[1])
}

func TestDijkstraWithPotentialsContextEx_FallbackThreshold(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 10, 1.0)
	g.AddEdgeWithReverse(1, 2, 10, 1.0)

	// Potentials causing significant negative reduced cost
	potentials := map[int64]float64{
		0: 0,
		1: 100, // reduced_cost(0->1) = 1 + 0 - 100 = -99
		2: 0,
	}

	// With any fallback threshold, should immediately fall back
	result := DijkstraWithPotentialsContextEx(context.Background(), g, 0, potentials, 1)

	assert.True(t, result.UsedBellmanFord)
}

func TestDijkstra_LargeGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	// Grid graph 50x50
	for i := 0; i < 50; i++ {
		for j := 0; j < 50; j++ {
			node := int64(i*50 + j)
			if i+1 < 50 {
				g.AddEdgeWithReverse(node, int64((i+1)*50+j), 10, float64(i+j+1))
			}
			if j+1 < 50 {
				g.AddEdgeWithReverse(node, int64(i*50+j+1), 10, float64(i+j+1))
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := DijkstraWithContext(ctx, g, 0)

	assert.False(t, result.Canceled)
	assert.False(t, result.UsedBellmanFord)
	// Check that we found distance to far corner
	farCorner := int64(49*50 + 49)
	assert.True(t, result.Distances[farCorner] < graph.Infinity)
}
