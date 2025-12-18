package algorithms

import (
	"context"

	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Push-Relabel Algorithm (Preflow-Push) - Optimized Implementation
// =============================================================================
//
// This implementation includes several optimizations:
//   - Index-based operations instead of node ID lookups in hot paths
//   - Bucket-based data structure for O(1) highest/lowest label selection
//   - Cached incoming edges list to avoid repeated allocations
//   - Adaptive global relabeling based on relabel count
//   - Gap heuristic with efficient tracking
//
// Time Complexity:
//   - FIFO variant: O(V³)
//   - Highest Label variant: O(V² √E)
//
// Space Complexity: O(V + E)
// =============================================================================

// PushRelabelResult contains the result of the Push-Relabel algorithm.
type PushRelabelResult struct {
	MaxFlow    float64
	Iterations int
	Canceled   bool
}

// =============================================================================
// Optimized Data Structures
// =============================================================================

// nodeData holds per-node data in a cache-friendly layout.
type nodeData struct {
	height     int
	excess     float64
	currentArc int
}

// bucketQueue implements a bucket-based priority queue for O(1) operations.
// Used for both Highest Label and Lowest Label selection strategies.
type bucketQueue struct {
	buckets     [][]int // buckets[height] = slice of node indices
	inBucket    []bool  // inBucket[nodeIdx] = is node in some bucket
	maxActive   int     // highest non-empty bucket index
	minActive   int     // lowest non-empty bucket index
	activeCount int     // total number of active nodes
}

// newBucketQueue creates a new bucket queue with the given capacity.
func newBucketQueue(maxHeight, nodeCount int) *bucketQueue {
	buckets := make([][]int, maxHeight+1)
	for i := range buckets {
		buckets[i] = make([]int, 0, 8) // Small initial capacity
	}
	return &bucketQueue{
		buckets:   buckets,
		inBucket:  make([]bool, nodeCount),
		maxActive: -1,
		minActive: maxHeight + 1,
	}
}

// push adds a node index to the bucket at the given height.
func (bq *bucketQueue) push(nodeIdx, height int) {
	if bq.inBucket[nodeIdx] {
		return // Already in queue
	}
	if height >= len(bq.buckets) || height < 0 {
		return
	}

	bq.buckets[height] = append(bq.buckets[height], nodeIdx)
	bq.inBucket[nodeIdx] = true
	bq.activeCount++

	if height > bq.maxActive {
		bq.maxActive = height
	}
	if height < bq.minActive {
		bq.minActive = height
	}
}

// popHighest removes and returns the node with the highest height.
func (bq *bucketQueue) popHighest() (int, bool) {
	for bq.maxActive >= 0 {
		bucket := bq.buckets[bq.maxActive]
		if len(bucket) > 0 {
			// Pop from the end (LIFO within bucket for cache locality)
			n := len(bucket) - 1
			nodeIdx := bucket[n]
			bq.buckets[bq.maxActive] = bucket[:n]
			bq.inBucket[nodeIdx] = false
			bq.activeCount--
			return nodeIdx, true
		}
		bq.maxActive--
	}
	return -1, false
}

// popLowest removes and returns the node with the lowest height.
func (bq *bucketQueue) popLowest() (int, bool) {
	for bq.minActive < len(bq.buckets) {
		bucket := bq.buckets[bq.minActive]
		if len(bucket) > 0 {
			n := len(bucket) - 1
			nodeIdx := bucket[n]
			bq.buckets[bq.minActive] = bucket[:n]
			bq.inBucket[nodeIdx] = false
			bq.activeCount--
			return nodeIdx, true
		}
		bq.minActive++
	}
	return -1, false
}

// remove removes a node from its bucket (used when height changes).
func (bq *bucketQueue) remove(nodeIdx, height int) {
	if !bq.inBucket[nodeIdx] {
		return
	}
	if height >= len(bq.buckets) || height < 0 {
		return
	}

	bucket := bq.buckets[height]
	for i, idx := range bucket {
		if idx == nodeIdx {
			// Swap with last and truncate
			bucket[i] = bucket[len(bucket)-1]
			bq.buckets[height] = bucket[:len(bucket)-1]
			bq.inBucket[nodeIdx] = false
			bq.activeCount--
			return
		}
	}
}

// updateHeight moves a node from old height bucket to new height bucket.
func (bq *bucketQueue) updateHeight(nodeIdx, oldHeight, newHeight int) {
	if bq.inBucket[nodeIdx] {
		bq.remove(nodeIdx, oldHeight)
	}
	bq.push(nodeIdx, newHeight)
}

// isEmpty returns true if there are no active nodes.
func (bq *bucketQueue) isEmpty() bool {
	return bq.activeCount == 0
}

// clear resets the bucket queue.
func (bq *bucketQueue) clear() {
	for i := range bq.buckets {
		bq.buckets[i] = bq.buckets[i][:0]
	}
	for i := range bq.inBucket {
		bq.inBucket[i] = false
	}
	bq.maxActive = -1
	bq.minActive = len(bq.buckets)
	bq.activeCount = 0
}

// =============================================================================
// Optimized Push-Relabel State
// =============================================================================

// prState holds the optimized state for Push-Relabel algorithm.
type prState struct {
	g       *graph.ResidualGraph
	source  int64
	sink    int64
	epsilon float64

	// Node mapping
	nodes     []int64       // Index → Node ID
	nodeIndex map[int64]int // Node ID → Index
	n         int           // Number of nodes

	// Per-node data (index-based access)
	data        []nodeData
	heightCount []int // Number of nodes at each height

	// Precomputed edge lists for each node (index-based)
	edgeLists [][]*graph.ResidualEdge

	// Source and sink indices
	sourceIdx int
	sinkIdx   int

	maxHeight int

	// Adaptive global relabeling
	relabelCount        int
	globalRelabelPeriod int

	// Statistics
	pushCount   int
	relabelOps  int
	globalCount int
}

// newPRState creates and initializes optimized Push-Relabel state.
func newPRState(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *prState {
	nodes := g.GetSortedNodes()
	n := len(nodes)

	// Build node index mapping
	nodeIndex := make(map[int64]int, n)
	for i, node := range nodes {
		nodeIndex[node] = i
	}

	maxHeight := 2*n - 1

	s := &prState{
		g:                   g,
		source:              source,
		sink:                sink,
		epsilon:             options.Epsilon,
		nodes:               nodes,
		nodeIndex:           nodeIndex,
		n:                   n,
		data:                make([]nodeData, n),
		heightCount:         make([]int, maxHeight+2),
		edgeLists:           make([][]*graph.ResidualEdge, n),
		sourceIdx:           nodeIndex[source],
		sinkIdx:             nodeIndex[sink],
		maxHeight:           maxHeight,
		globalRelabelPeriod: n, // Will be adjusted adaptively
	}

	// Precompute edge lists for each node
	for i, node := range nodes {
		s.edgeLists[i] = g.GetNeighborsList(node)
	}

	// Build incoming edges cache for globalRelabel
	g.BuildIncomingEdgesCache()

	return s
}

// initialize sets up the initial preflow.
func (s *prState) initialize() {
	// All heights start at 0, source height = n
	for i := range s.data {
		s.data[i].height = 0
		s.data[i].excess = 0
		s.data[i].currentArc = 0
	}
	s.data[s.sourceIdx].height = s.n

	// Count heights
	for i := range s.heightCount {
		s.heightCount[i] = 0
	}
	for i := range s.data {
		h := s.data[i].height
		if h <= s.maxHeight {
			s.heightCount[h]++
		}
	}

	// Saturate all edges from source
	for _, edge := range s.edgeLists[s.sourceIdx] {
		if edge.Capacity > s.epsilon {
			toIdx := s.nodeIndex[edge.To]
			flow := edge.Capacity
			s.g.UpdateFlow(s.source, edge.To, flow)
			s.data[toIdx].excess += flow
			s.data[s.sourceIdx].excess -= flow
		}
	}

	// Perform initial global relabeling for accurate heights
	s.globalRelabel()
}

// globalRelabel recomputes heights using reverse BFS from sink.
func (s *prState) globalRelabel() {
	s.globalCount++

	// Reset height counts
	for i := range s.heightCount {
		s.heightCount[i] = 0
	}

	// Initialize new heights to maxHeight + 1 (unreachable)
	newHeight := make([]int, s.n)
	for i := range newHeight {
		newHeight[i] = s.maxHeight + 1
	}
	newHeight[s.sinkIdx] = 0

	// BFS from sink using cached incoming edges
	queue := make([]int, 0, s.n)
	queue = append(queue, s.sinkIdx)
	head := 0

	for head < len(queue) {
		uIdx := queue[head]
		head++
		uHeight := newHeight[uIdx]
		u := s.nodes[uIdx]

		// Get cached incoming edges (edges pointing TO u)
		incomingList := s.g.GetIncomingEdgesListCached(u)
		for _, incoming := range incomingList {
			vIdx := s.nodeIndex[incoming.From]

			// Check if v can push to u (the edge from v to u has capacity)
			if newHeight[vIdx] > s.maxHeight && incoming.Edge.Capacity > s.epsilon {
				newHeight[vIdx] = uHeight + 1
				queue = append(queue, vIdx)
			}
		}
	}

	// Source always has height n
	newHeight[s.sourceIdx] = s.n

	// Apply new heights
	for i, h := range newHeight {
		s.data[i].height = h
		if h <= s.maxHeight {
			s.heightCount[h]++
		}
	}

	// Reset current arcs
	for i := range s.data {
		s.data[i].currentArc = 0
	}

	// Reset relabel count
	s.relabelCount = 0
}

// push attempts to push flow from node u to node v.
// Returns the amount of flow pushed.
func (s *prState) push(uIdx int, edge *graph.ResidualEdge) float64 {
	if edge.Capacity <= s.epsilon {
		return 0
	}

	vIdx := s.nodeIndex[edge.To]

	// Can only push to lower height
	if s.data[uIdx].height != s.data[vIdx].height+1 {
		return 0
	}

	// Calculate push amount
	delta := s.data[uIdx].excess
	if edge.Capacity < delta {
		delta = edge.Capacity
	}

	if delta <= s.epsilon {
		return 0
	}

	// Update graph
	s.g.UpdateFlow(s.nodes[uIdx], edge.To, delta)

	// Update excess
	s.data[uIdx].excess -= delta
	s.data[vIdx].excess += delta

	s.pushCount++

	return delta
}

// relabel increases the height of node u.
// Returns the new height, or -1 if node should be deactivated.
func (s *prState) relabel(uIdx int) int {
	oldHeight := s.data[uIdx].height
	if oldHeight > s.maxHeight {
		return -1
	}

	edges := s.edgeLists[uIdx]
	if len(edges) == 0 {
		s.heightCount[oldHeight]--
		s.data[uIdx].height = s.maxHeight + 1
		return -1
	}

	// Find minimum height among neighbors with residual capacity
	minHeight := s.maxHeight + 1
	for _, edge := range edges {
		if edge.Capacity > s.epsilon {
			vIdx := s.nodeIndex[edge.To]
			h := s.data[vIdx].height
			if h < minHeight {
				minHeight = h
			}
		}
	}

	if minHeight >= s.maxHeight {
		s.heightCount[oldHeight]--
		s.data[uIdx].height = s.maxHeight + 1
		return -1
	}

	newHeight := minHeight + 1
	if newHeight > s.maxHeight {
		s.heightCount[oldHeight]--
		s.data[uIdx].height = s.maxHeight + 1
		return -1
	}

	// Gap heuristic
	s.heightCount[oldHeight]--
	if s.heightCount[oldHeight] == 0 && oldHeight < s.n && oldHeight > 0 {
		s.applyGapHeuristic(oldHeight)
	}

	s.heightCount[newHeight]++
	s.data[uIdx].height = newHeight
	s.data[uIdx].currentArc = 0

	s.relabelOps++
	s.relabelCount++

	return newHeight
}

// applyGapHeuristic raises all nodes above the gap to maxHeight + 1.
func (s *prState) applyGapHeuristic(gapHeight int) {
	for i := 0; i < s.n; i++ {
		h := s.data[i].height
		if h > gapHeight && h <= s.maxHeight && i != s.sourceIdx {
			s.heightCount[h]--
			s.data[i].height = s.maxHeight + 1
		}
	}
}

// discharge processes node u until it has no excess or is deactivated.
// Returns true if the node is still active after discharge.
func (s *prState) discharge(uIdx int, activateFunc func(int)) bool {
	edges := s.edgeLists[uIdx]
	if len(edges) == 0 {
		return false
	}

	for s.data[uIdx].excess > s.epsilon && s.data[uIdx].height <= s.maxHeight {
		arc := s.data[uIdx].currentArc

		if arc >= len(edges) {
			// Need to relabel
			newHeight := s.relabel(uIdx)
			if newHeight < 0 {
				return false
			}

			// Check if we should do global relabel
			if s.relabelCount >= s.globalRelabelPeriod {
				s.globalRelabel()
			}
			continue
		}

		edge := edges[arc]
		vIdx := s.nodeIndex[edge.To]

		// Try to push
		if edge.Capacity > s.epsilon && s.data[uIdx].height == s.data[vIdx].height+1 {
			pushed := s.push(uIdx, edge)
			if pushed > s.epsilon && activateFunc != nil {
				// Activate v if it became active
				if vIdx != s.sourceIdx && vIdx != s.sinkIdx {
					activateFunc(vIdx)
				}
			}
		} else {
			s.data[uIdx].currentArc++
		}
	}

	return s.data[uIdx].excess > s.epsilon && s.data[uIdx].height <= s.maxHeight
}

// =============================================================================
// Push-Relabel FIFO Variant
// =============================================================================

// PushRelabel executes the Push-Relabel algorithm with FIFO vertex selection.
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

	state := newPRState(g, source, sink, options)
	state.initialize()

	// FIFO queue using slice (more efficient than list)
	queue := make([]int, 0, state.n)
	inQueue := make([]bool, state.n)

	// Add initially active nodes
	for i := 0; i < state.n; i++ {
		if i != state.sourceIdx && i != state.sinkIdx && state.data[i].excess > state.epsilon {
			queue = append(queue, i)
			inQueue[i] = true
		}
	}

	iterations := 0
	const checkInterval = 100

	for len(queue) > 0 {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &PushRelabelResult{
					MaxFlow:    state.data[state.sinkIdx].excess,
					Iterations: iterations,
					Canceled:   true,
				}
			default:
			}
		}

		// Pop from front
		uIdx := queue[0]
		queue = queue[1:]
		inQueue[uIdx] = false

		// Discharge
		stillActive := state.discharge(uIdx, func(vIdx int) {
			if !inQueue[vIdx] && state.data[vIdx].excess > state.epsilon {
				queue = append(queue, vIdx)
				inQueue[vIdx] = true
			}
		})

		// Re-add if still active
		if stillActive && !inQueue[uIdx] {
			queue = append(queue, uIdx)
			inQueue[uIdx] = true
		}

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    state.data[state.sinkIdx].excess,
		Iterations: iterations,
		Canceled:   false,
	}
}

