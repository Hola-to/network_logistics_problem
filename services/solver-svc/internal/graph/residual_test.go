package graph

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResidualGraph(t *testing.T) {
	rg := NewResidualGraph()

	if rg == nil {
		t.Fatal("NewResidualGraph returned nil")
	}

	if rg.Nodes == nil {
		t.Error("Nodes map is nil")
	}

	if rg.Edges == nil {
		t.Error("Edges map is nil")
	}

	if len(rg.Nodes) != 0 {
		t.Errorf("Expected empty nodes, got %d", len(rg.Nodes))
	}

	if len(rg.Edges) != 0 {
		t.Errorf("Expected empty edges, got %d", len(rg.Edges))
	}
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

			if got := rg.NodeCount(); got != tt.want {
				t.Errorf("NodeCount() = %d, want %d", got, tt.want)
			}
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

			if got := rg.EdgeCount(); got != tt.wantEdge {
				t.Errorf("EdgeCount() = %d, want %d", got, tt.wantEdge)
			}

			if got := rg.NodeCount(); got != tt.wantNode {
				t.Errorf("NodeCount() = %d, want %d", got, tt.wantNode)
			}

			// Verify edge properties
			for _, e := range tt.edges {
				edge := rg.GetEdge(e.from, e.to)
				if edge == nil {
					t.Errorf("Edge from %d to %d not found", e.from, e.to)
					continue
				}
				if edge.Capacity != e.capacity {
					t.Errorf("Edge capacity = %f, want %f", edge.Capacity, e.capacity)
				}
				if edge.Cost != e.cost {
					t.Errorf("Edge cost = %f, want %f", edge.Cost, e.cost)
				}
				if edge.OriginalCapacity != e.capacity {
					t.Errorf("Edge original capacity = %f, want %f", edge.OriginalCapacity, e.capacity)
				}
				if edge.IsReverse {
					t.Error("Forward edge marked as reverse")
				}
			}
		})
	}
}

func TestResidualGraph_AddEdgeWithReverse(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 5.0)

	// Check forward edge
	forward := rg.GetEdge(1, 2)
	if forward == nil {
		t.Fatal("Forward edge not found")
	}
	if forward.Capacity != 10.0 {
		t.Errorf("Forward capacity = %f, want 10.0", forward.Capacity)
	}
	if forward.Cost != 5.0 {
		t.Errorf("Forward cost = %f, want 5.0", forward.Cost)
	}
	if forward.IsReverse {
		t.Error("Forward edge marked as reverse")
	}

	// Check reverse edge
	reverse := rg.GetEdge(2, 1)
	if reverse == nil {
		t.Fatal("Reverse edge not found")
	}
	if reverse.Capacity != 0.0 {
		t.Errorf("Reverse capacity = %f, want 0.0", reverse.Capacity)
	}
	if reverse.Cost != -5.0 {
		t.Errorf("Reverse cost = %f, want -5.0", reverse.Cost)
	}
	if !reverse.IsReverse {
		t.Error("Reverse edge not marked as reverse")
	}
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
			if tt.wantNil && edge != nil {
				t.Error("Expected nil edge, got non-nil")
			}
			if !tt.wantNil && edge == nil {
				t.Error("Expected non-nil edge, got nil")
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
			if tt.wantNil && neighbors != nil && len(neighbors) > 0 {
				t.Errorf("Expected nil/empty neighbors, got %d", len(neighbors))
			}
			if !tt.wantNil {
				if neighbors == nil {
					t.Error("Expected non-nil neighbors, got nil")
				} else if len(neighbors) != tt.wantLen {
					t.Errorf("Neighbors count = %d, want %d", len(neighbors), tt.wantLen)
				}
			}
		})
	}
}

func TestResidualGraph_GetNodes(t *testing.T) {
	rg := NewResidualGraph()

	// Empty graph
	nodes := rg.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}

	// Add nodes
	rg.AddNode(1)
	rg.AddNode(2)
	rg.AddNode(3)

	nodes = rg.GetNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}

	// Check all nodes present
	nodeSet := make(map[int64]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}
	for _, expected := range []int64{1, 2, 3} {
		if !nodeSet[expected] {
			t.Errorf("Node %d not found in result", expected)
		}
	}
}

