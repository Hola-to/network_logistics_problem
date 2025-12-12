package algorithms

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"logistics/services/solver-svc/internal/graph"
)

func TestCapacityScalingMinCostFlow_Simple(t *testing.T) {
	g := graph.NewResidualGraph()

	// Simple graph:
	//   1 --10,cost=1--> 2 --10,cost=1--> 4
	//   |                                 ^
	//   +--5,cost=2--> 3 --5,cost=2-------+
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)
	g.AddNode(4)

	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 4, 10, 1)
	g.AddEdgeWithReverse(1, 3, 5, 2)
	g.AddEdgeWithReverse(3, 4, 5, 2)

	result := CapacityScalingMinCostFlow(g, 1, 4, 15, nil)

	if result.Flow < 15-graph.Epsilon {
		t.Errorf("Expected flow 15, got %f", result.Flow)
	}

	// Optimal cost: 10*2 + 5*4 = 40
	expectedCost := 40.0
	if result.Cost < expectedCost-1 || result.Cost > expectedCost+1 {
		t.Errorf("Expected cost ~%f, got %f", expectedCost, result.Cost)
	}
}

func TestCapacityScalingMinCostFlow_LargeCapacity(t *testing.T) {
	g := graph.NewResidualGraph()

	// Graph with large capacity (should activate Capacity Scaling)
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	largeCap := 1e7
	g.AddEdgeWithReverse(1, 2, largeCap, 1)
	g.AddEdgeWithReverse(2, 3, largeCap, 1)
	g.AddEdgeWithReverse(1, 3, largeCap/2, 3)

	result := CapacityScalingMinCostFlow(g, 1, 3, largeCap, nil)

	if result.Flow < largeCap-graph.Epsilon {
		t.Errorf("Expected flow %f, got %f", largeCap, result.Flow)
	}
}

func TestCapacityScalingMinCostFlow_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()

	// Large graph for cancellation testing
	for i := int64(1); i <= 100; i++ {
		g.AddNode(i)
	}
	for i := int64(1); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 1e8, float64(i))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	result := CapacityScalingMinCostFlowWithContext(ctx, g, 1, 100, 1e8, nil)

	if !result.Canceled {
		t.Log("Operation completed before timeout (may happen on fast systems)")
	}
}

func TestCapacityScalingMinCostFlow_NegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	// Graph with negative cycle - should return empty result
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, -5)
	g.AddEdgeWithReverse(3, 1, 10, -5) // Creates negative cycle

	result := CapacityScalingMinCostFlow(g, 1, 3, 10, nil)

	// Should return empty result due to negative cycle
	assert.Equal(t, 0.0, result.Flow)
	assert.False(t, result.Canceled)
}

func TestCapacityScalingMinCostFlow_EmptyGraph(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)
	// No edges

	result := CapacityScalingMinCostFlow(g, 1, 2, 10, nil)

	assert.Equal(t, 0.0, result.Flow)
}

func TestCapacityScalingMinCostFlow_ZeroCapacity(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 0, 1) // Zero capacity

	result := CapacityScalingMinCostFlow(g, 1, 2, 10, nil)

	assert.Equal(t, 0.0, result.Flow)
}

