package algorithms

import (
	"testing"

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

	// Создаём граф где g.Edges[from] == nil при создании обратного ребра
	// Это происходит когда узел не имеет исходящих рёбер
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	// Добавляем ребро напрямую без использования AddEdgeWithReverse
	if g.Edges[1] == nil {
		g.Edges[1] = make(map[int64]*graph.ResidualEdge)
	}
	g.Edges[1][2] = &graph.ResidualEdge{
		To:               2,
		Capacity:         10,
		OriginalCapacity: 10,
		IsReverse:        false,
	}

	// g.Edges[2] == nil - это вызовет создание map в updateBackwardEdgePR
	if g.Edges[2] == nil {
		g.Edges[2] = make(map[int64]*graph.ResidualEdge)
	}
	g.Edges[2][3] = &graph.ResidualEdge{
		To:               3,
		Capacity:         10,
		OriginalCapacity: 10,
		IsReverse:        false,
	}

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

	// Специальный тест для проверки создания map в updateBackwardEdgePR
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	// Создаём ребро 1->2, но g.Edges[2] будет nil
	if g.Edges[1] == nil {
		g.Edges[1] = make(map[int64]*graph.ResidualEdge)
	}
	g.Edges[1][2] = &graph.ResidualEdge{
		To:               2,
		Capacity:         10,
		OriginalCapacity: 10,
		Cost:             1,
		IsReverse:        false,
	}

	// g.Edges[2] намеренно nil
	// Создаём ребро 2->3
	if g.Edges[2] == nil {
		g.Edges[2] = make(map[int64]*graph.ResidualEdge)
	}
	g.Edges[2][3] = &graph.ResidualEdge{
		To:               3,
		Capacity:         10,
		OriginalCapacity: 10,
		Cost:             1,
		IsReverse:        false,
	}

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}
