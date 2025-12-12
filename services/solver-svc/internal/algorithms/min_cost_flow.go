// Package algorithms provides implementations of network flow algorithms.
//
// This file implements the Successive Shortest Path (SSP) algorithm for
// Min-Cost Max-Flow problems, along with utility functions for algorithm
// selection and result handling.
//
// The SSP algorithm works by repeatedly finding shortest paths from source
// to sink using reduced costs, then augmenting flow along these paths.
// Node potentials are maintained to ensure non-negative reduced costs,
// enabling the use of Dijkstra's algorithm.
//
// For graphs with very large capacities, the Capacity Scaling variant
// (implemented in capacity_scaling.go) may be more efficient.
//
// References:
//   - Ahuja, R.K., et al. "Network Flows" (1993), Chapter 9
//   - Kleinberg, J., Tardos, E. "Algorithm Design" (2005), Chapter 7
package algorithms

import (
	"context"

	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// ShortestPathResult defines the interface for shortest path algorithm results.
// This abstraction allows SSP to work with both Dijkstra and Bellman-Ford results.
type ShortestPathResult interface {
	// GetDistances returns the shortest distances from source to all nodes.
	// Unreachable nodes have distance = graph.Infinity.
	GetDistances() map[int64]float64

	// GetParent returns the parent map for path reconstruction.
	// parent[v] = u means the shortest path to v goes through u.
	// parent[source] = -1.
	GetParent() map[int64]int64
}

// MinCostFlowResult encapsulates the result of a min-cost flow computation.
type MinCostFlowResult struct {
	// Flow is the total flow successfully pushed from source to sink.
	Flow float64

	// Cost is the total cost incurred (sum of flow * cost for all edges used).
	Cost float64

	// Iterations counts the number of augmenting paths found.
	Iterations int

	// Paths contains the augmenting paths if ReturnPaths option was enabled.
	Paths []converter.PathWithFlow

	// Canceled indicates if the computation was interrupted by context cancellation.
	Canceled bool
}

// =============================================================================
// Main Entry Points
// =============================================================================

// MinCostMaxFlow finds the minimum cost maximum flow from source to sink.
// This is the primary entry point that automatically selects the best algorithm
// based on graph characteristics.
//
// Algorithm Selection:
//   - For graphs with max capacity > 10^6: uses Capacity Scaling
//   - Otherwise: uses Successive Shortest Path with potentials
//
// Parameters:
//   - g: Residual graph to compute flow on
//   - source: Source node ID
//   - sink: Sink node ID
//   - requiredFlow: Maximum flow to push (use math.MaxFloat64 for max flow)
//   - options: Solver options (nil uses defaults)
//
// Returns:
//   - MinCostFlowResult with optimal flow and minimum cost
//
// Example:
//
//	g := converter.ToResidualGraph(protoGraph)
//	result := MinCostMaxFlow(g, sourceID, sinkID, math.MaxFloat64, nil)
//	fmt.Printf("Optimal flow: %f with cost: %f\n", result.Flow, result.Cost)
func MinCostMaxFlow(g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	return MinCostMaxFlowWithContext(context.Background(), g, source, sink, requiredFlow, options)
}

// MinCostMaxFlowWithContext is the context-aware version of MinCostMaxFlow.
// Supports cancellation via context for long-running computations.
//
// The computation is periodically interrupted to check context status.
// If cancelled, returns partial results with Canceled=true.
func MinCostMaxFlowWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	// Select algorithm based on graph characteristics
	recommendation := RecommendMinCostAlgorithm(g)

	switch recommendation {
	case MinCostAlgorithmCapacityScaling:
		return CapacityScalingMinCostFlowWithContext(ctx, g, source, sink, requiredFlow, options)
	default:
		return SuccessiveShortestPathInternal(ctx, g, source, sink, requiredFlow, options)
	}
}

// SuccessiveShortestPath is an alias for MinCostMaxFlow for API compatibility.
func SuccessiveShortestPath(g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	return MinCostMaxFlow(g, source, sink, requiredFlow, options)
}

// SuccessiveShortestPathWithContext is an alias with context support.
func SuccessiveShortestPathWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	return MinCostMaxFlowWithContext(ctx, g, source, sink, requiredFlow, options)
}

// MinCostFlowWithAlgorithm allows explicit algorithm selection, overriding
// the automatic recommendation.
//
// Use this when you know the algorithm characteristics of your graph or
// when benchmarking different algorithms.
func MinCostFlowWithAlgorithm(ctx context.Context, g *graph.ResidualGraph, source, sink int64, requiredFlow float64, algorithm MinCostAlgorithmType, options *SolverOptions) *MinCostFlowResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	switch algorithm {
	case MinCostAlgorithmCapacityScaling:
		return CapacityScalingMinCostFlowWithContext(ctx, g, source, sink, requiredFlow, options)
	default:
		return SuccessiveShortestPathInternal(ctx, g, source, sink, requiredFlow, options)
	}
}

