// Package graph provides data structures and utilities for network flow algorithms.
package graph

import (
	"sort"
	"sync"

	"logistics/pkg/domain"
)

// =============================================================================
// Constants
// =============================================================================

// Epsilon is the tolerance for floating-point comparisons.
// Values smaller than Epsilon are considered zero.
// This is crucial for numerical stability in flow algorithms.
const Epsilon = domain.Epsilon

// Infinity represents an unreachable distance or unlimited capacity.
// Used as the initial distance in shortest path algorithms.
const Infinity = domain.Infinity

// =============================================================================
// Residual Edge
// =============================================================================

// ResidualEdge represents an edge in the residual graph.
//
// In the residual graph, each original edge (u, v) with capacity c and cost w
// is represented by two edges:
//   - Forward edge (u, v) with capacity c and cost w
//   - Backward edge (v, u) with capacity 0 and cost -w
//
// When flow f is pushed along (u, v):
//   - Forward edge capacity becomes c - f
//   - Backward edge capacity becomes f
//
// This allows the algorithm to "undo" flow decisions.
type ResidualEdge struct {
	// To is the destination node ID.
	To int64

	// Capacity is the current residual capacity.
	// For forward edges: OriginalCapacity - Flow
	// For backward edges: equals the flow on the corresponding forward edge
	Capacity float64

	// Cost is the cost per unit of flow.
	// For backward edges, this is the negative of the forward edge cost.
	Cost float64

	// Flow is the amount of flow currently on this edge.
	// Only meaningful for forward edges.
	Flow float64

	// OriginalCapacity is the initial capacity of the edge.
	// Used for reset operations and utilization calculations.
	OriginalCapacity float64

	// IsReverse indicates whether this is a backward (reverse) edge.
	// Reverse edges are created automatically and should not be counted
	// when computing statistics.
	IsReverse bool

	// Index is the position of this edge in the EdgesList slice.
	// Used for efficient edge lookup and current-arc optimization.
	Index int
}

// ResidualCapacity returns the remaining capacity on this edge.
// This is equivalent to accessing the Capacity field directly.
func (e *ResidualEdge) ResidualCapacity() float64 {
	return e.Capacity
}

// HasCapacity returns true if the edge has positive residual capacity.
// Uses Epsilon for floating-point comparison.
func (e *ResidualEdge) HasCapacity() bool {
	return e.Capacity > Epsilon
}

// =============================================================================
// Incoming Edge (for reverse traversal)
// =============================================================================

// IncomingEdge represents an edge for reverse graph traversal.
//
// Used by algorithms like Push-Relabel's globalRelabel that need to
// traverse edges in the reverse direction.
type IncomingEdge struct {
	// From is the source node of the edge.
	From int64

	// Edge is the edge data (points to node From).
	Edge *ResidualEdge
}

// =============================================================================
// Residual Graph
// =============================================================================

// ResidualGraph is the core data structure for network flow algorithms.
//
// It maintains both forward and backward edges, supporting efficient:
//   - Edge lookup by (from, to) pair: O(1)
//   - Neighbor iteration in deterministic order: O(degree)
//   - Incoming edge lookup for reverse traversal: O(1)
//
// # Edge Storage
//
// Edges are stored in two complementary structures:
//   - Edges: map for O(1) lookup by (from, to)
//   - EdgesList: slice for deterministic iteration order
//
// Both structures point to the same ResidualEdge objects.
//
// # Determinism
//
// Network flow algorithms can find different valid solutions depending on
// the order of edge traversal. To ensure deterministic results:
//   - Use GetNeighborsList() for iteration (not GetNeighbors())
//   - Use GetSortedNodes() for node iteration
//   - Use GetIncomingEdgesList() for reverse traversal
//
// # Thread Safety
//
// ResidualGraph is NOT thread-safe for concurrent writes. However,
// GetSortedNodes() is safe for concurrent reads due to internal locking.
// For full concurrent operations:
//   - Clone the graph for each goroutine
//   - Use SafeResidualGraph for shared read access
//
// # Example
//
//	g := NewResidualGraph()
//	g.AddNode(1)
//	g.AddNode(2)
//	g.AddNode(3)
//	g.AddEdgeWithReverse(1, 2, 10, 1.0)  // capacity=10, cost=1.0
//	g.AddEdgeWithReverse(2, 3, 5, 2.0)   // capacity=5, cost=2.0
//
//	// Find flow along path [1, 2, 3]
//	path := []int64{1, 2, 3}
//	minCap := FindMinCapacityOnPath(g, path)  // returns 5
//	AugmentPath(g, path, minCap)              // push 5 units
type ResidualGraph struct {
	// Nodes contains all node IDs in the graph.
	// The bool value is always true (used as a set).
	Nodes map[int64]bool

	// Edges provides O(1) edge lookup by (from, to) pair.
	// Edges[from][to] returns the ResidualEdge or nil.
	Edges map[int64]map[int64]*ResidualEdge

	// EdgesList provides deterministic edge iteration.
	// EdgesList[from] is a slice of edges sorted by insertion order.
	// This ensures algorithms produce the same results on every run.
	EdgesList map[int64][]*ResidualEdge

	//IncomingCache provides cache for EdgesList
	IncomingEdgesListCache map[int64][]IncomingEdge
	incomingCacheDirty     bool

	// ReverseEdges enables efficient reverse graph traversal.
	// ReverseEdges[to][from] points to the edge from 'from' to 'to'.
	ReverseEdges map[int64]map[int64]*ResidualEdge

	// sortedNodesMu protects sortedNodes cache for concurrent access.
	sortedNodesMu sync.Mutex

	// sortedNodes caches the sorted list of node IDs.
	// Invalidated when nodes are added.
	sortedNodes      []int64
	sortedNodesDirty bool
}

