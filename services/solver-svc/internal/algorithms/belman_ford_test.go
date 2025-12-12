package algorithms

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"logistics/services/solver-svc/internal/graph"
)

func TestBellmanFord(t *testing.T) {
	tests := []struct {
		name              string
		buildGraph        func() *graph.ResidualGraph
		source            int64
		wantDistances     map[int64]float64
		wantParent        map[int64]int64
		wantNegativeCycle bool
	}{
		{
			name: "simple_linear_graph",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 2.0)
				g.AddEdge(3, 4, 10, 3.0)
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 1,
				3: 3,
				4: 6,
			},
			wantNegativeCycle: false,
		},
		{
			name: "graph_with_multiple_paths",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Path 1->2->4: cost 5
				g.AddEdge(1, 2, 10, 2.0)
				g.AddEdge(2, 4, 10, 3.0)
				// Path 1->3->4: cost 4
				g.AddEdge(1, 3, 10, 1.0)
				g.AddEdge(3, 4, 10, 3.0)
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 2,
				3: 1,
				4: 4, // Shortest path via 3
			},
			wantNegativeCycle: false,
		},
		{
			name: "disconnected_nodes",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddNode(3) // Isolated node
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 1,
				3: graph.Infinity,
			},
			wantNegativeCycle: false,
		},
		{
			name: "graph_with_zero_cost_edges",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 0)
				g.AddEdge(2, 3, 10, 0)
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 0,
				3: 0,
			},
			wantNegativeCycle: false,
		},
		{
			name: "single_node",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
			},
			wantNegativeCycle: false,
		},
		{
			name: "diamond_graph",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				//     2
				//    / \
				//   1   4
				//    \ /
				//     3
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(1, 3, 10, 4.0)
				g.AddEdge(2, 4, 10, 2.0)
				g.AddEdge(3, 4, 10, 1.0)
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 1,
				3: 4,
				4: 3, // Via 2
			},
			wantNegativeCycle: false,
		},
		{
			name: "graph_with_negative_edges_no_cycle",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 5.0)
				g.AddEdge(1, 3, 10, 2.0)
				g.AddEdge(3, 2, 10, -2.0) // Negative edge
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 0, // Via 3 with negative edge
				3: 2,
			},
			wantNegativeCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			result := BellmanFord(g, tt.source)

			assert.Equal(t, tt.wantNegativeCycle, result.HasNegativeCycle, "negative cycle mismatch")

			for node, expectedDist := range tt.wantDistances {
				actualDist, exists := result.Distances[node]
				if expectedDist == graph.Infinity {
					assert.True(t, actualDist == graph.Infinity || !exists,
						"node %d should be unreachable", node)
				} else {
					require.True(t, exists, "node %d should have distance", node)
					assert.InDelta(t, expectedDist, actualDist, graph.Epsilon,
						"distance to node %d", node)
				}
			}
		})
	}
}

func TestBellmanFord_NegativeCycle(t *testing.T) {
	tests := []struct {
		name       string
		buildGraph func() *graph.ResidualGraph
		source     int64
		wantCycle  bool
	}{
		{
			name: "simple_negative_cycle",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Cycle: 1 -> 2 -> 3 -> 1 with total cost -1
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 1.0)
				g.AddEdge(3, 1, 10, -3.0) // Creates negative cycle
				return g
			},
			source:    1,
			wantCycle: true,
		},
		{
			name: "negative_cycle_not_reachable_from_source",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				// Cycle not reachable from source
				g.AddEdge(3, 4, 10, 1.0)
				g.AddEdge(4, 5, 10, 1.0)
				g.AddEdge(5, 3, 10, -5.0)
				return g
			},
			source:    1,
			wantCycle: false, // Cycle unreachable from source
		},
		{
			name: "reachable_negative_cycle",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 1.0)
				g.AddEdge(3, 4, 10, 1.0)
				g.AddEdge(4, 2, 10, -5.0) // Cycle 2->3->4->2
				return g
			},
			source:    1,
			wantCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			result := BellmanFord(g, tt.source)
			assert.Equal(t, tt.wantCycle, result.HasNegativeCycle)
		})
	}
}