// =============================================================================
// Successive Shortest Path Implementation
// =============================================================================

// SuccessiveShortestPathInternal implements the SSP algorithm with Johnson's
// potential technique for handling negative edge costs.
//
// Algorithm Overview:
//  1. Initialize potentials using Bellman-Ford from source (handles negative costs)
//  2. While flow < required and path exists:
//     a. Find shortest path using Dijkstra with reduced costs
//     b. Update potentials: π(v) += d(v)
//     c. Find bottleneck capacity and augment flow
//  3. Periodically reinitialize potentials for numerical stability
//
// Time Complexity: O(V*E + V*E*log(V) * F) where F = number of augmenting paths
// Space Complexity: O(V + E)
//
// The algorithm maintains the invariant that reduced costs c'(u,v) = c(u,v) + π(u) - π(v)
// are non-negative for all edges with positive residual capacity. This allows using
// Dijkstra's algorithm instead of Bellman-Ford for finding shortest paths.
func SuccessiveShortestPathInternal(ctx context.Context, g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	// Get sorted nodes for deterministic iteration
	nodes := g.GetSortedNodes()

	totalFlow := 0.0
	totalCost := 0.0
	iterations := 0
	var paths []converter.PathWithFlow

	// Initialize potentials to zero for all nodes
	potentials := make(map[int64]float64, len(nodes))
	for _, node := range nodes {
		potentials[node] = 0
	}

	// Compute initial potentials using Bellman-Ford
	// This handles any negative edge costs in the original graph
	initResult := BellmanFordWithContext(ctx, g, source)
	if initResult.Canceled {
		return &MinCostFlowResult{Canceled: true}
	}

	// If graph has negative cycles, min-cost flow is undefined
	if initResult.HasNegativeCycle {
		return &MinCostFlowResult{}
	}

	// Set initial potentials from Bellman-Ford distances
	for _, node := range nodes {
		if initResult.Distances[node] < graph.Infinity-graph.Epsilon {
			potentials[node] = initResult.Distances[node]
		}
	}

	// Frequency of context cancellation checks
	checkInterval := 50

	// Adaptive reinitializaiton interval for numerical stability
	// Larger graphs need less frequent reinitialization relative to their size
	reinitInterval := computeReinitInterval(len(nodes))

	// Track if we should use the initial BF result for the first path
	useInitialResult := true

	// Main SSP loop
	for totalFlow < requiredFlow-options.Epsilon {
		// Check iteration limit
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		// Periodic context cancellation check
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &MinCostFlowResult{
					Flow:       totalFlow,
					Cost:       totalCost,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		var spResult ShortestPathResult
		var shouldUpdatePotentials bool

		// Periodic full reinitialization for numerical stability
		// Over many iterations, floating-point errors can accumulate in potentials
		if iterations > 0 && iterations%reinitInterval == 0 {
			bfResult := BellmanFordWithContext(ctx, g, source)
			if bfResult.Canceled {
				return &MinCostFlowResult{
					Flow:       totalFlow,
					Cost:       totalCost,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			}
			if bfResult.HasNegativeCycle {
				// This shouldn't happen if the algorithm is correct
				break
			}

			// Complete potential reset
			for _, node := range nodes {
				if bfResult.Distances[node] < graph.Infinity-graph.Epsilon {
					potentials[node] = bfResult.Distances[node]
				}
			}
			spResult = bfResult
			shouldUpdatePotentials = false
		} else if useInitialResult {
			// Reuse the initial Bellman-Ford result for the first iteration
			spResult = initResult
			shouldUpdatePotentials = false
			useInitialResult = false
		} else {
			// Normal case: use Dijkstra with potentials
			dijkstraResult := DijkstraWithPotentialsContext(ctx, g, source, potentials)
			if dijkstraResult.Canceled {
				return &MinCostFlowResult{
					Flow:       totalFlow,
					Cost:       totalCost,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			}
			spResult = dijkstraResult
			shouldUpdatePotentials = true
		}

		distances := spResult.GetDistances()
		parent := spResult.GetParent()

		// No path to sink - flow is maximized
		if distances[sink] >= graph.Infinity-options.Epsilon {
			break
		}

		// Update potentials based on new distances
		if shouldUpdatePotentials {
			for _, node := range nodes {
				if distances[node] < graph.Infinity-graph.Epsilon {
					potentials[node] += distances[node]
				}
			}
		}

		// Reconstruct shortest path
		path := graph.ReconstructPath(parent, source, sink)
		if len(path) == 0 {
			break
		}

		// Find bottleneck capacity
		pathFlow := requiredFlow - totalFlow
		bottleneck := graph.FindMinCapacityOnPath(g, path)
		if bottleneck < pathFlow {
			pathFlow = bottleneck
		}

		if pathFlow <= options.Epsilon {
			break
		}

		// Compute path cost before augmentation (costs are on original edges)
		pathCost := computePathCost(g, path, pathFlow)

		// Augment flow along the path
		graph.AugmentPath(g, path, pathFlow)

		totalFlow += pathFlow
		totalCost += pathCost
		iterations++

		// Record path if requested
		if options.ReturnPaths {
			pathCopy := make([]int64, len(path))
			copy(pathCopy, path)
			paths = append(paths, converter.PathWithFlow{
				NodeIDs: pathCopy,
				Flow:    pathFlow,
			})
		}
	}

	return &MinCostFlowResult{
		Flow:       totalFlow,
		Cost:       totalCost,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}

// =============================================================================
// Bellman-Ford Based Implementation
// =============================================================================

// MinCostFlowBellmanFord implements min-cost flow using only Bellman-Ford
// for shortest path computations (no Dijkstra, no potentials).
//
// This is slower than the potentials-based version but is simpler and can
// handle any graph structure including those with negative cycles (which it
// will detect and abort).
//
// Use this when:
//   - Graph structure changes between iterations
//   - Debugging potential-related issues
//   - Educational purposes
//
// Time Complexity: O(V * E * F) where F = number of augmenting paths
func MinCostFlowBellmanFord(g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	return MinCostFlowBellmanFordWithContext(context.Background(), g, source, sink, requiredFlow, options)
}

// MinCostFlowBellmanFordWithContext is the context-aware version.
func MinCostFlowBellmanFordWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	totalFlow := 0.0
	totalCost := 0.0
	iterations := 0
	var paths []converter.PathWithFlow

	checkInterval := 50

	for totalFlow < requiredFlow-options.Epsilon {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &MinCostFlowResult{
					Flow:       totalFlow,
					Cost:       totalCost,
					Iterations: iterations,
					Paths:      paths,
					Canceled:   true,
				}
			default:
			}
		}

		// Run full Bellman-Ford each iteration
		bfResult := BellmanFordWithContext(ctx, g, source)
		if bfResult.Canceled {
			return &MinCostFlowResult{
				Flow:       totalFlow,
				Cost:       totalCost,
				Iterations: iterations,
				Paths:      paths,
				Canceled:   true,
			}
		}

		// Negative cycle detected - abort
		if bfResult.HasNegativeCycle {
			break
		}

		// No path to sink
		if bfResult.Distances[sink] >= graph.Infinity-options.Epsilon {
			break
		}

		path := graph.ReconstructPath(bfResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := requiredFlow - totalFlow
		bottleneck := graph.FindMinCapacityOnPath(g, path)
		if bottleneck < pathFlow {
			pathFlow = bottleneck
		}

		if pathFlow <= options.Epsilon {
			break
		}

		pathCost := computePathCost(g, path, pathFlow)
		graph.AugmentPath(g, path, pathFlow)

		totalFlow += pathFlow
		totalCost += pathCost
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

	return &MinCostFlowResult{
		Flow:       totalFlow,
		Cost:       totalCost,
		Iterations: iterations,
		Paths:      paths,
		Canceled:   false,
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// computePathCost calculates the total cost of pushing a given flow along a path.
// Cost = sum of (edge.Cost * flow) for all edges in the path.
func computePathCost(g *graph.ResidualGraph, path []int64, flow float64) float64 {
	cost := 0.0
	for i := 0; i < len(path)-1; i++ {
		edge := g.GetEdge(path[i], path[i+1])
		if edge != nil {
			cost += edge.Cost * flow
		}
	}
	return cost
}

// computeReinitInterval determines how often to reinitialize potentials
// using Bellman-Ford for numerical stability.
//
// The interval is adaptive based on graph size:
//   - Small graphs (< 50 nodes): every 100 iterations
//   - Medium graphs (50-500 nodes): every 200 iterations
//   - Large graphs (> 500 nodes): every 500 iterations
//
// This balances numerical stability against the overhead of Bellman-Ford.
func computeReinitInterval(nodeCount int) int {
	switch {
	case nodeCount < 50:
		return 100
	case nodeCount < 500:
		return 200
	default:
		return 500
	}
}
