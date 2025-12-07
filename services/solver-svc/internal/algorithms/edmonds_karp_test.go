package algorithms

import (
	"testing"

	"logistics/services/solver-svc/internal/graph"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdmondsKarp(t *testing.T) {
	tests := []struct {
		name         string
		setupGraph   func() *graph.ResidualGraph
		source       int64
		sink         int64
		expectedFlow float64
	}{
		{
			name: "simple_two_node",
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
			name: "linear_graph",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(2, 3, 5, 0)
				return g
			},
			source:       1,
			sink:         3,
			expectedFlow: 5,
		},
		{
			name: "parallel_paths",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 20,
		},
		{
			name: "bottleneck_in_middle",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 100, 0)
				g.AddEdgeWithReverse(2, 3, 1, 0) // Bottleneck
				g.AddEdgeWithReverse(3, 4, 100, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 1,
		},
		{
			name: "diamond_graph",
			setupGraph: func() *graph.ResidualGraph {
				// Diamond: 1 -> 2 -> 4
				//          1 -> 3 -> 4
				// Каждое ребро capacity 10, ожидаем flow 20
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 20, // Исправлено: два независимых пути по 10
		},
		{
			name: "diamond_with_cross_edge",
			setupGraph: func() *graph.ResidualGraph {
				// Diamond с перекрестным ребром 2->3
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 3, 5, 0) // Cross edge
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 15, 0) // Увеличена capacity для принятия потока из 2->3
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 20, // 1->2->4 (10) + 1->3->4 (10)
		},
		{
			name: "no_path",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddNode(1)
				g.AddNode(2)
				g.AddNode(3)
				g.AddNode(4)
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 0,
		},
		{
			name: "complex_network",
			setupGraph: func() *graph.ResidualGraph {
				// Классический пример из CLRS
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
			name: "multiple_sources_to_sink",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 4, 5, 0)
				g.AddEdgeWithReverse(2, 4, 5, 0)
				g.AddEdgeWithReverse(3, 4, 5, 0)
				return g
			},
			source:       1,
			sink:         4,
			expectedFlow: 5, // Только из узла 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setupGraph()
			opts := DefaultSolverOptions()

			result := EdmondsKarp(g, tt.source, tt.sink, opts)

			assert.InDelta(t, tt.expectedFlow, result.MaxFlow, 1e-9, "max flow mismatch")
		})
	}
}

func TestEdmondsKarp_WithOptions(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func() *graph.ResidualGraph
		opts       *SolverOptions
		checkFunc  func(t *testing.T, result *EdmondsKarpResult)
	}{
		{
			name: "max_iterations_limit",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Граф требует нескольких итераций
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			opts: &SolverOptions{
				Epsilon:       1e-9,
				MaxIterations: 1,
			},
			checkFunc: func(t *testing.T, result *EdmondsKarpResult) {
				assert.Equal(t, 1, result.Iterations)
				assert.Less(t, result.MaxFlow, 20.0) // Не успел найти весь поток
			},
		},
		{
			name: "return_paths_disabled",
			setupGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				return g
			},
			opts: &SolverOptions{
				Epsilon:     1e-9,
				ReturnPaths: false,
			},
			checkFunc: func(t *testing.T, result *EdmondsKarpResult) {
				assert.Empty(t, result.Paths)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setupGraph()
			result := EdmondsKarp(g, 1, 4, tt.opts)
			tt.checkFunc(t, result)
		})
	}
}

func TestEdmondsKarp_FlowConservation(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 4, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)

	EdmondsKarp(g, 1, 4, DefaultSolverOptions())

	// Проверяем сохранение потока для промежуточных узлов
	for _, node := range []int64{2, 3} {
		inFlow := 0.0
		outFlow := 0.0

		// Входящий поток
		for from := range g.Edges {
			if edge := g.GetEdge(from, node); edge != nil && !edge.IsReverse && edge.Flow > 0 {
				inFlow += edge.Flow
			}
		}

		// Исходящий поток
		if neighbors := g.GetNeighbors(node); neighbors != nil {
			for _, edge := range neighbors {
				if !edge.IsReverse && edge.Flow > 0 {
					outFlow += edge.Flow
				}
			}
		}

		assert.InDelta(t, inFlow, outFlow, 1e-9, "Flow conservation violated at node %d", node)
	}
}

