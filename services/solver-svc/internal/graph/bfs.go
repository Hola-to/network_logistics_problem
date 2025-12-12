// Package graph provides core data structures and algorithms for graph-based
// network flow computations.
//
// This file implements Breadth-First Search (BFS) variants used by flow algorithms:
//   - Standard BFS for finding augmenting paths (Edmonds-Karp)
//   - Level BFS for building level graphs (Dinic's algorithm)
//   - Reverse BFS for computing node heights (Push-Relabel)
//
// All BFS implementations use deterministic node ordering to ensure reproducible
// results regardless of map iteration order.
package graph

import (
	"logistics/pkg/domain"
)

// BFSResult encapsulates the result of a BFS traversal.
// Imported from domain package for consistency across the codebase.
type BFSResult = domain.BFSResult

// =============================================================================
// Queue Implementation
// =============================================================================

// Queue provides an efficient FIFO queue for BFS traversal.
// It uses a slice with a head pointer to avoid repeated allocations
// during typical BFS operations.
//
// The queue grows as needed but reuses underlying storage between operations.
// For optimal performance with large graphs, pre-allocate with NewQueue(expectedSize).
type Queue struct {
	data []int64 // Underlying storage
	head int     // Index of next element to dequeue
}

// NewQueue creates a new Queue with the specified initial capacity.
// The capacity should be set to the expected maximum queue size
// (typically the number of nodes in the graph for BFS).
//
// Example:
//
//	q := NewQueue(len(graph.Nodes))
//	q.Push(sourceID)
func NewQueue(capacity int) *Queue {
	return &Queue{
		data: make([]int64, 0, capacity),
		head: 0,
	}
}

// Push adds an element to the end of the queue.
// Amortized O(1) time complexity.
func (q *Queue) Push(v int64) {
	q.data = append(q.data, v)
}

// Pop removes and returns the element at the front of the queue.
// O(1) time complexity.
//
// Panics if the queue is empty. Always check Empty() before calling Pop().
func (q *Queue) Pop() int64 {
	v := q.data[q.head]
	q.head++
	return v
}

// Empty returns true if the queue contains no elements.
func (q *Queue) Empty() bool {
	return q.head >= len(q.data)
}

// Len returns the number of elements currently in the queue.
func (q *Queue) Len() int {
	return len(q.data) - q.head
}

// Reset clears the queue for reuse, keeping the underlying capacity.
// This is more efficient than creating a new queue.
func (q *Queue) Reset() {
	q.data = q.data[:0]
	q.head = 0
}

// =============================================================================
// Standard BFS
// =============================================================================

// BFS performs breadth-first search from source to sink.
// This is the standard BFS used by the Edmonds-Karp algorithm.
//
// The search only traverses edges with positive residual capacity.
// Returns as soon as the sink is found (early termination).
//
// Parameters:
//   - g: The residual graph to search
//   - source: Starting node ID
//   - sink: Target node ID
//
// Returns:
//   - BFSResult with Found=true if sink is reachable, parent map for path reconstruction
//
// Example:
//
//	result := BFS(g, sourceID, sinkID)
//	if result.Found {
//	    path := ReconstructPath(result.Parent, sourceID, sinkID)
//	}
func BFS(g *ResidualGraph, source, sink int64) *BFSResult {
	return BFSDeterministic(g, source, sink)
}

// BFSDeterministic performs BFS with deterministic neighbor ordering.
// This is the primary BFS implementation that guarantees reproducible results.
//
// The algorithm uses EdgesList (which maintains insertion order) rather than
// iterating over maps, ensuring the same path is found regardless of Go's
// map iteration randomization.
//
// Time Complexity: O(V + E)
// Space Complexity: O(V)
func BFSDeterministic(g *ResidualGraph, source, sink int64) *BFSResult {
	parent := make(map[int64]int64, len(g.Nodes))
	visited := make(map[int64]bool, len(g.Nodes))

	queue := NewQueue(len(g.Nodes))
	queue.Push(source)
	visited[source] = true
	parent[source] = -1

	for !queue.Empty() {
		u := queue.Pop()

		// Use EdgesList for deterministic ordering
		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			v := edge.To

			// Only traverse edges with positive residual capacity
			if !visited[v] && edge.Capacity > Epsilon {
				parent[v] = u
				visited[v] = true
				queue.Push(v)

				// Early termination when sink is found
				if v == sink {
					return &BFSResult{
						Found:   true,
						Parent:  parent,
						Visited: visited,
					}
				}
			}
		}
	}

	return &BFSResult{
		Found:   false,
		Parent:  parent,
		Visited: visited,
	}
}

// =============================================================================
// Level BFS (for Dinic's Algorithm)
// =============================================================================

// BFSLevel builds a level graph by computing BFS distances from the source.
// This is used by Dinic's algorithm to construct the layered network.
//
// A level graph partitions vertices into layers where:
//   - level[source] = 0
//   - level[v] = level[u] + 1 for edge (u,v) in the BFS tree
//   - Only edges going from level i to level i+1 are considered valid
//
// Parameters:
//   - g: The residual graph
//   - source: Starting node ID
//
// Returns:
//   - Map from node ID to its level (BFS distance from source)
//   - Unreachable nodes are not included in the map
//
// Example:
//
//	level := BFSLevel(g, sourceID)
//	if _, exists := level[sinkID]; exists {
//	    // Sink is reachable, proceed with blocking flow
//	}
func BFSLevel(g *ResidualGraph, source int64) map[int64]int {
	level := make(map[int64]int, len(g.Nodes))
	level[source] = 0

	queue := NewQueue(len(g.Nodes))
	queue.Push(source)

	for !queue.Empty() {
		u := queue.Pop()

		// Deterministic neighbor ordering via EdgesList
		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			v := edge.To

			// Only consider edges with positive capacity
			if _, exists := level[v]; !exists && edge.Capacity > Epsilon {
				level[v] = level[u] + 1
				queue.Push(v)
			}
		}
	}

	return level
}