func TestResidualGraph_Clone(t *testing.T) {
	original := NewResidualGraph()
	original.AddEdgeWithReverse(1, 2, 10.0, 5.0)
	original.AddEdgeWithReverse(2, 3, 20.0, 10.0)

	// Modify flow
	original.UpdateFlow(1, 2, 5.0)

	// Clone
	clone := original.Clone()

	// Verify independence
	if clone == original {
		t.Error("Clone is same object as original")
	}

	// Verify data equality
	if clone.NodeCount() != original.NodeCount() {
		t.Errorf("Clone nodes = %d, original = %d", clone.NodeCount(), original.NodeCount())
	}

	// Verify edge data
	origEdge := original.GetEdge(1, 2)
	cloneEdge := clone.GetEdge(1, 2)

	if origEdge == cloneEdge {
		t.Error("Edge is same object in clone")
	}

	if cloneEdge.Flow != origEdge.Flow {
		t.Errorf("Clone flow = %f, original = %f", cloneEdge.Flow, origEdge.Flow)
	}

	// Modify clone, original should not change
	clone.UpdateFlow(2, 3, 10.0)
	cloneEdge23 := clone.GetEdge(2, 3)
	origEdge23 := original.GetEdge(2, 3)

	if cloneEdge23.Flow == origEdge23.Flow && cloneEdge23.Flow != 0 {
		t.Error("Modifying clone affected original")
	}
}

func TestResidualGraph_Reset(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.AddEdgeWithReverse(2, 3, 20.0, 2.0)

	// Add flow
	rg.UpdateFlow(1, 2, 5.0)
	rg.UpdateFlow(2, 3, 5.0)

	// Verify flow exists
	if rg.GetEdge(1, 2).Flow != 5.0 {
		t.Error("Flow not set before reset")
	}

	// Reset
	rg.Reset()

	// Verify flow is zero
	edge12 := rg.GetEdge(1, 2)
	if edge12.Flow != 0 {
		t.Errorf("Flow after reset = %f, want 0", edge12.Flow)
	}

	// Verify capacity restored
	if edge12.Capacity != 10.0 {
		t.Errorf("Capacity after reset = %f, want 10.0", edge12.Capacity)
	}

	// Verify reverse edge capacity reset
	reverse := rg.GetEdge(2, 1)
	if reverse.Capacity != 0 {
		t.Errorf("Reverse capacity after reset = %f, want 0", reverse.Capacity)
	}
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
			if edge.Flow != tt.wantFlow {
				t.Errorf("Flow = %f, want %f", edge.Flow, tt.wantFlow)
			}
			if edge.Capacity != tt.wantCapacity {
				t.Errorf("Capacity = %f, want %f", edge.Capacity, tt.wantCapacity)
			}

			reverse := rg.GetEdge(2, 1)
			if reverse.Capacity != tt.wantReverseCapacity {
				t.Errorf("Reverse capacity = %f, want %f", reverse.Capacity, tt.wantReverseCapacity)
			}
		})
	}
}