func TestEdmondsKarp_CapacityConstraints(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 5, 0)

	EdmondsKarp(g, 1, 3, DefaultSolverOptions())

	// Проверяем, что поток не превышает capacity
	for _, edges := range g.Edges {
		for _, edge := range edges {
			if !edge.IsReverse {
				assert.LessOrEqual(t, edge.Flow, edge.OriginalCapacity+1e-9,
					"Flow exceeds capacity on edge")
			}
		}
	}
}

func TestEdmondsKarp_BipartiteMatching(t *testing.T) {
	// Двудольное сопоставление через max flow
	g := graph.NewResidualGraph()
	// Source -> Left nodes
	g.AddEdgeWithReverse(0, 1, 1, 0)
	g.AddEdgeWithReverse(0, 2, 1, 0)
	g.AddEdgeWithReverse(0, 3, 1, 0)
	// Left -> Right edges
	g.AddEdgeWithReverse(1, 4, 1, 0)
	g.AddEdgeWithReverse(1, 5, 1, 0)
	g.AddEdgeWithReverse(2, 4, 1, 0)
	g.AddEdgeWithReverse(3, 5, 1, 0)
	g.AddEdgeWithReverse(3, 6, 1, 0)
	// Right nodes -> Sink
	g.AddEdgeWithReverse(4, 7, 1, 0)
	g.AddEdgeWithReverse(5, 7, 1, 0)
	g.AddEdgeWithReverse(6, 7, 1, 0)

	result := EdmondsKarp(g, 0, 7, DefaultSolverOptions())

	// Максимальное паросочетание = 3 (все левые узлы могут быть сопоставлены)
	assert.InDelta(t, 3.0, result.MaxFlow, 1e-9)
}

func TestEdmondsKarp_ZeroCapacityEdges(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 0, 0) // Zero capacity
	g.AddEdgeWithReverse(2, 3, 10, 0)

	result := EdmondsKarp(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestEdmondsKarp_SelfLoop(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 1, 10, 0) // Self-loop
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := EdmondsKarp(g, 1, 2, DefaultSolverOptions())

	// Self-loop не должен влиять на результат
	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestEdmondsKarp_ReturnPaths(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	opts := &SolverOptions{
		Epsilon:     1e-9,
		ReturnPaths: true,
	}

	result := EdmondsKarp(g, 1, 3, opts)

	require.NotEmpty(t, result.Paths)
	assert.Equal(t, []int64{1, 2, 3}, result.Paths[0])
}

func TestEdmondsKarp_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	// Передаём nil options - должен использовать DefaultSolverOptions
	result := EdmondsKarp(g, 1, 2, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestEdmondsKarp_PathFlowBelowEpsilon(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где capacity очень маленькая (меньше epsilon)
	g.AddEdgeWithReverse(1, 2, 1e-12, 0) // Capacity < default epsilon (1e-9)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	opts := &SolverOptions{
		Epsilon: 1e-9,
	}

	result := EdmondsKarp(g, 1, 3, opts)

	// Поток должен быть 0, так как pathFlow <= epsilon
	assert.InDelta(t, 0.0, result.MaxFlow, 1e-9)
	assert.Equal(t, 0, result.Iterations) // Ни одной итерации не должно быть
}

func TestEdmondsKarp_EmptyPathAfterBFS(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где BFS найдёт путь, но capacity = 0
	// Это приводит к ситуации когда pathFlow = 0 и break
	g.AddEdgeWithReverse(1, 2, 0, 0) // Zero capacity
	g.AddEdgeWithReverse(2, 3, 10, 0)

	result := EdmondsKarp(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestEdmondsKarp_ReconstructPathEmpty(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где BFS найдёт что-то, но путь не восстановится
	// Это происходит когда граф несвязный после насыщения
	g.AddEdgeWithReverse(1, 2, 5, 0)
	g.AddEdgeWithReverse(2, 3, 5, 0)

	opts := DefaultSolverOptions()
	opts.ReturnPaths = true

	// Пускаем первый поток - он насытит путь
	result1 := EdmondsKarp(g, 1, 3, opts)
	assert.Equal(t, 5.0, result1.MaxFlow)

	// Теперь граф насыщен, следующий BFS не найдёт путь
	// Это покрывает break после "bfsResult.Found == false"
}

func TestEdmondsKarp_PathFlowZeroAfterSaturation(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф с очень маленькой capacity на одном ребре
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 1e-15, 0) // Меньше epsilon
	g.AddEdgeWithReverse(3, 4, 10, 0)

	opts := &SolverOptions{
		Epsilon: 1e-9,
	}

	result := EdmondsKarp(g, 1, 4, opts)

	// pathFlow будет <= epsilon, сработает break
	assert.Equal(t, 0.0, result.MaxFlow)
}
