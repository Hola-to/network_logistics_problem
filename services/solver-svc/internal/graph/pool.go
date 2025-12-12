// Package graph provides data structures and utilities for network flow algorithms.
//
// This package contains:
//   - ResidualGraph: The core graph representation for flow algorithms
//   - GraphPool: Memory pooling for efficient graph allocation
//   - BFS utilities: Breadth-first search implementations
//   - Path utilities: Path reconstruction and augmentation
//
// # Memory Management
//
// Network flow algorithms often need to create temporary graphs and data structures.
// The GraphPool provides efficient memory reuse through sync.Pool, reducing GC pressure
// in high-throughput scenarios.
//
// # Thread Safety
//
// ResidualGraph is NOT thread-safe. Each goroutine should work with its own graph.
// Use Clone() or CloneToPooled() for concurrent operations.
// For shared read access, use SafeResidualGraph.
//
// # Example
//
//	// Using the global pool
//	pool := graph.GetPool()
//	g := pool.AcquireGraph()
//	defer pool.ReleaseGraph(g)
//
//	g.AddEdgeWithReverse(1, 2, 10, 0)
//	// ... use graph ...
package graph

import (
	"sync"
)

// =============================================================================
// Graph Pool
// =============================================================================

// GraphPool provides memory pooling for ResidualGraph and related data structures.
//
// Using a pool significantly reduces memory allocations and GC pressure when
// processing many graphs or running algorithms repeatedly.
//
// The pool is safe for concurrent use from multiple goroutines.
//
// # Usage
//
// For single graph operations:
//
//	pool := graph.GetPool()  // Use global pool
//	g := pool.AcquireGraph()
//	defer pool.ReleaseGraph(g)
//	// ... use g ...
//
// For request-scoped resources:
//
//	resources := NewPooledResources()
//	defer resources.Release()
//	g := resources.Graph()
//	dist := resources.FloatMap()
//	// ... use resources ...
//
// # Implementation Notes
//
// The pool uses sync.Pool internally, which means:
//   - Objects may be garbage collected if not in use
//   - Objects are not pre-allocated
//   - The pool grows and shrinks based on demand
type GraphPool struct {
	graphs      sync.Pool
	int64Slices sync.Pool
	int64Maps   sync.Pool
	floatMaps   sync.Pool
	boolMaps    sync.Pool
	intMaps     sync.Pool
}

// globalPool is the singleton pool instance.
// Initialized at package load time.
var globalPool = &GraphPool{
	graphs: sync.Pool{
		New: func() any {
			return &ResidualGraph{
				Nodes:        make(map[int64]bool, 64),
				Edges:        make(map[int64]map[int64]*ResidualEdge, 64),
				EdgesList:    make(map[int64][]*ResidualEdge, 64),
				ReverseEdges: make(map[int64]map[int64]*ResidualEdge, 64),
			}
		},
	},
	int64Slices: sync.Pool{
		New: func() any {
			s := make([]int64, 0, 128)
			return &s
		},
	},
	int64Maps: sync.Pool{
		New: func() any {
			return make(map[int64]int64, 64)
		},
	},
	floatMaps: sync.Pool{
		New: func() any {
			return make(map[int64]float64, 64)
		},
	},
	boolMaps: sync.Pool{
		New: func() any {
			return make(map[int64]bool, 64)
		},
	},
	intMaps: sync.Pool{
		New: func() any {
			return make(map[int64]int, 64)
		},
	},
}

// GetPool returns the global graph pool.
//
// The global pool is thread-safe and should be used for most operations.
// Creating custom pools is rarely necessary.
func GetPool() *GraphPool {
	return globalPool
}

// =============================================================================
// Graph Pool Methods
// =============================================================================

// AcquireGraph obtains a ResidualGraph from the pool.
//
// The returned graph is cleared and ready for use.
// Call ReleaseGraph() when done to return it to the pool.
//
// # Example
//
//	g := pool.AcquireGraph()
//	defer pool.ReleaseGraph(g)
//	g.AddNode(1)
//	g.AddEdgeWithReverse(1, 2, 10, 0)
func (p *GraphPool) AcquireGraph() *ResidualGraph {
	return p.graphs.Get().(*ResidualGraph)
}

