// Package graph provides core data structures and algorithms for graph operations.
//
// This file provides utilities for path manipulation in flow networks:
//   - Path reconstruction from parent pointers
//   - Bottleneck capacity computation
//   - Flow augmentation along paths
package graph

import (
	"logistics/pkg/domain"
)

// ReconstructPath builds a path from source to sink using the parent map
// produced by BFS or other shortest path algorithms.
//
// The parent map encodes a shortest path tree where parent[v] = u means
// the path to v goes through u. Starting from sink, we follow parent
// pointers back to source and reverse to get the forward path.
//
// Parameters:
//   - parent: Map from node ID to parent node ID (parent[source] = -1)
//   - source: Starting node of the path
//   - sink: Ending node of the path
//
// Returns:
//   - Slice of node IDs from source to sink (inclusive)
//   - Empty slice if no path exists (sink not in parent map or not connected)
//
// Example:
//
//	result := BFS(g, 1, 5)
//	if result.Found {
//	    path := ReconstructPath(result.Parent, 1, 5)
//	    // path might be [1, 3, 4, 5]
//	}
//
// Implementation delegates to pkg/domain for consistency across packages.
func ReconstructPath(parent map[int64]int64, source, sink int64) []int64 {
	return domain.ReconstructPath(parent, source, sink)
}

// FindMinCapacityOnPath finds the minimum residual capacity along a path.
// This value (the "bottleneck") determines the maximum flow that can be
// pushed along this path without exceeding any edge's capacity.
//
// Parameters:
//   - g: The residual graph containing edge capacities
//   - path: Sequence of node IDs from source to sink
//
// Returns:
//   - Minimum capacity among all edges in the path
//   - 0 if path is invalid (< 2 nodes) or any edge doesn't exist
//
// Example:
//
//	path := []int64{1, 3, 5}
//	bottleneck := FindMinCapacityOnPath(g, path)
//	// bottleneck = min(cap[1->3], cap[3->5])
//
// Time Complexity: O(|path|)
func FindMinCapacityOnPath(g *ResidualGraph, path []int64) float64 {
	if len(path) < 2 {
		return 0
	}

	minCapacity := Infinity

	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]

		edge := g.GetEdge(from, to)
		if edge == nil {
			// Edge doesn't exist - invalid path
			return 0
		}

		if edge.Capacity < minCapacity {
			minCapacity = edge.Capacity
		}
	}

	// Guard against returning Infinity if path was empty
	if minCapacity == Infinity {
		return 0
	}

	return minCapacity
}

// AugmentPath pushes flow along a path and updates residual capacities.
//
// For each edge (u, v) in the path:
//   - Decreases capacity of (u, v) by flow
//   - Increases capacity of reverse edge (v, u) by flow
//   - Updates flow counters on both edges
//
// This maintains the residual graph property where the capacity of a reverse
// edge represents the amount of flow that can be "cancelled" by pushing
// flow in the opposite direction.
//
// Parameters:
//   - g: The residual graph (modified in place)
//   - path: Sequence of node IDs from source to sink
//   - flow: Amount of flow to push (should be <= bottleneck capacity)
//
// Example:
//
//	path := []int64{1, 3, 5}
//	bottleneck := FindMinCapacityOnPath(g, path)
//	AugmentPath(g, path, bottleneck)
//	// Now cap[1->3] decreased, cap[3->1] increased, etc.
//
// Note: This function assumes the path is valid and flow <= bottleneck.
// Calling with flow > bottleneck may result in negative capacities.
//
// Time Complexity: O(|path|)
func AugmentPath(g *ResidualGraph, path []int64, flow float64) {
	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]
		g.UpdateFlow(from, to, flow)
	}
}
