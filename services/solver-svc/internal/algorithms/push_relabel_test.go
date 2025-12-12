package algorithms

import (
	"context"
	"testing"
	"time"

	"logistics/services/solver-svc/internal/graph"

	"github.com/stretchr/testify/assert"
)

func TestPushRelabel(t *testing.T) {
	tests := []struct {
		name         string
		setupGraph   func() *graph.ResidualGraph
		source       int64
		sink         int64
		expectedFlow float64
	}{
		{
			name: "simple_edge",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				return g
			},
			source:       1,
			sink:         2,
			expectedFlow: 10,
		},
		{
			name: "linear_chain",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(2, 3, 5, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 5,
		},
		{
			name: "diamond",
			setupGraph: func() *graph.ResidualGraph {
				// Стандартный diamond без пересечений
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 20, // Исправлено
		},
		{
			name: "complex_network",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 16, 0)
				g.AddEdgeWithReverse(0, 2, 13, 0)
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 12, 0)
				g.AddEdgeWithReverse(2, 1, 4, 0)
				g.AddEdgeWithReverse(2, 4, 14, 0)
				g.AddEdgeWithReverse(3, 2, 9, 0)
				g.AddEdgeWithReverse(3, 5, 20, 0)
				g.AddEdgeWithReverse(4, 3, 7, 0)
				g.AddEdgeWithReverse(4, 5, 4, 0)
				return g
			},
			source:       0,
			sink:         5,
			expectedFlow: 23,
		},
		{
			name: "no_path",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				g.AddNode(2)
				return g
			},
			source:       1,
			sink:         2,
			expectedFlow: 0,
		},
		{
			name: "dense_graph",
			setupGraph: func() *graph.ResidualGraph {
				// Полный граф на 4 вершинах (кроме source-sink)
				g := graph.NewResidualGraph()
				// Source edges
				g.AddEdgeWithReverse(0, 1, 10, 0)
				g.AddEdgeWithReverse(0, 2, 10, 0)
				// Middle connections
				g.AddEdgeWithReverse(1, 2, 5, 0)
				g.AddEdgeWithReverse(2, 1, 5, 0)
				// Sink edges
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 3, 10, 0)
				return g
			},
			source:       0,
			sink:         3,
			expectedFlow: 20, // Исправлено: 0->1->3 (10) + 0->2->3 (10)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setupGraph()
			opts := DefaultSolverOptions()

			result := PushRelabel(g, tt.source, tt.sink, opts)

			assert.InDelta(t, tt.expectedFlow, result.MaxFlow, 1e-9)
		})
	}
}

func TestPushRelabel_VsOtherAlgorithms(t *testing.T) {
	graphs := []struct {
		name  string
		setup func() *graph.ResidualGraph
		s, t  int64
	}{
		{
			name: "test_1",
			setup: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 5, 0)
				g.AddEdgeWithReverse(2, 4, 5, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			s: 1, t: 4,
		},
		{
			name: "test_2",
			setup: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 100, 0)
				g.AddEdgeWithReverse(0, 2, 100, 0)
				g.AddEdgeWithReverse(1, 3, 50, 0)
				g.AddEdgeWithReverse(2, 3, 50, 0)
				return g
			},
			s: 0, t: 3,
		},
	}

	for _, tc := range graphs {
		t.Run(tc.name, func(t *testing.T) {
			opts := DefaultSolverOptions()

			g1 := tc.setup()
			ekResult := EdmondsKarp(g1, tc.s, tc.t, opts)

			g2 := tc.setup()
			prResult := PushRelabel(g2, tc.s, tc.t, opts)

			assert.InDelta(t, ekResult.MaxFlow, prResult.MaxFlow, 1e-9,
				"Push-Relabel and Edmonds-Karp should give same result")
		})
	}
}

