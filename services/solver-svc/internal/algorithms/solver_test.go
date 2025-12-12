package algorithms

import (
	"context"
	"math"
	"testing"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolve(t *testing.T) {
	// Create graph with known answer
	// Simple: 1 -> 2 -> 3, all capacity 10
	setupGraph := func() *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		g.AddEdgeWithReverse(1, 2, 10, 1)
		g.AddEdgeWithReverse(2, 3, 10, 1)
		return g
	}

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"edmonds_karp", commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{"dinic", commonv1.Algorithm_ALGORITHM_DINIC},
		{"push_relabel", commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
		{"min_cost", commonv1.Algorithm_ALGORITHM_MIN_COST},
		{"ford_fulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
		{"unspecified_defaults_to_dinic", commonv1.Algorithm_ALGORITHM_UNSPECIFIED},
	}

	for _, tt := range algorithms {
		t.Run(tt.name, func(t *testing.T) {
			g := setupGraph()
			ctx := context.Background()
			opts := DefaultSolverOptions()

			result := Solve(ctx, g, 1, 3, tt.algo, opts)

			assert.InDelta(t, 10.0, result.MaxFlow, 1e-9, "max flow mismatch")
			assert.Equal(t, commonv1.FlowStatus_FLOW_STATUS_OPTIMAL, result.Status)
			assert.NoError(t, result.Error)
		})
	}
}

func TestSolve_ComplexGraph(t *testing.T) {
	// Diamond graph: max flow = 20
	setupGraph := func() *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		g.AddEdgeWithReverse(1, 2, 10, 1)
		g.AddEdgeWithReverse(1, 3, 10, 1)
		g.AddEdgeWithReverse(2, 4, 10, 1)
		g.AddEdgeWithReverse(3, 4, 10, 1)
		return g
	}

	algorithms := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	for _, algo := range algorithms {
		t.Run(algo.String(), func(t *testing.T) {
			g := setupGraph()
			ctx := context.Background()
			result := Solve(ctx, g, 1, 4, algo, DefaultSolverOptions())

			assert.InDelta(t, 20.0, result.MaxFlow, 1e-9)
		})
	}
}

func TestSolve_WithOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	opts := &SolverOptions{
		Epsilon:       1e-6,
		MaxIterations: 100,
		ReturnPaths:   true,
	}

	ctx := context.Background()
	result := Solve(ctx, g, 1, 3, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, opts)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
	assert.NotEmpty(t, result.Paths)
}

func TestSolve_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	ctx := context.Background()
	result := Solve(ctx, g, 1, 2, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestSolve_Validation(t *testing.T) {
	tests := []struct {
		name      string
		graph     *graph.ResidualGraph
		source    int64
		sink      int64
		wantError bool
	}{
		{
			name:      "nil_graph",
			graph:     nil,
			source:    1,
			sink:      2,
			wantError: true,
		},
		{
			name: "source_not_found",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(2)
				return g
			}(),
			source:    1,
			sink:      2,
			wantError: true,
		},
		{
			name: "sink_not_found",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				return g
			}(),
			source:    1,
			sink:      2,
			wantError: true,
		},
		{
			name: "source_equals_sink",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				return g
			}(),
			source:    1,
			sink:      1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := Solve(ctx, tt.graph, tt.source, tt.sink, commonv1.Algorithm_ALGORITHM_DINIC, nil)

			if tt.wantError {
				assert.Equal(t, commonv1.FlowStatus_FLOW_STATUS_ERROR, result.Status)
				assert.Error(t, result.Error)
			}
		})
	}
}

func TestSolve_ContextCancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i < 1000; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := Solve(ctx, g, 1, 1000, commonv1.Algorithm_ALGORITHM_DINIC, nil)

	assert.Equal(t, commonv1.FlowStatus_FLOW_STATUS_ERROR, result.Status)
	assert.Error(t, result.Error)
}

func TestSolve_Timeout(t *testing.T) {
	g := graph.NewResidualGraph()
	// Create graph that takes some time
	for i := int64(1); i < 100; i++ {
		for j := i + 1; j <= 100; j++ {
			g.AddEdgeWithReverse(i, j, float64(i), float64(j))
		}
	}

	opts := &SolverOptions{
		Epsilon: graph.Epsilon,
		Timeout: 1 * time.Microsecond, // Very short timeout
	}

	ctx := context.Background()
	result := Solve(ctx, g, 1, 100, commonv1.Algorithm_ALGORITHM_DINIC, opts)

	// May or may not timeout depending on speed
	// Just check it doesn't panic
	assert.NotNil(t, result)
}

func TestSolveMinCost_NoPrecomputation(t *testing.T) {
	// Test that min-cost algorithm works without Edmonds-Karp precomputation
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(1, 3, 10, 2)
	g.AddEdgeWithReverse(2, 4, 10, 1)
	g.AddEdgeWithReverse(3, 4, 10, 1)

	ctx := context.Background()
	result := Solve(ctx, g, 1, 4, commonv1.Algorithm_ALGORITHM_MIN_COST, nil)

	assert.Equal(t, commonv1.FlowStatus_FLOW_STATUS_OPTIMAL, result.Status)
	assert.InDelta(t, 20.0, result.MaxFlow, graph.Epsilon)
	// Should prefer cheaper paths
	assert.True(t, result.TotalCost < 60) // Max would be 20*3=60, optimal is less
}

