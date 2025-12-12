package algorithms

import (
	"context"
	"math"
	"testing"
	"time"

	"logistics/services/solver-svc/internal/graph"

	"github.com/stretchr/testify/assert"
)

func TestFordFulkerson_SimpleGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	// S(0) -> A(1) -> T(2)
	g.AddEdgeWithReverse(0, 1, 10, 0)
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := FordFulkerson(g, 0, 2, nil)

	if math.Abs(result.MaxFlow-10) > 1e-9 {
		t.Errorf("Expected flow 10, got %f", result.MaxFlow)
	}
}

func TestFordFulkerson_DiamondGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	// Diamond shape
	//     1
	//    / \
	//   0   3
	//    \ /
	//     2
	g.AddEdgeWithReverse(0, 1, 10, 0)
	g.AddEdgeWithReverse(0, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	result := FordFulkerson(g, 0, 3, nil)

	if math.Abs(result.MaxFlow-20) > 1e-9 {
		t.Errorf("Expected flow 20, got %f", result.MaxFlow)
	}
}

func TestFordFulkerson_BottleneckGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	// S -> A -> B -> T, bottleneck at A->B
	g.AddEdgeWithReverse(0, 1, 100, 0)
	g.AddEdgeWithReverse(1, 2, 1, 0) // Bottleneck
	g.AddEdgeWithReverse(2, 3, 100, 0)

	result := FordFulkerson(g, 0, 3, nil)

	if math.Abs(result.MaxFlow-1) > 1e-9 {
		t.Errorf("Expected flow 1, got %f", result.MaxFlow)
	}
}

func TestFordFulkerson_ClassicExample(t *testing.T) {
	// Классический пример из учебников
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

	result := FordFulkerson(g, 0, 5, nil)

	// Максимальный поток = 23
	if math.Abs(result.MaxFlow-23) > 1e-9 {
		t.Errorf("Expected flow 23, got %f", result.MaxFlow)
	}
}

func TestFordFulkerson_MatchesEdmondsKarp(t *testing.T) {
	// Проверяем, что результаты совпадают
	createGraph := func() *graph.ResidualGraph {
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
	}

	ffResult := FordFulkerson(createGraph(), 0, 5, nil)
	ekResult := EdmondsKarp(createGraph(), 0, 5, nil)

	if math.Abs(ffResult.MaxFlow-ekResult.MaxFlow) > 1e-9 {
		t.Errorf("Ford-Fulkerson (%f) != Edmonds-Karp (%f)", ffResult.MaxFlow, ekResult.MaxFlow)
	}
}

func TestFordFulkerson_NoPath(t *testing.T) {
	g := graph.NewResidualGraph()

	// Разъединённый граф
	g.AddNode(0)
	g.AddNode(1)
	g.AddNode(2)
	g.AddEdgeWithReverse(0, 1, 10, 0)
	// Нет пути до 2

	result := FordFulkerson(g, 0, 2, nil)

	if result.MaxFlow != 0 {
		t.Errorf("Expected flow 0, got %f", result.MaxFlow)
	}
}

func TestFordFulkerson_WithMaxIterations(t *testing.T) {
	g := graph.NewResidualGraph()

	g.AddEdgeWithReverse(0, 1, 1000000, 0)
	g.AddEdgeWithReverse(1, 2, 1000000, 0)

	options := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 10, // Ограничиваем итерации
	}

	result := FordFulkerson(g, 0, 2, options)

	// Должен остановиться после 10 итераций
	if result.Iterations > 10 {
		t.Errorf("Expected <= 10 iterations, got %d", result.Iterations)
	}
}

func TestFordFulkersonIterative_LargeGraph(t *testing.T) {
	// Тест итеративной версии на большом графе (чтобы избежать stack overflow)
	g := graph.NewResidualGraph()

	// Длинная цепочка: 0 -> 1 -> 2 -> ... -> 999 -> 1000
	n := 1000
	for i := 0; i < n; i++ {
		g.AddEdgeWithReverse(int64(i), int64(i+1), 10, 0)
	}

	result := FordFulkersonIterative(g, 0, int64(n), nil)

	if math.Abs(result.MaxFlow-10) > 1e-9 {
		t.Errorf("Expected flow 10, got %f", result.MaxFlow)
	}
}

// Benchmark для сравнения
func BenchmarkFordFulkerson(b *testing.B) {
	for i := 0; i < b.N; i++ {
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

		FordFulkerson(g, 0, 5, nil)
	}
}