// =============================================================================
// Push-Relabel Highest Label Variant
// =============================================================================

// PushRelabelHighestLabel executes Push-Relabel with Highest Label selection.
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

	state := newPRState(g, source, sink, options)
	state.initialize()

	// Bucket queue for highest label selection
	bq := newBucketQueue(state.maxHeight, state.n)

	// Add initially active nodes
	for i := 0; i < state.n; i++ {
		if i != state.sourceIdx && i != state.sinkIdx {
			if state.data[i].excess > state.epsilon && state.data[i].height <= state.maxHeight {
				bq.push(i, state.data[i].height)
			}
		}
	}

	iterations := 0
	const checkInterval = 100

	for !bq.isEmpty() {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &PushRelabelResult{
					MaxFlow:    state.data[state.sinkIdx].excess,
					Iterations: iterations,
					Canceled:   true,
				}
			default:
			}
		}

		uIdx, ok := bq.popHighest()
		if !ok {
			break
		}

		// Skip if no longer active
		if state.data[uIdx].excess <= state.epsilon || state.data[uIdx].height > state.maxHeight {
			continue
		}

		oldHeight := state.data[uIdx].height

		// Discharge
		stillActive := state.discharge(uIdx, func(vIdx int) {
			if state.data[vIdx].excess > state.epsilon && state.data[vIdx].height <= state.maxHeight {
				bq.push(vIdx, state.data[vIdx].height)
			}
		})

		// Re-add if still active (possibly at new height)
		if stillActive {
			newHeight := state.data[uIdx].height
			if newHeight != oldHeight {
				bq.push(uIdx, newHeight)
			} else {
				bq.push(uIdx, oldHeight)
			}
		}

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    state.data[state.sinkIdx].excess,
		Iterations: iterations,
		Canceled:   false,
	}
}

