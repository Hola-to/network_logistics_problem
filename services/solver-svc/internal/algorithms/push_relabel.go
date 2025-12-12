package algorithms

import (
	"container/heap"
	"context"

	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Push-Relabel Algorithm (Preflow-Push)
// =============================================================================
//
// The Push-Relabel algorithm (also known as Preflow-Push) computes maximum flow
// by maintaining a preflow and gradually converting it to a valid flow.
// Unlike augmenting path methods, it works locally on vertices.
//
// Time Complexity:
//   - FIFO variant: O(V³)
//   - Highest Label variant: O(V² √E)
//   - Lowest Label variant: O(V² E)
//
// Space Complexity: O(V + E)
//
// Key Concepts:
//   - Preflow: Allows excess flow at vertices (more inflow than outflow)
//   - Height function: Labels vertices; flow only goes from higher to lower
//   - Push: Moves excess flow to lower neighbors
//   - Relabel: Increases height when no valid push is possible
//
// Optimizations Implemented:
//   - Gap heuristic: When a height has no vertices, cut off higher vertices
//   - Global relabeling: Periodically recompute heights via BFS from sink
//   - Current arc optimization: Skip recently saturated edges
//   - Highest/Lowest label selection: Better vertex selection strategies
//
// This implementation provides three variants:
//   - PushRelabel: FIFO vertex selection
//   - PushRelabelHighestLabel: Highest active vertex first
//   - PushRelabelLowestLabel: Lowest active vertex first
//
// References:
//   - Goldberg, A.V. & Tarjan, R.E. (1988). "A new approach to the maximum-flow problem"
//   - Cherkassky, B.V. & Goldberg, A.V. (1997). "On implementing push-relabel method
//     for the maximum flow problem"
// =============================================================================

// PushRelabelResult contains the result of the Push-Relabel algorithm.
type PushRelabelResult struct {
	// MaxFlow is the maximum flow value computed.
	MaxFlow float64

	// Iterations is the number of push/relabel operations performed.
	Iterations int

	// Canceled indicates whether the operation was canceled via context.
	Canceled bool
}

// =============================================================================
// Data Structures
// =============================================================================

// prNode represents a vertex entry in the priority queue with versioning
// to handle stale entries efficiently.
type prNode struct {
	id      int64
	height  int
	version int
}

// maxHeap implements a max-heap for Highest Label vertex selection.
// Uses versioning to efficiently handle height updates without removal.
type maxHeap struct {
	items       []prNode
	nodeVersion map[int64]int // Current version of each node
}

func newMaxHeap(capacity int) *maxHeap {
	return &maxHeap{
		items:       make([]prNode, 0, capacity),
		nodeVersion: make(map[int64]int, capacity),
	}
}

func (h *maxHeap) Len() int           { return len(h.items) }
func (h *maxHeap) Less(i, j int) bool { return h.items[i].height > h.items[j].height }
func (h *maxHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *maxHeap) Push(x any) {
	h.items = append(h.items, x.(prNode))
}

func (h *maxHeap) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}

// push adds a vertex to the heap with a new version.
func (h *maxHeap) push(id int64, height int) {
	h.nodeVersion[id]++
	heap.Push(h, prNode{id: id, height: height, version: h.nodeVersion[id]})
}

// pop extracts a vertex, skipping stale entries.
func (h *maxHeap) pop() (int64, bool) {
	for h.Len() > 0 {
		item := heap.Pop(h).(prNode)
		// Check if this entry is still valid
		if item.version == h.nodeVersion[item.id] {
			return item.id, true
		}
		// Stale entry - skip
	}
	return 0, false
}

// =============================================================================
// Algorithm State
// =============================================================================

// pushRelabelState holds the mutable state during algorithm execution.
type pushRelabelState struct {
	g       *graph.ResidualGraph
	source  int64
	sink    int64
	n       int
	nodes   []int64       // Deterministic node ordering
	nodeIdx map[int64]int // Node ID to index mapping

	height      []int     // height[nodeIdx[v]]
	excess      []float64 // excess[nodeIdx[v]]
	heightCount []int     // Number of vertices at each height

	maxHeight int
	epsilon   float64

	// Current arc optimization
	currentArc []int // Index of next edge to try for each vertex
}

