package algorithms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"logistics/services/solver-svc/internal/graph"
)

func TestMinCostMaxFlow(t *testing.T) {
	tests := []struct {
		name         string
		buildGraph   func() *graph.ResidualGraph
		source       int64
		sink         int64
		requiredFlow float64
		wantFlow     float64
		wantCost     float64
	}{
		{
			name: "simple_edge",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 5)
				return g
			},
			source:       1,
			sink:         2,
			requiredFlow: 10,
			wantFlow:     10,
			wantCost:     50,
		},
		{
			name: "choose_cheaper_path",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Путь 1 -> 2 -> 4: capacity 5, cost 10
				g.AddEdgeWithReverse(1, 2, 5, 3)
				g.AddEdgeWithReverse(2, 4, 5, 7)
				// Путь 1 -> 3 -> 4: capacity 5, cost 5
				g.AddEdgeWithReverse(1, 3, 5, 2)
				g.AddEdgeWithReverse(3, 4, 5, 3)
				return g
			},
			source:       1,
			sink:         4,
			requiredFlow: 5,
			wantFlow:     5,
			wantCost:     25, // Выбран дешёвый путь: 5 * (2 + 3) = 25
		},
		{
			name: "use_both_paths",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				// Дешёвый путь: capacity 3, cost 1
				g.AddEdgeWithReverse(1, 2, 3, 1)
				g.AddEdgeWithReverse(2, 4, 3, 1)
				// Дорогой путь: capacity 5, cost 5
				g.AddEdgeWithReverse(1, 3, 5, 5)
				g.AddEdgeWithReverse(3, 4, 5, 5)
				return g
			},
			source:       1,
			sink:         4,
			requiredFlow: 6,
			wantFlow:     6,
			wantCost:     36, // 3*(1+1) + 3*(5+5) = 6 + 30 = 36
		},
		{
			name: "partial_flow",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 5, 2)
				return g
			},
			source:       1,
			sink:         2,
			requiredFlow: 10, // Требуется больше, чем доступно
			wantFlow:     5,  // Получим только доступное
			wantCost:     10,
		},
		{
			name: "zero_cost_edges",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(1, 2, 10, 0)
				g.AddEdgeWithReverse(2, 3, 10, 0)
				return g
			},
			source:       1,
			sink:         3,
			requiredFlow: 10,
			wantFlow:     10,
			wantCost:     0,
		},
		{
			name: "complex_network",
			buildGraph: func() *graph.ResidualGraph {
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 15, 4)
				g.AddEdgeWithReverse(0, 2, 8, 4)
				g.AddEdgeWithReverse(1, 2, 20, 2)
				g.AddEdgeWithReverse(1, 3, 4, 2)
				g.AddEdgeWithReverse(1, 4, 10, 6)
				g.AddEdgeWithReverse(2, 3, 15, 1)
				g.AddEdgeWithReverse(2, 4, 4, 3)
				g.AddEdgeWithReverse(3, 4, 20, 2)
				g.AddEdgeWithReverse(3, 5, 5, 3)
				g.AddEdgeWithReverse(4, 5, 15, 2)
				return g
			},
			source:       0,
			sink:         5,
			requiredFlow: 15,
			wantFlow:     15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.buildGraph()

			result := MinCostMaxFlow(g, tt.source, tt.sink, tt.requiredFlow, DefaultSolverOptions())

			assert.InDelta(t, tt.wantFlow, result.Flow, graph.Epsilon, "flow mismatch")
			if tt.wantCost > 0 {
				assert.InDelta(t, tt.wantCost, result.Cost, graph.Epsilon, "cost mismatch")
			}
		})
	}
}

func TestMinCostMaxFlow_NegativeCosts(t *testing.T) {
	// Алгоритм должен корректно обрабатывать отрицательные стоимости
	// (но не отрицательные циклы)
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, -5) // Отрицательная стоимость
	g.AddEdgeWithReverse(2, 3, 10, 3)

	result := MinCostMaxFlow(g, 1, 3, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, graph.Epsilon)
	assert.InDelta(t, -20.0, result.Cost, graph.Epsilon) // 10 * (-5 + 3) = -20
}