// NewResidualGraph creates a new empty residual graph.
//
// The graph is ready to use immediately. Add nodes with AddNode()
// and edges with AddEdgeWithReverse().
func NewResidualGraph() *ResidualGraph {
	return &ResidualGraph{
		Nodes:                  make(map[int64]bool),
		Edges:                  make(map[int64]map[int64]*ResidualEdge),
		EdgesList:              make(map[int64][]*ResidualEdge),
		ReverseEdges:           make(map[int64]map[int64]*ResidualEdge),
		IncomingEdgesListCache: make(map[int64][]IncomingEdge),
		incomingCacheDirty:     true,
		sortedNodesDirty:       true,
	}
}

// =============================================================================
// Graph Modification
// =============================================================================

// Clear removes all nodes and edges from the graph.
//
// The graph can be reused after clearing. This is more efficient than
// creating a new graph when using pooling.
func (rg *ResidualGraph) Clear() {
	clear(rg.Nodes)
	for k := range rg.Edges {
		clear(rg.Edges[k])
		delete(rg.Edges, k)
	}
	for k := range rg.EdgesList {
		rg.EdgesList[k] = rg.EdgesList[k][:0]
		delete(rg.EdgesList, k)
	}
	for k := range rg.ReverseEdges {
		clear(rg.ReverseEdges[k])
		delete(rg.ReverseEdges, k)
	}
	for k := range rg.IncomingEdgesListCache {
		delete(rg.IncomingEdgesListCache, k)
	}
	rg.incomingCacheDirty = true

	rg.sortedNodesMu.Lock()
	rg.sortedNodes = rg.sortedNodes[:0]
	rg.sortedNodesDirty = true
	rg.sortedNodesMu.Unlock()
}

// AddNode adds a node to the graph.
//
// If the node already exists, this is a no-op.
// Nodes are added implicitly when adding edges, but explicit addition
// is useful for isolated nodes or pre-allocation.
func (rg *ResidualGraph) AddNode(id int64) {
	if !rg.Nodes[id] {
		rg.Nodes[id] = true
		rg.markSortedNodesDirty()
	}
}

// ensureNode adds a node if it doesn't exist (internal helper).
func (rg *ResidualGraph) ensureNode(id int64) {
	if !rg.Nodes[id] {
		rg.Nodes[id] = true
		rg.markSortedNodesDirty()
	}
}

// markSortedNodesDirty marks the sorted nodes cache as dirty.
// Thread-safe helper method.
func (rg *ResidualGraph) markSortedNodesDirty() {
	rg.sortedNodesMu.Lock()
	rg.sortedNodesDirty = true
	rg.sortedNodesMu.Unlock()
}

// invalidateIncomingCache invalidates the incoming edges cache.
func (rg *ResidualGraph) invalidateIncomingCache() {
	rg.incomingCacheDirty = true
}