// newPushRelabelState initializes the algorithm state.
func newPushRelabelState(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *pushRelabelState {
	nodes := g.GetSortedNodes()
	n := len(nodes)

	nodeIdx := make(map[int64]int, n)
	for i, node := range nodes {
		nodeIdx[node] = i
	}

	return &pushRelabelState{
		g:           g,
		source:      source,
		sink:        sink,
		n:           n,
		nodes:       nodes,
		nodeIdx:     nodeIdx,
		height:      make([]int, n),
		excess:      make([]float64, n),
		heightCount: make([]int, 2*n+1),
		currentArc:  make([]int, n),
		maxHeight:   2*n - 1,
		epsilon:     options.Epsilon,
	}
}

// Height/excess accessors for cleaner code
func (s *pushRelabelState) getHeight(node int64) int            { return s.height[s.nodeIdx[node]] }
func (s *pushRelabelState) setHeight(node int64, h int)         { s.height[s.nodeIdx[node]] = h }
func (s *pushRelabelState) getExcess(node int64) float64        { return s.excess[s.nodeIdx[node]] }
func (s *pushRelabelState) addExcess(node int64, delta float64) { s.excess[s.nodeIdx[node]] += delta }
func (s *pushRelabelState) getCurrentArc(node int64) int        { return s.currentArc[s.nodeIdx[node]] }
func (s *pushRelabelState) setCurrentArc(node int64, arc int)   { s.currentArc[s.nodeIdx[node]] = arc }

// =============================================================================
// Push-Relabel FIFO Variant
// =============================================================================

// PushRelabel executes the Push-Relabel algorithm with FIFO vertex selection.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *PushRelabelResult containing max flow and iteration count
func PushRelabel(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	return PushRelabelWithContext(context.Background(), g, source, sink, options)
}

// PushRelabelWithContext executes Push-Relabel with context cancellation.
func PushRelabelWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	if len(g.Nodes) == 0 {
		return &PushRelabelResult{MaxFlow: 0, Iterations: 0}
	}

	state := newPushRelabelState(g, source, sink, options)
	state.initialize()

	// FIFO queue of active vertices
	queue := make([]int64, 0, state.n)
	inQueue := make(map[int64]bool, state.n)

	// Add initially active vertices
	for _, node := range state.nodes {
		if node != source && node != sink && state.getExcess(node) > state.epsilon {
			queue = append(queue, node)
			inQueue[node] = true
		}
	}

	iterations := 0
	globalRelabelFreq := state.n
	const checkInterval = 100

	for len(queue) > 0 {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		// Context check
		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &PushRelabelResult{
					MaxFlow:    state.getExcess(sink),
					Iterations: iterations,
					Canceled:   true,
				}
			default:
			}
		}

		// Periodic global relabeling
		if iterations > 0 && iterations%globalRelabelFreq == 0 {
			state.globalRelabel()
			// Rebuild queue
			queue = queue[:0]
			for k := range inQueue {
				delete(inQueue, k)
			}
			for _, node := range state.nodes {
				if node != source && node != sink {
					if state.getExcess(node) > state.epsilon && state.getHeight(node) <= state.maxHeight {
						queue = append(queue, node)
						inQueue[node] = true
					}
				}
			}
			// Check if queue became empty after rebuild
			if len(queue) == 0 {
				break
			}
		}

		// Pop vertex from queue
		u := queue[0]
		queue = queue[1:]
		delete(inQueue, u)

		// Discharge the vertex
		state.discharge(u, func(v int64) {
			if v != source && v != sink && !inQueue[v] && state.getExcess(v) > state.epsilon {
				queue = append(queue, v)
				inQueue[v] = true
			}
		})

		// Re-add if still active
		if state.getExcess(u) > state.epsilon && state.getHeight(u) <= state.maxHeight {
			if !inQueue[u] {
				queue = append(queue, u)
				inQueue[u] = true
			}
		}

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    state.getExcess(sink),
		Iterations: iterations,
		Canceled:   false,
	}
}