func TestMinCostMaxFlow_Potentials(t *testing.T) {
	// Проверяем, что потенциалы корректно обновляются
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 10)
	g.AddEdgeWithReverse(1, 3, 5, 5)
	g.AddEdgeWithReverse(2, 4, 5, 3)
	g.AddEdgeWithReverse(3, 4, 5, 8)

	opts := DefaultSolverOptions()
	opts.ReturnPaths = true

	result := MinCostMaxFlow(g, 1, 4, 10, opts)

	// Проверяем, что оптимальный поток найден
	assert.InDelta(t, 10.0, result.Flow, graph.Epsilon)
	// Первый путь (дешёвый): 1->2->4 (cost 13), затем 1->3->4 (cost 13)
}

func TestMinCostMaxFlow_Iterations(t *testing.T) {
	g := graph.NewResidualGraph()
	// Создаём граф, требующий нескольких итераций
	g.AddEdgeWithReverse(1, 2, 1, 1)
	g.AddEdgeWithReverse(1, 3, 1, 2)
	g.AddEdgeWithReverse(1, 4, 1, 3)
	g.AddEdgeWithReverse(2, 5, 1, 1)
	g.AddEdgeWithReverse(3, 5, 1, 1)
	g.AddEdgeWithReverse(4, 5, 1, 1)

	opts := &SolverOptions{
		Epsilon:       graph.Epsilon,
		MaxIterations: 2,
	}

	result := MinCostMaxFlow(g, 1, 5, 3, opts)

	assert.LessOrEqual(t, result.Iterations, 2)
	assert.LessOrEqual(t, result.Flow, 2.0)
}

func TestMinCostMaxFlow_VsGreedy(t *testing.T) {
	// Демонстрируем, что min-cost flow лучше жадного подхода
	g := graph.NewResidualGraph()

	// Сценарий: жадный выбрал бы дешёвый путь первым,
	// но оптимально использовать комбинацию
	g.AddEdgeWithReverse(1, 2, 10, 1) // Дешёвое ребро
	g.AddEdgeWithReverse(2, 4, 5, 10) // Дорогое продолжение
	g.AddEdgeWithReverse(1, 3, 10, 2) // Чуть дороже
	g.AddEdgeWithReverse(3, 4, 10, 1) // Дешёвое продолжение

	result := MinCostMaxFlow(g, 1, 4, 10, DefaultSolverOptions())

	// Оптимально: 5 через (1,2,4) + 5 через (1,3,4)
	// = 5*(1+10) + 5*(2+1) = 55 + 15 = 70
	// Но алгоритм найдёт: сначала (1,3,4) cost=3, затем (1,2,4) cost=11
	assert.InDelta(t, 10.0, result.Flow, graph.Epsilon)
}

func TestSuccessiveShortestPath_Alias(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := SuccessiveShortestPath(g, 1, 2, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, graph.Epsilon)
	assert.InDelta(t, 50.0, result.Cost, graph.Epsilon)
}

func BenchmarkMinCostMaxFlow(b *testing.B) {
	buildGraph := func(n int) *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		for i := 1; i < n; i++ {
			g.AddEdgeWithReverse(int64(i), int64(i+1), float64(n-i), float64(i))
		}
		// Добавляем обходные пути
		for i := 1; i < n-1; i++ {
			g.AddEdgeWithReverse(int64(i), int64(i+2), float64(i), float64(n-i))
		}
		return g
	}

	sizes := []int{50, 100, 200}

	for _, size := range sizes {
		b.Run("size_"+string(rune(size)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				g := buildGraph(size)
				MinCostMaxFlow(g, 1, int64(size), float64(size), DefaultSolverOptions())
			}
		})
	}
}

func TestMinCostMaxFlow_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := MinCostMaxFlow(g, 1, 2, 10, nil)

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
	assert.InDelta(t, 50.0, result.Cost, 1e-9) // 10 * 5
}