// =============================================================================
// BFS with Callback
// =============================================================================

// BFSWithCallback performs BFS and invokes a callback for each visited node.
// The callback receives the node ID and its level (distance from source).
//
// If the callback returns false, the BFS terminates early.
// This is useful for:
//   - Finding all nodes within a certain distance
//   - Computing reachability with early termination
//   - Custom traversal logic
//
// Parameters:
//   - g: The residual graph
//   - source: Starting node ID
//   - callback: Function called for each visited node; return false to stop
//
// Example:
//
//	// Find all nodes within 3 hops of source
//	nearby := make([]int64, 0)
//	BFSWithCallback(g, source, func(node int64, level int) bool {
//	    if level > 3 {
//	        return false // Stop BFS
//	    }
//	    nearby = append(nearby, node)
//	    return true
//	})
func BFSWithCallback(g *ResidualGraph, source int64, callback func(node int64, level int) bool) {
	visited := make(map[int64]bool, len(g.Nodes))
	level := make(map[int64]int, len(g.Nodes))

	queue := NewQueue(len(g.Nodes))
	queue.Push(source)
	visited[source] = true
	level[source] = 0

	for !queue.Empty() {
		u := queue.Pop()

		// Invoke callback; stop if it returns false
		if !callback(u, level[u]) {
			return
		}

		neighbors := g.GetNeighborsList(u)
		for _, edge := range neighbors {
			v := edge.To
			if !visited[v] && edge.Capacity > Epsilon {
				visited[v] = true
				level[v] = level[u] + 1
				queue.Push(v)
			}
		}
	}
}

// =============================================================================
// Reverse BFS (for Push-Relabel)
// =============================================================================

// BFSReverse performs backward BFS from the sink to compute node heights.
// This is used by the Push-Relabel algorithm for global relabeling.
//
// The search traverses edges in reverse direction: for edge (u,v), we check
// if we can reach u from v. This requires looking at incoming edges.
//
// Parameters:
//   - g: The residual graph
//   - sink: The sink node to start from
//
// Returns:
//   - Map from node ID to height (reverse BFS distance from sink)
//   - Nodes not reverse-reachable from sink are not included
//
// The heights are used in Push-Relabel to determine valid push directions:
// flow can only be pushed from higher to lower vertices.
func BFSReverse(g *ResidualGraph, sink int64) map[int64]int {
	height := make(map[int64]int, len(g.Nodes))
	height[sink] = 0

	queue := NewQueue(len(g.Nodes))
	queue.Push(sink)

	for !queue.Empty() {
		u := queue.Pop()

		// Get incoming edges in deterministic order
		incomingEdges := g.GetIncomingEdgesList(u)
		for _, incoming := range incomingEdges {
			v := incoming.From

			// Check if the reverse edge has capacity (can push from v to u)
			if _, exists := height[v]; !exists && incoming.Edge.Capacity > Epsilon {
				height[v] = height[u] + 1
				queue.Push(v)
			}
		}
	}

	return height
}

// =============================================================================
// Multi-Path BFS
// =============================================================================

// BFSAllPaths finds all shortest paths from source to sink.
// This is useful for analyzing network structure or finding alternative routes.
//
// The algorithm:
//  1. Build level graph using BFSLevel
//  2. DFS on level graph to enumerate all paths
//
// Parameters:
//   - g: The residual graph
//   - source: Starting node ID
//   - sink: Target node ID
//   - maxPaths: Maximum number of paths to find (0 uses default of 100)
//
// Returns:
//   - Slice of paths, where each path is a slice of node IDs from source to sink
//   - Empty slice if sink is unreachable
//
// Warning: The number of shortest paths can be exponential in graph size.
// Always use a reasonable maxPaths limit.
func BFSAllPaths(g *ResidualGraph, source, sink int64, maxPaths int) [][]int64 {
	if maxPaths <= 0 {
		maxPaths = 100
	}

	// Build level graph
	level := BFSLevel(g, source)
	if _, exists := level[sink]; !exists {
		return nil // Sink not reachable
	}

	var paths [][]int64
	var currentPath []int64

	// DFS on level graph to enumerate paths
	var dfs func(node int64)
	dfs = func(node int64) {
		if len(paths) >= maxPaths {
			return
		}

		currentPath = append(currentPath, node)

		if node == sink {
			// Found a complete path - copy it
			pathCopy := make([]int64, len(currentPath))
			copy(pathCopy, currentPath)
			paths = append(paths, pathCopy)
		} else {
			// Continue DFS along level graph edges
			neighbors := g.GetNeighborsList(node)
			for _, edge := range neighbors {
				v := edge.To
				// Only follow edges that go to next level
				if level[v] == level[node]+1 && edge.Capacity > Epsilon {
					dfs(v)
				}
			}
		}

		// Backtrack
		currentPath = currentPath[:len(currentPath)-1]
	}

	dfs(source)
	return paths
}
