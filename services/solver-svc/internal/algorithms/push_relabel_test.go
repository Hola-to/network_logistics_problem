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
				g := graph.NewResidualGraph()
				g.AddEdgeWithReverse(0, 1, 10, 0)
				g.AddEdgeWithReverse(0, 2, 10, 0)
				g.AddEdgeWithReverse(1, 2, 5, 0)
				g.AddEdgeWithReverse(2, 1, 5, 0)
				g.AddEdgeWithReverse(1, 3, 10, 0)
				g.AddEdgeWithReverse(2, 3, 10, 0)
				return g
			},
			source:       0,
			sink:         3,
			expectedFlow: 20,
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

func TestPushRelabel_AllVariants(t *testing.T) {
	setupGraph := func() *graph.ResidualGraph {
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

	t.Run("FIFO", func(t *testing.T) {
		g := setupGraph()
		result := PushRelabel(g, 0, 5, nil)
		assert.InDelta(t, 23.0, result.MaxFlow, 1e-9)
	})

	t.Run("HighestLabel", func(t *testing.T) {
		g := setupGraph()
		result := PushRelabelHighestLabel(g, 0, 5, nil)
		assert.InDelta(t, 23.0, result.MaxFlow, 1e-9)
	})

	t.Run("LowestLabel", func(t *testing.T) {
		g := setupGraph()
		result := PushRelabelLowestLabel(g, 0, 5, nil)
		assert.InDelta(t, 23.0, result.MaxFlow, 1e-9)
	})
}

func TestPushRelabel_ExcessHandling(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 5, 0)
	g.AddEdgeWithReverse(2, 1, 5, 0)

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 5.0, result.MaxFlow, 1e-9)
}

func TestPushRelabel_HeightFunction(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)
	g.AddEdgeWithReverse(3, 4, 10, 0)

	result := PushRelabel(g, 1, 4, DefaultSolverOptions())

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

	for i := int64(1); i <= 10; i++ {
		g.AddEdgeWithReverse(i, i+1, 10, 0)
	}

	opts := &SolverOptions{
		Epsilon:       1e-9,
		MaxIterations: 2,
	}

	result := PushRelabel(g, 1, 11, opts)

	assert.LessOrEqual(t, result.Iterations, 2)
}

func TestPushRelabel_NoPath(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)
	g.AddEdgeWithReverse(1, 2, 10, 0)

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
}

func TestPushRelabel_EmptyGraph(t *testing.T) {
	g := graph.NewResidualGraph()

	result := PushRelabel(g, 1, 2, DefaultSolverOptions())

	assert.Equal(t, 0.0, result.MaxFlow)
	assert.Equal(t, 0, result.Iterations)
}

