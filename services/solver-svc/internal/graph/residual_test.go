package graph

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResidualGraph(t *testing.T) {
	rg := NewResidualGraph()

	require.NotNil(t, rg)
	assert.NotNil(t, rg.Nodes)
	assert.NotNil(t, rg.Edges)
	assert.NotNil(t, rg.EdgesList)
	assert.NotNil(t, rg.ReverseEdges)
	assert.NotNil(t, rg.IncomingEdgesListCache)
	assert.True(t, rg.incomingCacheDirty)
	assert.Empty(t, rg.Nodes)
	assert.Empty(t, rg.Edges)
}

func TestResidualGraph_AddNode(t *testing.T) {
	tests := []struct {
		name    string
		nodeIDs []int64
		want    int
	}{
		{
			name:    "single node",
			nodeIDs: []int64{1},
			want:    1,
		},
		{
			name:    "multiple nodes",
			nodeIDs: []int64{1, 2, 3, 4, 5},
			want:    5,
		},
		{
			name:    "duplicate nodes",
			nodeIDs: []int64{1, 1, 1, 2, 2},
			want:    2,
		},
		{
			name:    "negative node IDs",
			nodeIDs: []int64{-1, -2, 0, 1, 2},
			want:    5,
		},
		{
			name:    "large node IDs",
			nodeIDs: []int64{1000000, 2000000, 3000000},
			want:    3,
		},
		{
			name:    "empty",
			nodeIDs: []int64{},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := NewResidualGraph()

			for _, id := range tt.nodeIDs {
				rg.AddNode(id)
			}

			assert.Equal(t, tt.want, rg.NodeCount())
		})
	}
}

func TestResidualGraph_AddEdge(t *testing.T) {
	tests := []struct {
		name  string
		edges []struct {
			from, to       int64
			capacity, cost float64
		}
		wantEdge int
		wantNode int
	}{
		{
			name: "single edge",
			edges: []struct {
				from, to       int64
				capacity, cost float64
			}{
				{1, 2, 10.0, 1.0},
			},
			wantEdge: 1,
			wantNode: 2,
		},
		{
			name: "multiple edges",
			edges: []struct {
				from, to       int64
				capacity, cost float64
			}{
				{1, 2, 10.0, 1.0},
				{2, 3, 20.0, 2.0},
				{3, 4, 30.0, 3.0},
			},
			wantEdge: 3,
			wantNode: 4,
		},
		{
			name: "parallel edges (same from different to)",
			edges: []struct {
				from, to       int64
				capacity, cost float64
			}{
				{1, 2, 10.0, 1.0},
				{1, 3, 15.0, 1.5},
				{1, 4, 20.0, 2.0},
			},
			wantEdge: 3,
			wantNode: 4,
		},
		{
			name: "zero capacity edge",
			edges: []struct {
				from, to       int64
				capacity, cost float64
			}{
				{1, 2, 0.0, 1.0},
			},
			wantEdge: 1,
			wantNode: 2,
		},
		{
			name: "negative cost edge",
			edges: []struct {
				from, to       int64
				capacity, cost float64
			}{
				{1, 2, 10.0, -5.0},
			},
			wantEdge: 1,
			wantNode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := NewResidualGraph()

			for _, e := range tt.edges {
				rg.AddEdge(e.from, e.to, e.capacity, e.cost)
			}

			assert.Equal(t, tt.wantEdge, rg.EdgeCount())
			assert.Equal(t, tt.wantNode, rg.NodeCount())

			for _, e := range tt.edges {
				edge := rg.GetEdge(e.from, e.to)
				require.NotNil(t, edge)
				assert.Equal(t, e.capacity, edge.Capacity)
				assert.Equal(t, e.cost, edge.Cost)
				assert.Equal(t, e.capacity, edge.OriginalCapacity)
				assert.False(t, edge.IsReverse)
			}
		})
	}
}