func TestBellmanFordWithPotentials(t *testing.T) {
	tests := []struct {
		name       string
		buildGraph func() *graph.ResidualGraph
		source     int64
		potentials map[int64]float64
		wantDist   map[int64]float64
	}{
		{
			name: "simple_with_zero_potentials",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 3.0)
				g.AddEdge(2, 3, 10, 2.0)
				return g
			},
			source:     1,
			potentials: map[int64]float64{1: 0, 2: 0, 3: 0},
			wantDist:   map[int64]float64{1: 0, 2: 3, 3: 5},
		},
		{
			name: "with_initialized_potentials",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 5.0)
				g.AddEdge(2, 3, 10, 3.0)
				return g
			},
			source:     1,
			potentials: map[int64]float64{1: 0, 2: 5, 3: 8},
			wantDist: map[int64]float64{
				1: 0,
				2: 0, // reduced cost = 5 + 0 - 5 = 0
				3: 0, // reduced cost = 3 + 5 - 8 = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			result := BellmanFordWithPotentials(g, tt.source, tt.potentials)

			for node, expected := range tt.wantDist {
				assert.InDelta(t, expected, result.Distances[node], graph.Epsilon,
					"distance to node %d", node)
			}
		})
	}
}

func TestFindShortestPath(t *testing.T) {
	tests := []struct {
		name       string
		buildGraph func() *graph.ResidualGraph
		source     int64
		sink       int64
		wantPath   []int64
		wantCost   float64
		wantFound  bool
	}{
		{
			name: "simple_path",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 2.0)
				return g
			},
			source:    1,
			sink:      3,
			wantPath:  []int64{1, 2, 3},
			wantCost:  3.0,
			wantFound: true,
		},
		{
			name: "no_path",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddNode(3)
				return g
			},
			source:    1,
			sink:      3,
			wantPath:  nil,
			wantCost:  0,
			wantFound: false,
		},
		{
			name: "choose_shorter_path",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Long path: 1->2->3->4, cost=6
				g.AddEdge(1, 2, 10, 2.0)
				g.AddEdge(2, 3, 10, 2.0)
				g.AddEdge(3, 4, 10, 2.0)
				// Short path: 1->4, cost=5
				g.AddEdge(1, 4, 10, 5.0)
				return g
			},
			source:    1,
			sink:      4,
			wantPath:  []int64{1, 4},
			wantCost:  5.0,
			wantFound: true,
		},
		{
			name: "source_equals_sink",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				return g
			},
			source:    1,
			sink:      1,
			wantPath:  []int64{1},
			wantCost:  0,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			path, cost, found := FindShortestPath(g, tt.source, tt.sink)

			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantPath, path)
				assert.InDelta(t, tt.wantCost, cost, graph.Epsilon)
			}
		})
	}
}

func TestBellmanFordToSink(t *testing.T) {
	tests := []struct {
		name         string
		buildGraph   func() *graph.ResidualGraph
		source       int64
		sink         int64
		wantDistance float64
		wantCycle    bool
	}{
		{
			name: "simple_path_to_sink",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 2.0)
				g.AddEdge(3, 4, 10, 3.0)
				return g
			},
			source:       1,
			sink:         4,
			wantDistance: 6.0,
			wantCycle:    false,
		},
		{
			name: "early_termination_when_sink_stable",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Direct path to sink
				g.AddEdge(1, 2, 10, 1.0)
				// Longer paths that won't improve sink distance
				g.AddEdge(1, 3, 10, 5.0)
				g.AddEdge(3, 2, 10, 5.0)
				return g
			},
			source:       1,
			sink:         2,
			wantDistance: 1.0,
			wantCycle:    false,
		},
		{
			name: "sink_unreachable",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddNode(3) // Disconnected sink
				return g
			},
			source:       1,
			sink:         3,
			wantDistance: graph.Infinity,
			wantCycle:    false,
		},
		{
			name: "negative_cycle_on_path_to_sink",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 1.0)
				g.AddEdge(3, 2, 10, -5.0) // Negative cycle 2->3->2
				g.AddEdge(3, 4, 10, 1.0)
				return g
			},
			source:    1,
			sink:      4,
			wantCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			ctx := context.Background()
			result := BellmanFordToSink(ctx, g, tt.source, tt.sink)

			assert.Equal(t, tt.wantCycle, result.HasNegativeCycle)
			if !tt.wantCycle {
				if tt.wantDistance == graph.Infinity {
					assert.True(t, result.Distances[tt.sink] >= graph.Infinity-graph.Epsilon)
				} else {
					assert.InDelta(t, tt.wantDistance, result.Distances[tt.sink], graph.Epsilon)
				}
			}
		})
	}
}