// AddEdge adds a forward edge to the graph.
//
// If an edge already exists between the same nodes:
//   - If the existing edge is a reverse edge, it's converted to a forward edge
//   - Otherwise, the capacity is accumulated
//
// For most use cases, prefer AddEdgeWithReverse() which handles both directions.
//
// Parameters:
//   - from: Source node ID
//   - to: Destination node ID
//   - capacity: Maximum flow capacity
//   - cost: Cost per unit of flow
func (rg *ResidualGraph) AddEdge(from, to int64, capacity, cost float64) {
	rg.ensureNode(from)
	rg.ensureNode(to)

	if rg.Edges[from] == nil {
		rg.Edges[from] = make(map[int64]*ResidualEdge)
	}

	if existing := rg.Edges[from][to]; existing != nil {
		if existing.IsReverse {
			// Convert reverse edge to forward edge
			// This happens when the reverse edge was created first
			existing.OriginalCapacity = capacity
			existing.Capacity = capacity
			existing.Cost = cost
			existing.IsReverse = false
			return
		}
		// Accumulate capacity for parallel edges
		existing.Capacity += capacity
		existing.OriginalCapacity += capacity
		return
	}

	// Create new forward edge
	edge := &ResidualEdge{
		To:               to,
		Capacity:         capacity,
		Cost:             cost,
		Flow:             0,
		OriginalCapacity: capacity,
		IsReverse:        false,
		Index:            len(rg.EdgesList[from]),
	}

	rg.Edges[from][to] = edge
	rg.EdgesList[from] = append(rg.EdgesList[from], edge)
	rg.addReverseIndex(from, to, edge)
}

// AddReverseEdge adds a backward edge for flow cancellation.
//
// Reverse edges have:
//   - Initial capacity of 0 (increases as flow is pushed)
//   - Negative cost (pushing flow backward refunds the cost)
//
// This is typically called internally by AddEdgeWithReverse().
func (rg *ResidualGraph) AddReverseEdge(from, to int64, cost float64) {
	rg.ensureNode(from)
	rg.ensureNode(to)

	if rg.Edges[from] == nil {
		rg.Edges[from] = make(map[int64]*ResidualEdge)
	}

	// Don't overwrite existing edge
	if existing := rg.Edges[from][to]; existing != nil {
		return
	}

	edge := &ResidualEdge{
		To:               to,
		Capacity:         0,
		Cost:             -cost, // Negative cost for refund
		Flow:             0,
		OriginalCapacity: 0,
		IsReverse:        true,
		Index:            len(rg.EdgesList[from]),
	}

	if rg.ReverseEdges[to] == nil {
		rg.ReverseEdges[to] = make(map[int64]*ResidualEdge)
	}
	rg.ReverseEdges[to][from] = edge
	rg.incomingCacheDirty = true

	rg.Edges[from][to] = edge
	rg.EdgesList[from] = append(rg.EdgesList[from], edge)
	rg.addReverseIndex(from, to, edge)
}

// addReverseIndex adds an entry to the reverse edge index (internal helper).
func (rg *ResidualGraph) addReverseIndex(from, to int64, edge *ResidualEdge) {
	if rg.ReverseEdges[to] == nil {
		rg.ReverseEdges[to] = make(map[int64]*ResidualEdge)
	}
	rg.ReverseEdges[to][from] = edge
}

// AddEdgeWithReverse adds both forward and backward edges.
//
// This is the recommended method for adding edges to a flow network.
// It creates:
//   - Forward edge (from → to) with the specified capacity and cost
//   - Backward edge (to → from) with 0 capacity and negative cost
//
// Parameters:
//   - from: Source node ID
//   - to: Destination node ID
//   - capacity: Maximum flow capacity
//   - cost: Cost per unit of flow
//
// Example:
//
//	g.AddEdgeWithReverse(1, 2, 10, 1.5)
//	// Creates edge 1→2 with capacity=10, cost=1.5
//	// Creates edge 2→1 with capacity=0, cost=-1.5
func (rg *ResidualGraph) AddEdgeWithReverse(from, to int64, capacity, cost float64) {
	rg.AddEdge(from, to, capacity, cost)
	rg.AddReverseEdge(to, from, cost)
}

// =============================================================================
// Edge Access
// =============================================================================

// GetEdge returns the edge from 'from' to 'to', or nil if not found.
//
// Time complexity: O(1)
func (rg *ResidualGraph) GetEdge(from, to int64) *ResidualEdge {
	if rg.Edges[from] == nil {
		return nil
	}
	return rg.Edges[from][to]
}

// GetNeighbors returns all outgoing edges from a node as a map.
//
// WARNING: Iterating over the returned map is non-deterministic.
// Use GetNeighborsList() for deterministic iteration.
//
// Time complexity: O(1)
func (rg *ResidualGraph) GetNeighbors(node int64) map[int64]*ResidualEdge {
	return rg.Edges[node]
}

