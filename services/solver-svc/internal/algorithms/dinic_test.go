package algorithms

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"logistics/services/solver-svc/internal/graph"
)

func TestDinic(t *testing.T) {
	tests := []struct {
		name        string
		buildGraph  func() *graph.ResidualGraph
		source      int64
		sink        int64
		wantMaxFlow float64
	}{
		{
			name: "simple_two_node",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				return g
			},
			source:      1,
			sink:        2,
			wantMaxFlow: 10,
		},
		{
			name: "linear_chain",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 5, 0)
				g.AddEdgeWithReverse(2, 3, 5, 0)
				g.AddEdgeWithReverse(3, 4, 5, 0)
				return g
			},
			source:      1,
			sink:        4,
			wantMaxFlow: 5,
		},
		{
			name: "complex_network_cormen",
			buildGraph: func() *graph.ResidualGraph {
				// Пример из CLRS (Cormen)
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
			source:      0,
			sink:        5,
			wantMaxFlow: 23,
		},
		{
			name: "unit_capacity_graph",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Граф с единичными пропускными способностями
				g.AddEdgeWithReverse(1, 2, 1, 0)
				g.AddEdgeWithReverse(1, 3, 1, 0)
				g.AddEdgeWithReverse(2, 3, 1, 0)
				g.AddEdgeWithReverse(2, 4, 1, 0)
				g.AddEdgeWithReverse(3, 4, 1, 0)
				return g
			},
			source:      1,
			sink:        4,
			wantMaxFlow: 2,
		},
		{
			name: "multiple_augmenting_paths",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// 10 параллельных путей
				for i := int64(1); i <= 10; i++ {
					g.AddEdgeWithReverse(0, i, 1, 0)
					g.AddEdgeWithReverse(i, 11, 1, 0)
				}
				return g
			},
			source:      0,
			sink:        11,
			wantMaxFlow: 10,
		},
		{
			name: "layered_graph",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Слоистый граф для тестирования level graph
				// Layer 0: source (0)
				// Layer 1: 1, 2
				// Layer 2: 3, 4
				// Layer 3: sink (5)
				g.AddEdgeWithReverse(0, 1, 5, 0)
				g.AddEdgeWithReverse(0, 2, 5, 0)
				g.AddEdgeWithReverse(1, 3, 3, 0)
				g.AddEdgeWithReverse(1, 4, 3, 0)
				g.AddEdgeWithReverse(2, 3, 3, 0)
				g.AddEdgeWithReverse(2, 4, 3, 0)
				g.AddEdgeWithReverse(3, 5, 5, 0)
				g.AddEdgeWithReverse(4, 5, 5, 0)
				return g
			},
			source:      0,
			sink:        5,
			wantMaxFlow: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()

			result := Dinic(g, tt.source, tt.sink, DefaultSolverOptions())

			assert.InDelta(t, tt.wantMaxFlow, result.MaxFlow, graph.Epsilon)
		})
	}
}

func TestDinic_VsEdmondsKarp(t *testing.T) {
	// Сравниваем результаты Dinic и Edmonds-Karp
	testCases := []struct {
		name       string
		buildGraph func() *graph.ResidualGraph
		source     int64
		sink       int64
	}{
		{
			name: "random_graph_1",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(1, 3, 5, 0)
				g.AddEdgeWithReverse(2, 3, 15, 0)
				g.AddEdgeWithReverse(2, 4, 10, 0)
				g.AddEdgeWithReverse(3, 4, 10, 0)
				return g
			},
			source: 1,
			sink:   4,
		},
		{
			name: "random_graph_2",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 7, 0)
				g.AddEdgeWithReverse(0, 2, 4, 0)
				g.AddEdgeWithReverse(1, 2, 3, 0)
				g.AddEdgeWithReverse(1, 3, 5, 0)
				g.AddEdgeWithReverse(2, 3, 6, 0)
				g.AddEdgeWithReverse(2, 4, 2, 0)
				g.AddEdgeWithReverse(3, 4, 8, 0)
				return g
			},
			source: 0,
			sink:   4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g1 := tc.buildGraph()
			g2 := tc.buildGraph()

			resultDinic := Dinic(g1, tc.source, tc.sink, DefaultSolverOptions())
			resultEK := EdmondsKarp(g2, tc.source, tc.sink, DefaultSolverOptions())

			assert.InDelta(t, resultEK.MaxFlow, resultDinic.MaxFlow, graph.Epsilon,
				"Dinic and Edmonds-Karp should produce the same max flow")
		})
	}
}

func TestDinic_LevelGraph(t *testing.T) {
	// Тестируем корректность построения графа уровней
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 4, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)
	g.AddEdgeWithReverse(4, 5, 10, 0)

	levels := graph.BFSLevel(g, 1)

	assert.Equal(t, 0, levels[1])
	assert.Equal(t, 1, levels[2])
	assert.Equal(t, 1, levels[3])
	assert.Equal(t, 2, levels[4])
	assert.Equal(t, 3, levels[5])
}

