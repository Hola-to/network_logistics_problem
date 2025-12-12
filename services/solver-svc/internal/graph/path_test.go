package graph

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReconstructPath(t *testing.T) {
	tests := []struct {
		name   string
		parent map[int64]int64
		source int64
		sink   int64
		want   []int64
	}{
		{
			name: "simple path",
			parent: map[int64]int64{
				1: -1,
				2: 1,
				3: 2,
				4: 3,
			},
			source: 1,
			sink:   4,
			want:   []int64{1, 2, 3, 4},
		},
		{
			name: "direct edge",
			parent: map[int64]int64{
				1: -1,
				2: 1,
			},
			source: 1,
			sink:   2,
			want:   []int64{1, 2},
		},
		{
			name: "sink not reachable",
			parent: map[int64]int64{
				1: -1,
				2: 1,
			},
			source: 1,
			sink:   5,
			want:   nil,
		},
		{
			name:   "empty parent",
			parent: map[int64]int64{},
			source: 1,
			sink:   2,
			want:   nil,
		},
		{
			name: "long path",
			parent: map[int64]int64{
				1: -1,
				2: 1,
				3: 2,
				4: 3,
				5: 4,
				6: 5,
			},
			source: 1,
			sink:   6,
			want:   []int64{1, 2, 3, 4, 5, 6},
		},
		{
			name: "source equals sink",
			parent: map[int64]int64{
				1: -1,
			},
			source: 1,
			sink:   1,
			want:   []int64{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReconstructPath(tt.parent, tt.source, tt.sink)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconstructPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindMinCapacityOnPath(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *ResidualGraph
		path  []int64
		want  float64
	}{
		{
			name: "uniform capacity",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 10.0, 1.0)
				rg.AddEdge(2, 3, 10.0, 1.0)
				rg.AddEdge(3, 4, 10.0, 1.0)
				return rg
			},
			path: []int64{1, 2, 3, 4},
			want: 10.0,
		},
		{
			name: "bottleneck in middle",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 10.0, 1.0)
				rg.AddEdge(2, 3, 3.0, 1.0) // Bottleneck
				rg.AddEdge(3, 4, 10.0, 1.0)
				return rg
			},
			path: []int64{1, 2, 3, 4},
			want: 3.0,
		},
		{
			name: "bottleneck at start",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 2.0, 1.0) // Bottleneck
				rg.AddEdge(2, 3, 10.0, 1.0)
				rg.AddEdge(3, 4, 10.0, 1.0)
				return rg
			},
			path: []int64{1, 2, 3, 4},
			want: 2.0,
		},
		{
			name: "bottleneck at end",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 10.0, 1.0)
				rg.AddEdge(2, 3, 10.0, 1.0)
				rg.AddEdge(3, 4, 1.0, 1.0) // Bottleneck
				return rg
			},
			path: []int64{1, 2, 3, 4},
			want: 1.0,
		},
		{
			name: "single edge path",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 5.0, 1.0)
				return rg
			},
			path: []int64{1, 2},
			want: 5.0,
		},
		{
			name: "missing edge in path",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 10.0, 1.0)
				// Missing edge 2 -> 3
				rg.AddEdge(3, 4, 10.0, 1.0)
				return rg
			},
			path: []int64{1, 2, 3, 4},
			want: 0.0,
		},
		{
			name: "empty path",
			setup: func() *ResidualGraph {
				return NewResidualGraph()
			},
			path: []int64{},
			want: 0.0,
		},
		{
			name: "single node path",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddNode(1)
				return rg
			},
			path: []int64{1},
			want: 0.0,
		},
		{
			name: "zero capacity edge",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdge(1, 2, 10.0, 1.0)
				rg.AddEdge(2, 3, 0.0, 1.0)
				rg.AddEdge(3, 4, 10.0, 1.0)
				return rg
			},
			path: []int64{1, 2, 3, 4},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := tt.setup()
			got := FindMinCapacityOnPath(rg, tt.path)
			if got != tt.want {
				t.Errorf("FindMinCapacityOnPath() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestAugmentPath(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *ResidualGraph
		path      []int64
		flow      float64
		wantFlows map[EdgeKey]float64
		wantCaps  map[EdgeKey]float64
	}{
		{
			name: "simple augment",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
				rg.AddEdgeWithReverse(2, 3, 10.0, 1.0)
				return rg
			},
			path: []int64{1, 2, 3},
			flow: 5.0,
			wantFlows: map[EdgeKey]float64{
				{1, 2}: 5.0,
				{2, 3}: 5.0,
			},
			wantCaps: map[EdgeKey]float64{
				{1, 2}: 5.0,
				{2, 3}: 5.0,
				{2, 1}: 5.0, // Reverse
				{3, 2}: 5.0, // Reverse
			},
		},
		{
			name: "multiple augments",
			setup: func() *ResidualGraph {
				rg := NewResidualGraph()
				rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
				rg.AddEdgeWithReverse(2, 3, 10.0, 1.0)
				// First augment
				AugmentPath(rg, []int64{1, 2, 3}, 3.0)
				return rg
			},
			path: []int64{1, 2, 3},
			flow: 4.0,
			wantFlows: map[EdgeKey]float64{
				{1, 2}: 7.0,
				{2, 3}: 7.0,
			},
			wantCaps: map[EdgeKey]float64{
				{1, 2}: 3.0,
				{2, 3}: 3.0,
				{2, 1}: 7.0,
				{3, 2}: 7.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := tt.setup()
			AugmentPath(rg, tt.path, tt.flow)

			for key, wantFlow := range tt.wantFlows {
				edge := rg.GetEdge(key.From, key.To)
				if edge == nil {
					t.Errorf("Edge %d->%d not found", key.From, key.To)
					continue
				}
				if edge.Flow != wantFlow {
					t.Errorf("Flow on %d->%d = %f, want %f", key.From, key.To, edge.Flow, wantFlow)
				}
			}

			for key, wantCap := range tt.wantCaps {
				edge := rg.GetEdge(key.From, key.To)
				if edge == nil {
					t.Errorf("Edge %d->%d not found", key.From, key.To)
					continue
				}
				if edge.Capacity != wantCap {
					t.Errorf("Capacity on %d->%d = %f, want %f", key.From, key.To, edge.Capacity, wantCap)
				}
			}
		})
	}
}