func TestResidualGraph_AddEdgeWithReverse(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 5.0)

	forward := rg.GetEdge(1, 2)
	require.NotNil(t, forward)
	assert.Equal(t, 10.0, forward.Capacity)
	assert.Equal(t, 5.0, forward.Cost)
	assert.False(t, forward.IsReverse)

	reverse := rg.GetEdge(2, 1)
	require.NotNil(t, reverse)
	assert.Equal(t, 0.0, reverse.Capacity)
	assert.Equal(t, -5.0, reverse.Cost)
	assert.True(t, reverse.IsReverse)
}

func TestResidualGraph_GetEdge(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 20.0, 2.0)

	tests := []struct {
		name     string
		from, to int64
		wantNil  bool
	}{
		{"existing edge", 1, 2, false},
		{"another existing edge", 2, 3, false},
		{"non-existing edge", 1, 3, true},
		{"reversed non-existing", 2, 1, true},
		{"unknown nodes", 99, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := rg.GetEdge(tt.from, tt.to)
			if tt.wantNil {
				assert.Nil(t, edge)
			} else {
				assert.NotNil(t, edge)
			}
		})
	}
}

func TestResidualGraph_GetNeighbors(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(1, 3, 20.0, 2.0)
	rg.AddEdge(1, 4, 30.0, 3.0)
	rg.AddEdge(2, 5, 40.0, 4.0)

	tests := []struct {
		name    string
		nodeID  int64
		wantLen int
		wantNil bool
	}{
		{"node with 3 neighbors", 1, 3, false},
		{"node with 1 neighbor", 2, 1, false},
		{"node with no neighbors", 5, 0, true},
		{"unknown node", 99, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neighbors := rg.GetNeighbors(tt.nodeID)
			if tt.wantNil {
				assert.True(t, neighbors == nil)
			} else {
				assert.NotNil(t, neighbors)
				assert.Equal(t, tt.wantLen, len(neighbors))
			}
		})
	}
}

func TestResidualGraph_GetNodes(t *testing.T) {
	rg := NewResidualGraph()

	nodes := rg.GetNodes()
	assert.Empty(t, nodes)

	rg.AddNode(1)
	rg.AddNode(2)
	rg.AddNode(3)

	nodes = rg.GetNodes()
	assert.Len(t, nodes, 3)

	nodeSet := make(map[int64]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}
	for _, expected := range []int64{1, 2, 3} {
		assert.True(t, nodeSet[expected])
	}
}

func TestResidualGraph_Clone(t *testing.T) {
	original := NewResidualGraph()
	original.AddEdgeWithReverse(1, 2, 10.0, 5.0)
	original.AddEdgeWithReverse(2, 3, 20.0, 10.0)
	original.UpdateFlow(1, 2, 5.0)

	clone := original.Clone()

	assert.True(t, clone != original)
	assert.Equal(t, original.NodeCount(), clone.NodeCount())

	origEdge := original.GetEdge(1, 2)
	cloneEdge := clone.GetEdge(1, 2)

	assert.True(t, origEdge != cloneEdge)
	assert.Equal(t, origEdge.Flow, cloneEdge.Flow)

	clone.UpdateFlow(2, 3, 10.0)
	cloneEdge23 := clone.GetEdge(2, 3)
	origEdge23 := original.GetEdge(2, 3)

	assert.NotEqual(t, cloneEdge23.Flow, origEdge23.Flow)
}

func TestResidualGraph_Reset(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.AddEdgeWithReverse(2, 3, 20.0, 2.0)
	rg.UpdateFlow(1, 2, 5.0)
	rg.UpdateFlow(2, 3, 5.0)

	assert.Equal(t, 5.0, rg.GetEdge(1, 2).Flow)

	rg.Reset()

	edge12 := rg.GetEdge(1, 2)
	assert.Equal(t, 0.0, edge12.Flow)
	assert.Equal(t, 10.0, edge12.Capacity)

	reverse := rg.GetEdge(2, 1)
	assert.Equal(t, 0.0, reverse.Capacity)
}