// =============================================================================
// Push-Relabel Lowest Label Variant
// =============================================================================

// PushRelabelLowestLabel executes Push-Relabel with Lowest Label selection.
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

	state := newPRState(g, source, sink, options)
	state.initialize()

	// Bucket queue for lowest label selection
	bq := newBucketQueue(state.maxHeight, state.n)

	// Add initially active nodes
	for i := 0; i < state.n; i++ {
		if i != state.sourceIdx && i != state.sinkIdx {
			if state.data[i].excess > state.epsilon && state.data[i].height <= state.maxHeight {
				bq.push(i, state.data[i].height)
			}
		}
	}

	iterations := 0
	const checkInterval = 100

	for !bq.isEmpty() {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		if iterations%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return &PushRelabelResult{
					MaxFlow:    state.data[state.sinkIdx].excess,
					Iterations: iterations,
					Canceled:   true,
				}
			default:
			}
		}

		uIdx, ok := bq.popLowest()
		if !ok {
			break
		}

		// Skip if no longer active
		if state.data[uIdx].excess <= state.epsilon || state.data[uIdx].height > state.maxHeight {
			continue
		}

		oldHeight := state.data[uIdx].height

		// Discharge
		stillActive := state.discharge(uIdx, func(vIdx int) {
			if state.data[vIdx].excess > state.epsilon && state.data[vIdx].height <= state.maxHeight {
				bq.push(vIdx, state.data[vIdx].height)
			}
		})

		// Re-add if still active
		if stillActive {
			newHeight := state.data[uIdx].height
			if newHeight != oldHeight {
				bq.push(uIdx, newHeight)
			} else {
				bq.push(uIdx, oldHeight)
			}
		}

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    state.data[state.sinkIdx].excess,
		Iterations: iterations,
		Canceled:   false,
	}
}