func TestBellmanFordToSink_ContextCancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	// Large graph to ensure cancellation happens
	for i := int64(1); i < 1000; i++ {
		g.AddEdge(i, i+1, 10, 1.0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()

	// Give context time to expire
	time.Sleep(10 * time.Microsecond)

	result := BellmanFordToSink(ctx, g, 1, 1000)

	assert.True(t, result.Canceled)
}

func TestBellmanFord_LargeGraph(t *testing.T) {
	g := graph.NewResidualGraph()
	n := 1000

	// Create linear graph
	for i := 1; i < n; i++ {
		g.AddEdge(int64(i), int64(i+1), 10, 1.0)
	}

	result := BellmanFord(g, 1)

	assert.False(t, result.HasNegativeCycle)
	assert.InDelta(t, float64(n-1), result.Distances[int64(n)], graph.Epsilon)
}

func TestBellmanFord_ComplexNegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	// Main path
	g.AddEdge(1, 2, 10, 1.0)
	g.AddEdge(2, 3, 10, 1.0)
	g.AddEdge(3, 4, 10, 1.0)
	g.AddEdge(4, 5, 10, 1.0)

	// Positive cycle
	g.AddEdge(2, 6, 10, 1.0)
	g.AddEdge(6, 2, 10, 1.0) // Cycle 2->6->2 = +2

	// Negative cycle
	g.AddEdge(3, 7, 10, 1.0)
	g.AddEdge(7, 8, 10, 1.0)
	g.AddEdge(8, 3, 10, -5.0) // Cycle 3->7->8->3 = -3

	result := BellmanFord(g, 1)
	assert.True(t, result.HasNegativeCycle)
}

func TestBellmanFordWithPotentials_NegativeCycleDetection(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)
	g.AddEdgeWithReverse(3, 2, 10, -5) // Negative cost creates cycle

	potentials := map[int64]float64{
		1: 0,
		2: 0,
		3: 0,
	}

	result := BellmanFordWithPotentials(g, 1, potentials)

	assert.True(t, result.HasNegativeCycle)
}

func TestFindShortestPath_NegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, -5)
	g.AddEdgeWithReverse(3, 2, 10, -5) // Negative cycle 2-3-2
	g.AddEdgeWithReverse(3, 4, 10, 1)

	path, cost, found := FindShortestPath(g, 1, 4)

	assert.False(t, found)
	assert.Nil(t, path)
	assert.Equal(t, 0.0, cost)
}

func TestBellmanFordWithPotentials_ReducedCostNegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 2)
	g.AddEdgeWithReverse(2, 3, 10, 2)
	g.AddEdgeWithReverse(3, 1, 10, -10) // Cycle with very negative cost
	g.AddEdgeWithReverse(3, 4, 10, 1)

	potentials := map[int64]float64{
		1: 0,
		2: 2,
		3: 4,
		4: 5,
	}

	result := BellmanFordWithPotentials(g, 1, potentials)

	// Reduced cost for 3->1: -10 + 4 - 0 = -6 (negative)
	assert.True(t, result.HasNegativeCycle)
}

