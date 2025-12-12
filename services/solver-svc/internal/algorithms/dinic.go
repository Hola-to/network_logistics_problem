package algorithms

import (
	"context"

	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Dinic's Algorithm (Dinitz's Algorithm)
// =============================================================================
//
// Dinic's algorithm finds the maximum flow in a flow network. It improves upon
// Ford-Fulkerson by using BFS to construct level graphs and finding blocking
// flows, reducing the number of augmenting path searches.
//
// Time Complexity: O(V² × E) general case, O(E × √V) for unit capacity graphs
// Space Complexity: O(V + E)
//
// Key Features:
//   - Level graph construction using BFS
//   - Blocking flow computation in each phase
//   - Current arc optimization for efficiency
//   - Optimal for unit capacity networks and bipartite matching
//
// Algorithm Phases:
//  1. BFS from source to build level graph (assigns levels to vertices)
//  2. Find blocking flow using DFS with current arc optimization
//  3. Repeat until sink is unreachable from source
//
// References:
//   - Dinitz, Y. (1970). "Algorithm for solution of a problem of maximum flow
//     in a network with power estimation"
//   - Even, S. & Tarjan, R.E. (1975). "Network flow and testing graph connectivity"
// =============================================================================

// DinicResult contains the result of Dinic's algorithm.
type DinicResult struct {
	// MaxFlow is the maximum flow value computed.
	MaxFlow float64

	// Iterations is the number of BFS phases executed.
	Iterations int

	// Paths contains the augmenting paths found (if ReturnPaths option is enabled).
	Paths []converter.PathWithFlow

	// Canceled indicates whether the operation was canceled via context.
	Canceled bool
}

// Dinic executes Dinic's algorithm without context cancellation support.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options (nil for defaults)
//
// Returns:
//   - *DinicResult containing max flow and optional paths
func Dinic(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *DinicResult {
	return DinicWithContext(context.Background(), g, source, sink, options)
}

// DinicWithContext executes Dinic's algorithm with context cancellation.
// Uses deterministic ordering for reproducible results.
//
// Parameters:
//   - ctx: Context for cancellation support
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *DinicResult containing max flow, iteration count, paths, and cancellation status
func DinicWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *DinicResult {
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
				return &DinicResult{
					MaxFlow:    maxFlow,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		// Phase 1: Build level graph using BFS
		level := bfsLevelDeterministic(g, source)

		// If sink is unreachable, we've found maximum flow
		if _, exists := level[sink]; !exists {
			break
		}

		// Phase 2: Find blocking flow
		blockingFlow, blockingPaths := findBlockingFlow(g, source, sink, level, options)

		if blockingFlow <= options.Epsilon {
			break
		}

		maxFlow += blockingFlow

		if options.ReturnPaths {
			paths = append(paths, blockingPaths...)
		}

		iterations++
	}

	return &DinicResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}

// findBlockingFlow finds a blocking flow in the level graph.
// A blocking flow saturates at least one edge on every path from source to sink.
//
// Uses iterative DFS with current arc optimization to achieve efficient performance.
func findBlockingFlow(g *graph.ResidualGraph, source, sink int64, level map[int64]int, options *SolverOptions) (float64, []converter.PathWithFlow) {
	totalFlow := 0.0
	var paths []converter.PathWithFlow

	// Current arc optimization: track next edge to try for each node
	currentArc := make(map[int64]int)

	for {
		// Find an augmenting path and its flow
		path, pathFlow := dfsBlockingPath(g, source, sink, level, currentArc, options.Epsilon)

		if pathFlow <= options.Epsilon {
			break
		}

		totalFlow += pathFlow

		if options.ReturnPaths && len(path) > 0 {
			pathCopy := make([]int64, len(path))
			copy(pathCopy, path)
			paths = append(paths, converter.PathWithFlow{
				NodeIDs: pathCopy,
				Flow:    pathFlow,
			})
		}
	}

	return totalFlow, paths
}

// dfsBlockingPath finds one augmenting path using iterative DFS and augments it.
// Returns the path and the amount of flow pushed.
//
// The iterative implementation avoids stack overflow on deep graphs.
func dfsBlockingPath(g *graph.ResidualGraph, source, sink int64, level map[int64]int, currentArc map[int64]int, epsilon float64) ([]int64, float64) {
	type stackItem struct {
		node    int64
		pathIdx int
	}

	// Preallocate path and minCap slices
	path := make([]int64, 0, 64)
	minCap := make([]float64, 0, 64)

	stack := []stackItem{{node: source, pathIdx: 0}}
	path = append(path, source)
	minCap = append(minCap, graph.Infinity)

	for len(stack) > 0 {
		current := &stack[len(stack)-1]
		u := current.node

		// Found path to sink
		if u == sink {
			bottleneck := minCap[len(minCap)-1]

			// Augment path: update flows along the path
			for i := 0; i < len(path)-1; i++ {
				g.UpdateFlow(path[i], path[i+1], bottleneck)
			}

			// Return a copy of the path
			result := make([]int64, len(path))
			copy(result, path)
			return result, bottleneck
		}

		// Get edges from current node (deterministic order via EdgesList)
		edges := g.GetNeighborsList(u)
		startIdx := currentArc[u]

		advanced := false
		for i := startIdx; i < len(edges); i++ {
			edge := edges[i]
			v := edge.To

			// Check level graph constraints and capacity
			if level[v] != level[u]+1 || edge.Capacity <= epsilon {
				continue
			}

			// Update current arc
			currentArc[u] = i

			// Compute bottleneck to v
			newMinCap := minCap[len(minCap)-1]
			if edge.Capacity < newMinCap {
				newMinCap = edge.Capacity
			}

			// Push v onto path and stack
			path = append(path, v)
			minCap = append(minCap, newMinCap)
			stack = append(stack, stackItem{node: v, pathIdx: len(path) - 1})

			advanced = true
			break
		}

		if !advanced {
			// No valid edge found - backtrack
			currentArc[u] = len(edges) // Mark all edges as processed

			// Remove node from level graph (dead end optimization)
			delete(level, u)

			// Pop from stack and path
			stack = stack[:len(stack)-1]
			path = path[:len(path)-1]
			minCap = minCap[:len(minCap)-1]
		}
	}

	return nil, 0
}

// bfsLevelDeterministic constructs a level graph using BFS with deterministic ordering.
// Returns a map from node ID to its level (distance from source).
func bfsLevelDeterministic(g *graph.ResidualGraph, source int64) map[int64]int {
	level := make(map[int64]int, len(g.Nodes))
	level[source] = 0

	queue := make([]int64, 0, len(g.Nodes))
	queue = append(queue, source)
	head := 0

	for head < len(queue) {
		u := queue[head]
		head++

		// Use EdgesList for deterministic ordering
		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			v := edge.To
			if _, exists := level[v]; !exists && edge.Capacity > graph.Epsilon {
				level[v] = level[u] + 1
				queue = append(queue, v)
			}
		}
	}

	return level
}