// GetNeighborsList returns all outgoing edges from a node as a slice.
//
// The slice is in insertion order, providing deterministic iteration.
// This should be used in algorithms to ensure reproducible results.
//
// Time complexity: O(1)
func (rg *ResidualGraph) GetNeighborsList(node int64) []*ResidualEdge {
	return rg.EdgesList[node]
}

// GetIncomingEdges returns all incoming edges to a node as a map.
//
// WARNING: Iterating over the returned map is non-deterministic.
// Use GetIncomingEdgesList() for deterministic iteration.
//
// Time complexity: O(1)
func (rg *ResidualGraph) GetIncomingEdges(to int64) map[int64]*ResidualEdge {
	return rg.ReverseEdges[to]
}

// GetIncomingEdgesList returns all incoming edges to a node in deterministic order.
//
// The returned slice is sorted by source node ID for reproducibility.
//
// Time complexity: O(in-degree × log(in-degree)) for sorting
func (rg *ResidualGraph) GetIncomingEdgesList(to int64) []IncomingEdge {
	incoming := rg.ReverseEdges[to]
	if incoming == nil {
		return nil
	}

	result := make([]IncomingEdge, 0, len(incoming))
	for from, edge := range incoming {
		result = append(result, IncomingEdge{From: from, Edge: edge})
	}

	// Sort by source node ID for determinism
	sort.Slice(result, func(i, j int) bool {
		return result[i].From < result[j].From
	})

	return result
}

// GetEdgesFrom returns all edges originating from a node.
// Equivalent to GetNeighborsList().
func (rg *ResidualGraph) GetEdgesFrom(from int64) []*ResidualEdge {
	return rg.EdgesList[from]
}

// =============================================================================
// Node Access
// =============================================================================

// GetNodes returns all node IDs in deterministic (sorted) order.
//
// This is equivalent to GetSortedNodes() and should be used for
// deterministic iteration over nodes.
func (rg *ResidualGraph) GetNodes() []int64 {
	return rg.GetSortedNodes()
}

// GetSortedNodes returns node IDs sorted in ascending order.
//
// The result is cached for efficiency. The cache is invalidated when
// nodes are added. This method is safe for concurrent use.
//
// Time complexity: O(1) if cached, O(n log n) otherwise
func (rg *ResidualGraph) GetSortedNodes() []int64 {
	rg.sortedNodesMu.Lock()
	defer rg.sortedNodesMu.Unlock()

	if rg.sortedNodesDirty || len(rg.sortedNodes) != len(rg.Nodes) {
		rg.sortedNodes = make([]int64, 0, len(rg.Nodes))
		for node := range rg.Nodes {
			rg.sortedNodes = append(rg.sortedNodes, node)
		}
		sort.Slice(rg.sortedNodes, func(i, j int) bool {
			return rg.sortedNodes[i] < rg.sortedNodes[j]
		})
		rg.sortedNodesDirty = false
	}

	return rg.sortedNodes
}

// NodeCount returns the number of nodes in the graph.
func (rg *ResidualGraph) NodeCount() int {
	return len(rg.Nodes)
}

// EdgeCount returns the total number of edges (including reverse edges).
func (rg *ResidualGraph) EdgeCount() int {
	count := 0
	for _, edges := range rg.EdgesList {
		count += len(edges)
	}
	return count
}

// =============================================================================
// Flow Operations
// =============================================================================

// UpdateFlow pushes flow along an edge and updates the residual graph.
//
// This operation:
//   - Decreases the forward edge capacity by 'flow'
//   - Increases the forward edge flow by 'flow'
//   - Increases the backward edge capacity by 'flow'
//
// The backward edge is created if it doesn't exist.
//
// Parameters:
//   - from: Source node of the edge
//   - to: Destination node of the edge
//   - flow: Amount of flow to push (must be positive)
func (rg *ResidualGraph) UpdateFlow(from, to int64, flow float64) {
	// Update forward edge
	if edge := rg.GetEdge(from, to); edge != nil {
		edge.Flow += flow
		edge.Capacity -= flow
	}

	// Update or create backward edge
	if backEdge := rg.GetEdge(to, from); backEdge != nil {
		backEdge.Capacity += flow
	} else {
		// Create backward edge if it doesn't exist
		if rg.Edges[to] == nil {
			rg.Edges[to] = make(map[int64]*ResidualEdge)
		}
		forwardEdge := rg.GetEdge(from, to)
		cost := 0.0
		if forwardEdge != nil {
			cost = -forwardEdge.Cost
		}
		newEdge := &ResidualEdge{
			To:               from,
			Capacity:         flow,
			Cost:             cost,
			Flow:             0,
			OriginalCapacity: 0,
			IsReverse:        true,
			Index:            len(rg.EdgesList[to]),
		}
		rg.Edges[to][from] = newEdge
		rg.EdgesList[to] = append(rg.EdgesList[to], newEdge)
		rg.addReverseIndex(to, from, newEdge)
	}
}

