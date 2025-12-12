package algorithms

import (
	"context"

	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Edmonds-Karp Algorithm
// =============================================================================
//
// The Edmonds-Karp algorithm is an implementation of the Ford-Fulkerson method
// using BFS to find augmenting paths. By always choosing the shortest augmenting
// path (in terms of number of edges), it guarantees polynomial time complexity.
//
// Time Complexity: O(V × E²)
// Space Complexity: O(V + E)
//
// Key Features:
//   - Uses BFS for finding augmenting paths (shortest path in unweighted graph)
//   - Guaranteed polynomial time (unlike basic Ford-Fulkerson)
//   - Simple to implement and understand
//   - Good for educational purposes and medium-sized graphs
//
// Comparison with other algorithms:
//   - Slower than Dinic for large graphs (O(V × E²) vs O(V² × E))
//   - More predictable than Ford-Fulkerson with DFS
//   - Simpler than Push-Relabel
//
// References:
//   - Edmonds, J. & Karp, R.M. (1972). "Theoretical improvements in
//     algorithmic efficiency for network flow problems"
// =============================================================================

// EdmondsKarpResult contains the result of the Edmonds-Karp algorithm.
type EdmondsKarpResult struct {
	// MaxFlow is the maximum flow value computed.
	MaxFlow float64

	// Iterations is the number of augmenting paths found.
	Iterations int

	// Paths contains the augmenting paths found (if ReturnPaths option is enabled).
	Paths []converter.PathWithFlow

	// Canceled indicates whether the operation was canceled via context.
	Canceled bool
}

// EdmondsKarp executes the Edmonds-Karp algorithm without context cancellation.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options (nil for defaults)
//
// Returns:
//   - *EdmondsKarpResult containing max flow and optional paths
func EdmondsKarp(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *EdmondsKarpResult {
	return EdmondsKarpWithContext(context.Background(), g, source, sink, options)
}

// EdmondsKarpWithContext executes the Edmonds-Karp algorithm with context cancellation.
// Uses deterministic BFS for reproducible results.
//
// Parameters:
//   - ctx: Context for cancellation support
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *EdmondsKarpResult containing max flow, iterations, paths, and cancellation status
func EdmondsKarpWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *EdmondsKarpResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	maxFlow := 0.0
	iterations := 0
	var paths []converter.PathWithFlow

	const checkInterval = 100

	for options.MaxIterations <= 0 || iterations < options.MaxIterations {
		// Periodic context check
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &EdmondsKarpResult{
					MaxFlow:    maxFlow,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		// Find shortest augmenting path using BFS
		bfsResult := graph.BFSDeterministic(g, source, sink)
		if !bfsResult.Found {
			break
		}

		// Reconstruct the path
		path := graph.ReconstructPath(bfsResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		// Find bottleneck capacity
		pathFlow := graph.FindMinCapacityOnPath(g, path)
		if pathFlow <= options.Epsilon {
			break
		}

		// Augment flow along the path
		graph.AugmentPath(g, path, pathFlow)

		maxFlow += pathFlow
		iterations++

		if options.ReturnPaths {
			pathCopy := make([]int64, len(path))
			copy(pathCopy, path)
			paths = append(paths, converter.PathWithFlow{
				NodeIDs: pathCopy,
				Flow:    pathFlow,
			})
		}
	}

	return &EdmondsKarpResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}
