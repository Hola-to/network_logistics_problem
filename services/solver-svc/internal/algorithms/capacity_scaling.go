// Package algorithms provides implementations of various network flow algorithms.
//
// This file implements the Capacity Scaling algorithm for Min-Cost Max-Flow problems.
// Capacity Scaling is particularly efficient for graphs with large capacity values,
// achieving O(E² log U) complexity where U is the maximum edge capacity.
//
// The algorithm works in phases, starting with a large scaling factor (delta) and
// progressively halving it. In each phase, only edges with capacity >= delta are
// considered, which significantly reduces the number of augmenting paths needed.
//
// Reference: Ahuja, R.K., Magnanti, T.L., and Orlin, J.B. "Network Flows: Theory,
// Algorithms, and Applications" (1993), Chapter 10.
package algorithms

import (
	"container/heap"
	"context"
	"math"

	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// CapacityScalingThreshold defines the minimum max-capacity value for which
// Capacity Scaling algorithm should be preferred over Successive Shortest Path.
// For graphs with max capacity below this threshold, SSP is typically faster.
const CapacityScalingThreshold = 1e6

// =============================================================================
// Capacity Scaling Min-Cost Flow Algorithm
// =============================================================================

// CapacityScalingMinCostFlow implements the Capacity Scaling algorithm for
// finding minimum cost maximum flow.
//
// Algorithm Overview:
//  1. Find the maximum capacity U in the graph
//  2. Initialize delta = largest power of 2 <= U
//  3. Initialize node potentials using Bellman-Ford from source
//  4. For each scaling phase (while delta >= 1):
//     a. Find shortest paths only considering edges with capacity >= delta
//     b. Augment flow along these paths
//     c. Halve delta and repeat
//  5. Finish with standard SSP for remaining fractional flow
//
// Time Complexity: O(E² log U) where U = max capacity
// Space Complexity: O(V + E)
//
// Parameters:
//   - g: The residual graph to find flow in
//   - source: Source node ID
//   - sink: Sink node ID
//   - requiredFlow: Maximum amount of flow to push (use math.MaxFloat64 for max flow)
//   - options: Solver configuration options
//
// Returns:
//   - MinCostFlowResult containing flow value, cost, paths, and status
//
// Example:
//
//	g := graph.NewResidualGraph()
//	g.AddEdgeWithReverse(1, 2, 1000000, 1.0)  // Large capacity
//	g.AddEdgeWithReverse(2, 3, 1000000, 2.0)
//	result := CapacityScalingMinCostFlow(g, 1, 3, math.MaxFloat64, nil)
//	fmt.Printf("Flow: %f, Cost: %f\n", result.Flow, result.Cost)
func CapacityScalingMinCostFlow(g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	return CapacityScalingMinCostFlowWithContext(context.Background(), g, source, sink, requiredFlow, options)
}

// CapacityScalingMinCostFlowWithContext is the context-aware version of
// CapacityScalingMinCostFlow that supports cancellation and timeouts.
//
// The context is checked periodically during execution. If cancelled,
// the function returns immediately with partial results and Canceled=true.
//
// Parameters:
//   - ctx: Context for cancellation/timeout control
//   - g: The residual graph
//   - source: Source node ID
//   - sink: Sink node ID
//   - requiredFlow: Maximum flow to push
//   - options: Solver options (nil uses defaults)
//
// Returns:
//   - MinCostFlowResult with partial results if cancelled
func CapacityScalingMinCostFlowWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	// Find maximum capacity in the graph to determine initial delta
	maxCap := findMaxCapacity(g)
	if maxCap <= options.Epsilon {
		return &MinCostFlowResult{}
	}

	// Initialize delta as the largest power of 2 not exceeding maxCap
	// This ensures we start with the coarsest granularity
	delta := computeInitialDelta(maxCap)

	// Initialize node potentials using Bellman-Ford
	// Potentials are used to transform edge costs to non-negative reduced costs,
	// enabling the use of Dijkstra's algorithm in subsequent iterations
	potentials := initializePotentials(ctx, g, source)
	if potentials == nil {
		return &MinCostFlowResult{
			Canceled: ctx.Err() != nil,
		}
	}

	totalFlow := 0.0
	totalCost := 0.0
	iterations := 0
	var paths []converter.PathWithFlow

	// Frequency of context cancellation checks
	checkInterval := 20

	// Main scaling loop: process phases with decreasing delta
	for delta >= 1.0 && totalFlow < requiredFlow-options.Epsilon {
		// Check for context cancellation between phases
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

		// Process current delta phase
		phaseResult := processScalingPhase(ctx, g, source, sink, requiredFlow-totalFlow,
			delta, potentials, options, iterations, checkInterval)

		totalFlow += phaseResult.flow
		totalCost += phaseResult.cost
		iterations += phaseResult.iterations
		paths = append(paths, phaseResult.paths...)

		if phaseResult.canceled {
			return &MinCostFlowResult{
				Flow:       totalFlow,
				Cost:       totalCost,
				Iterations: iterations,
				Paths:      paths,
				Canceled:   true,
			}
		}

		// Halve delta for the next phase
		delta /= 2
	}

	// Final pass: use standard SSP for any remaining flow (handles delta < 1)
	if totalFlow < requiredFlow-options.Epsilon {
		finalResult := finishWithSSP(ctx, g, source, sink, requiredFlow-totalFlow, potentials, options)
		if finalResult.Canceled {
			return &MinCostFlowResult{
				Flow:       totalFlow,
				Cost:       totalCost,
				Iterations: iterations,
				Paths:      paths,
				Canceled:   true,
			}
		}

		totalFlow += finalResult.Flow
		totalCost += finalResult.Cost
		iterations += finalResult.Iterations
		paths = append(paths, finalResult.Paths...)
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
// Phase Processing
// =============================================================================

// phaseResult holds the results from processing a single scaling phase.
type phaseResult struct {
	flow       float64
	cost       float64
	iterations int
	paths      []converter.PathWithFlow
	canceled   bool
}

// processScalingPhase processes a single delta-phase of the capacity scaling algorithm.
// It repeatedly finds shortest paths in the delta-network and augments flow until
// no more augmenting paths exist in the current delta-network.
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: Residual graph
//   - source, sink: Source and sink node IDs
//   - remainingFlow: Maximum additional flow to push in this phase
//   - delta: Current scaling factor
//   - potentials: Node potentials for reduced cost computation
//   - options: Solver options
//   - startIterations: Initial iteration count (for MaxIterations check)
//   - checkInterval: How often to check context cancellation
//
// Returns:
//   - phaseResult containing flow pushed, cost incurred, and paths found
func processScalingPhase(
	ctx context.Context,
	g *graph.ResidualGraph,
	source, sink int64,
	remainingFlow, delta float64,
	potentials map[int64]float64,
	options *SolverOptions,
	startIterations, checkInterval int,
) phaseResult {
	result := phaseResult{}
	iterations := 0

	// Safety limit to prevent infinite loops in case of numerical issues
	maxPhaseIterations := len(g.Nodes) * len(g.Nodes)

	for result.flow < remainingFlow-options.Epsilon && iterations < maxPhaseIterations {
		// Check iteration limit
		if options.MaxIterations > 0 && startIterations+iterations >= options.MaxIterations {
			break
		}

		// Periodically check for context cancellation
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				result.canceled = true
				result.iterations = iterations
				return result
			default:
			}
		}

		// Find shortest path considering only edges with capacity >= delta
		spResult := dijkstraWithDeltaNetwork(ctx, g, source, sink, potentials, delta, options.Epsilon)
		if spResult.Canceled {
			result.canceled = true
			result.iterations = iterations
			return result
		}

		// No path found in delta-network - phase complete
		if spResult.Distances[sink] >= graph.Infinity-options.Epsilon {
			break
		}

		// Update potentials based on shortest path distances
		updatePotentials(g, potentials, spResult.Distances)

		// Reconstruct the shortest path
		path := graph.ReconstructPath(spResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		// Find the bottleneck capacity along the path
		pathFlow := findPathFlowWithDelta(g, path, remainingFlow-result.flow, delta, options.Epsilon)
		if pathFlow <= options.Epsilon {
			break
		}

		// Augment flow along the path and compute cost
		pathCost := augmentPathWithCost(g, path, pathFlow)

		result.flow += pathFlow
		result.cost += pathCost
		iterations++

		// Record path if requested
		if options.ReturnPaths {
			result.paths = append(result.paths, converter.PathWithFlow{
				NodeIDs: copyPath(path),
				Flow:    pathFlow,
			})
		}
	}

	result.iterations = iterations
	return result
}

// =============================================================================
// Helper Functions for Capacity Scaling
// =============================================================================

// computeInitialDelta computes the initial delta value as the largest power of 2
// that does not exceed maxCap. This provides optimal scaling behavior.
//
// Example: maxCap = 1000 -> delta = 512 (2^9)
func computeInitialDelta(maxCap float64) float64 {
	if maxCap <= 0 {
		return 0
	}
	delta := 1.0
	for delta*2 <= maxCap {
		delta *= 2
	}
	return delta
}

// initializePotentials computes initial node potentials using the Bellman-Ford
// algorithm. Potentials represent shortest path distances from source and are
// used to compute reduced costs: reducedCost(u,v) = cost(u,v) + π(u) - π(v).
//
// With proper potentials, all reduced costs are non-negative on shortest paths,
// enabling the use of Dijkstra's algorithm instead of Bellman-Ford in subsequent
// iterations.
//
// Returns nil if the context is cancelled during computation.
func initializePotentials(ctx context.Context, g *graph.ResidualGraph, source int64) map[int64]float64 {
	initResult := BellmanFordWithContext(ctx, g, source)
	if initResult.Canceled {
		return nil
	}

	if initResult.HasNegativeCycle {
		return nil
	}

	potentials := make(map[int64]float64, len(g.Nodes))
	for node := range g.Nodes {
		potentials[node] = 0
	}

	for node, dist := range initResult.Distances {
		if dist < graph.Infinity-graph.Epsilon {
			potentials[node] = dist
		}
	}

	return potentials
}

// updatePotentials updates the potential function after finding shortest paths.
// This maintains the property that reduced costs remain non-negative.
//
// The update rule is: π'(v) = π(v) + d(v) for all reachable vertices v,
// where d(v) is the shortest path distance computed with reduced costs.
func updatePotentials(g *graph.ResidualGraph, potentials map[int64]float64, distances map[int64]float64) {
	for node := range g.Nodes {
		if distances[node] < graph.Infinity-graph.Epsilon {
			potentials[node] += distances[node]
		}
	}
}

// dijkstraWithDeltaNetwork runs Dijkstra's algorithm on the delta-restricted network.
// Only edges with residual capacity >= delta are considered during the search.
//
// The algorithm uses reduced costs: reducedCost(u,v) = cost(u,v) + π(u) - π(v).
// With proper potentials, reduced costs are non-negative, making Dijkstra valid.
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: Residual graph
//   - source, sink: Source and sink node IDs
//   - potentials: Node potential function
//   - delta: Minimum edge capacity to consider
//   - epsilon: Numerical tolerance
//
// Returns:
//   - DijkstraResult with distances and parent pointers
func dijkstraWithDeltaNetwork(
	ctx context.Context,
	g *graph.ResidualGraph,
	source, sink int64,
	potentials map[int64]float64,
	delta, epsilon float64,
) *DijkstraResult {
	// Use sorted nodes for deterministic behavior
	nodes := g.GetSortedNodes()

	dist := make(map[int64]float64, len(nodes))
	parent := make(map[int64]int64, len(nodes))

	// Initialize distances to infinity
	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	// Priority queue for Dijkstra
	pq := make(dijkstraPQ, 0, len(nodes))
	heap.Init(&pq)
	heap.Push(&pq, &dijkstraPQItem{node: source, distance: 0})

	checkInterval := 100
	iterations := 0

	for pq.Len() > 0 {
		// Periodic cancellation check
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &DijkstraResult{Distances: dist, Parent: parent, Canceled: true}
			default:
			}
		}
		iterations++

		// Extract minimum distance node
		current := heap.Pop(&pq).(*dijkstraPQItem)
		u := current.node

		// Skip stale entries
		if current.distance > dist[u]+epsilon {
			continue
		}

		// Early termination if sink is reached
		if u == sink {
			break
		}

		// Explore neighbors in deterministic order via EdgesList
		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			// Skip edges with capacity below delta threshold
			if edge.Capacity < delta-epsilon {
				continue
			}

			v := edge.To

			// Compute reduced cost using potentials
			reducedCost := edge.Cost + potentials[u] - potentials[v]

			// Clamp small negative values caused by numerical errors
			if reducedCost < 0 {
				reducedCost = 0
			}

			newDist := dist[u] + reducedCost
			if newDist < dist[v]-epsilon {
				dist[v] = newDist
				parent[v] = u
				heap.Push(&pq, &dijkstraPQItem{node: v, distance: newDist})
			}
		}
	}

	return &DijkstraResult{Distances: dist, Parent: parent, Canceled: false}
}