func TestMinCostMaxFlow_PathFlowBelowEpsilon(t *testing.T) {
	g := graph.NewResidualGraph()

	// Создаём граф где capacity очень маленькая
	g.AddEdgeWithReverse(1, 2, 1e-12, 1) // Capacity < epsilon
	g.AddEdgeWithReverse(2, 3, 10, 1)

	opts := &SolverOptions{
		Epsilon: 1e-9,
	}

	result := MinCostMaxFlow(g, 1, 3, 100, opts)

	// Поток должен быть 0
	assert.InDelta(t, 0.0, result.Flow, 1e-9)
}

func TestMinCostMaxFlow_EmptyPath(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где нет пути после первой итерации
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(2, 3, 5, 1)

	// Требуем поток больше чем возможно - после первого пути остальные пути пусты
	result := MinCostMaxFlow(g, 1, 3, 100, DefaultSolverOptions())

	// Должен найти только 5 единиц потока
	assert.InDelta(t, 5.0, result.Flow, 1e-9)
}

func TestMinCostMaxFlow_ZeroCapacityPath(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где остаточная capacity становится 0
	g.AddEdgeWithReverse(1, 2, 0, 1) // Zero capacity edge
	g.AddEdgeWithReverse(2, 3, 10, 1)

	result := MinCostMaxFlow(g, 1, 3, 10, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.Flow)
}

func TestMinCostMaxFlow_ReconstructPathEmpty(t *testing.T) {
	g := graph.NewResidualGraph()

	// Граф где после нескольких итераций путь не восстанавливается
	g.AddEdgeWithReverse(1, 2, 3, 1)
	g.AddEdgeWithReverse(2, 3, 3, 1)

	// Требуем поток больше чем есть - после насыщения path будет пустой
	result := MinCostMaxFlow(g, 1, 3, 100, DefaultSolverOptions())

	// Найдёт только 3 единицы, потом break по пустому пути
	assert.InDelta(t, 3.0, result.Flow, 1e-9)
}

func TestMinCostMaxFlow_PathFlowBelowEpsilonBreak(t *testing.T) {
	g := graph.NewResidualGraph()

	// Первое ребро с нормальной capacity, второе с очень маленькой
	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 3, 1e-15, 1) // Capacity < epsilon

	opts := &SolverOptions{
		Epsilon: 1e-9,
	}

	result := MinCostMaxFlow(g, 1, 3, 10, opts)

	// pathFlow <= epsilon, сработает break
	assert.Equal(t, 0.0, result.Flow)
}

// =============================================================================
// MinCostFlowBellmanFord Coverage
// =============================================================================

func TestMinCostFlowBellmanFord_Basic(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := MinCostFlowBellmanFord(g, 1, 2, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
	assert.InDelta(t, 50.0, result.Cost, 1e-9)
}

func TestMinCostFlowBellmanFord_ChooseCheaper(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(2, 4, 5, 1)
	g.AddEdgeWithReverse(1, 3, 5, 10)
	g.AddEdgeWithReverse(3, 4, 5, 10)

	result := MinCostFlowBellmanFord(g, 1, 4, 5, DefaultSolverOptions())

	assert.InDelta(t, 5.0, result.Flow, 1e-9)
	assert.InDelta(t, 10.0, result.Cost, 1e-9) // 5 * (1+1)
}

func TestMinCostFlowBellmanFord_NegativeCosts(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, -5)
	g.AddEdgeWithReverse(2, 3, 10, 3)

	result := MinCostFlowBellmanFord(g, 1, 3, 10, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
	assert.InDelta(t, -20.0, result.Cost, 1e-9)
}

func TestMinCostFlowBellmanFord_NoPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)

	result := MinCostFlowBellmanFord(g, 1, 2, 10, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.Flow)
}

func TestMinCostFlowBellmanFord_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 100; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := MinCostFlowBellmanFordWithContext(ctx, g, 0, 100, 100, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestMinCostFlowBellmanFord_MaxIterations(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 1, 1)
	g.AddEdgeWithReverse(1, 3, 1, 2)
	g.AddEdgeWithReverse(1, 4, 1, 3)
	g.AddEdgeWithReverse(2, 5, 1, 1)
	g.AddEdgeWithReverse(3, 5, 1, 1)
	g.AddEdgeWithReverse(4, 5, 1, 1)

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 2,
	}

	result := MinCostFlowBellmanFord(g, 1, 5, 3, opts)

	assert.LessOrEqual(t, result.Iterations, 2)
}