func TestShouldUseCapacityScaling(t *testing.T) {
	tests := []struct {
		name     string
		maxCap   float64
		expected bool
	}{
		{"Small capacity", 1000, false},
		{"Medium capacity", 1e5, false},
		{"Large capacity", 1e7, true},
		{"Very large capacity", 1e9, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := graph.NewResidualGraph()
			g.AddNode(1)
			g.AddNode(2)
			g.AddEdgeWithReverse(1, 2, tt.maxCap, 1)

			result := ShouldUseCapacityScaling(g)
			if result != tt.expected {
				t.Errorf("ShouldUseCapacityScaling() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRecommendMinCostAlgorithm(t *testing.T) {
	t.Run("Large capacity recommends CapacityScaling", func(t *testing.T) {
		g := graph.NewResidualGraph()
		g.AddNode(1)
		g.AddNode(2)
		g.AddEdgeWithReverse(1, 2, 1e7, 1)

		rec := RecommendMinCostAlgorithm(g)
		if rec != MinCostAlgorithmCapacityScaling {
			t.Errorf("Expected CapacityScaling, got %s", rec)
		}
	})

	t.Run("Small graph recommends SSP", func(t *testing.T) {
		g := graph.NewResidualGraph()
		g.AddNode(1)
		g.AddNode(2)
		g.AddEdgeWithReverse(1, 2, 100, 1)

		rec := RecommendMinCostAlgorithm(g)
		if rec != MinCostAlgorithmSSP {
			t.Errorf("Expected SSP, got %s", rec)
		}
	})
}

func TestComputeInitialDelta(t *testing.T) {
	tests := []struct {
		maxCap float64
		want   float64
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 4},
		{5, 4},
		{1000, 512},
		{1024, 1024},
		{1025, 1024},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := computeInitialDelta(tt.maxCap)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInitializePotentials_ContextCancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i < 500; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	potentials := initializePotentials(ctx, g, 1)

	assert.Nil(t, potentials)
}

func TestInitializePotentials_NegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, -5)
	g.AddEdgeWithReverse(3, 1, 10, -5)

	ctx := context.Background()
	potentials := initializePotentials(ctx, g, 1)

	assert.Nil(t, potentials)
}

func TestCapacityScalingMinCostFlow_WithReturnPaths(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)

	opts := DefaultSolverOptions()
	opts.ReturnPaths = true

	result := CapacityScalingMinCostFlow(g, 1, 3, 10, opts)

	assert.InDelta(t, 10.0, result.Flow, graph.Epsilon)
	assert.NotEmpty(t, result.Paths)
}

func TestCapacityScalingMinCostFlow_MaxIterations(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i < 20; i++ {
		g.AddEdgeWithReverse(i, i+1, 1e7, float64(i))
	}

	opts := DefaultSolverOptions()
	opts.MaxIterations = 10

	result := CapacityScalingMinCostFlow(g, 1, 20, 1e7, opts)

	// Should stop early due to iteration limit
	assert.LessOrEqual(t, result.Iterations, 10)
}

func TestCapacityScalingMinCostFlow_PartialFlow(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(2, 3, 5, 1)

	// Request more than available
	result := CapacityScalingMinCostFlow(g, 1, 3, 100, nil)

	assert.InDelta(t, 5.0, result.Flow, graph.Epsilon)
	assert.InDelta(t, 10.0, result.Cost, graph.Epsilon) // 5 * (1+1)
}

func TestMinCostAlgorithmType_String(t *testing.T) {
	tests := []struct {
		algo MinCostAlgorithmType
		want string
	}{
		{MinCostAlgorithmSSP, "SuccessiveShortestPath"},
		{MinCostAlgorithmCapacityScaling, "CapacityScaling"},
		{MinCostAlgorithmType(999), "Unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.algo.String())
	}
}

func TestFinishWithSSP(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)

	potentials := map[int64]float64{1: 0, 2: 1, 3: 2}
	opts := DefaultSolverOptions()

	result := finishWithSSP(context.Background(), g, 1, 3, 10, potentials, opts)

	assert.InDelta(t, 10.0, result.Flow, graph.Epsilon)
	assert.False(t, result.Canceled)
}

func TestFinishWithSSP_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1)
	}

	potentials := make(map[int64]float64)
	for i := int64(1); i <= 100; i++ {
		potentials[i] = float64(i - 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := finishWithSSP(ctx, g, 1, 100, math.MaxFloat64, potentials, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestFindMaxCapacity(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 100, 1)
	g.AddEdgeWithReverse(2, 3, 50, 1)
	g.AddEdgeWithReverse(3, 4, 200, 1)

	maxCap := findMaxCapacity(g)

	assert.Equal(t, 200.0, maxCap)
}

func TestFindMaxCapacity_EmptyGraph(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)

	maxCap := findMaxCapacity(g)

	assert.Equal(t, 0.0, maxCap)
}

// =============================================================================
// CapacityScalingMinCostFlow Coverage
// =============================================================================

func TestCapacityScaling_Basic(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := CapacityScalingMinCostFlow(g, 1, 2, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
	assert.InDelta(t, 50.0, result.Cost, 1e-9)
}

func TestCapacityScaling_LargeCapacities(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 1e8, 1)
	g.AddEdgeWithReverse(2, 3, 1e8, 1)

	result := CapacityScalingMinCostFlow(g, 1, 3, 1e8, DefaultSolverOptions())

	assert.InDelta(t, 1e8, result.Flow, 1)
}

func TestCapacityScaling_SmallCapacities(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 1, 1)
	g.AddEdgeWithReverse(2, 3, 1, 1)

	result := CapacityScalingMinCostFlow(g, 1, 3, 1, DefaultSolverOptions())

	assert.InDelta(t, 1.0, result.Flow, 1e-9)
}

func TestCapacityScaling_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 1e6, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := CapacityScalingMinCostFlowWithContext(ctx, g, 0, 100, 1e6, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestCapacityScaling_NoPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)

	result := CapacityScalingMinCostFlow(g, 1, 2, 10, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.Flow)
}

func TestCapacityScaling_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := CapacityScalingMinCostFlow(g, 1, 2, 10, nil)

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
}

func TestCapacityScaling_Diamond(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(1, 3, 5, 2)
	g.AddEdgeWithReverse(2, 4, 5, 1)
	g.AddEdgeWithReverse(3, 4, 5, 1)

	result := CapacityScalingMinCostFlow(g, 1, 4, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
}

func TestCapacityScaling_NegativeCosts(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, -5)
	g.AddEdgeWithReverse(2, 3, 10, 3)

	result := CapacityScalingMinCostFlow(g, 1, 3, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
	assert.InDelta(t, -20.0, result.Cost, 1e-9)
}

func TestCapacityScaling_MaxIterations(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i <= 10; i++ {
		g.AddEdgeWithReverse(0, i, 1, float64(i))
		g.AddEdgeWithReverse(i, 11, 1, 1)
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 2,
	}

	result := CapacityScalingMinCostFlow(g, 0, 11, 10, opts)

	assert.GreaterOrEqual(t, result.Flow, 0.0)
	assert.GreaterOrEqual(t, result.Iterations, 0)
}

func TestCapacityScaling_ReturnPaths(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(2, 3, 5, 1)

	opts := &SolverOptions{
		Epsilon:     1e-9,
		ReturnPaths: true,
	}

	result := CapacityScalingMinCostFlow(g, 1, 3, 5, opts)

	assert.NotEmpty(t, result.Paths)
}

func TestCapacityScaling_VerySmallDelta(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 0.5, 1)
	g.AddEdgeWithReverse(2, 3, 0.5, 1)

	result := CapacityScalingMinCostFlow(g, 1, 3, 0.5, DefaultSolverOptions())

	assert.InDelta(t, 0.5, result.Flow, 1e-9)
}

// =============================================================================
// Algorithm Recommendation Coverage
// =============================================================================

func TestRecommendMinCostAlgorithm_SmallCapacity(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 100, 1)

	algo := RecommendMinCostAlgorithm(g)

	assert.Equal(t, MinCostAlgorithmSSP, algo)
}

func TestRecommendMinCostAlgorithm_LargeCapacity(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 1e7, 1)

	algo := RecommendMinCostAlgorithm(g)

	assert.Equal(t, MinCostAlgorithmCapacityScaling, algo)
}
