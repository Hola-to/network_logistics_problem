package algorithms

import (
	"container/heap"
	"context"

	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Dijkstra's Algorithm
// =============================================================================
//
// Dijkstra's algorithm finds the shortest paths from a single source vertex to
// all other vertices in a graph with non-negative edge weights.
//
// Time Complexity: O((V + E) log V) with binary heap
// Space Complexity: O(V)
//
// Use Cases:
//   - Finding shortest paths in graphs with non-negative weights
//   - Successive shortest path algorithms (with potentials to handle negative costs)
//   - General single-source shortest path problems
//
// Important:
//   - Standard Dijkstra cannot handle negative edge weights correctly
//   - This implementation includes automatic fallback to Bellman-Ford when
//     negative edges are detected
//
// References:
//   - Dijkstra, E. W. (1959). "A note on two problems in connexion with graphs"
// =============================================================================

// DijkstraResult contains the result of Dijkstra's algorithm.
type DijkstraResult struct {
	// Distances maps each node to its shortest distance from the source.
	Distances map[int64]float64

	// Parent maps each node to its predecessor on the shortest path.
	Parent map[int64]int64

	// Canceled indicates whether the operation was canceled via context.
	Canceled bool

	// UsedBellmanFord indicates whether the algorithm fell back to Bellman-Ford
	// due to negative edge weights being detected.
	UsedBellmanFord bool
}

// GetDistances implements the ShortestPathResult interface.
func (r *DijkstraResult) GetDistances() map[int64]float64 {
	return r.Distances
}

// GetParent implements the ShortestPathResult interface.
func (r *DijkstraResult) GetParent() map[int64]int64 {
	return r.Parent
}

// priorityQueueItem represents an element in the priority queue.
type priorityQueueItem struct {
	node     int64
	distance float64
	index    int // Index in the heap for updates
}

// priorityQueue implements heap.Interface for Dijkstra's algorithm.
// It is a min-heap based on distance, with tie-breaking by node ID for determinism.
type priorityQueue []*priorityQueueItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Primary sort: by distance (min-heap)
	if pq[i].distance != pq[j].distance {
		return pq[i].distance < pq[j].distance
	}
	// Secondary sort: by node ID for deterministic ordering
	return pq[i].node < pq[j].node
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*priorityQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // Avoid memory leak
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// update modifies the distance of an item in the priority queue.
func (pq *priorityQueue) update(item *priorityQueueItem, distance float64) {
	item.distance = distance
	heap.Fix(pq, item.index)
}

// Dijkstra executes Dijkstra's algorithm without context cancellation support.
// Automatically falls back to Bellman-Ford if negative edges are detected.
//
// Parameters:
//   - g: The residual graph to search
//   - source: The source node ID
//
// Returns:
//   - *DijkstraResult containing distances and parent pointers
func Dijkstra(g *graph.ResidualGraph, source int64) *DijkstraResult {
	return DijkstraWithContext(context.Background(), g, source)
}

// DijkstraWithContext executes Dijkstra's algorithm with context cancellation.
// Uses deterministic ordering for reproducible results.
//
// Parameters:
//   - ctx: Context for cancellation support
//   - g: The residual graph to search
//   - source: The source node ID
//
// Returns:
//   - *DijkstraResult with distances, parents, and status flags
func DijkstraWithContext(ctx context.Context, g *graph.ResidualGraph, source int64) *DijkstraResult {
	nodes := g.GetSortedNodes()

	dist := make(map[int64]float64, len(nodes))
	parent := make(map[int64]int64, len(nodes))

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	pq := make(priorityQueue, 0, len(nodes))
	heap.Init(&pq)
	heap.Push(&pq, &priorityQueueItem{
		node:     source,
		distance: 0,
	})

	const checkInterval = 100
	iterations := 0

	for pq.Len() > 0 {
		// Periodic context check
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &DijkstraResult{
					Distances: dist,
					Parent:    parent,
					Canceled:  true,
				}
			default:
			}
		}
		iterations++

		current := heap.Pop(&pq).(*priorityQueueItem)
		u := current.node

		// Skip stale entries (already processed with a better distance)
		if current.distance > dist[u]+graph.Epsilon {
			continue
		}

		// Use EdgesList for deterministic edge ordering
		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			if edge.Capacity <= graph.Epsilon {
				continue
			}

			v := edge.To

			// Detect negative edge weight - fallback to Bellman-Ford
			if edge.Cost < -graph.Epsilon {
				bfResult := BellmanFordWithContext(ctx, g, source)
				return &DijkstraResult{
					Distances:       bfResult.Distances,
					Parent:          bfResult.Parent,
					Canceled:        bfResult.Canceled,
					UsedBellmanFord: true,
				}
			}

			newDist := dist[u] + edge.Cost

			if newDist < dist[v]-graph.Epsilon {
				dist[v] = newDist
				parent[v] = u
				heap.Push(&pq, &priorityQueueItem{
					node:     v,
					distance: newDist,
				})
			}
		}
	}

	return &DijkstraResult{
		Distances: dist,
		Parent:    parent,
		Canceled:  false,
	}
}

// DijkstraWithPotentials executes Dijkstra using reduced costs based on potentials.
// This allows Dijkstra to work correctly in graphs that originally had negative costs,
// as long as the potentials make all reduced costs non-negative.
//
// The reduced cost of edge (u,v) is: cost(u,v) + potential[u] - potential[v]
//
// Parameters:
//   - g: The residual graph
//   - source: The source node ID
//   - potentials: Map of node potentials
//
// Returns:
//   - *DijkstraResult with distances based on reduced costs
func DijkstraWithPotentials(g *graph.ResidualGraph, source int64, potentials map[int64]float64) *DijkstraResult {
	return DijkstraWithPotentialsContext(context.Background(), g, source, potentials)
}