func TestBellmanFordWithContext_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i < 500; i++ {
		g.AddEdge(i, i+1, 10, 1.0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := BellmanFordWithContext(ctx, g, 1)

	assert.True(t, result.Canceled)
}

func TestBellmanFordWithPotentialsContext_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i < 500; i++ {
		g.AddEdge(i, i+1, 10, 1.0)
	}

	potentials := make(map[int64]float64)
	for i := int64(1); i <= 500; i++ {
		potentials[i] = 0
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := BellmanFordWithPotentialsContext(ctx, g, 1, potentials)

	assert.True(t, result.Canceled)
}

func TestBellmanFordWithPotentials_NegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdge(1, 2, 10, 1)
	g.AddEdge(2, 3, 10, 1)
	g.AddEdge(3, 1, 10, -5) // Creates cycle with cost = 1+1-5 = -3

	potentials := map[int64]float64{1: 0, 2: 0, 3: 0}

	result := BellmanFordWithPotentials(g, 1, potentials)

	assert.True(t, result.HasNegativeCycle)
}

func TestBellmanFordWithPotentials_NoNegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)

	potentials := map[int64]float64{1: 0, 2: 0, 3: 0}

	result := BellmanFordWithPotentials(g, 1, potentials)

	assert.False(t, result.HasNegativeCycle)
}

func TestBellmanFordWithPotentials_ReducedCosts(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 5)
	g.AddEdgeWithReverse(2, 3, 10, 3)

	// Set potentials such that reduced costs are different
	potentials := map[int64]float64{1: 0, 2: 5, 3: 8}

	result := BellmanFordWithPotentials(g, 1, potentials)

	assert.False(t, result.HasNegativeCycle)
	assert.InDelta(t, 0.0, result.Distances[1], 1e-9)
}

func TestBellmanFordWithPotentials_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()

	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1)
	}

	potentials := make(map[int64]float64)
	for i := int64(0); i <= 100; i++ {
		potentials[i] = 0
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := BellmanFordWithPotentialsContext(ctx, g, 0, potentials)

	assert.True(t, result.Canceled)
}

// =============================================================================
// BellmanFord Edge Cases
// =============================================================================

func TestBellmanFord_EmptyGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	result := BellmanFord(g, 1)

	assert.NotNil(t, result)
	assert.False(t, result.HasNegativeCycle)
}

func TestBellmanFord_SingleNode(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)

	result := BellmanFord(g, 1)

	assert.InDelta(t, 0.0, result.Distances[1], 1e-9)
}

func TestBellmanFord_DisconnectedNodes(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)
	g.AddEdgeWithReverse(1, 2, 10, 5)
	// Node 3 is disconnected

	result := BellmanFord(g, 1)

	assert.InDelta(t, 0.0, result.Distances[1], 1e-9)
	assert.InDelta(t, 5.0, result.Distances[2], 1e-9)
	assert.InDelta(t, graph.Infinity, result.Distances[3], 1e-9)
}

func TestBellmanFord_NegativeEdgeCost(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, -5)
	g.AddEdgeWithReverse(2, 3, 10, 3)

	result := BellmanFord(g, 1)

	assert.InDelta(t, -5.0, result.Distances[2], 1e-9)
	assert.InDelta(t, -2.0, result.Distances[3], 1e-9)
}

func TestBellmanFord_MultipleRelaxations(t *testing.T) {
	g := graph.NewResidualGraph()

	// Graph where path 1->2->4 is shorter than 1->3->4
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(1, 3, 10, 5)
	g.AddEdgeWithReverse(2, 4, 10, 1)
	g.AddEdgeWithReverse(3, 4, 10, 1)

	result := BellmanFord(g, 1)

	assert.InDelta(t, 2.0, result.Distances[4], 1e-9) // Path 1->2->4
}

func TestBellmanFord_ZeroCapacityEdge(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 0, 5) // Zero capacity
	g.AddEdgeWithReverse(2, 3, 10, 3)

	result := BellmanFord(g, 1)

	// Should not traverse zero capacity edge
	assert.InDelta(t, graph.Infinity, result.Distances[2], 1e-9)
}