func TestSolveMinCost_FindsMaxFlow(t *testing.T) {
	// Test that min-cost finds the same max flow as other algorithms
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(0, 1, 16, 1)
	g.AddEdgeWithReverse(0, 2, 13, 2)
	g.AddEdgeWithReverse(1, 3, 12, 1)
	g.AddEdgeWithReverse(2, 4, 14, 1)
	g.AddEdgeWithReverse(3, 5, 20, 1)
	g.AddEdgeWithReverse(4, 5, 4, 1)
	g.AddEdgeWithReverse(4, 3, 7, 1)

	ctx := context.Background()

	// Get max flow from Dinic
	g1 := g.Clone()
	dinicResult := Solve(ctx, g1, 0, 5, commonv1.Algorithm_ALGORITHM_DINIC, nil)

	// Get max flow from Min-Cost
	g2 := g.Clone()
	mcResult := Solve(ctx, g2, 0, 5, commonv1.Algorithm_ALGORITHM_MIN_COST, nil)

	assert.InDelta(t, dinicResult.MaxFlow, mcResult.MaxFlow, graph.Epsilon)
}

func TestMinCostFlowWithAlgorithm(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)

	ctx := context.Background()

	// Test SSP
	g1 := g.Clone()
	sspResult := MinCostFlowWithAlgorithm(ctx, g1, 1, 3, math.MaxFloat64, MinCostAlgorithmSSP, nil)
	assert.InDelta(t, 10.0, sspResult.Flow, graph.Epsilon)

	// Test Capacity Scaling (will use SSP internally for small graph)
	g2 := g.Clone()
	csResult := MinCostFlowWithAlgorithm(ctx, g2, 1, 3, math.MaxFloat64, MinCostAlgorithmCapacityScaling, nil)
	assert.InDelta(t, 10.0, csResult.Flow, graph.Epsilon)
}

func TestGetAlgorithmInfo(t *testing.T) {
	tests := []struct {
		algo        commonv1.Algorithm
		name        string
		supportsMin bool
		supportsNeg bool
	}{
		{commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, "Edmonds-Karp", false, false},
		{commonv1.Algorithm_ALGORITHM_DINIC, "Dinic", false, false},
		{commonv1.Algorithm_ALGORITHM_PUSH_RELABEL, "Push-Relabel (FIFO with Highest Label option)", false, false},
		{commonv1.Algorithm_ALGORITHM_MIN_COST, "Min-Cost Max-Flow (SSP + Capacity Scaling)", true, true},
		{commonv1.Algorithm_ALGORITHM_FORD_FULKERSON, "Ford-Fulkerson", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetAlgorithmInfo(tt.algo)

			require.NotNil(t, info)
			assert.Equal(t, tt.name, info.Name)
			assert.Equal(t, tt.supportsMin, info.SupportsMinCost)
			assert.Equal(t, tt.supportsNeg, info.SupportsNegativeCosts)
			assert.NotEmpty(t, info.TimeComplexity)
			assert.NotEmpty(t, info.SpaceComplexity)
		})
	}
}

func TestGetAlgorithmInfo_Unknown(t *testing.T) {
	info := GetAlgorithmInfo(commonv1.Algorithm(999))
	assert.Nil(t, info)
}

func TestGetAllAlgorithms(t *testing.T) {
	infos := GetAllAlgorithms()

	assert.Len(t, infos, 5)

	names := make(map[string]bool)
	for _, info := range infos {
		names[info.Name] = true
	}

	assert.True(t, names["Edmonds-Karp"])
	assert.True(t, names["Dinic"])
}

func TestDefaultSolverOptions(t *testing.T) {
	opts := DefaultSolverOptions()

	assert.NotNil(t, opts)
	assert.Equal(t, graph.Epsilon, opts.Epsilon)
	assert.Equal(t, 0, opts.MaxIterations)
	assert.False(t, opts.ReturnPaths)
	assert.NotNil(t, opts.Pool)
	assert.Equal(t, 30*time.Second, opts.Timeout)
}

func TestSolverOptions_Chaining(t *testing.T) {
	pool := graph.GetPool()
	opts := DefaultSolverOptions().
		WithTimeout(10 * time.Second).
		WithPool(pool).
		WithReturnPaths(true).
		WithMaxIterations(100)

	assert.Equal(t, 10*time.Second, opts.Timeout)
	assert.Equal(t, pool, opts.Pool)
	assert.True(t, opts.ReturnPaths)
	assert.Equal(t, 100, opts.MaxIterations)
}