// DijkstraWithPotentialsContext is the context-aware version with configurable fallback threshold.
func DijkstraWithPotentialsContext(ctx context.Context, g *graph.ResidualGraph, source int64, potentials map[int64]float64) *DijkstraResult {
	return DijkstraWithPotentialsContextEx(ctx, g, source, potentials, DefaultNegativeEdgeFallbackThreshold)
}

// DefaultNegativeEdgeFallbackThreshold is the default number of negative reduced costs
// to tolerate before falling back to Bellman-Ford.
const DefaultNegativeEdgeFallbackThreshold = 3

// DijkstraWithPotentialsContextEx provides full control over fallback behavior.
//
// This function uses Johnson's technique with node potentials to handle graphs
// that originally had negative edge costs. The reduced cost of an edge (u,v) is:
//
//	reducedCost(u,v) = cost(u,v) + potential[u] - potential[v]
//
// With proper potentials (computed by Bellman-Ford), all reduced costs should be
// non-negative. However, due to floating-point errors or improper potential updates,
// small negative values may occur.
//
// Fallback Behavior:
//   - Tiny negative values (> -Epsilon): clamped to 0 (numerical noise)
//   - Significant negative values: immediate fallback to Bellman-Ford
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: The residual graph
//   - source: The source node ID
//   - potentials: Map of node potentials
//   - fallbackThreshold: Ignored in current implementation (kept for API compatibility)
//
// Returns:
//   - *DijkstraResult with distances based on reduced costs
func DijkstraWithPotentialsContextEx(ctx context.Context, g *graph.ResidualGraph, source int64, potentials map[int64]float64, fallbackThreshold int) *DijkstraResult {
	nodes := g.GetSortedNodes()

	dist := make(map[int64]float64, len(nodes))
	parent := make(map[int64]int64, len(nodes))
	items := make(map[int64]*priorityQueueItem, len(nodes))

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	pq := make(priorityQueue, 0, len(nodes))
	heap.Init(&pq)

	startItem := &priorityQueueItem{
		node:     source,
		distance: 0,
	}
	heap.Push(&pq, startItem)
	items[source] = startItem

	const checkInterval = 100
	iterations := 0
	usedFallback := false

	for pq.Len() > 0 {
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &DijkstraResult{
					Distances: dist,
					Parent:    parent,
					Canceled:  true,
				}
			default:
			}
		}
		iterations++

		current := heap.Pop(&pq).(*priorityQueueItem)
		u := current.node

		// Skip stale entries
		if current.distance > dist[u]+graph.Epsilon {
			continue
		}

		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			if edge.Capacity <= graph.Epsilon {
				continue
			}

			v := edge.To

			// Compute reduced cost using potentials
			potU := potentials[u]
			potV := potentials[v]
			reducedCost := edge.Cost + potU - potV

			// Handle negative reduced costs
			if reducedCost < -graph.Epsilon {
				// Significant negative reduced cost detected.
				// This indicates either:
				// 1. Potentials are not properly maintained
				// 2. Graph structure changed after potential computation
				// 3. Numerical instability accumulated over iterations
				//
				// Fall back to Bellman-Ford which handles negative edges correctly.
				bfResult := BellmanFordWithPotentialsContext(ctx, g, source, potentials)
				return &DijkstraResult{
					Distances:       bfResult.Distances,
					Parent:          bfResult.Parent,
					Canceled:        bfResult.Canceled,
					UsedBellmanFord: true,
				}
			}

			// Clamp tiny negative values to zero (numerical noise)
			if reducedCost < 0 {
				reducedCost = 0
				usedFallback = true // Mark that we had numerical issues
			}

			newDist := dist[u] + reducedCost

			if newDist < dist[v]-graph.Epsilon {
				dist[v] = newDist
				parent[v] = u

				// Update existing item or add new one
				if item, exists := items[v]; exists && item.index >= 0 {
					pq.update(item, newDist)
				} else {
					newItem := &priorityQueueItem{
						node:     v,
						distance: newDist,
					}
					heap.Push(&pq, newItem)
					items[v] = newItem
				}
			}
		}
	}

	return &DijkstraResult{
		Distances:       dist,
		Parent:          parent,
		Canceled:        false,
		UsedBellmanFord: usedFallback,
	}
}

// DijkstraWithFallback explicitly checks for negative weights before running Dijkstra.
// If negative weights exist, it uses Bellman-Ford instead.
//
// Parameters:
//   - ctx: Context for cancellation
//   - g: The residual graph
//   - source: The source node ID
//
// Returns:
//   - *DijkstraResult with appropriate algorithm used
func DijkstraWithFallback(ctx context.Context, g *graph.ResidualGraph, source int64) *DijkstraResult {
	// Pre-check for negative weights in deterministic order
	nodes := g.GetSortedNodes()
	hasNegativeWeights := false

	for _, u := range nodes {
		edges := g.GetNeighborsList(u)
		for _, edge := range edges {
			if edge.Capacity > graph.Epsilon && edge.Cost < -graph.Epsilon {
				hasNegativeWeights = true
				break
			}
		}
		if hasNegativeWeights {
			break
		}
	}

	if hasNegativeWeights {
		bfResult := BellmanFordWithContext(ctx, g, source)
		return &DijkstraResult{
			Distances:       bfResult.Distances,
			Parent:          bfResult.Parent,
			Canceled:        bfResult.Canceled,
			UsedBellmanFord: true,
		}
	}

	return DijkstraWithContext(ctx, g, source)
}