// GetFlowOnEdge returns the current flow on an edge.
//
// Returns 0 if the edge doesn't exist.
func (rg *ResidualGraph) GetFlowOnEdge(from, to int64) float64 {
	if edge := rg.GetEdge(from, to); edge != nil {
		return edge.Flow
	}
	return 0
}

// GetTotalFlow computes the total flow leaving the source node.
//
// This is the standard way to determine the flow value after running
// a max-flow algorithm.
func (rg *ResidualGraph) GetTotalFlow(source int64) float64 {
	totalFlow := 0.0
	for _, edge := range rg.EdgesList[source] {
		if !edge.IsReverse && edge.Flow > 0 {
			totalFlow += edge.Flow
		}
	}
	return totalFlow
}

// GetTotalCost computes the total cost of all flow in the graph.
//
// Only forward edges with positive flow contribute to the cost.
// The result is deterministic due to sorted node iteration.
func (rg *ResidualGraph) GetTotalCost() float64 {
	totalCost := 0.0
	nodes := rg.GetSortedNodes()
	for _, from := range nodes {
		for _, edge := range rg.EdgesList[from] {
			if !edge.IsReverse && edge.Flow > 0 {
				totalCost += edge.Flow * edge.Cost
			}
		}
	}
	return totalCost
}

// =============================================================================
// Graph Operations
// =============================================================================

// Clone creates a deep copy of the graph.
//
// The cloned graph is completely independent and can be modified
// without affecting the original.
//
// Use CloneToPooled() for better performance when using pooling.
func (rg *ResidualGraph) Clone() *ResidualGraph {
	clone := NewResidualGraph()

	for node := range rg.Nodes {
		clone.Nodes[node] = true
	}

	for from, edges := range rg.EdgesList {
		clone.Edges[from] = make(map[int64]*ResidualEdge, len(edges))
		clone.EdgesList[from] = make([]*ResidualEdge, len(edges))

		for i, edge := range edges {
			clonedEdge := &ResidualEdge{
				To:               edge.To,
				Capacity:         edge.Capacity,
				Cost:             edge.Cost,
				Flow:             edge.Flow,
				OriginalCapacity: edge.OriginalCapacity,
				IsReverse:        edge.IsReverse,
				Index:            edge.Index,
			}
			clone.Edges[from][edge.To] = clonedEdge
			clone.EdgesList[from][i] = clonedEdge
			clone.addReverseIndex(from, edge.To, clonedEdge)
		}
	}

	clone.sortedNodesDirty = true
	return clone
}

// CloneToPooled creates a deep copy using a graph from the pool.
//
// This is more efficient than Clone() when the pool has available graphs.
// The caller is responsible for returning the cloned graph to the pool
// when done.
//
// Example:
//
//	pool := graph.GetPool()
//	cloned := g.CloneToPooled(pool)
//	defer pool.ReleaseGraph(cloned)
//	// ... use cloned ...
func (rg *ResidualGraph) CloneToPooled(pool *GraphPool) *ResidualGraph {
	clone := pool.AcquireGraph()

	for node := range rg.Nodes {
		clone.Nodes[node] = true
	}

	for from, edges := range rg.EdgesList {
		clone.Edges[from] = make(map[int64]*ResidualEdge, len(edges))
		clone.EdgesList[from] = make([]*ResidualEdge, 0, len(edges))

		for _, edge := range edges {
			clonedEdge := &ResidualEdge{
				To:               edge.To,
				Capacity:         edge.Capacity,
				Cost:             edge.Cost,
				Flow:             edge.Flow,
				OriginalCapacity: edge.OriginalCapacity,
				IsReverse:        edge.IsReverse,
				Index:            len(clone.EdgesList[from]),
			}
			clone.Edges[from][edge.To] = clonedEdge
			clone.EdgesList[from] = append(clone.EdgesList[from], clonedEdge)
			clone.addReverseIndex(from, edge.To, clonedEdge)
		}
	}

	clone.sortedNodesDirty = true
	return clone
}