// ReleaseGraph returns a ResidualGraph to the pool.
//
// The graph is cleared before being pooled.
// After calling this method, the graph must not be used.
//
// It is safe to pass nil to this method.
func (p *GraphPool) ReleaseGraph(g *ResidualGraph) {
	if g == nil {
		return
	}
	g.Clear()
	p.graphs.Put(g)
}

// AcquireInt64Slice obtains a []int64 slice from the pool.
//
// The returned slice has length 0 but may have non-zero capacity.
// Call ReleaseInt64Slice() when done.
func (p *GraphPool) AcquireInt64Slice() *[]int64 {
	return p.int64Slices.Get().(*[]int64)
}

// ReleaseInt64Slice returns a []int64 slice to the pool.
//
// The slice is reset to length 0 before pooling.
// It is safe to pass nil.
func (p *GraphPool) ReleaseInt64Slice(s *[]int64) {
	if s == nil {
		return
	}
	*s = (*s)[:0]
	p.int64Slices.Put(s)
}

// AcquireInt64Map obtains a map[int64]int64 from the pool.
//
// The returned map is cleared and ready for use.
// Call ReleaseInt64Map() when done.
func (p *GraphPool) AcquireInt64Map() map[int64]int64 {
	return p.int64Maps.Get().(map[int64]int64)
}

// ReleaseInt64Map returns a map[int64]int64 to the pool.
//
// The map is cleared before pooling.
// It is safe to pass nil.
func (p *GraphPool) ReleaseInt64Map(m map[int64]int64) {
	if m == nil {
		return
	}
	clear(m)
	p.int64Maps.Put(m)
}

// AcquireFloatMap obtains a map[int64]float64 from the pool.
//
// The returned map is cleared and ready for use.
// Call ReleaseFloatMap() when done.
func (p *GraphPool) AcquireFloatMap() map[int64]float64 {
	return p.floatMaps.Get().(map[int64]float64)
}

// ReleaseFloatMap returns a map[int64]float64 to the pool.
//
// The map is cleared before pooling.
// It is safe to pass nil.
func (p *GraphPool) ReleaseFloatMap(m map[int64]float64) {
	if m == nil {
		return
	}
	clear(m)
	p.floatMaps.Put(m)
}

// AcquireBoolMap obtains a map[int64]bool from the pool.
//
// The returned map is cleared and ready for use.
// Call ReleaseBoolMap() when done.
func (p *GraphPool) AcquireBoolMap() map[int64]bool {
	return p.boolMaps.Get().(map[int64]bool)
}

// ReleaseBoolMap returns a map[int64]bool to the pool.
//
// The map is cleared before pooling.
// It is safe to pass nil.
func (p *GraphPool) ReleaseBoolMap(m map[int64]bool) {
	if m == nil {
		return
	}
	clear(m)
	p.boolMaps.Put(m)
}

// AcquireIntMap obtains a map[int64]int from the pool.
//
// The returned map is cleared and ready for use.
// Call ReleaseIntMap() when done.
func (p *GraphPool) AcquireIntMap() map[int64]int {
	return p.intMaps.Get().(map[int64]int)
}

// ReleaseIntMap returns a map[int64]int to the pool.
//
// The map is cleared before pooling.
// It is safe to pass nil.
func (p *GraphPool) ReleaseIntMap(m map[int64]int) {
	if m == nil {
		return
	}
	clear(m)
	p.intMaps.Put(m)
}

// =============================================================================
// Pooled Resources
// =============================================================================