func TestResidualGraph_UpdateFlow(t *testing.T) {
	tests := []struct {
		name                string
		capacity            float64
		flowToAdd           float64
		wantFlow            float64
		wantCapacity        float64
		wantReverseCapacity float64
	}{
		{
			name:                "partial flow",
			capacity:            10.0,
			flowToAdd:           3.0,
			wantFlow:            3.0,
			wantCapacity:        7.0,
			wantReverseCapacity: 3.0,
		},
		{
			name:                "full capacity flow",
			capacity:            10.0,
			flowToAdd:           10.0,
			wantFlow:            10.0,
			wantCapacity:        0.0,
			wantReverseCapacity: 10.0,
		},
		{
			name:                "zero flow",
			capacity:            10.0,
			flowToAdd:           0.0,
			wantFlow:            0.0,
			wantCapacity:        10.0,
			wantReverseCapacity: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := NewResidualGraph()
			rg.AddEdgeWithReverse(1, 2, tt.capacity, 1.0)
			rg.UpdateFlow(1, 2, tt.flowToAdd)

			edge := rg.GetEdge(1, 2)
			assert.Equal(t, tt.wantFlow, edge.Flow)
			assert.Equal(t, tt.wantCapacity, edge.Capacity)

			reverse := rg.GetEdge(2, 1)
			assert.Equal(t, tt.wantReverseCapacity, reverse.Capacity)
		})
	}
}

func TestResidualGraph_UpdateFlowCreatesReverseEdge(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 5.0)

	rg.UpdateFlow(1, 2, 3.0)

	reverse := rg.GetEdge(2, 1)
	require.NotNil(t, reverse)
	assert.Equal(t, 3.0, reverse.Capacity)
	assert.Equal(t, -5.0, reverse.Cost)
	assert.True(t, reverse.IsReverse)
}

func TestResidualGraph_GetFlowOnEdge(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.UpdateFlow(1, 2, 5.0)

	tests := []struct {
		name     string
		from, to int64
		want     float64
	}{
		{"existing edge with flow", 1, 2, 5.0},
		{"reverse edge", 2, 1, 0.0},
		{"non-existing edge", 1, 3, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rg.GetFlowOnEdge(tt.from, tt.to)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResidualGraph_GetTotalFlow(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*ResidualGraph)
		source int64
		want   float64
	}{
		{
			name: "single outgoing edge",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
				rg.UpdateFlow(1, 2, 5.0)
			},
			source: 1,
			want:   5.0,
		},
		{
			name: "multiple outgoing edges",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
				rg.AddEdgeWithReverse(1, 3, 20.0, 2.0)
				rg.UpdateFlow(1, 2, 5.0)
				rg.UpdateFlow(1, 3, 8.0)
			},
			source: 1,
			want:   13.0,
		},
		{
			name: "no flow",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
			},
			source: 1,
			want:   0.0,
		},
		{
			name:   "empty graph",
			setup:  func(rg *ResidualGraph) {},
			source: 1,
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := NewResidualGraph()
			tt.setup(rg)
			got := rg.GetTotalFlow(tt.source)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResidualGraph_GetTotalCost(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*ResidualGraph)
		want  float64
	}{
		{
			name: "single edge with flow",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 5.0)
				rg.UpdateFlow(1, 2, 3.0)
			},
			want: 15.0,
		},
		{
			name: "multiple edges with flow",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 2.0)
				rg.AddEdgeWithReverse(2, 3, 10.0, 3.0)
				rg.UpdateFlow(1, 2, 5.0)
				rg.UpdateFlow(2, 3, 4.0)
			},
			want: 22.0,
		},
		{
			name: "no flow",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 5.0)
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := NewResidualGraph()
			tt.setup(rg)
			got := rg.GetTotalCost()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResidualEdge_ResidualCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity float64
		want     float64
	}{
		{"no flow", 10.0, 10.0},
		{"partial flow", 7.0, 7.0},
		{"full flow", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &ResidualEdge{Capacity: tt.capacity}
			assert.Equal(t, tt.want, edge.ResidualCapacity())
		})
	}
}

func TestResidualEdge_HasCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity float64
		want     bool
	}{
		{"has capacity", 7.0, true},
		{"no capacity", 0.0, false},
		{"epsilon capacity", Epsilon / 2, false},
		{"just above epsilon", Epsilon * 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &ResidualEdge{Capacity: tt.capacity}
			assert.Equal(t, tt.want, edge.HasCapacity())
		})
	}
}