func TestSolve_AllAlgorithmsConsistent(t *testing.T) {
	graphs := []struct {
		name         string
		setup        func() *graph.ResidualGraph
		source, sink int64
	}{
		{
			name: "simple",
			setup: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				return g
			},
			source: 1, sink: 2,
		},
		{
			name: "diamond",
			setup: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source: 1, sink: 4,
		},
		{
			name: "complex",
			setup: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 16, 0)
				g.AddEdgeWithReverse(0, 2, 13, 0)
				g.AddEdgeWithReverse(1, 3, 12, 0)
				g.AddEdgeWithReverse(2, 4, 14, 0)
				g.AddEdgeWithReverse(3, 5, 20, 0)
				g.AddEdgeWithReverse(4, 5, 4, 0)
				return g
			},
			source: 0, sink: 5,
		},
	}

	algos := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	for _, tc := range graphs {
		t.Run(tc.name, func(t *testing.T) {
			var referenceFlow float64
			ctx := context.Background()

			for i, algo := range algos {
				g := tc.setup()
				result := Solve(ctx, g, tc.source, tc.sink, algo, DefaultSolverOptions())

				if i == 0 {
					referenceFlow = result.MaxFlow
				} else {
					assert.InDelta(t, referenceFlow, result.MaxFlow, 1e-9,
						"Algorithm %s gave different result", algo.String())
				}
			}
		})
	}
}

func TestRecommendAlgorithm(t *testing.T) {
	tests := []struct {
		name             string
		nodeCount        int
		edgeCount        int
		needMinCost      bool
		hasNegativeCosts bool
		wantAlgo         commonv1.Algorithm
	}{
		{
			name:      "small_graph",
			nodeCount: 10,
			edgeCount: 20,
			wantAlgo:  commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		},
		{
			name:      "large_sparse_graph",
			nodeCount: 500,
			edgeCount: 1000,
			wantAlgo:  commonv1.Algorithm_ALGORITHM_DINIC,
		},
		{
			name:      "large_dense_graph",
			nodeCount: 200,
			edgeCount: 30000, // > 50% density
			wantAlgo:  commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
		},
		{
			name:        "need_min_cost",
			nodeCount:   100,
			edgeCount:   200,
			needMinCost: true,
			wantAlgo:    commonv1.Algorithm_ALGORITHM_MIN_COST,
		},
		{
			name:             "has_negative_costs",
			nodeCount:        100,
			edgeCount:        200,
			hasNegativeCosts: true,
			wantAlgo:         commonv1.Algorithm_ALGORITHM_MIN_COST,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algo := RecommendAlgorithm(tt.nodeCount, tt.edgeCount, tt.needMinCost, tt.hasNegativeCosts)
			assert.Equal(t, tt.wantAlgo, algo)
		})
	}
}

func TestValidateGraph(t *testing.T) {
	tests := []struct {
		name      string
		graph     *graph.ResidualGraph
		source    int64
		sink      int64
		wantError error
	}{
		{
			name:      "nil_graph",
			graph:     nil,
			source:    1,
			sink:      2,
			wantError: ErrNilGraph,
		},
		{
			name: "source_not_in_graph",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(2)
				g.AddNode(3)
				return g
			}(),
			source:    1,
			sink:      2,
			wantError: ErrSourceNotFound,
		},
		{
			name: "sink_not_in_graph",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				return g
			}(),
			source:    1,
			sink:      2,
			wantError: ErrSinkNotFound,
		},
		{
			name: "source_equals_sink",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				return g
			}(),
			source:    1,
			sink:      1,
			wantError: ErrSourceEqualSink,
		},
		{
			name: "valid_graph",
			graph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				return g
			}(),
			source:    1,
			sink:      2,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGraph(tt.graph, tt.source, tt.sink)

			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolverPool_BatchSolve(t *testing.T) {
	pool := NewSolverPool(4)

	createGraph := func() *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		g.AddEdgeWithReverse(1, 2, 10, 1)
		g.AddEdgeWithReverse(2, 3, 10, 1)
		return g
	}

	tasks := []BatchTask{
		{TaskID: "task1", Graph: createGraph(), Source: 1, Sink: 3, Algorithm: commonv1.Algorithm_ALGORITHM_DINIC},
		{TaskID: "task2", Graph: createGraph(), Source: 1, Sink: 3, Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{TaskID: "task3", Graph: createGraph(), Source: 1, Sink: 3, Algorithm: commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
	}

	ctx := context.Background()
	results := pool.BatchSolve(ctx, tasks)

	assert.Len(t, results, 3)
	for i, result := range results {
		assert.Equal(t, tasks[i].TaskID, result.TaskID)
		assert.InDelta(t, 10.0, result.Result.MaxFlow, graph.Epsilon)
	}
}

func TestSolverPool_AcquireRelease(t *testing.T) {
	pool := NewSolverPool(2)
	ctx := context.Background()

	// Should succeed
	err := pool.Acquire(ctx)
	assert.NoError(t, err)

	err = pool.Acquire(ctx)
	assert.NoError(t, err)

	// Pool is full, should timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	err = pool.Acquire(ctxTimeout)
	assert.Error(t, err)

	// Release one
	pool.Release()

	// Should succeed now
	err = pool.Acquire(ctx)
	assert.NoError(t, err)

	// Clean up
	pool.Release()
	pool.Release()
}