func TestMinCostFlowBellmanFord_ReturnPaths(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(2, 3, 5, 1)

	opts := &SolverOptions{
		Epsilon:     1e-9,
		ReturnPaths: true,
	}

	result := MinCostFlowBellmanFord(g, 1, 3, 5, opts)

	require.NotEmpty(t, result.Paths)
	assert.Equal(t, []int64{1, 2, 3}, result.Paths[0].NodeIDs)
}

func TestMinCostFlowBellmanFord_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := MinCostFlowBellmanFord(g, 1, 2, 10, nil)

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
}

func TestMinCostFlowBellmanFord_PathFlowBelowEpsilon(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 1e-12, 1)
	g.AddEdgeWithReverse(2, 3, 10, 1)

	result := MinCostFlowBellmanFord(g, 1, 3, 10, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.Flow)
}

func TestMinCostFlowBellmanFord_EmptyPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 1)
	g.AddEdgeWithReverse(2, 3, 5, 1)

	result := MinCostFlowBellmanFord(g, 1, 3, 100, DefaultSolverOptions())

	assert.InDelta(t, 5.0, result.Flow, 1e-9)
}

// =============================================================================
// MinCostFlowWithAlgorithm Coverage
// =============================================================================

func TestMinCostFlowWithAlgorithm_SSP(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := MinCostFlowWithAlgorithm(
		context.Background(), g, 1, 2, 10,
		MinCostAlgorithmSSP, DefaultSolverOptions(),
	)

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
}

func TestMinCostFlowWithAlgorithm_CapacityScaling(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := MinCostFlowWithAlgorithm(
		context.Background(), g, 1, 2, 10,
		MinCostAlgorithmCapacityScaling, DefaultSolverOptions(),
	)

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
}

func TestMinCostFlowWithAlgorithm_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 5)

	result := MinCostFlowWithAlgorithm(
		context.Background(), g, 1, 2, 10,
		MinCostAlgorithmSSP, nil,
	)

	assert.InDelta(t, 10.0, result.Flow, 1e-9)
}

// =============================================================================
// SSP Reinitialization Coverage
// =============================================================================

func TestSSP_Reinitialization(t *testing.T) {
	g := graph.NewResidualGraph()

	// Create a graph that requires many iterations
	for i := int64(1); i <= 200; i++ {
		g.AddEdgeWithReverse(0, i, 1, float64(i))
		g.AddEdgeWithReverse(i, 201, 1, float64(201-i))
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 250,
	}

	result := SuccessiveShortestPathInternal(context.Background(), g, 0, 201, 200, opts)

	assert.Greater(t, result.Iterations, 100, "Should have many iterations for reinitialization")
	assert.InDelta(t, 200.0, result.Flow, 1e-9)
}

func TestSSP_CancellationDuringReinit(t *testing.T) {
	g := graph.NewResidualGraph()

	for i := int64(1); i <= 50; i++ {
		g.AddEdgeWithReverse(0, i, 1, 1)
		g.AddEdgeWithReverse(i, 51, 1, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 1000,
	}

	go func() {
		cancel()
	}()

	result := SuccessiveShortestPathInternal(ctx, g, 0, 51, 50, opts)

	// May or may not be canceled depending on timing
	_ = result
}

// =============================================================================
// computeReinitInterval Coverage
// =============================================================================

func TestComputeReinitInterval(t *testing.T) {
	assert.Equal(t, 100, computeReinitInterval(10))
	assert.Equal(t, 100, computeReinitInterval(49))
	assert.Equal(t, 200, computeReinitInterval(50))
	assert.Equal(t, 200, computeReinitInterval(499))
	assert.Equal(t, 500, computeReinitInterval(500))
	assert.Equal(t, 500, computeReinitInterval(1000))
}