type EdgeKey struct {
	From, To int64
}

func TestAugmentPath_EmptyPath(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)

	// Should not panic
	AugmentPath(rg, []int64{}, 5.0)
	AugmentPath(rg, []int64{1}, 5.0)

	// Original edge unchanged
	edge := rg.GetEdge(1, 2)
	if edge.Flow != 0 {
		t.Errorf("Empty path should not modify flow, got %f", edge.Flow)
	}
}

func TestAugmentPath_LongPath(t *testing.T) {
	rg := NewResidualGraph()

	path := make([]int64, 100)
	for i := int64(0); i < 100; i++ {
		path[i] = i
		if i > 0 {
			rg.AddEdgeWithReverse(i-1, i, 100.0, 1.0)
		}
	}

	AugmentPath(rg, path, 50.0)

	// Check all edges
	for i := int64(0); i < 99; i++ {
		edge := rg.GetEdge(i, i+1)
		if edge.Flow != 50.0 {
			t.Errorf("Flow on %d->%d = %f, want 50.0", i, i+1, edge.Flow)
		}
		if edge.Capacity != 50.0 {
			t.Errorf("Capacity on %d->%d = %f, want 50.0", i, i+1, edge.Capacity)
		}
	}
}

func TestFindMinCapacityOnPath_InfinityCapacity(t *testing.T) {
	g := NewResidualGraph()

	// Create edge with Infinity capacity
	g.AddNode(1)
	g.AddNode(2)
	if g.Edges[1] == nil {
		g.Edges[1] = make(map[int64]*ResidualEdge)
	}
	edge := &ResidualEdge{
		To:               2,
		Capacity:         Infinity,
		Cost:             0,
		Flow:             0,
		OriginalCapacity: Infinity,
		IsReverse:        false,
	}
	g.Edges[1][2] = edge
	g.EdgesList[1] = append(g.EdgesList[1], edge)

	path := []int64{1, 2}

	result := FindMinCapacityOnPath(g, path)

	// When capacity = Infinity, minCapacity stays Infinity and returns 0
	assert.Equal(t, 0.0, result)
}

func TestFindMinCapacityOnPath_AllEdgesInfinity(t *testing.T) {
	g := NewResidualGraph()

	g.AddNode(1)
	g.AddNode(2)
	g.AddNode(3)

	if g.Edges[1] == nil {
		g.Edges[1] = make(map[int64]*ResidualEdge)
	}
	edge1 := &ResidualEdge{
		To:               2,
		Capacity:         Infinity,
		OriginalCapacity: Infinity,
	}
	g.Edges[1][2] = edge1
	g.EdgesList[1] = append(g.EdgesList[1], edge1)

	if g.Edges[2] == nil {
		g.Edges[2] = make(map[int64]*ResidualEdge)
	}
	edge2 := &ResidualEdge{
		To:               3,
		Capacity:         Infinity,
		OriginalCapacity: Infinity,
	}
	g.Edges[2][3] = edge2
	g.EdgesList[2] = append(g.EdgesList[2], edge2)

	path := []int64{1, 2, 3}

	result := FindMinCapacityOnPath(g, path)

	assert.Equal(t, 0.0, result)
}

func TestFindMinCapacityOnPath_MixedCapacity(t *testing.T) {
	g := NewResidualGraph()
	g.AddEdge(1, 2, 10, 0)
	g.AddEdge(2, 3, Infinity, 0)
	g.AddEdge(3, 4, 5, 0)

	path := []int64{1, 2, 3, 4}

	result := FindMinCapacityOnPath(g, path)

	assert.Equal(t, 5.0, result) // Minimum of 10, Infinity, 5
}