// PooledResources manages a set of pooled resources for a single request.
//
// This is useful when an algorithm needs multiple temporary data structures.
// All resources are tracked and can be released with a single call to Release().
//
// # Usage Pattern
//
//	func processRequest(g *ResidualGraph) {
//	    resources := NewPooledResources()
//	    defer resources.Release()  // Always release!
//
//	    dist := resources.FloatMap()
//	    parent := resources.Int64Map()
//	    visited := resources.BoolMap()
//
//	    // ... use resources ...
//	}
//
// # Thread Safety
//
// PooledResources is NOT thread-safe. Each goroutine should have its own instance.
type PooledResources struct {
	pool        *GraphPool
	graphs      []*ResidualGraph
	int64Maps   []map[int64]int64
	floatMaps   []map[int64]float64
	boolMaps    []map[int64]bool
	intMaps     []map[int64]int
	int64Slices []*[]int64
}

// NewPooledResources creates a new resource container using the global pool.
//
// Always call Release() when done, typically via defer:
//
//	resources := NewPooledResources()
//	defer resources.Release()
func NewPooledResources() *PooledResources {
	return &PooledResources{
		pool: globalPool,
	}
}

// NewPooledResourcesWithPool creates a resource container with a custom pool.
//
// This is useful when you need isolated pooling or custom pool configuration.
func NewPooledResourcesWithPool(pool *GraphPool) *PooledResources {
	if pool == nil {
		pool = globalPool
	}
	return &PooledResources{
		pool: pool,
	}
}

// Graph acquires a ResidualGraph and tracks it for automatic release.
//
// The returned graph is cleared and ready for use.
// Do not manually release graphs obtained this way.
func (pr *PooledResources) Graph() *ResidualGraph {
	g := pr.pool.AcquireGraph()
	pr.graphs = append(pr.graphs, g)
	return g
}

// Int64Map acquires a map[int64]int64 and tracks it for automatic release.
func (pr *PooledResources) Int64Map() map[int64]int64 {
	m := pr.pool.AcquireInt64Map()
	pr.int64Maps = append(pr.int64Maps, m)
	return m
}

// FloatMap acquires a map[int64]float64 and tracks it for automatic release.
func (pr *PooledResources) FloatMap() map[int64]float64 {
	m := pr.pool.AcquireFloatMap()
	pr.floatMaps = append(pr.floatMaps, m)
	return m
}

// BoolMap acquires a map[int64]bool and tracks it for automatic release.
func (pr *PooledResources) BoolMap() map[int64]bool {
	m := pr.pool.AcquireBoolMap()
	pr.boolMaps = append(pr.boolMaps, m)
	return m
}

// IntMap acquires a map[int64]int and tracks it for automatic release.
func (pr *PooledResources) IntMap() map[int64]int {
	m := pr.pool.AcquireIntMap()
	pr.intMaps = append(pr.intMaps, m)
	return m
}

// Int64Slice acquires a []int64 slice and tracks it for automatic release.
func (pr *PooledResources) Int64Slice() *[]int64 {
	s := pr.pool.AcquireInt64Slice()
	pr.int64Slices = append(pr.int64Slices, s)
	return s
}

// Release returns all tracked resources to the pool.
//
// After calling Release(), do not use any resources obtained from this container.
// It is safe to call Release() multiple times.
func (pr *PooledResources) Release() {
	// Release all tracked resources
	for _, g := range pr.graphs {
		pr.pool.ReleaseGraph(g)
	}
	for _, m := range pr.int64Maps {
		pr.pool.ReleaseInt64Map(m)
	}
	for _, m := range pr.floatMaps {
		pr.pool.ReleaseFloatMap(m)
	}
	for _, m := range pr.boolMaps {
		pr.pool.ReleaseBoolMap(m)
	}
	for _, m := range pr.intMaps {
		pr.pool.ReleaseIntMap(m)
	}
	for _, s := range pr.int64Slices {
		pr.pool.ReleaseInt64Slice(s)
	}

	// Clear tracking slices (keep capacity for reuse)
	pr.graphs = pr.graphs[:0]
	pr.int64Maps = pr.int64Maps[:0]
	pr.floatMaps = pr.floatMaps[:0]
	pr.boolMaps = pr.boolMaps[:0]
	pr.intMaps = pr.intMaps[:0]
	pr.int64Slices = pr.int64Slices[:0]
}

// Reset is an alias for Release() for consistency with other Reset methods.
func (pr *PooledResources) Reset() {
	pr.Release()
}