func TestResidualGraph_Concurrency(t *testing.T) {
	rg := NewResidualGraph()

	for i := int64(0); i < 100; i++ {
		rg.AddNode(i)
	}
	for i := int64(0); i < 99; i++ {
		rg.AddEdgeWithReverse(i, i+1, 100.0, 1.0)
	}

	_ = rg.GetSortedNodes()
	rg.BuildIncomingEdgesCache()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = rg.GetSortedNodes()
				_ = rg.GetNeighbors(50)
				_ = rg.GetEdge(25, 26)
				_ = rg.NodeCount()
				_ = rg.EdgeCount()
				_ = rg.GetIncomingEdgesListCached(50)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestResidualGraph_AddEdge_OverwriteReverse(t *testing.T) {
	g := NewResidualGraph()

	g.AddReverseEdge(1, 2, 5.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.True(t, edge.IsReverse)
	assert.Equal(t, 0.0, edge.Capacity)

	g.AddEdge(1, 2, 10.0, 3.0)

	edge = g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.False(t, edge.IsReverse)
	assert.Equal(t, 10.0, edge.Capacity)
	assert.Equal(t, 10.0, edge.OriginalCapacity)
	assert.Equal(t, 3.0, edge.Cost)
}

func TestResidualGraph_AddEdge_ParallelEdges(t *testing.T) {
	g := NewResidualGraph()

	g.AddEdge(1, 2, 10.0, 5.0)
	g.AddEdge(1, 2, 7.0, 3.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.Equal(t, 17.0, edge.Capacity)
	assert.Equal(t, 17.0, edge.OriginalCapacity)
	assert.Equal(t, 5.0, edge.Cost)
}

func TestResidualGraph_AddReverseEdge_ExistingForward(t *testing.T) {
	g := NewResidualGraph()

	g.AddEdge(1, 2, 10.0, 5.0)
	g.AddReverseEdge(1, 2, 3.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.False(t, edge.IsReverse)
	assert.Equal(t, 10.0, edge.Capacity)
	assert.Equal(t, 5.0, edge.Cost)
}

func TestResidualGraph_GetNeighborsList(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(1, 3, 20, 2)
	rg.AddEdgeWithReverse(1, 4, 30, 3)

	neighbors := rg.GetNeighborsList(1)

	assert.Len(t, neighbors, 3)
	assert.Equal(t, int64(2), neighbors[0].To)
	assert.Equal(t, int64(3), neighbors[1].To)
	assert.Equal(t, int64(4), neighbors[2].To)
}

func TestResidualGraph_GetNeighborsList_Empty(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)

	neighbors := rg.GetNeighborsList(1)
	assert.Empty(t, neighbors)
}

func TestResidualGraph_GetNeighborsList_Unknown(t *testing.T) {
	rg := NewResidualGraph()

	neighbors := rg.GetNeighborsList(999)
	assert.Nil(t, neighbors)
}

func TestResidualGraph_GetIncomingEdgesList(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 4, 10, 1)
	rg.AddEdgeWithReverse(2, 4, 20, 2)
	rg.AddEdgeWithReverse(3, 4, 30, 3)

	incoming := rg.GetIncomingEdgesList(4)

	assert.Len(t, incoming, 3)
	assert.Equal(t, int64(1), incoming[0].From)
	assert.Equal(t, int64(2), incoming[1].From)
	assert.Equal(t, int64(3), incoming[2].From)
}

func TestResidualGraph_GetIncomingEdgesList_Empty(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)

	incoming := rg.GetIncomingEdgesList(1)
	assert.Nil(t, incoming)
}

// =============================================================================
// IncomingEdgesListCache Tests
// =============================================================================

func TestResidualGraph_BuildIncomingEdgesCache(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 4, 10, 1)
	rg.AddEdgeWithReverse(2, 4, 20, 2)
	rg.AddEdgeWithReverse(3, 4, 30, 3)

	assert.True(t, rg.incomingCacheDirty)

	rg.BuildIncomingEdgesCache()

	assert.False(t, rg.incomingCacheDirty)
	assert.Len(t, rg.IncomingEdgesListCache[4], 3)
}

