package algorithms

import (
	"context"

	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Ford-Fulkerson Algorithm
// =============================================================================
//
// The Ford-Fulkerson algorithm computes maximum flow by repeatedly finding
// augmenting paths and pushing flow along them. This implementation uses DFS
// to find paths.
//
// Time Complexity: O(E × max_flow) - can be very slow for large max_flow values
// Space Complexity: O(V + E)
//
// IMPORTANT LIMITATIONS:
//   - May not terminate for irrational capacities
//   - Time complexity depends on max_flow value, not just graph size
//   - Consider using Edmonds-Karp or Dinic for production use
//
// Advantages:
//   - Simple to implement and understand
//   - Good for educational purposes
//   - Works well when max_flow is small relative to graph size
//
// This implementation provides both recursive and iterative DFS versions.
// The iterative version (FordFulkersonIterative) is recommended for large graphs
// to avoid stack overflow.
//
// References:
//   - Ford, L.R. & Fulkerson, D.R. (1956). "Maximal flow through a network"
// =============================================================================

// FordFulkersonResult contains the result of the Ford-Fulkerson algorithm.
type FordFulkersonResult struct {
	// MaxFlow is the maximum flow value computed.
	MaxFlow float64

	// Iterations is the number of augmenting paths found.
	Iterations int

	// Paths contains the augmenting paths found (if ReturnPaths option is enabled).
	Paths []converter.PathWithFlow

	// Canceled indicates whether the operation was canceled via context.
	Canceled bool
}

// FordFulkerson executes the Ford-Fulkerson algorithm using iterative DFS.
// This is the recommended version as it avoids stack overflow on deep graphs.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options (nil for defaults)
//
// Returns:
//   - *FordFulkersonResult containing max flow and optional paths
func FordFulkerson(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *FordFulkersonResult {
	// Use iterative version by default to avoid stack overflow
	return FordFulkersonIterative(g, source, sink, options)
}

// FordFulkersonWithContext executes Ford-Fulkerson with context cancellation.
func FordFulkersonWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *FordFulkersonResult {
	return FordFulkersonIterativeWithContext(ctx, g, source, sink, options)
}

// FordFulkersonRecursive executes Ford-Fulkerson using recursive DFS.
// WARNING: May cause stack overflow on graphs with deep paths (>10000 nodes).
// Use FordFulkersonIterative for large graphs.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *FordFulkersonResult containing max flow and optional paths
func FordFulkersonRecursive(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *FordFulkersonResult {
	return FordFulkersonRecursiveWithContext(context.Background(), g, source, sink, options)
}

// FordFulkersonRecursiveWithContext is the context-aware recursive version.
func FordFulkersonRecursiveWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *FordFulkersonResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	// Set a reasonable default max iterations for safety
	maxIter := options.MaxIterations
	if maxIter <= 0 {
		maxIter = 1_000_000
	}

	maxFlow := 0.0
	iterations := 0
	var paths []converter.PathWithFlow

	const checkInterval = 100

	for iterations < maxIter {
		// Periodic context check
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &FordFulkersonResult{
					MaxFlow:    maxFlow,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		visited := make(map[int64]bool)
		parent := make(map[int64]int64)

		// Find path using recursive DFS
		found := dfsPathRecursive(g, source, sink, visited, parent, options.Epsilon)
		if !found {
			break
		}

		path := graph.ReconstructPath(parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := graph.FindMinCapacityOnPath(g, path)
		if pathFlow <= options.Epsilon {
			break
		}

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

	return &FordFulkersonResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}

// dfsPathRecursive finds an augmenting path using recursive DFS.
// Uses deterministic edge ordering via EdgesList.
func dfsPathRecursive(g *graph.ResidualGraph, current, sink int64, visited map[int64]bool, parent map[int64]int64, epsilon float64) bool {
	if current == sink {
		return true
	}

	visited[current] = true

	// Use EdgesList for deterministic ordering
	neighbors := g.GetNeighborsList(current)
	for _, edge := range neighbors {
		next := edge.To
		if !visited[next] && edge.Capacity > epsilon {
			parent[next] = current

			if dfsPathRecursive(g, next, sink, visited, parent, epsilon) {
				return true
			}
		}
	}

	return false
}

// FordFulkersonIterative executes Ford-Fulkerson using iterative DFS.
// This version is safe for graphs with deep paths.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *FordFulkersonResult containing max flow and optional paths
func FordFulkersonIterative(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *FordFulkersonResult {
	return FordFulkersonIterativeWithContext(context.Background(), g, source, sink, options)
}

// FordFulkersonIterativeWithContext is the context-aware iterative version.
func FordFulkersonIterativeWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *FordFulkersonResult {
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
				return &FordFulkersonResult{
					MaxFlow:    maxFlow,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		// Find path using iterative DFS
		path, found := dfsPathIterative(g, source, sink, options.Epsilon)
		if !found {
			break
		}

		pathFlow := graph.FindMinCapacityOnPath(g, path)
		if pathFlow <= options.Epsilon {
			break
		}

		graph.AugmentPath(g, path, pathFlow)

		maxFlow += pathFlow
		iterations++

		if options.ReturnPaths {
			paths = append(paths, converter.PathWithFlow{
				NodeIDs: path,
				Flow:    pathFlow,
			})
		}
	}

	return &FordFulkersonResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}

// dfsPathIterative finds an augmenting path using iterative DFS.
// Uses explicit stack to avoid recursion depth issues.
func dfsPathIterative(g *graph.ResidualGraph, source, sink int64, epsilon float64) ([]int64, bool) {
	type stackItem struct {
		node    int64
		parent  int64
		edgeIdx int // Index of next edge to try
	}

	visited := make(map[int64]bool)
	parent := make(map[int64]int64)

	// Initialize stack with source
	stack := []stackItem{{node: source, parent: -1, edgeIdx: 0}}

	for len(stack) > 0 {
		current := &stack[len(stack)-1]

		// First visit to this node
		if !visited[current.node] {
			visited[current.node] = true
			parent[current.node] = current.parent

			// Check if we reached the sink
			if current.node == sink {
				return graph.ReconstructPath(parent, source, sink), true
			}
		}

		// Get edges in deterministic order
		neighbors := g.GetNeighborsList(current.node)

		// Find next unvisited neighbor with capacity
		found := false
		for i := current.edgeIdx; i < len(neighbors); i++ {
			edge := neighbors[i]
			next := edge.To

			if !visited[next] && edge.Capacity > epsilon {
				current.edgeIdx = i + 1 // Remember position for backtrack
				stack = append(stack, stackItem{node: next, parent: current.node, edgeIdx: 0})
				found = true
				break
			}
		}

		if !found {
			// No more neighbors - backtrack
			stack = stack[:len(stack)-1]
		}
	}

	return nil, false
}