func TestPushRelabel_CreateBackEdge(t *testing.T) {
	g := graph.NewResidualGraph()
	g.AddEdge(1, 2, 10, 1)
	g.AddEdge(2, 3, 10, 1)

	result := PushRelabel(g, 1, 3, DefaultSolverOptions())

	assert.InDelta(t, 10.0, result.MaxFlow, 1e-9)

	reverseEdge := g.GetEdge(2, 1)
	assert.NotNil(t, reverseEdge, "Reverse edge should be created")
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
// BucketQueue Coverage
// =============================================================================

func TestBucketQueue_Basic(t *testing.T) {
	bq := newBucketQueue(10, 5)

	bq.push(0, 5)
	bq.push(1, 3)
	bq.push(2, 7)
	bq.push(3, 3)

	// Pop highest should return node 2 (height 7)
	idx, ok := bq.popHighest()
	assert.True(t, ok)
	assert.Equal(t, 2, idx)

	// Pop highest should return node 0 (height 5)
	idx, ok = bq.popHighest()
	assert.True(t, ok)
	assert.Equal(t, 0, idx)

	// Pop lowest from remaining should return 1 or 3 (both height 3)
	idx, ok = bq.popLowest()
	assert.True(t, ok)
	assert.True(t, idx == 1 || idx == 3)
}

func TestBucketQueue_UpdateHeight(t *testing.T) {
	bq := newBucketQueue(10, 3)

	bq.push(0, 2)
	bq.push(1, 3)

	// Update node 0 from height 2 to height 5
	bq.updateHeight(0, 2, 5)

	// Pop highest should now return node 0 (height 5)
	idx, ok := bq.popHighest()
	assert.True(t, ok)
	assert.Equal(t, 0, idx)
}

func TestBucketQueue_Empty(t *testing.T) {
	bq := newBucketQueue(10, 5)

	assert.True(t, bq.isEmpty())

	bq.push(0, 3)
	assert.False(t, bq.isEmpty())

	bq.popHighest()
	assert.True(t, bq.isEmpty())
}

func TestBucketQueue_PopEmpty(t *testing.T) {
	bq := newBucketQueue(10, 5)

	_, ok := bq.popHighest()
	assert.False(t, ok)

	_, ok = bq.popLowest()
	assert.False(t, ok)
}

func TestBucketQueue_DuplicatePush(t *testing.T) {
	bq := newBucketQueue(10, 3)

	bq.push(0, 5)
	bq.push(0, 5) // Duplicate - should be ignored

	assert.Equal(t, 1, bq.activeCount)

	idx, ok := bq.popHighest()
	assert.True(t, ok)
	assert.Equal(t, 0, idx)

	_, ok = bq.popHighest()
	assert.False(t, ok, "Should be empty after single pop")
}

func TestBucketQueue_Remove(t *testing.T) {
	bq := newBucketQueue(10, 3)

	bq.push(0, 3)
	bq.push(1, 5)
	bq.push(2, 3)

	bq.remove(0, 3)

	assert.Equal(t, 2, bq.activeCount)

	// Pop all remaining
	idx1, _ := bq.popHighest()
	idx2, _ := bq.popHighest()

	assert.Equal(t, 1, idx1) // Height 5
	assert.Equal(t, 2, idx2) // Height 3, remaining after remove
}

func TestBucketQueue_Clear(t *testing.T) {
	bq := newBucketQueue(10, 5)

	bq.push(0, 3)
	bq.push(1, 5)
	bq.push(2, 7)

	bq.clear()

	assert.True(t, bq.isEmpty())
	assert.Equal(t, 0, bq.activeCount)

	_, ok := bq.popHighest()
	assert.False(t, ok)
}

func TestBucketQueue_InvalidHeight(t *testing.T) {
	bq := newBucketQueue(10, 5)

	// Should not panic with invalid heights
	bq.push(0, -1)
	bq.push(1, 100) // Beyond maxHeight

	assert.Equal(t, 0, bq.activeCount)
}

func TestBucketQueue_RemoveNotInBucket(t *testing.T) {
	bq := newBucketQueue(10, 3)

	bq.push(0, 3)

	// Remove node that's not in bucket - should not panic
	bq.remove(1, 3)
	bq.remove(0, 5) // Wrong height

	assert.Equal(t, 1, bq.activeCount)
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

// =============================================================================
// Grid and Complete Graph Tests
// =============================================================================

func TestPushRelabel_Grid(t *testing.T) {
	g := graph.NewResidualGraph()
	n := 5

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			id := int64(i*n + j)
			g.AddNode(id)

			if j < n-1 {
				g.AddEdgeWithReverse(id, id+1, 10, 0)
			}
			if i < n-1 {
				g.AddEdgeWithReverse(id, id+int64(n), 10, 0)
			}
		}
	}

	source := int64(0)
	sink := int64(n*n - 1)

	resultFIFO := PushRelabel(g.Clone(), source, sink, nil)
	resultHighest := PushRelabelHighestLabel(g.Clone(), source, sink, nil)
	resultLowest := PushRelabelLowestLabel(g.Clone(), source, sink, nil)

	assert.InDelta(t, resultFIFO.MaxFlow, resultHighest.MaxFlow, 1e-9)
	assert.InDelta(t, resultFIFO.MaxFlow, resultLowest.MaxFlow, 1e-9)
}

func TestPushRelabel_Complete(t *testing.T) {
	g := graph.NewResidualGraph()
	n := 20

	for i := 0; i < n; i++ {
		g.AddNode(int64(i))
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			g.AddEdgeWithReverse(int64(i), int64(j), 10, 0)
		}
	}

	result := PushRelabel(g, 0, int64(n-1), nil)

	assert.True(t, result.MaxFlow > 0)
}

// =============================================================================
// Consistency Tests
// =============================================================================

func TestPushRelabel_ConsistencyWithDinic(t *testing.T) {
	testCases := []struct {
		name  string
		setup func() *graph.ResidualGraph
		s, t  int64
	}{
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
			s: 1, t: 4,
		},
		{
			name: "complex",
			setup: func() *graph.ResidualGraph {
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
			s: 0, t: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gDinic := tc.setup()
			gPR := tc.setup()

			dinicResult := Dinic(gDinic, tc.s, tc.t, nil)
			prResult := PushRelabel(gPR, tc.s, tc.t, nil)

			assert.InDelta(t, dinicResult.MaxFlow, prResult.MaxFlow, 1e-9,
				"Dinic and Push-Relabel should produce same max flow")
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkPushRelabel_Complete50(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		g := graph.NewResidualGraph()
		n := 50

		for i := 0; i < n; i++ {
			g.AddNode(int64(i))
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				g.AddEdgeWithReverse(int64(i), int64(j), float64(10+i+j), 0)
			}
		}
		b.StartTimer()

		PushRelabel(g, 0, int64(n-1), nil)
	}
}

func BenchmarkPushRelabelHighestLabel_Complete50(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		g := graph.NewResidualGraph()
		n := 50

		for i := 0; i < n; i++ {
			g.AddNode(int64(i))
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				g.AddEdgeWithReverse(int64(i), int64(j), float64(10+i+j), 0)
			}
		}
		b.StartTimer()

		PushRelabelHighestLabel(g, 0, int64(n-1), nil)
	}
}

func BenchmarkPushRelabelLowestLabel_Complete50(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		g := graph.NewResidualGraph()
		n := 50

		for i := 0; i < n; i++ {
			g.AddNode(int64(i))
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				g.AddEdgeWithReverse(int64(i), int64(j), float64(10+i+j), 0)
			}
		}
		b.StartTimer()

		PushRelabelLowestLabel(g, 0, int64(n-1), nil)
	}
}