func TestResidualGraph_GetIncomingEdgesListCached(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 4, 10, 1)
	rg.AddEdgeWithReverse(2, 4, 20, 2)
	rg.AddEdgeWithReverse(3, 4, 30, 3)

	// First call builds cache
	incoming := rg.GetIncomingEdgesListCached(4)
	assert.Len(t, incoming, 3)
	assert.False(t, rg.incomingCacheDirty)

	// Second call uses cache
	incoming2 := rg.GetIncomingEdgesListCached(4)
	assert.Equal(t, incoming, incoming2)
}

func TestResidualGraph_GetIncomingEdgesListCached_Empty(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)

	incoming := rg.GetIncomingEdgesListCached(1)
	assert.Nil(t, incoming)
}

func TestResidualGraph_IncomingCacheInvalidation(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 3, 10, 1)

	rg.BuildIncomingEdgesCache()
	assert.False(t, rg.incomingCacheDirty)

	// Adding new edge should invalidate cache
	rg.AddEdgeWithReverse(2, 3, 20, 2)
	assert.True(t, rg.incomingCacheDirty)

	// GetIncomingEdgesListCached should rebuild
	incoming := rg.GetIncomingEdgesListCached(3)
	assert.Len(t, incoming, 2)
	assert.False(t, rg.incomingCacheDirty)
}

func TestResidualGraph_CacheSorting(t *testing.T) {
	rg := NewResidualGraph()
	// Add in non-sorted order
	rg.AddEdgeWithReverse(5, 10, 10, 1)
	rg.AddEdgeWithReverse(1, 10, 10, 1)
	rg.AddEdgeWithReverse(3, 10, 10, 1)

	incoming := rg.GetIncomingEdgesListCached(10)

	assert.Len(t, incoming, 3)
	// Should be sorted by From
	assert.Equal(t, int64(1), incoming[0].From)
	assert.Equal(t, int64(3), incoming[1].From)
	assert.Equal(t, int64(5), incoming[2].From)
}

func TestResidualGraph_CacheClear(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.BuildIncomingEdgesCache()

	assert.False(t, rg.incomingCacheDirty)
	assert.NotEmpty(t, rg.IncomingEdgesListCache)

	rg.Clear()

	assert.True(t, rg.incomingCacheDirty)
	assert.Empty(t, rg.IncomingEdgesListCache)
}

// =============================================================================
// GetSortedNodes Tests
// =============================================================================

func TestResidualGraph_GetSortedNodes(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(5)
	rg.AddNode(1)
	rg.AddNode(3)
	rg.AddNode(2)
	rg.AddNode(4)

	sorted := rg.GetSortedNodes()

	assert.Equal(t, []int64{1, 2, 3, 4, 5}, sorted)
}

func TestResidualGraph_GetSortedNodes_Cached(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(3)
	rg.AddNode(1)
	rg.AddNode(2)

	sorted1 := rg.GetSortedNodes()
	sorted2 := rg.GetSortedNodes()

	assert.Equal(t, sorted1, sorted2)

	rg.AddNode(4)
	sorted3 := rg.GetSortedNodes()

	assert.Equal(t, []int64{1, 2, 3, 4}, sorted3)
}

func TestResidualGraph_CloneToPooled(t *testing.T) {
	pool := GetPool()
	original := NewResidualGraph()
	original.AddEdgeWithReverse(1, 2, 10, 5)
	original.AddEdgeWithReverse(2, 3, 20, 10)
	original.UpdateFlow(1, 2, 5)

	clone := original.CloneToPooled(pool)
	defer pool.ReleaseGraph(clone)

	assert.Equal(t, original.NodeCount(), clone.NodeCount())

	origEdge := original.GetEdge(1, 2)
	cloneEdge := clone.GetEdge(1, 2)

	assert.True(t, origEdge != cloneEdge)
	assert.Equal(t, origEdge.Flow, cloneEdge.Flow)
	assert.Equal(t, origEdge.Capacity, cloneEdge.Capacity)
}