// findPathFlowWithDelta computes the flow amount to push along a path.
// The flow is the minimum of:
//   - The remaining required flow
//   - The minimum edge capacity along the path
//   - Optionally rounded down to a multiple of delta (for integer scaling)
//
// Parameters:
//   - g: Residual graph
//   - path: Sequence of node IDs from source to sink
//   - remainingFlow: Maximum additional flow allowed
//   - delta: Current scaling factor (for rounding)
//   - epsilon: Numerical tolerance
//
// Returns:
//   - Flow amount to push (may be 0 if path is invalid)
func findPathFlowWithDelta(g *graph.ResidualGraph, path []int64, remainingFlow, delta, epsilon float64) float64 {
	pathFlow := remainingFlow

	// Find bottleneck capacity
	for i := 0; i < len(path)-1; i++ {
		edge := g.GetEdge(path[i], path[i+1])
		if edge == nil {
			return 0
		}
		if edge.Capacity < pathFlow {
			pathFlow = edge.Capacity
		}
	}

	// Round down to multiple of delta for integer capacity graphs
	// This maintains the scaling property and avoids fractional flows in early phases
	// NOTE: Do not add epsilon before floor - this caused incorrect rounding up
	if delta >= 1.0 && pathFlow >= delta {
		pathFlow = math.Floor(pathFlow/delta) * delta
	}

	return pathFlow
}