// =============================================================================
// Push-Relabel Highest Label Variant
// =============================================================================

// PushRelabelHighestLabel executes Push-Relabel with Highest Label selection.
// This variant achieves O(V² √E) time complexity.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *PushRelabelResult containing max flow and iteration count
func PushRelabelHighestLabel(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	return PushRelabelHighestLabelWithContext(context.Background(), g, source, sink, options)
}

// PushRelabelHighestLabelWithContext is the context-aware Highest Label variant.
func PushRelabelHighestLabelWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	if len(g.Nodes) == 0 {
		return &PushRelabelResult{MaxFlow: 0, Iterations: 0}
	}

	state := newPushRelabelState(g, source, sink, options)
	state.initialize()

	// Max-heap with versioning for efficient updates
	activeHeap := newMaxHeap(state.n)

	// Add initially active vertices
	for _, node := range state.nodes {
		if node != source && node != sink && state.getExcess(node) > state.epsilon {
			activeHeap.push(node, state.getHeight(node))
		}
	}

	iterations := 0
	globalRelabelFreq := state.n
	const checkInterval = 100

	for activeHeap.Len() > 0 {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &PushRelabelResult{
					MaxFlow:    state.getExcess(sink),
					Iterations: iterations,
					Canceled:   true,
				}
			default:
			}
			if activeHeap.Len() == 0 {
				break
			}
		}

		// Periodic global relabeling
		if iterations > 0 && iterations%globalRelabelFreq == 0 {
			state.globalRelabel()
			// Rebuild heap
			activeHeap = newMaxHeap(state.n)
			for _, node := range state.nodes {
				if node != source && node != sink {
					if state.getExcess(node) > state.epsilon && state.getHeight(node) <= state.maxHeight {
						activeHeap.push(node, state.getHeight(node))
					}
				}
			}
		}

		// Get highest active vertex
		u, ok := activeHeap.pop()
		if !ok {
			break
		}

		// Skip if no longer active
		if state.getExcess(u) <= state.epsilon || state.getHeight(u) > state.maxHeight {
			continue
		}

		// Discharge
		state.discharge(u, func(v int64) {
			if v != source && v != sink && state.getExcess(v) > state.epsilon {
				activeHeap.push(v, state.getHeight(v))
			}
		})

		// Re-add if still active
		if state.getExcess(u) > state.epsilon && state.getHeight(u) <= state.maxHeight {
			activeHeap.push(u, state.getHeight(u))
		}

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    state.getExcess(sink),
		Iterations: iterations,
		Canceled:   false,
	}
}

// =============================================================================
// Push-Relabel Lowest Label Variant
// =============================================================================

// PushRelabelLowestLabel executes Push-Relabel with Lowest Label selection.
//
// Parameters:
//   - g: The residual graph (will be modified)
//   - source: The source node ID
//   - sink: The sink node ID
//   - options: Solver options
//
// Returns:
//   - *PushRelabelResult containing max flow and iteration count
func PushRelabelLowestLabel(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	return PushRelabelLowestLabelWithContext(context.Background(), g, source, sink, options)
}