func TestDinic_BlockingFlow(t *testing.T) {
	// Проверяем, что blocking flow находит все увеличивающие пути на одном уровне
	g := graph.NewResidualGraph()
	// Граф с двумя блокирующими потоками
	g.AddEdgeWithReverse(1, 2, 2, 0)
	g.AddEdgeWithReverse(1, 3, 2, 0)
	g.AddEdgeWithReverse(2, 4, 2, 0)
	g.AddEdgeWithReverse(3, 4, 2, 0)

	opts := DefaultSolverOptions()
	opts.ReturnPaths = true

	result := Dinic(g, 1, 4, opts)

	assert.InDelta(t, 4.0, result.MaxFlow, graph.Epsilon)
	// Должны быть найдены оба пути за одну фазу
}

func TestDinic_Iterations(t *testing.T) {
	// Проверяем количество итераций (фаз)
	tests := []struct {
		name          string
		buildGraph    func() *graph.ResidualGraph
		source        int64
		sink          int64
		maxIterations int
	}{
		{
			name: "single_path_single_iteration",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				return g
			},
			source:        1,
			sink:          2,
			maxIterations: 1,
		},
		{
			name: "parallel_paths_single_iteration",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 5, 0)
				g.AddEdgeWithReverse(1, 3, 5, 0)
				g.AddEdgeWithReverse(2, 4, 5, 0)
				g.AddEdgeWithReverse(3, 4, 5, 0)
				return g
			},
			source:        1,
			sink:          4,
			maxIterations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()
			result := Dinic(g, tt.source, tt.sink, DefaultSolverOptions())

			assert.LessOrEqual(t, result.Iterations, tt.maxIterations)
		})
	}
}

func BenchmarkDinic(b *testing.B) {
	// Создаём большой граф для бенчмарка
	buildLargeGraph := func(n int) *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		for i := 1; i < n; i++ {
			g.AddEdgeWithReverse(int64(i), int64(i+1), float64(i%10+1), 0)
			if i > 1 {
				g.AddEdgeWithReverse(int64(i-1), int64(i+1), float64(i%5+1), 0)
			}
		}
		return g
	}

	sizes := []int{100, 500, 1000}

	for _, size := range sizes {
		b.Run(
			"size_"+string(rune(size)),
			func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					g := buildLargeGraph(size)
					Dinic(g, 1, int64(size), DefaultSolverOptions())
				}
			},
		)
	}
}

func TestDinic_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := Dinic(g, 1, 2, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestDinic_NodeWithoutNeighbors(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где промежуточный узел не имеет исходящих рёбер в нужном направлении
	// 1 -> 2, но 2 не имеет рёбер к sink (3)
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)
	g.AddEdgeWithReverse(1, 2, 10, 0)
	// Нет ребра 2 -> 3

	result := Dinic(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestDinic_DeadEndInDFS(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где DFS попадает в тупик
	// 1 -> 2 -> 3 (но 3 не sink и не имеет выхода к sink)
	// 1 -> 4 (sink)
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)
	// 3 - тупик, нет ребра к sink (4)
	g.AddEdgeWithReverse(1, 4, 5, 0) // Путь к sink

	result := Dinic(g, 1, 4, DefaultSolverOptions())

	// Должен найти только путь 1->4
	assert.InDelta(t, 5.0, result.MaxFlow, 1e-9)
}

func TestDinic_NeighborsNil(t *testing.T) {
	g := graph.NewResidualGraph()

	// Минимальный граф с узлом без исходящих рёбер
	g.AddNode(1)
	g.AddNode(2)
	// Не добавляем рёбра - GetNeighbors вернёт nil

	result := Dinic(g, 1, 2, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestDinic_DFSNeighborsNil(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где в процессе DFS достигается узел без исходящих рёбер
	// 1 -> 2 (но узел 2 не имеет исходящих рёбер)
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3) // sink

	// Добавляем только ребро 1->2, но НЕ добавляем 2->3
	// Это значит GetNeighbors(2) вернёт nil (или пустую map без нужного ребра)
	if g.Edges[1] == nil {
		g.Edges[1] = make(map[int64]*graph.ResidualEdge)
	}
	g.Edges[1][2] = &graph.ResidualEdge{
		To:               2,
		Capacity:         10,
		OriginalCapacity: 10,
		IsReverse:        false,
	}
	// Узел 2 не имеет исходящих рёбер - g.Edges[2] == nil

	result := Dinic(g, 1, 3, DefaultSolverOptions())

	// Не найдёт путь, так как 2 не имеет соседей
	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestDinic_DFSDeadEndReturnsZero(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где DFS заходит в тупик
	// 1 -> 2 -> 3 (но 3 не sink и нет пути к sink)
	// sink = 4
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)
	// 3 не имеет пути к 4
	g.AddNode(4)

	result := Dinic(g, 1, 4, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}