func TestResidualGraph_GetAllEdges(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(2, 3, 20, 2)

	allEdges := rg.GetAllEdges()

	assert.Len(t, allEdges, 2)
	for _, edge := range allEdges {
		assert.False(t, edge.IsReverse)
	}
}

func TestResidualGraph_Clear(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(2, 3, 20, 2)
	rg.UpdateFlow(1, 2, 5)
	rg.BuildIncomingEdgesCache()

	assert.Equal(t, 3, rg.NodeCount())
	assert.False(t, rg.incomingCacheDirty)

	rg.Clear()

	assert.Equal(t, 0, rg.NodeCount())
	assert.Equal(t, 0, rg.EdgeCount())
	assert.Empty(t, rg.IncomingEdgesListCache)
	assert.True(t, rg.incomingCacheDirty)
}

func TestSafeResidualGraph(t *testing.T) {
	sg := NewSafeResidualGraph()

	sg.WithWriteLock(func(g *ResidualGraph) {
		g.AddNode(1)
		g.AddNode(2)
		g.AddEdgeWithReverse(1, 2, 10, 1)
	})

	var nodeCount int
	sg.WithReadLock(func(g *ResidualGraph) {
		nodeCount = g.NodeCount()
	})

	assert.Equal(t, 2, nodeCount)
}

func TestSafeResidualGraph_CloneUnsafe(t *testing.T) {
	sg := NewSafeResidualGraph()

	sg.WithWriteLock(func(g *ResidualGraph) {
		g.AddEdgeWithReverse(1, 2, 10, 1)
	})

	clone := sg.CloneUnsafe()

	assert.Equal(t, 2, clone.NodeCount())
	assert.NotNil(t, clone.GetEdge(1, 2))
}

func TestSafeResidualGraph_ClonePooled(t *testing.T) {
	pool := GetPool()
	sg := NewSafeResidualGraph()

	sg.WithWriteLock(func(g *ResidualGraph) {
		g.AddEdgeWithReverse(1, 2, 10, 1)
	})

	clone := sg.ClonePooled(pool)
	defer pool.ReleaseGraph(clone)

	assert.Equal(t, 2, clone.NodeCount())
}

func TestSafeResidualGraph_ConcurrentReads(t *testing.T) {
	sg := NewSafeResidualGraph()

	sg.WithWriteLock(func(g *ResidualGraph) {
		for i := int64(1); i <= 100; i++ {
			g.AddNode(i)
		}
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sg.WithReadLock(func(g *ResidualGraph) {
				_ = g.NodeCount()
				_ = g.GetSortedNodes()
			})
		}()
	}

	wg.Wait()
}

func TestResidualGraph_AntiParallelEdges(t *testing.T) {
	g := NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 1)
	g.AddEdgeWithReverse(2, 1, 5, 2)

	edge12 := g.GetEdge(1, 2)
	edge21 := g.GetEdge(2, 1)

	require.NotNil(t, edge12)
	require.NotNil(t, edge21)

	assert.False(t, edge12.IsReverse)
	assert.False(t, edge21.IsReverse)
	assert.Equal(t, 10.0, edge12.Capacity)
	assert.Equal(t, 5.0, edge21.Capacity)
	assert.Equal(t, 1.0, edge12.Cost)
	assert.Equal(t, 2.0, edge21.Cost)
}

func TestResidualGraph_AntiParallelFlow(t *testing.T) {
	g := NewResidualGraph()

	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 1, 5, 0)
	g.AddEdgeWithReverse(2, 3, 10, 0)

	g.UpdateFlow(1, 2, 8)
	g.UpdateFlow(2, 3, 8)

	edge12 := g.GetEdge(1, 2)
	edge21 := g.GetEdge(2, 1)

	assert.Equal(t, 8.0, edge12.Flow)
	assert.Equal(t, 2.0, edge12.Capacity)

	assert.False(t, edge21.IsReverse)
	assert.Equal(t, 13.0, edge21.Capacity)
}