// PushRelabelLowestLabelWithContext is the context-aware Lowest Label variant.
func PushRelabelLowestLabelWithContext(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	if len(g.Nodes) == 0 {
		return &PushRelabelResult{MaxFlow: 0, Iterations: 0}
	}

	state := newPushRelabelState(g, source, sink, options)
	state.initialize()

	// Bucket-based structure for each height level
	buckets := make([][]int64, 2*state.n+1)
	for i := range buckets {
		buckets[i] = make([]int64, 0)
	}

	inBucket := make(map[int64]bool, state.n)
	minActiveHeight := state.maxHeight + 1

	// Initialize buckets
	for _, node := range state.nodes {
		if node != source && node != sink {
			if state.getExcess(node) > state.epsilon && state.getHeight(node) <= state.maxHeight {
				h := state.getHeight(node)
				buckets[h] = append(buckets[h], node)
				inBucket[node] = true
				if h < minActiveHeight {
					minActiveHeight = h
				}
			}
		}
	}

	iterations := 0
	globalRelabelFreq := state.n
	const checkInterval = 100

	for minActiveHeight <= state.maxHeight {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &PushRelabelResult{
					MaxFlow:    state.getExcess(sink),
					Iterations: iterations,
					Canceled:   true,
				}
			default:
			}
		}

		// Periodic global relabeling
		if iterations > 0 && iterations%globalRelabelFreq == 0 {
			state.globalRelabel()
			// Rebuild buckets
			for i := range buckets {
				buckets[i] = buckets[i][:0]
			}
			if minActiveHeight > state.maxHeight {
				break
			}
			for k := range inBucket {
				delete(inBucket, k)
			}
			minActiveHeight = state.maxHeight + 1

			for _, node := range state.nodes {
				if node != source && node != sink {
					if state.getExcess(node) > state.epsilon && state.getHeight(node) <= state.maxHeight {
						h := state.getHeight(node)
						buckets[h] = append(buckets[h], node)
						inBucket[node] = true
						if h < minActiveHeight {
							minActiveHeight = h
						}
					}
				}
			}
		}

		// Find non-empty bucket with minimum height
		for minActiveHeight <= state.maxHeight && len(buckets[minActiveHeight]) == 0 {
			minActiveHeight++
		}

		if minActiveHeight > state.maxHeight {
			break
		}

		// Extract vertex from bucket
		bucket := buckets[minActiveHeight]
		u := bucket[len(bucket)-1]
		buckets[minActiveHeight] = bucket[:len(bucket)-1]
		delete(inBucket, u)

		// Skip if no longer valid
		if state.getExcess(u) <= state.epsilon || state.getHeight(u) != minActiveHeight {
			continue
		}

		// Discharge
		state.discharge(u, func(v int64) {
			if v != source && v != sink && !inBucket[v] {
				if state.getExcess(v) > state.epsilon && state.getHeight(v) <= state.maxHeight {
					h := state.getHeight(v)
					buckets[h] = append(buckets[h], v)
					inBucket[v] = true
					if h < minActiveHeight {
						minActiveHeight = h
					}
				}
			}
		})

		// Re-add if still active
		if state.getExcess(u) > state.epsilon && state.getHeight(u) <= state.maxHeight {
			if !inBucket[u] {
				h := state.getHeight(u)
				buckets[h] = append(buckets[h], u)
				inBucket[u] = true
			}
		}

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    state.getExcess(sink),
		Iterations: iterations,
		Canceled:   false,
	}
}

// =============================================================================
// Core Operations
// =============================================================================

// initialize sets up the initial preflow and heights.
func (s *pushRelabelState) initialize() {
	// Initialize heights: source = n, others = 0
	for i := range s.height {
		s.height[i] = 0
	}
	s.setHeight(s.source, s.n)

	// Count vertices at each height
	for i := range s.heightCount {
		s.heightCount[i] = 0
	}
	for _, node := range s.nodes {
		h := s.getHeight(node)
		if h <= s.maxHeight {
			s.heightCount[h]++
		}
	}

	// Initial push from source: saturate all outgoing edges
	edges := s.g.GetNeighborsList(s.source)
	for _, edge := range edges {
		if edge.Capacity > s.epsilon {
			flow := edge.Capacity
			s.g.UpdateFlow(s.source, edge.To, flow)
			s.addExcess(edge.To, flow)
			s.addExcess(s.source, -flow)
		}
	}

	// Global relabel for accurate initial heights
	s.globalRelabel()
}

