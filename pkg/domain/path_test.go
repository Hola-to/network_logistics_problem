package domain

import (
	"testing"
)

func TestReconstructPath(t *testing.T) {
	tests := []struct {
		name     string
		parent   map[int64]int64
		source   int64
		sink     int64
		expected []int64
	}{
		{
			name: "simple path",
			parent: map[int64]int64{
				1: -1,
				2: 1,
				3: 2,
			},
			source:   1,
			sink:     3,
			expected: []int64{1, 2, 3},
		},
		{
			name: "direct path",
			parent: map[int64]int64{
				1: -1,
				2: 1,
			},
			source:   1,
			sink:     2,
			expected: []int64{1, 2},
		},
		{
			name:     "sink not in parent",
			parent:   map[int64]int64{1: -1},
			source:   1,
			sink:     3,
			expected: nil,
		},
		{
			name:     "empty parent",
			parent:   map[int64]int64{},
			source:   1,
			sink:     2,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReconstructPath(tt.parent, tt.source, tt.sink)
			if !int64SliceEqual(result, tt.expected) {
				t.Errorf("ReconstructPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindMinCapacityOnPath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, CurrentFlow: 3})
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 5, CurrentFlow: 2})

	tests := []struct {
		name     string
		path     []int64
		expected float64
	}{
		{
			name:     "normal path",
			path:     []int64{1, 2, 3},
			expected: 3, // min(10-3, 5-2) = min(7, 3) = 3
		},
		{
			name:     "single node",
			path:     []int64{1},
			expected: 0,
		},
		{
			name:     "empty path",
			path:     []int64{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindMinCapacityOnPath(g, tt.path)
			if !FloatEquals(result, tt.expected) {
				t.Errorf("FindMinCapacityOnPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindMinCapacityOnPath_MissingEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10})
	// No edge from 2 to 3

	path := []int64{1, 2, 3}
	result := FindMinCapacityOnPath(g, path)
	if result != 0 {
		t.Errorf("FindMinCapacityOnPath() with missing edge = %v, want 0", result)
	}
}

func TestCalculatePathCost(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Cost: 5})
	g.AddEdge(&Edge{From: 2, To: 3, Cost: 3})

	tests := []struct {
		name     string
		path     []int64
		expected float64
	}{
		{
			name:     "normal path",
			path:     []int64{1, 2, 3},
			expected: 8,
		},
		{
			name:     "single edge",
			path:     []int64{1, 2},
			expected: 5,
		},
		{
			name:     "single node",
			path:     []int64{1},
			expected: 0,
		},
		{
			name:     "empty path",
			path:     []int64{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePathCost(g, tt.path)
			if !FloatEquals(result, tt.expected) {
				t.Errorf("CalculatePathCost() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculatePathLength(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Length: 100})
	g.AddEdge(&Edge{From: 2, To: 3, Length: 50})

	path := []int64{1, 2, 3}
	result := CalculatePathLength(g, path)
	if !FloatEquals(result, 150) {
		t.Errorf("CalculatePathLength() = %v, want 150", result)
	}
}

func TestAugmentPath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, CurrentFlow: 0})
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10, CurrentFlow: 0})
	g.AddEdge(&Edge{From: 2, To: 1, Capacity: 0, CurrentFlow: 0}) // reverse edge

	path := []int64{1, 2, 3}
	AugmentPath(g, path, 5)

	edge12, _ := g.GetEdge(1, 2)
	if !FloatEquals(edge12.CurrentFlow, 5) {
		t.Errorf("Edge 1->2 flow = %v, want 5", edge12.CurrentFlow)
	}

	edge23, _ := g.GetEdge(2, 3)
	if !FloatEquals(edge23.CurrentFlow, 5) {
		t.Errorf("Edge 2->3 flow = %v, want 5", edge23.CurrentFlow)
	}

	// Check reverse edge was decremented
	edge21, _ := g.GetEdge(2, 1)
	if !FloatEquals(edge21.CurrentFlow, -5) {
		t.Errorf("Reverse edge 2->1 flow = %v, want -5", edge21.CurrentFlow)
	}
}

func TestCreatePath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Cost: 5, Length: 100})
	g.AddEdge(&Edge{From: 2, To: 3, Cost: 3, Length: 50})

	path := CreatePath(g, []int64{1, 2, 3}, 10)

	if len(path.Nodes) != 3 {
		t.Errorf("path.Nodes length = %d, want 3", len(path.Nodes))
	}
	if !FloatEquals(path.Flow, 10) {
		t.Errorf("path.Flow = %v, want 10", path.Flow)
	}
	if !FloatEquals(path.Cost, 80) { // (5+3) * 10
		t.Errorf("path.Cost = %v, want 80", path.Cost)
	}
	if !FloatEquals(path.Length, 150) {
		t.Errorf("path.Length = %v, want 150", path.Length)
	}
}

func int64SliceEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
