// Package algorithms provides implementations of various network flow algorithms
// including max-flow and min-cost flow algorithms with support for context cancellation,
// deterministic execution, and performance optimizations.
package algorithms

import (
	"context"

	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Bellman-Ford Algorithm
// =============================================================================
//
// The Bellman-Ford algorithm computes shortest paths from a single source vertex
// to all other vertices in a weighted graph. Unlike Dijkstra's algorithm, it can
// handle graphs with negative edge weights and detect negative cycles.
//
// Time Complexity: O(V * E)
// Space Complexity: O(V)
//
// Use Cases:
//   - Finding shortest paths in graphs with negative weights
//   - Detecting negative cycles (arbitrage detection, etc.)
//   - Initializing potentials for successive shortest path algorithms
//
// Algorithm:
//   1. Initialize distances: dist[source] = 0, dist[v] = ∞ for all v ≠ source
//   2. Repeat V-1 times: relax all edges
//   3. Check for negative cycles by attempting one more relaxation
//
// References:
//   - Bellman, R. (1958). "On a routing problem"
//   - Ford, L.R. (1956). "Network Flow Theory"
// =============================================================================

// BellmanFordResult contains the result of the Bellman-Ford algorithm.
// It provides shortest path distances, parent pointers for path reconstruction,
// and information about negative cycles and cancellation status.
type BellmanFordResult struct {
	// Distances maps each node to its shortest distance from the source.
	// Unreachable nodes have distance equal to graph.Infinity.
	Distances map[int64]float64

	// Parent maps each node to its predecessor on the shortest path.
	// The source node and unreachable nodes have parent = -1.
	Parent map[int64]int64

	// HasNegativeCycle indicates whether a negative-weight cycle was detected.
	// If true, the distances may not be valid.
	HasNegativeCycle bool

	// Canceled indicates whether the operation was canceled via context.
	Canceled bool
}

// GetDistances implements the ShortestPathResult interface.
// Returns the map of shortest distances from the source to all nodes.
func (r *BellmanFordResult) GetDistances() map[int64]float64 {
	return r.Distances
}

// GetParent implements the ShortestPathResult interface.
// Returns the map of parent pointers for path reconstruction.
func (r *BellmanFordResult) GetParent() map[int64]int64 {
	return r.Parent
}

// BellmanFord executes the Bellman-Ford algorithm without context cancellation support.
// This is a convenience wrapper around BellmanFordWithContext using context.Background().
//
// Parameters:
//   - g: The residual graph to search
//   - source: The source node ID
//
// Returns:
//   - *BellmanFordResult containing distances, parents, and cycle detection result
func BellmanFord(g *graph.ResidualGraph, source int64) *BellmanFordResult {
	return BellmanFordWithContext(context.Background(), g, source)
}

// BellmanFordWithContext executes the Bellman-Ford algorithm with context cancellation.
// The algorithm processes nodes and edges in a deterministic order to ensure
// reproducible results across multiple runs.
//
// Parameters:
//   - ctx: Context for cancellation support
//   - g: The residual graph to search
//   - source: The source node ID
//
// Returns:
//   - *BellmanFordResult containing distances, parents, cycle info, and cancellation status
//
// Context Cancellation:
//
//	The algorithm checks for cancellation every 100 iterations.
//	If canceled, returns partial results with Canceled = true.
func BellmanFordWithContext(ctx context.Context, g *graph.ResidualGraph, source int64) *BellmanFordResult {
	// Get sorted nodes for deterministic iteration order
	nodes := g.GetSortedNodes()
	n := len(nodes)

	// Initialize distance and parent maps
	dist := make(map[int64]float64, n)
	parent := make(map[int64]int64, n)

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	// Context check interval - balance between responsiveness and performance
	const checkInterval = 100

	// Main loop: relax all edges V-1 times
	for i := 0; i < n-1; i++ {
		// Periodic context check
		if i%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &BellmanFordResult{
					Distances:        dist,
					Parent:           parent,
					HasNegativeCycle: false,
					Canceled:         true,
				}
			default:
			}
		}

		// Early termination if no updates occurred
		updated := relaxAllEdgesDeterministic(g, nodes, dist, parent)
		if !updated {
			break
		}
	}

	// Check for negative cycles by attempting one more relaxation
	hasNegativeCycle := checkNegativeCycleDeterministic(g, nodes, dist)

	return &BellmanFordResult{
		Distances:        dist,
		Parent:           parent,
		HasNegativeCycle: hasNegativeCycle,
		Canceled:         false,
	}
}