func TestResidualGraph_UpdateFlowCreatesReverseEdge(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 5.0) // No reverse edge

	// Update flow should create reverse edge
	rg.UpdateFlow(1, 2, 3.0)

	reverse := rg.GetEdge(2, 1)
	if reverse == nil {
		t.Fatal("Reverse edge not created")
	}
	if reverse.Capacity != 3.0 {
		t.Errorf("Reverse capacity = %f, want 3.0", reverse.Capacity)
	}
	if reverse.Cost != -5.0 {
		t.Errorf("Reverse cost = %f, want -5.0", reverse.Cost)
	}
	if !reverse.IsReverse {
		t.Error("Created edge not marked as reverse")
	}
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
			if got != tt.want {
				t.Errorf("GetFlowOnEdge(%d, %d) = %f, want %f", tt.from, tt.to, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("GetTotalFlow() = %f, want %f", got, tt.want)
			}
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
				rg.UpdateFlow(1, 2, 3.0) // 3 * 5 = 15
			},
			want: 15.0,
		},
		{
			name: "multiple edges with flow",
			setup: func(rg *ResidualGraph) {
				rg.AddEdgeWithReverse(1, 2, 10.0, 2.0)
				rg.AddEdgeWithReverse(2, 3, 10.0, 3.0)
				rg.UpdateFlow(1, 2, 5.0) // 5 * 2 = 10
				rg.UpdateFlow(2, 3, 4.0) // 4 * 3 = 12
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
			if got != tt.want {
				t.Errorf("GetTotalCost() = %f, want %f", got, tt.want)
			}
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
		{"zero capacity", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &ResidualEdge{
				Capacity: tt.capacity,
			}
			if got := edge.ResidualCapacity(); got != tt.want {
				t.Errorf("ResidualCapacity() = %f, want %f", got, tt.want)
			}
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
			edge := &ResidualEdge{
				Capacity: tt.capacity,
			}
			if got := edge.HasCapacity(); got != tt.want {
				t.Errorf("HasCapacity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResidualGraph_Concurrency(t *testing.T) {
	rg := NewResidualGraph()

	// Add initial structure
	for i := int64(0); i < 100; i++ {
		rg.AddNode(i)
	}
	for i := int64(0); i < 99; i++ {
		rg.AddEdgeWithReverse(i, i+1, 100.0, 1.0)
	}

	// Pre-compute sorted nodes to avoid write during concurrent reads
	_ = rg.GetSortedNodes()

	done := make(chan bool)

	// Multiple goroutines reading (теперь действительно только чтение)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = rg.GetSortedNodes() // Теперь возвращает cached
				_ = rg.GetNeighbors(50)
				_ = rg.GetEdge(25, 26)
				_ = rg.NodeCount()
				_ = rg.EdgeCount()
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

	// Сначала добавляем reverse ребро
	g.AddReverseEdge(1, 2, 5.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.True(t, edge.IsReverse)
	assert.Equal(t, 0.0, edge.Capacity)

	// Теперь добавляем прямое ребро - должно перезаписать reverse
	g.AddEdge(1, 2, 10.0, 3.0)

	edge = g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.False(t, edge.IsReverse, "Should be converted to forward edge")
	assert.Equal(t, 10.0, edge.Capacity)
	assert.Equal(t, 10.0, edge.OriginalCapacity)
	assert.Equal(t, 3.0, edge.Cost)
}

func TestResidualGraph_AddEdge_ParallelEdges(t *testing.T) {
	g := NewResidualGraph()

	// Добавляем первое ребро
	g.AddEdge(1, 2, 10.0, 5.0)

	// Добавляем параллельное ребро - capacity должна суммироваться
	g.AddEdge(1, 2, 7.0, 3.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.Equal(t, 17.0, edge.Capacity, "Capacities should sum")
	assert.Equal(t, 17.0, edge.OriginalCapacity)
	assert.Equal(t, 5.0, edge.Cost, "Cost should remain from first edge")
}

func TestResidualGraph_AddReverseEdge_ExistingForward(t *testing.T) {
	g := NewResidualGraph()

	// Сначала добавляем прямое ребро
	g.AddEdge(1, 2, 10.0, 5.0)

	// Пытаемся добавить reverse - НЕ должно перезаписать прямое
	g.AddReverseEdge(1, 2, 3.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.False(t, edge.IsReverse, "Forward edge should not be overwritten")
	assert.Equal(t, 10.0, edge.Capacity)
	assert.Equal(t, 5.0, edge.Cost)
}

func TestResidualGraph_AddReverseEdge_ExistingReverse(t *testing.T) {
	g := NewResidualGraph()

	// Добавляем reverse ребро
	g.AddReverseEdge(1, 2, 5.0)

	// Пытаемся добавить ещё одно reverse - должно остаться первое
	g.AddReverseEdge(1, 2, 10.0)

	edge := g.GetEdge(1, 2)
	require.NotNil(t, edge)
	assert.True(t, edge.IsReverse)
	assert.Equal(t, -5.0, edge.Cost, "Cost should remain from first reverse edge")
}

func TestResidualGraph_AntiParallelEdges(t *testing.T) {
	g := NewResidualGraph()

	// Добавляем ребро 1->2
	g.AddEdgeWithReverse(1, 2, 10, 1)

	// Добавляем anti-parallel ребро 2->1
	g.AddEdgeWithReverse(2, 1, 5, 2)

	// Проверяем, что оба прямых ребра существуют
	edge12 := g.GetEdge(1, 2)
	edge21 := g.GetEdge(2, 1)

	require.NotNil(t, edge12, "Edge 1->2 should exist")
	require.NotNil(t, edge21, "Edge 2->1 should exist")

	// Оба должны быть прямыми (не reverse)
	assert.False(t, edge12.IsReverse, "Edge 1->2 should be forward")
	assert.False(t, edge21.IsReverse, "Edge 2->1 should be forward")

	// Проверяем capacity и cost
	assert.Equal(t, 10.0, edge12.Capacity)
	assert.Equal(t, 5.0, edge21.Capacity)
	assert.Equal(t, 1.0, edge12.Cost)
	assert.Equal(t, 2.0, edge21.Cost)
}

func TestResidualGraph_AntiParallelFlow(t *testing.T) {
	g := NewResidualGraph()

	// Граф: 1 <-> 2 -> 3
	g.AddEdgeWithReverse(1, 2, 10, 0)
	g.AddEdgeWithReverse(2, 1, 5, 0) // Anti-parallel
	g.AddEdgeWithReverse(2, 3, 10, 0)

	// Пускаем поток 8 по пути 1->2->3
	g.UpdateFlow(1, 2, 8)
	g.UpdateFlow(2, 3, 8)

	edge12 := g.GetEdge(1, 2)
	edge21 := g.GetEdge(2, 1)

	// 1->2: capacity осталось 2, flow = 8
	assert.Equal(t, 8.0, edge12.Flow)
	assert.Equal(t, 2.0, edge12.Capacity)

	// 2->1: остаётся прямым ребром, capacity увеличилась на flow (для возможности отмены)
	assert.False(t, edge21.IsReverse)
	assert.Equal(t, 13.0, edge21.Capacity) // 5 original + 8 cancellation
}

func TestResidualGraph_GetNeighborsList(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(1, 3, 20, 2)
	rg.AddEdgeWithReverse(1, 4, 30, 3)

	neighbors := rg.GetNeighborsList(1)

	assert.Len(t, neighbors, 3)
	// Verify they're in insertion order
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
	// Should be sorted by From ID
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

func TestResidualGraph_GetSortedNodes(t *testing.T) {
	rg := NewResidualGraph()
	// Add nodes in random order
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

	// First call computes
	sorted1 := rg.GetSortedNodes()
	// Second call should return cached
	sorted2 := rg.GetSortedNodes()

	assert.Equal(t, sorted1, sorted2)

	// Add new node - should invalidate cache
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

	// Verify data equality
	assert.Equal(t, original.NodeCount(), clone.NodeCount())

	origEdge := original.GetEdge(1, 2)
	cloneEdge := clone.GetEdge(1, 2)

	assert.True(t, origEdge != cloneEdge, "Should be different pointer objects")

	assert.Equal(t, origEdge.Flow, cloneEdge.Flow)
	assert.Equal(t, origEdge.Capacity, cloneEdge.Capacity)
}

func TestResidualGraph_GetAllEdges(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(2, 3, 20, 2)

	allEdges := rg.GetAllEdges()

	assert.Len(t, allEdges, 2) // Only forward edges
	for _, edge := range allEdges {
		assert.False(t, edge.IsReverse)
	}
}

func TestResidualGraph_Clear(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(2, 3, 20, 2)
	rg.UpdateFlow(1, 2, 5)

	assert.Equal(t, 3, rg.NodeCount())

	rg.Clear()

	assert.Equal(t, 0, rg.NodeCount())
	assert.Equal(t, 0, rg.EdgeCount())
}

func TestSafeResidualGraph(t *testing.T) {
	sg := NewSafeResidualGraph()

	// Write operations
	sg.WithWriteLock(func(g *ResidualGraph) {
		g.AddNode(1)
		g.AddNode(2)
		g.AddEdgeWithReverse(1, 2, 10, 1)
	})

	// Read operations
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