// Reset clears all flow and restores original capacities.
//
// This allows rerunning algorithms on the same graph structure
// without recreating it.
func (rg *ResidualGraph) Reset() {
	for _, edges := range rg.EdgesList {
		for _, edge := range edges {
			if edge.IsReverse {
				edge.Capacity = 0
			} else {
				edge.Capacity = edge.OriginalCapacity
			}
			edge.Flow = 0
		}
	}
}

// GetAllEdges returns all forward (non-reverse) edges in deterministic order.
//
// Useful for exporting graph structure or computing statistics.
func (rg *ResidualGraph) GetAllEdges() []*ResidualEdge {
	var result []*ResidualEdge
	nodes := rg.GetSortedNodes()
	for _, from := range nodes {
		for _, edge := range rg.EdgesList[from] {
			if !edge.IsReverse {
				result = append(result, edge)
			}
		}
	}
	return result
}

// =============================================================================
// Thread-Safe Wrapper
// =============================================================================

// SafeResidualGraph provides thread-safe access to a ResidualGraph.
//
// Use this when the graph needs to be accessed from multiple goroutines.
// Write operations use an exclusive lock, read operations use a shared lock.
//
// For compute-intensive algorithms, it's usually better to clone the graph
// and work on the clone without locking.
//
// # Example
//
//	safe := NewSafeResidualGraph()
//	safe.WithWriteLock(func(g *ResidualGraph) {
//	    g.AddEdgeWithReverse(1, 2, 10, 0)
//	})
//
//	var maxFlow float64
//	safe.WithReadLock(func(g *ResidualGraph) {
//	    maxFlow = g.GetTotalFlow(1)
//	})
type SafeResidualGraph struct {
	mu    sync.RWMutex
	graph *ResidualGraph
}

// NewSafeResidualGraph creates a new thread-safe graph wrapper.
func NewSafeResidualGraph() *SafeResidualGraph {
	return &SafeResidualGraph{
		graph: NewResidualGraph(),
	}
}

// WithReadLock executes a function with read lock held.
//
// Multiple goroutines can hold read locks simultaneously.
// The function must not modify the graph.
func (sg *SafeResidualGraph) WithReadLock(fn func(*ResidualGraph)) {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	fn(sg.graph)
}

// WithWriteLock executes a function with exclusive write lock held.
//
// Only one goroutine can hold the write lock at a time.
func (sg *SafeResidualGraph) WithWriteLock(fn func(*ResidualGraph)) {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	fn(sg.graph)
}

// CloneUnsafe returns a deep copy for local work.
//
// The returned graph is independent and can be used without locking.
// This is the recommended pattern for running algorithms concurrently.
func (sg *SafeResidualGraph) CloneUnsafe() *ResidualGraph {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return sg.graph.Clone()
}

// ClonePooled returns a deep copy from the pool.
//
// More efficient than CloneUnsafe() when using pooling.
// The caller must return the graph to the pool when done.
func (sg *SafeResidualGraph) ClonePooled(pool *GraphPool) *ResidualGraph {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return sg.graph.CloneToPooled(pool)
}

// BuildIncomingEdgesCache clears the old cache and builds a new one.
func (rg *ResidualGraph) BuildIncomingEdgesCache() {
	if !rg.incomingCacheDirty {
		return
	}

	// Clear old cache
	for k := range rg.IncomingEdgesListCache {
		delete(rg.IncomingEdgesListCache, k)
	}

	// Build new cache
	for to, incoming := range rg.ReverseEdges {
		if len(incoming) == 0 {
			continue
		}

		list := make([]IncomingEdge, 0, len(incoming))
		for from, edge := range incoming {
			list = append(list, IncomingEdge{From: from, Edge: edge})
		}

		// Sort for determinism
		sort.Slice(list, func(i, j int) bool {
			return list[i].From < list[j].From
		})

		rg.IncomingEdgesListCache[to] = list
	}

	rg.incomingCacheDirty = false
}

// GetIncomingEdgesListCached returns a cached list of incoming edges
func (rg *ResidualGraph) GetIncomingEdgesListCached(to int64) []IncomingEdge {
	if rg.incomingCacheDirty {
		rg.BuildIncomingEdgesCache()
	}
	return rg.IncomingEdgesListCache[to]
}