// BellmanFordWithPotentials executes Bellman-Ford using reduced costs based on potentials.
// This is used in successive shortest path algorithms where potentials are maintained
// to ensure non-negative reduced costs for Dijkstra's algorithm.
//
// The reduced cost of an edge (u, v) is: cost(u,v) + potential[u] - potential[v]
//
// Parameters:
//   - g: The residual graph
//   - source: The source node ID
//   - potentials: Map of node potentials (typically from previous shortest path computation)
//
// Returns:
//   - *BellmanFordResult with distances based on reduced costs
func BellmanFordWithPotentials(g *graph.ResidualGraph, source int64, potentials map[int64]float64) *BellmanFordResult {
	return BellmanFordWithPotentialsContext(context.Background(), g, source, potentials)
}

// BellmanFordWithPotentialsContext is the context-aware version of BellmanFordWithPotentials.
func BellmanFordWithPotentialsContext(ctx context.Context, g *graph.ResidualGraph, source int64, potentials map[int64]float64) *BellmanFordResult {
	nodes := g.GetSortedNodes()
	n := len(nodes)

	dist := make(map[int64]float64, n)
	parent := make(map[int64]int64, n)

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	const checkInterval = 100

	for i := 0; i < n-1; i++ {
		if i%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &BellmanFordResult{
					Distances:        dist,
					Parent:           parent,
					HasNegativeCycle: false,
					Canceled:         true,
				}
			default:
			}
		}

		updated := false

		// Iterate over nodes in deterministic order
		for _, u := range nodes {
			if dist[u] >= graph.Infinity-graph.Epsilon {
				continue
			}

			// Use EdgesList for deterministic edge order
			edges := g.GetNeighborsList(u)
			for _, edge := range edges {
				if edge.Capacity > graph.Epsilon {
					v := edge.To

					// Compute reduced cost using potentials
					reducedCost := edge.Cost + potentials[u] - potentials[v]
					newDist := dist[u] + reducedCost

					if newDist < dist[v]-graph.Epsilon {
						dist[v] = newDist
						parent[v] = u
						updated = true
					}
				}
			}
		}

		if !updated {
			break
		}
	}

	hasNegativeCycle := checkNegativeCycleWithPotentialsDeterministic(g, nodes, dist, potentials)

	return &BellmanFordResult{
		Distances:        dist,
		Parent:           parent,
		HasNegativeCycle: hasNegativeCycle,
		Canceled:         false,
	}
}

// BellmanFordToSink is an optimized version that terminates early when the sink
// distance becomes stable. This is useful when only the source-to-sink shortest
// path is needed and we want to avoid unnecessary iterations.
//
// Early Termination Conditions:
//  1. No updates occurred in an iteration (standard early exit)
//  2. Sink distance hasn't improved for several consecutive iterations
//
// The stability check helps avoid running all V-1 iterations when the sink
// is reached early and its distance won't change further.
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: The residual graph
//   - source: The source node ID
//   - sink: The target node ID
//
// Returns:
//   - *BellmanFordResult with early termination optimization
func BellmanFordToSink(ctx context.Context, g *graph.ResidualGraph, source, sink int64) *BellmanFordResult {
	nodes := g.GetSortedNodes()
	n := len(nodes)

	dist := make(map[int64]float64, n)
	parent := make(map[int64]int64, n)

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	const checkInterval = 100
	sinkStableIterations := 0    // Count of iterations where sink distance didn't improve
	const stabilityThreshold = 2 // Exit after this many stable iterations

	for i := 0; i < n-1; i++ {
		if i%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &BellmanFordResult{
					Distances:        dist,
					Parent:           parent,
					HasNegativeCycle: false,
					Canceled:         true,
				}
			default:
			}
		}

		prevSinkDist := dist[sink]
		updated := relaxAllEdgesDeterministic(g, nodes, dist, parent)

		// Standard early termination: no updates means we're done
		if !updated {
			break
		}

		// Check sink distance stability for early exit
		if dist[sink] < graph.Infinity-graph.Epsilon {
			// Sink is reachable, check if distance improved
			if dist[sink] >= prevSinkDist-graph.Epsilon {
				// Distance didn't improve
				sinkStableIterations++
				if sinkStableIterations >= stabilityThreshold {
					// Sink distance is stable, no need to continue
					// (but we still need to check for negative cycles)
					break
				}
			} else {
				// Distance improved, reset counter
				sinkStableIterations = 0
			}
		}
	}

	hasNegativeCycle := checkNegativeCycleDeterministic(g, nodes, dist)

	return &BellmanFordResult{
		Distances:        dist,
		Parent:           parent,
		HasNegativeCycle: hasNegativeCycle,
		Canceled:         false,
	}
}

