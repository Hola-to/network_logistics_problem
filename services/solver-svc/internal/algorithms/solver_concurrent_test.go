// internal/algorithms/solver_concurrent_test.go
package algorithms

import (
	"context"
	"sync"
	"testing"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"
)

func TestSolverConcurrency(t *testing.T) {
	// Создаём тестовый граф
	createTestGraph := func() *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		g.AddNode(1)
		g.AddNode(2)
		g.AddNode(3)
		g.AddNode(4)
		g.AddEdgeWithReverse(1, 2, 10, 1)
		g.AddEdgeWithReverse(1, 3, 10, 2)
		g.AddEdgeWithReverse(2, 4, 10, 1)
		g.AddEdgeWithReverse(3, 4, 10, 1)
		return g
	}

	pool := NewSolverPool(10)

	t.Run("ConcurrentSolves", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				g := createTestGraph()
				result := pool.SolvePooled(ctx, g, 1, 4, commonv1.Algorithm_ALGORITHM_DINIC, nil)

				if result.Error != nil {
					errors <- result.Error
					return
				}

				if result.MaxFlow != 20 {
					errors <- &testError{msg: "unexpected max flow"}
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		g := createTestGraph()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Отменяем сразу

		result := Solve(ctx, g, 1, 4, commonv1.Algorithm_ALGORITHM_DINIC, nil)

		if result.Status != commonv1.FlowStatus_FLOW_STATUS_ERROR {
			t.Errorf("expected timeout status, got %v", result.Status)
		}
	})

	t.Run("PoolExhaustion", func(t *testing.T) {
		smallPool := NewSolverPool(2)

		var wg sync.WaitGroup
		started := make(chan struct{}, 10)
		blocked := make(chan struct{})

		// Занимаем все слоты пула
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				ctx := context.Background()
				if err := smallPool.Acquire(ctx); err != nil {
					t.Error(err)
					return
				}
				started <- struct{}{}
				<-blocked // Ждём сигнала
				smallPool.Release()
			}()
		}

		// Ждём пока оба слота заняты
		<-started
		<-started

		// Пытаемся получить ещё один слот с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := smallPool.Acquire(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("expected deadline exceeded, got %v", err)
		}

		// Освобождаем
		close(blocked)
		wg.Wait()
	})
}

func TestGraphPoolConcurrency(t *testing.T) {
	pool := graph.GetPool()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			g := pool.AcquireGraph()
			defer pool.ReleaseGraph(g)

			// Модифицируем граф
			g.AddNode(1)
			g.AddNode(2)
			g.AddEdgeWithReverse(1, 2, 10, 1)

			// Проверяем
			if g.NodeCount() != 2 {
				t.Errorf("unexpected node count")
			}
		}()
	}

	wg.Wait()
}

func BenchmarkConcurrentSolves(b *testing.B) {
	pool := NewSolverPool(8)

	createTestGraph := func() *graph.ResidualGraph {
		g := graph.NewResidualGraph()
		for i := int64(1); i <= 100; i++ {
			g.AddNode(i)
		}
		for i := int64(1); i < 100; i++ {
			g.AddEdgeWithReverse(i, i+1, 10, 1)
			if i+10 <= 100 {
				g.AddEdgeWithReverse(i, i+10, 5, 2)
			}
		}
		return g
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := context.Background()
			g := createTestGraph()
			pool.SolvePooled(ctx, g, 1, 100, commonv1.Algorithm_ALGORITHM_DINIC, nil)
		}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
