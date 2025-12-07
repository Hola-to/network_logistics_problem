package algorithms

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolve(t *testing.T) {
	// Создаем граф, для которого мы ТОЧНО знаем ответ
	// Simple: 1 -> 2 -> 3, все capacity 10
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
		{"unspecified_defaults_to_edmonds_karp", commonv1.Algorithm_ALGORITHM_UNSPECIFIED},
	}

	for _, tt := range algorithms {
		t.Run(tt.name, func(t *testing.T) {
			g := setupGraph()
			opts := DefaultSolverOptions()

			result := Solve(g, 1, 3, tt.algo, opts)

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
			result := Solve(g, 1, 4, algo, DefaultSolverOptions())

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

	result := Solve(g, 1, 3, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, opts)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
	assert.NotEmpty(t, result.Paths)
}

func TestSolve_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := Solve(g, 1, 2, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
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
		{commonv1.Algorithm_ALGORITHM_PUSH_RELABEL, "Push-Relabel", false, false},
		{commonv1.Algorithm_ALGORITHM_MIN_COST, "Min-Cost Max-Flow", true, true},
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

func TestGetAllAlgorithms(t *testing.T) {
	infos := GetAllAlgorithms()

	assert.Len(t, infos, 5)

	names := make(map[string]bool)
	for _, info := range infos {
		names[info.Name] = true
	}

	assert.True(t, names["Edmonds-Karp"])
	assert.True(t, names["Dinic"])
	assert.True(t, names["Push-Relabel"])
	assert.True(t, names["Min-Cost Max-Flow"])
	assert.True(t, names["Ford-Fulkerson"])
}

func TestDefaultSolverOptions(t *testing.T) {
	opts := DefaultSolverOptions()

	assert.NotNil(t, opts)
	assert.Equal(t, graph.Epsilon, opts.Epsilon)
	assert.Equal(t, 0, opts.MaxIterations)
	assert.False(t, opts.ReturnPaths)
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

			for i, algo := range algos {
				g := tc.setup()
				result := Solve(g, tc.source, tc.sink, algo, DefaultSolverOptions())

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