// relaxAllEdgesDeterministic performs one iteration of edge relaxation in deterministic order.
// Returns true if any distance was updated, false otherwise.
func relaxAllEdgesDeterministic(g *graph.ResidualGraph, nodes []int64, dist map[int64]float64, parent map[int64]int64) bool {
	updated := false

	for _, u := range nodes {
		// Skip unreachable nodes
		if dist[u] >= graph.Infinity-graph.Epsilon {
			continue
		}

		// Use EdgesList for deterministic edge ordering
		edges := g.GetNeighborsList(u)
		for _, edge := range edges {
			// Only consider edges with positive residual capacity
			if edge.Capacity > graph.Epsilon {
				v := edge.To
				newDist := dist[u] + edge.Cost

				// Relaxation: update if we found a shorter path
				if newDist < dist[v]-graph.Epsilon {
					dist[v] = newDist
					parent[v] = u
					updated = true
				}
			}
		}
	}

	return updated
}

// checkNegativeCycleDeterministic checks for negative-weight cycles.
// A negative cycle exists if we can still relax any edge after V-1 iterations.
func checkNegativeCycleDeterministic(g *graph.ResidualGraph, nodes []int64, dist map[int64]float64) bool {
	for _, u := range nodes {
		if dist[u] >= graph.Infinity-graph.Epsilon {
			continue
		}

		edges := g.GetNeighborsList(u)
		for _, edge := range edges {
			if edge.Capacity > graph.Epsilon {
				v := edge.To
				if dist[u]+edge.Cost < dist[v]-graph.Epsilon {
					return true
				}
			}
		}
	}
	return false
}

// checkNegativeCycleWithPotentialsDeterministic checks for negative cycles using reduced costs.
func checkNegativeCycleWithPotentialsDeterministic(g *graph.ResidualGraph, nodes []int64, dist map[int64]float64, potentials map[int64]float64) bool {
	for _, u := range nodes {
		if dist[u] >= graph.Infinity-graph.Epsilon {
			continue
		}

		edges := g.GetNeighborsList(u)
		for _, edge := range edges {
			if edge.Capacity > graph.Epsilon {
				v := edge.To
				reducedCost := edge.Cost + potentials[u] - potentials[v]
				if dist[u]+reducedCost < dist[v]-graph.Epsilon {
					return true
				}
			}
		}
	}
	return false
}

// FindShortestPath finds the shortest path from source to sink using Bellman-Ford.
// Returns the path as a slice of node IDs, the total cost, and a success flag.
//
// Parameters:
//   - g: The residual graph
//   - source: The source node ID
//   - sink: The target node ID
//
// Returns:
//   - path: Slice of node IDs from source to sink (empty if no path exists)
//   - cost: Total cost of the path
//   - found: True if a path was found without negative cycles
func FindShortestPath(g *graph.ResidualGraph, source, sink int64) ([]int64, float64, bool) {
	result := BellmanFord(g, source)

	if result.HasNegativeCycle {
		return nil, 0, false
	}

	if result.Distances[sink] >= graph.Infinity-graph.Epsilon {
		return nil, 0, false
	}

	path := graph.ReconstructPath(result.Parent, source, sink)
	return path, result.Distances[sink], len(path) > 0
}