func BenchmarkEdmondsKarp(b *testing.B) {
	for i := 0; i < b.N; i++ {
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

		EdmondsKarp(g, 0, 5, nil)
	}
}

// =============================================================================
// FordFulkersonRecursive Coverage
// =============================================================================

func TestFordFulkersonRecursive_Basic(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	result := FordFulkersonRecursive(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestFordFulkersonRecursive_Diamond(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(1, 3, 10, 0)
	g.AddEdgeWithReverse(2, 4, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)

	result := FordFulkersonRecursive(g, 1, 4, DefaultSolverOptions())

	assert.InDelta(t, 20.0, result.MaxFlow, 1e-9)
}

func TestFordFulkersonRecursive_NoPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)

	result := FordFulkersonRecursive(g, 1, 2, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestFordFulkersonRecursive_Cancellation(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 1000; i++ {
		g.AddEdgeWithReverse(i, i+1, 1, 0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := FordFulkersonRecursiveWithContext(ctx, g, 0, 1000, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}

func TestFordFulkersonRecursive_MaxIterationsInternal(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 1000000, 0)

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 0, // Unlimited - but internal limit is 1_000_000
	}

	result := FordFulkersonRecursive(g, 1, 2, opts)

	assert.InDelta(t, 1000000.0, result.MaxFlow, 1e-9)
}

func TestFordFulkersonRecursive_MaxIterationsLimit(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 1000000, 0)

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 5,
	}

	result := FordFulkersonRecursive(g, 1, 2, opts)

	assert.LessOrEqual(t, result.Iterations, 5)
}

func TestFordFulkersonRecursive_ReturnPaths(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 0)
	g.AddEdgeWithReverse(2, 3, 5, 0)

	opts := &SolverOptions{
		Epsilon:     1e-9,
		ReturnPaths: true,
	}

	result := FordFulkersonRecursive(g, 1, 3, opts)

	assert.NotEmpty(t, result.Paths)
	assert.Equal(t, []int64{1, 2, 3}, result.Paths[0].NodeIDs)
}

func TestFordFulkersonRecursive_PathFlowBelowEpsilon(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 1e-12, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	result := FordFulkersonRecursive(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestFordFulkersonRecursive_NilOptions(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := FordFulkersonRecursive(g, 1, 2, nil)

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)
}

func TestFordFulkersonRecursive_EmptyPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 5, 0)
	g.AddEdgeWithReverse(2, 3, 5, 0)

	result := FordFulkersonRecursive(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 5.0, result.MaxFlow, 1e-9)

	// Second call - no more path
	result2 := FordFulkersonRecursive(g, 1, 3, DefaultSolverOptions())
	assert.Equal(t, 0.0, result2.MaxFlow)
}

func TestFordFulkersonRecursive_ComplexNetwork(t *testing.T) {
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

	result := FordFulkersonRecursive(g, 0, 5, DefaultSolverOptions())

	assert.InDelta(t, 23.0, result.MaxFlow, 1e-9)
}

// =============================================================================
// dfsPathRecursive Coverage
// =============================================================================

func TestDfsPathRecursive_Basic(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	visited := make(map[int64]bool)
	parent := make(map[int64]int64)

	found := dfsPathRecursive(g, 1, 3, visited, parent, 1e-9)

	assert.True(t, found)
	assert.Equal(t, int64(1), parent[2])
	assert.Equal(t, int64(2), parent[3])
}

func TestDfsPathRecursive_NoPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	visited := make(map[int64]bool)
	parent := make(map[int64]int64)

	found := dfsPathRecursive(g, 1, 3, visited, parent, 1e-9)

	assert.False(t, found)
}

func TestDfsPathRecursive_AlreadyVisited(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	visited := map[int64]bool{2: true} // Already visited
	parent := make(map[int64]int64)

	found := dfsPathRecursive(g, 1, 3, visited, parent, 1e-9)

	assert.False(t, found)
}

func TestDfsPathRecursive_ZeroCapacity(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 0, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	visited := make(map[int64]bool)
	parent := make(map[int64]int64)

	found := dfsPathRecursive(g, 1, 3, visited, parent, 1e-9)

	assert.False(t, found)
}

func TestFordFulkersonIterative_ContextTimeout(t *testing.T) {
	g := graph.NewResidualGraph()
	for i := int64(0); i < 1000; i++ {
		g.AddEdgeWithReverse(i, i+1, 1, 0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	result := FordFulkersonIterativeWithContext(ctx, g, 0, 1000, DefaultSolverOptions())

	assert.True(t, result.Canceled)
}
