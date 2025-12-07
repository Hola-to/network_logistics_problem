package algorithms

import (
	"testing"

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
				// Путь 1->2->4: стоимость 5
				g.AddEdge(1, 2, 10, 2.0)
				g.AddEdge(2, 4, 10, 3.0)
				// Путь 1->3->4: стоимость 4
				g.AddEdge(1, 3, 10, 1.0)
				g.AddEdge(3, 4, 10, 3.0)
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 2,
				3: 1,
				4: 4, // Выбран кратчайший путь через 3
			},
			wantNegativeCycle: false,
		},
		{
			name: "disconnected_nodes",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddNode(3) // Изолированный узел
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
				4: 3, // Через 2
			},
			wantNegativeCycle: false,
		},
		{
			name: "graph_with_negative_edges_no_cycle",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 5.0)
				g.AddEdge(1, 3, 10, 2.0)
				g.AddEdge(3, 2, 10, -2.0) // Отрицательное ребро
				return g
			},
			source: 1,
			wantDistances: map[int64]float64{
				1: 0,
				2: 0, // Через 3 с отрицательным ребром
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
				// Цикл: 1 -> 2 -> 3 -> 1 с общей стоимостью -1
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 1.0)
				g.AddEdge(3, 1, 10, -3.0) // Создаёт отрицательный цикл
				return g
			},
			source:    1,
			wantCycle: true,
		},
		{
			name: "negative_cycle_not_from_source",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				// Цикл не достижим из источника, но всё равно должен быть обнаружен
				// если проверяем все рёбра
				g.AddEdge(3, 4, 10, 1.0)
				g.AddEdge(4, 5, 10, 1.0)
				g.AddEdge(5, 3, 10, -5.0)
				return g
			},
			source:    1,
			wantCycle: false, // Цикл недостижим из source
		},
		{
			name: "reachable_negative_cycle",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdge(1, 2, 10, 1.0)
				g.AddEdge(2, 3, 10, 1.0)
				g.AddEdge(3, 4, 10, 1.0)
				g.AddEdge(4, 2, 10, -5.0) // Цикл 2->3->4->2
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
				// Длинный путь: 1->2->3->4, cost=6
				g.AddEdge(1, 2, 10, 2.0)
				g.AddEdge(2, 3, 10, 2.0)
				g.AddEdge(3, 4, 10, 2.0)
				// Короткий путь: 1->4, cost=5
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

func TestBellmanFord_LargeGraph(t *testing.T) {
	// Тест на большом графе для проверки производительности
	g := graph.NewResidualGraph()
	n := 1000

	// Создаём линейный граф
	for i := 1; i < n; i++ {
		g.AddEdge(int64(i), int64(i+1), 10, 1.0)
	}

	result := BellmanFord(g, 1)

	assert.False(t, result.HasNegativeCycle)
	assert.InDelta(t, float64(n-1), result.Distances[int64(n)], graph.Epsilon)
}

func TestBellmanFord_ComplexNegativeCycle(t *testing.T) {
	// Сложный граф с несколькими циклами, один из которых отрицательный
	g := graph.NewResidualGraph()

	// Основной путь
	g.AddEdge(1, 2, 10, 1.0)
	g.AddEdge(2, 3, 10, 1.0)
	g.AddEdge(3, 4, 10, 1.0)
	g.AddEdge(4, 5, 10, 1.0)

	// Положительный цикл
	g.AddEdge(2, 6, 10, 1.0)
	g.AddEdge(6, 2, 10, 1.0) // Цикл 2->6->2 = +2

	// Отрицательный цикл
	g.AddEdge(3, 7, 10, 1.0)
	g.AddEdge(7, 8, 10, 1.0)
	g.AddEdge(8, 3, 10, -5.0) // Цикл 3->7->8->3 = -3

	result := BellmanFord(g, 1)
	assert.True(t, result.HasNegativeCycle)
}

func TestBellmanFordWithPotentials_NegativeCycleDetection(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф с отрицательным циклом, который обнаруживается через reduced costs
	// 1 -> 2 (cost 1)
	// 2 -> 3 (cost 1)
	// 3 -> 2 (cost -5) - создаёт negative cycle 2-3-2
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)
	g.AddEdgeWithReverse(3, 2, 10, -5) // Negative cost создаёт cycle

	// Инициализируем потенциалы
	potentials := map[int64]float64{
		1: 0,
		2: 0,
		3: 0,
	}

	result := BellmanFordWithPotentials(g, 1, potentials)

	// Должен обнаружить negative cycle
	assert.True(t, result.HasNegativeCycle)
}

func TestFindShortestPath_NegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф с negative cycle, достижимым из source
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 10, -5)
	g.AddEdgeWithReverse(3, 2, 10, -5) // Negative cycle 2-3-2
	g.AddEdgeWithReverse(3, 4, 10, 1)

	path, cost, found := FindShortestPath(g, 1, 4)

	// Не должен найти путь из-за negative cycle
	assert.False(t, found)
	assert.Nil(t, path)
	assert.Equal(t, 0.0, cost)
}

func TestBellmanFordWithPotentials_ReducedCostNegativeCycle(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где negative cycle обнаруживается только через reduced costs
	// С потенциалами, которые делают reduced cost отрицательным
	g.AddEdgeWithReverse(1, 2, 10, 2)
	g.AddEdgeWithReverse(2, 3, 10, 2)
	g.AddEdgeWithReverse(3, 1, 10, -10) // Cycle с очень отрицательной стоимостью
	g.AddEdgeWithReverse(3, 4, 10, 1)

	// Потенциалы которые уже учитывают часть стоимости
	potentials := map[int64]float64{
		1: 0,
		2: 2,
		3: 4,
		4: 5,
	}

	result := BellmanFordWithPotentials(g, 1, potentials)

	// Reduced cost для 3->1: -10 + 4 - 0 = -6 (negative)
	// Это создаёт negative cycle в терминах reduced costs
	assert.True(t, result.HasNegativeCycle)
}