// globalRelabel recomputes heights using reverse BFS from sink.
// Uses deterministic ordering for reproducible results.
//
// The algorithm traverses the graph in reverse: for each node u, we find
// all nodes v that have an edge TO u with positive residual capacity.
// This means v can push flow to u, so height[v] = height[u] + 1.
//
// Implementation Note:
// GetIncomingEdgesList(u) returns edges where the edge goes FROM some node TO u.
// The Edge.Capacity is the capacity of that forward edge (from -> u).
func (s *pushRelabelState) globalRelabel() {
	// Reset height counts
	for i := range s.heightCount {
		s.heightCount[i] = 0
	}

	// Initialize new heights to maxHeight + 1 (unreachable)
	newHeight := make([]int, s.n)
	for i := range newHeight {
		newHeight[i] = s.maxHeight + 1
	}
	newHeight[s.nodeIdx[s.sink]] = 0

	// BFS from sink using reverse edges
	queue := make([]int64, 0, s.n)
	queue = append(queue, s.sink)
	head := 0

	for head < len(queue) {
		u := queue[head]
		head++

		uHeight := newHeight[s.nodeIdx[u]]

		// Get all edges that point TO u (i.e., edges v -> u)
		// For each such edge, if it has capacity, then v can reach u
		incomingList := s.g.GetIncomingEdgesList(u)
		for _, incoming := range incomingList {
			v := incoming.From
			vIdx := s.nodeIdx[v]

			// incoming.Edge represents the edge v -> u
			// If this edge has capacity, v can push flow to u
			if newHeight[vIdx] > s.maxHeight && incoming.Edge.Capacity > s.epsilon {
				newHeight[vIdx] = uHeight + 1
				queue = append(queue, v)
			}
		}
	}

	// Source always has height n
	newHeight[s.nodeIdx[s.source]] = s.n

	// Apply new heights
	for i, h := range newHeight {
		s.height[i] = h
		if h <= s.maxHeight {
			s.heightCount[h]++
		}
	}

	// Reset current arcs
	for i := range s.currentArc {
		s.currentArc[i] = 0
	}
}

// discharge processes a vertex until it has no excess or cannot push.
func (s *pushRelabelState) discharge(u int64, onActivate func(int64)) {
	edges := s.g.GetNeighborsList(u)
	if edges == nil {
		return
	}

	for s.getExcess(u) > s.epsilon && s.getHeight(u) <= s.maxHeight {
		currentArc := s.getCurrentArc(u)

		if currentArc >= len(edges) {
			if !s.relabel(u) {
				break
			}
			s.setCurrentArc(u, 0)
			continue
		}

		edge := edges[currentArc]
		v := edge.To

		if edge.Capacity > s.epsilon && s.getHeight(u) == s.getHeight(v)+1 {
			delta := min(s.getExcess(u), edge.Capacity)
			s.g.UpdateFlow(u, v, delta)
			s.addExcess(u, -delta)
			s.addExcess(v, delta)

			if onActivate != nil {
				onActivate(v)
			}
		} else {
			s.setCurrentArc(u, currentArc+1)
		}
	}
}

// relabel increases the height of a vertex.
func (s *pushRelabelState) relabel(u int64) bool {
	oldHeight := s.getHeight(u)
	if oldHeight > s.maxHeight {
		return false
	}

	edges := s.g.GetNeighborsList(u)
	if edges == nil {
		s.heightCount[oldHeight]--
		s.setHeight(u, s.maxHeight+1)
		return false
	}

	// Find minimum height among neighbors with residual capacity
	minHeight := s.maxHeight + 1
	for _, edge := range edges {
		if edge.Capacity > s.epsilon {
			h := s.getHeight(edge.To)
			if h < minHeight {
				minHeight = h
			}
		}
	}

	if minHeight >= s.maxHeight {
		s.heightCount[oldHeight]--
		s.setHeight(u, s.maxHeight+1)
		return false
	}

	newHeight := minHeight + 1
	if newHeight > s.maxHeight {
		s.heightCount[oldHeight]--
		s.setHeight(u, s.maxHeight+1)
		return false
	}

	// Gap heuristic: if this height becomes empty, vertices above are unreachable
	s.heightCount[oldHeight]--
	if s.heightCount[oldHeight] == 0 && oldHeight < s.n {
		s.applyGapHeuristic(oldHeight)
	}

	s.heightCount[newHeight]++
	s.setHeight(u, newHeight)

	return true
}

// applyGapHeuristic raises all vertices above the gap to maxHeight + 1.
func (s *pushRelabelState) applyGapHeuristic(gapHeight int) {
	for i, node := range s.nodes {
		h := s.height[i]
		if h > gapHeight && h <= s.maxHeight && node != s.source {
			s.heightCount[h]--
			s.height[i] = s.maxHeight + 1
		}
	}
}