// =============================================================================
// Recursive DFS Implementation (for comparison and small graphs)
// =============================================================================

// dinicDFSRecursive is a recursive DFS implementation for blocking flow.
// Use only for small graphs due to stack depth limitations.
//
// Returns the flow pushed and the path taken.
func dinicDFSRecursive(
	g *graph.ResidualGraph,
	u, sink int64,
	pushed float64,
	level map[int64]int,
	iter map[int64]int,
	epsilon float64,
) (float64, []int64) {
	if u == sink {
		return pushed, []int64{sink}
	}

	neighbors := g.GetNeighborsList(u)
	if neighbors == nil {
		return 0, nil
	}

	for ; iter[u] < len(neighbors); iter[u]++ {
		edge := neighbors[iter[u]]
		v := edge.To

		if level[v] != level[u]+1 || edge.Capacity <= epsilon {
			continue
		}

		canPush := min(pushed, edge.Capacity)
		flow, path := dinicDFSRecursive(g, v, sink, canPush, level, iter, epsilon)

		if flow > epsilon {
			g.UpdateFlow(u, v, flow)
			return flow, append([]int64{u}, path...)
		}
	}

	// Dead end - remove from level graph only after exhausting all edges
	delete(level, u)
	return 0, nil
}

// =============================================================================
// Dinic with Callback
// =============================================================================

// PathCallback is a function called for each augmenting path found.
type PathCallback func(path []int64, flow float64)

// DinicWithCallback executes Dinic's algorithm with a callback for each path.
// Useful for real-time path processing or streaming results.
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: The residual graph
//   - source: Source node ID
//   - sink: Sink node ID
//   - options: Solver options
//   - callback: Function called for each path (can be nil)
//
// Returns:
//   - *DinicResult with max flow and collected paths
func DinicWithCallback(
	ctx context.Context,
	g *graph.ResidualGraph,
	source, sink int64,
	options *SolverOptions,
	callback PathCallback,
) *DinicResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	maxFlow := 0.0
	iterations := 0
	var paths []converter.PathWithFlow

	const checkInterval = 50

	for options.MaxIterations <= 0 || iterations < options.MaxIterations {
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &DinicResult{
					MaxFlow:    maxFlow,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		level := bfsLevelDeterministic(g, source)
		if _, exists := level[sink]; !exists {
			break
		}

		currentArc := make(map[int64]int)

		for {
			path, flow := dfsBlockingPath(g, source, sink, level, currentArc, options.Epsilon)
			if flow <= options.Epsilon {
				break
			}

			maxFlow += flow

			// Call callback for real-time processing
			if callback != nil {
				callback(path, flow)
			}

			if options.ReturnPaths && len(path) > 0 {
				pathCopy := make([]int64, len(path))
				copy(pathCopy, path)
				paths = append(paths, converter.PathWithFlow{
					NodeIDs: pathCopy,
					Flow:    flow,
				})
			}
		}

		iterations++
	}

	return &DinicResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}