// augmentPathWithCost pushes flow along a path and returns the total cost incurred.
// Updates the residual graph by decreasing forward edge capacities and increasing
// backward edge capacities.
//
// Parameters:
//   - g: Residual graph (modified in place)
//   - path: Path from source to sink
//   - flow: Amount of flow to push
//
// Returns:
//   - Total cost of pushing this flow (sum of edge costs * flow)
func augmentPathWithCost(g *graph.ResidualGraph, path []int64, flow float64) float64 {
	pathCost := 0.0
	for i := 0; i < len(path)-1; i++ {
		edge := g.GetEdge(path[i], path[i+1])
		if edge != nil {
			pathCost += edge.Cost * flow
		}
	}
	graph.AugmentPath(g, path, flow)
	return pathCost
}

// finishWithSSP completes the flow computation using standard Successive Shortest Path.
// This is called after capacity scaling phases complete (when delta < 1) to handle
// any remaining fractional flow requirements.
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: Residual graph
//   - source, sink: Source and sink node IDs
//   - requiredFlow: Remaining flow to push
//   - potentials: Current node potentials
//   - options: Solver options
//
// Returns:
//   - MinCostFlowResult with the additional flow pushed
func finishWithSSP(
	ctx context.Context,
	g *graph.ResidualGraph,
	source, sink int64,
	requiredFlow float64,
	potentials map[int64]float64,
	options *SolverOptions,
) *MinCostFlowResult {
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

		// Use Dijkstra with potentials (no delta threshold)
		dijkstraResult := dijkstraWithPotentialsDeterministic(ctx, g, source, potentials, options.Epsilon)
		if dijkstraResult.Canceled {
			return &MinCostFlowResult{
				Flow:       totalFlow,
				Cost:       totalCost,
				Iterations: iterations,
				Paths:      paths,
				Canceled:   true,
			}
		}

		if dijkstraResult.Distances[sink] >= graph.Infinity-options.Epsilon {
			break
		}

		updatePotentials(g, potentials, dijkstraResult.Distances)

		path := graph.ReconstructPath(dijkstraResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := min(requiredFlow-totalFlow, graph.FindMinCapacityOnPath(g, path))
		if pathFlow <= options.Epsilon {
			break
		}

		pathCost := augmentPathWithCost(g, path, pathFlow)

		totalFlow += pathFlow
		totalCost += pathCost
		iterations++

		if options.ReturnPaths {
			paths = append(paths, converter.PathWithFlow{
				NodeIDs: copyPath(path),
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

// dijkstraWithPotentialsDeterministic runs Dijkstra's algorithm with potentials
// and deterministic node ordering. Unlike dijkstraWithDeltaNetwork, this version
// considers all edges regardless of capacity (no delta threshold).
//
// This is used in the final SSP phase after capacity scaling completes.
func dijkstraWithPotentialsDeterministic(
	ctx context.Context,
	g *graph.ResidualGraph,
	source int64,
	potentials map[int64]float64,
	epsilon float64,
) *DijkstraResult {
	nodes := g.GetSortedNodes()

	dist := make(map[int64]float64, len(nodes))
	parent := make(map[int64]int64, len(nodes))

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	pq := make(dijkstraPQ, 0, len(nodes))
	heap.Init(&pq)
	heap.Push(&pq, &dijkstraPQItem{node: source, distance: 0})

	checkInterval := 100
	iterations := 0

	for pq.Len() > 0 {
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &DijkstraResult{Distances: dist, Parent: parent, Canceled: true}
			default:
			}
		}
		iterations++

		current := heap.Pop(&pq).(*dijkstraPQItem)
		u := current.node

		if current.distance > dist[u]+epsilon {
			continue
		}

		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			// Consider all edges with positive capacity
			if edge.Capacity <= epsilon {
				continue
			}

			v := edge.To
			reducedCost := edge.Cost + potentials[u] - potentials[v]
			if reducedCost < 0 {
				reducedCost = 0
			}

			newDist := dist[u] + reducedCost
			if newDist < dist[v]-epsilon {
				dist[v] = newDist
				parent[v] = u
				heap.Push(&pq, &dijkstraPQItem{node: v, distance: newDist})
			}
		}
	}

	return &DijkstraResult{Distances: dist, Parent: parent, Canceled: false}
}

// =============================================================================
// Priority Queue for Dijkstra
// =============================================================================

// dijkstraPQItem represents an item in the Dijkstra priority queue.
type dijkstraPQItem struct {
	node     int64   // Node ID
	distance float64 // Distance from source
	index    int     // Index in the heap (managed by container/heap)
}

// dijkstraPQ implements a min-heap priority queue for Dijkstra's algorithm.
type dijkstraPQ []*dijkstraPQItem

func (pq dijkstraPQ) Len() int { return len(pq) }

func (pq dijkstraPQ) Less(i, j int) bool {
	// Min-heap by distance; break ties by node ID for determinism
	if pq[i].distance == pq[j].distance {
		return pq[i].node < pq[j].node
	}
	return pq[i].distance < pq[j].distance
}

func (pq dijkstraPQ) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *dijkstraPQ) Push(x any) {
	item := x.(*dijkstraPQItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *dijkstraPQ) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return item
}

// =============================================================================
// Utility Functions
// =============================================================================

// findMaxCapacity finds the maximum original capacity among all forward edges.
// This is used to determine the initial delta for capacity scaling.
func findMaxCapacity(g *graph.ResidualGraph) float64 {
	maxCap := 0.0
	for _, edges := range g.EdgesList {
		for _, edge := range edges {
			if !edge.IsReverse && edge.OriginalCapacity > maxCap {
				maxCap = edge.OriginalCapacity
			}
		}
	}
	return maxCap
}

// copyPath creates a deep copy of a path slice.
func copyPath(path []int64) []int64 {
	result := make([]int64, len(path))
	copy(result, path)
	return result
}

// =============================================================================
// Algorithm Selection Helpers
// =============================================================================

// MinCostAlgorithmType enumerates the available min-cost flow algorithms.
type MinCostAlgorithmType int

const (
	// MinCostAlgorithmSSP selects Successive Shortest Path algorithm.
	// Best for: small to medium graphs, graphs with small capacities.
	MinCostAlgorithmSSP MinCostAlgorithmType = iota

	// MinCostAlgorithmCapacityScaling selects Capacity Scaling algorithm.
	// Best for: large graphs with high capacity values (> 10^6).
	MinCostAlgorithmCapacityScaling
)

// String returns the algorithm name for logging/debugging.
func (t MinCostAlgorithmType) String() string {
	switch t {
	case MinCostAlgorithmSSP:
		return "SuccessiveShortestPath"
	case MinCostAlgorithmCapacityScaling:
		return "CapacityScaling"
	default:
		return "Unknown"
	}
}

// ShouldUseCapacityScaling determines if Capacity Scaling is recommended
// for the given graph based on maximum edge capacity.
//
// Capacity Scaling is preferred when:
//   - Maximum capacity exceeds CapacityScalingThreshold (10^6)
//   - The graph is large enough to benefit from scaling phases
//
// Returns true if Capacity Scaling is recommended.
func ShouldUseCapacityScaling(g *graph.ResidualGraph) bool {
	return findMaxCapacity(g) > CapacityScalingThreshold
}

// RecommendMinCostAlgorithm analyzes graph characteristics and recommends
// the most suitable min-cost flow algorithm.
//
// Decision factors:
//   - Maximum edge capacity (high capacity favors Capacity Scaling)
//   - Graph size (very small graphs don't benefit from scaling)
//
// Returns the recommended algorithm type.
func RecommendMinCostAlgorithm(g *graph.ResidualGraph) MinCostAlgorithmType {
	if ShouldUseCapacityScaling(g) {
		return MinCostAlgorithmCapacityScaling
	}
	return MinCostAlgorithmSSP
}