func TestPushRelabel_ExcessHandling(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 5, 0)
	g.AddEdgeWithReverse(2, 1, 5, 0) // Обратное ребро для возврата excess

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 5.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_HeightFunction(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)

	result := PushRelabel(g, 1, 4, DefaultSolverOptions())

	// Линейный граф должен дать поток 10
	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := PushRelabel(g, 1, 2, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_MaxIterationsBreak(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф который требует много итераций
	for i := int64(1); i <= 10; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 2, // Очень маленький лимит
	}

	result := PushRelabel(g, 1, 11, opts)

	// Должен остановиться после 2 итераций
	assert.LessOrEqual(t, result.Iterations, 2)
}

func TestPushRelabel_RelabelReturnsFalse(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где узел становится "изолированным" после проталкивания
	// и relabel не может найти соседа с capacity
	g.AddNode(1) // source
	g.AddNode(2) // intermediate
	g.AddNode(3) // sink

	// Только ребро 1->2, но 2 не имеет выхода к 3
	g.AddEdgeWithReverse(1, 2, 10, 0)
	// Нет ребра 2->3

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	// Поток = 0, так как нет пути к sink
	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestPushRelabel_RelabelNoValidNeighbor(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где после relabel нет валидных соседей (minH == MaxInt32)
	// 1 -> 2, 2 имеет только обратное ребро к 1
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddNode(3) // sink, не связан

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestPushRelabel_CreateBackEdgeFromNil(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdge(1, 2, 10, 1)
	g.AddEdge(2, 3, 10, 1)

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_CreateNewReverseEdge(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где backEdge не существует и нужно создать новое
	// Используем AddEdge вместо AddEdgeWithReverse чтобы не создавать reverse
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	// Добавляем только прямые рёбра
	g.AddEdge(1, 2, 10, 1)
	g.AddEdge(2, 3, 10, 1)

	// При первом push из 1 в 2, нужно будет создать reverse edge 2->1
	// Но g.Edges[2][1] не существует, сработает else ветка

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	// Должен найти поток 10
	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)

	// Проверяем, что reverse edge был создан
	reverseEdge := g.GetEdge(2, 1)
	assert.NotNil(t, reverseEdge, "Reverse edge should be created")
}

func TestPushRelabel_UpdateBackwardEdgeCreatesMap(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdge(1, 2, 10, 1)
	g.AddEdge(2, 3, 10, 1)

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

// =============================================================================
// PushRelabelHighestLabel Coverage
// =============================================================================

func TestPushRelabelHighestLabel_Basic(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := PushRelabelHighestLabel(g, 1, 2, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelHighestLabel_Diamond(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 4, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)

	result := PushRelabelHighestLabel(g, 1, 4, DefaultSolverOptions())

	assert.InDelta(t, 20.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelHighestLabel_Complex(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(0, 1, 16, 0)
	g.AddEdgeWithReverse(0, 2, 13, 0)
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 12, 0)
	g.AddEdgeWithReverse(2, 1, 4, 0)
	g.AddEdgeWithReverse(2, 4, 14, 0)
	g.AddEdgeWithReverse(3, 2, 9, 0)
	g.AddEdgeWithReverse(3, 5, 20, 0)
	g.AddEdgeWithReverse(4, 3, 7, 0)
	g.AddEdgeWithReverse(4, 5, 4, 0)

	result := PushRelabelHighestLabel(g, 0, 5, DefaultSolverOptions())

	assert.InDelta(t, 23.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelHighestLabel_EmptyGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	result := PushRelabelHighestLabel(g, 1, 2, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
	assert.Equal(t, 0, result.Iterations)
}

func TestPushRelabelHighestLabel_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := PushRelabelHighestLabelWithContext(ctx, g, 0, 100, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestPushRelabelHighestLabel_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := PushRelabelHighestLabel(g, 1, 2, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelHighestLabel_MaxIterations(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i <= 20; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 5,
	}

	result := PushRelabelHighestLabel(g, 1, 21, opts)

	assert.LessOrEqual(t, result.Iterations, 5)
}

func TestPushRelabelHighestLabel_NoExcess(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)

	result := PushRelabelHighestLabel(g, 1, 2, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

// =============================================================================
// PushRelabelLowestLabel Coverage
// =============================================================================

func TestPushRelabelLowestLabel_Basic(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := PushRelabelLowestLabel(g, 1, 2, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelLowestLabel_Diamond(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 4, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)

	result := PushRelabelLowestLabel(g, 1, 4, DefaultSolverOptions())

	assert.InDelta(t, 20.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelLowestLabel_Complex(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(0, 1, 16, 0)
	g.AddEdgeWithReverse(0, 2, 13, 0)
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 12, 0)
	g.AddEdgeWithReverse(2, 1, 4, 0)
	g.AddEdgeWithReverse(2, 4, 14, 0)
	g.AddEdgeWithReverse(3, 2, 9, 0)
	g.AddEdgeWithReverse(3, 5, 20, 0)
	g.AddEdgeWithReverse(4, 3, 7, 0)
	g.AddEdgeWithReverse(4, 5, 4, 0)

	result := PushRelabelLowestLabel(g, 0, 5, DefaultSolverOptions())

	assert.InDelta(t, 23.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelLowestLabel_EmptyGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	result := PushRelabelLowestLabel(g, 1, 2, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestPushRelabelLowestLabel_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := PushRelabelLowestLabelWithContext(ctx, g, 0, 100, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestPushRelabelLowestLabel_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := PushRelabelLowestLabel(g, 1, 2, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabelLowestLabel_MaxIterations(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(1); i <= 20; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 5,
	}

	result := PushRelabelLowestLabel(g, 1, 21, opts)

	assert.LessOrEqual(t, result.Iterations, 5)
}

// =============================================================================
// MaxHeap Coverage
// =============================================================================

func TestMaxHeap_Operations(t *testing.T) {
	h := newMaxHeap(10)

	h.push(1, 5)
	h.push(2, 10)
	h.push(3, 3)

	id, ok := h.pop()
	assert.True(t, ok)
	assert.Equal(t, int64(2), id)

	id, ok = h.pop()
	assert.True(t, ok)
	assert.Equal(t, int64(1), id)

	id, ok = h.pop()
	assert.True(t, ok)
	assert.Equal(t, int64(3), id)

	_, ok = h.pop()
	assert.False(t, ok)
}

func TestMaxHeap_Versioning(t *testing.T) {
	h := newMaxHeap(10)

	h.push(1, 5)
	h.push(1, 10) // Update same node

	id, ok := h.pop()
	assert.True(t, ok)
	assert.Equal(t, int64(1), id)

	_, ok = h.pop()
	assert.False(t, ok, "Stale entry should be skipped")
}

func TestMaxHeap_EmptyPop(t *testing.T) {
	h := newMaxHeap(10)

	_, ok := h.pop()
	assert.False(t, ok)
}

func TestMaxHeap_LenAndSwap(t *testing.T) {
	h := newMaxHeap(10)

	assert.Equal(t, 0, h.Len())

	h.push(1, 5)
	h.push(2, 10)

	assert.Equal(t, 2, h.Len())
	assert.True(t, h.Less(0, 1) || h.Less(1, 0)) // One should be greater
}

// =============================================================================
// Gap Heuristic and Global Relabel
// =============================================================================

func TestPushRelabel_GapHeuristic(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 4, 5, 0)
	g.AddEdgeWithReverse(3, 4, 5, 0)
	g.AddEdgeWithReverse(4, 5, 10, 0)

	result := PushRelabel(g, 1, 5, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_GlobalRelabel(t *testing.T) {
	g := graph.NewResidualGraph()

	n := 20
	for i := 0; i < n; i++ {
		g.AddEdgeWithReverse(int64(i), int64(i+1), 10, 0)
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 10000,
	}

	result := PushRelabel(g, 0, int64(n), opts)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_DischargeNilEdges(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	g.AddEdgeWithReverse(1, 2, 10, 0)
	// Node 2 has no outgoing edges to sink

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestPushRelabel_RelabelNoValidNeighbors(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddNode(3)

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestPushRelabel_ContextTimeout(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	result := PushRelabelWithContext(ctx, g, 0, 100, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestMinCostMaxFlow_ContextTimeout(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	result := MinCostMaxFlowWithContext(ctx, g, 0, 100, 100, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}
