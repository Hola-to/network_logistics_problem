package graph

import (
	"testing"
)

func TestBFS_SimpleGraph(t *testing.T) {
	rg := NewResidualGraph()
	// Create: 1 -> 2 -> 3 -> 4
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.AddEdgeWithReverse(2, 3, 10.0, 1.0)
	rg.AddEdgeWithReverse(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	if !result.Found {
		t.Error("Path should be found")
	}

	if !result.Visited[1] || !result.Visited[2] || !result.Visited[3] || !result.Visited[4] {
		t.Error("All nodes should be visited")
	}

	// Check parent chain
	if result.Parent[4] != 3 {
		t.Errorf("Parent of 4 = %d, want 3", result.Parent[4])
	}
	if result.Parent[3] != 2 {
		t.Errorf("Parent of 3 = %d, want 2", result.Parent[3])
	}
	if result.Parent[2] != 1 {
		t.Errorf("Parent of 2 = %d, want 1", result.Parent[2])
	}
}

func TestBFS_NoPath(t *testing.T) {
	rg := NewResidualGraph()
	// Disconnected: 1 -> 2, 3 -> 4
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	if result.Found {
		t.Error("Path should not be found")
	}

	if !result.Visited[1] || !result.Visited[2] {
		t.Error("Reachable nodes should be visited")
	}

	if result.Visited[3] || result.Visited[4] {
		t.Error("Unreachable nodes should not be visited")
	}
}

func TestBFS_ZeroCapacityEdge(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 0.0, 1.0) // Zero capacity
	rg.AddEdge(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	if result.Found {
		t.Error("Path should not be found (zero capacity edge)")
	}
}

func TestBFS_SaturatedPath(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.AddEdgeWithReverse(2, 3, 10.0, 1.0)

	// Saturate edge 2->3
	rg.UpdateFlow(2, 3, 10.0)

	result := BFS(rg, 1, 3)

	if result.Found {
		t.Error("Path should not be found (saturated edge)")
	}
}

func TestBFS_MultiplePathsFindsAny(t *testing.T) {
	rg := NewResidualGraph()
	// Diamond: 1 -> 2 -> 4
	//          1 -> 3 -> 4
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(1, 3, 10.0, 1.0)
	rg.AddEdge(2, 4, 10.0, 1.0)
	rg.AddEdge(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	if !result.Found {
		t.Error("Path should be found")
	}

	// Parent of 4 should be 2 or 3 (BFS finds shortest)
	parent4 := result.Parent[4]
	if parent4 != 2 && parent4 != 3 {
		t.Errorf("Parent of 4 should be 2 or 3, got %d", parent4)
	}
}

func TestBFS_SourceEqualsSink(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)
	rg.AddEdge(1, 2, 10.0, 1.0)

	result := BFS(rg, 1, 1)

	// Source == sink, technically found immediately but depends on implementation
	if result.Found {
		// OK: found immediately
		if result.Parent[1] != -1 {
			t.Errorf("Parent of source should be -1, got %d", result.Parent[1])
		}
	}
	// If not found, also OK as BFS doesn't check source==sink immediately
}

func TestBFS_LargeGraph(t *testing.T) {
	rg := NewResidualGraph()

	// Linear graph: 0 -> 1 -> 2 -> ... -> 999
	for i := int64(0); i < 1000; i++ {
		rg.AddNode(i)
		if i > 0 {
			rg.AddEdge(i-1, i, 100.0, 1.0)
		}
	}

	result := BFS(rg, 0, 999)

	if !result.Found {
		t.Error("Path should be found in linear graph")
	}

	// Verify path length
	pathLen := 0
	current := int64(999)
	for current != 0 {
		parent, exists := result.Parent[current]
		if !exists {
			t.Fatal("Parent chain broken")
		}
		current = parent
		pathLen++
	}

	if pathLen != 999 {
		t.Errorf("Path length = %d, want 999", pathLen)
	}
}

func TestBFSLevel_SimpleGraph(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(1, 3, 10.0, 1.0)
	rg.AddEdge(2, 4, 10.0, 1.0)
	rg.AddEdge(3, 4, 10.0, 1.0)
	rg.AddEdge(4, 5, 10.0, 1.0)

	levels := BFSLevel(rg, 1)

	expected := map[int64]int{
		1: 0,
		2: 1,
		3: 1,
		4: 2,
		5: 3,
	}

	for node, wantLevel := range expected {
		if gotLevel, exists := levels[node]; !exists {
			t.Errorf("Node %d not in levels", node)
		} else if gotLevel != wantLevel {
			t.Errorf("Level of node %d = %d, want %d", node, gotLevel, wantLevel)
		}
	}
}

func TestBFSLevel_Disconnected(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddNode(3) // Disconnected node

	levels := BFSLevel(rg, 1)

	if _, exists := levels[3]; exists {
		t.Error("Disconnected node should not have level")
	}
}

func TestBFSLevel_ZeroCapacity(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 0.0, 1.0) // Zero capacity
	rg.AddEdge(3, 4, 10.0, 1.0)

	levels := BFSLevel(rg, 1)

	if _, exists := levels[3]; exists {
		t.Error("Node behind zero capacity edge should not have level")
	}
	if _, exists := levels[4]; exists {
		t.Error("Node behind zero capacity edge should not have level")
	}
}

func TestBFSLevel_SingleNode(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)

	levels := BFSLevel(rg, 1)

	if levels[1] != 0 {
		t.Errorf("Source level = %d, want 0", levels[1])
	}
	if len(levels) != 1 {
		t.Errorf("Levels count = %d, want 1", len(levels))
	}
}

func TestBFSLevel_Cycle(t *testing.T) {
	rg := NewResidualGraph()
	// Triangle: 1 -> 2 -> 3 -> 1
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 10.0, 1.0)
	rg.AddEdge(3, 1, 10.0, 1.0)

	levels := BFSLevel(rg, 1)

	if levels[1] != 0 {
		t.Errorf("Level of 1 = %d, want 0", levels[1])
	}
	if levels[2] != 1 {
		t.Errorf("Level of 2 = %d, want 1", levels[2])
	}
	if levels[3] != 2 {
		t.Errorf("Level of 3 = %d, want 2", levels[3])
	}
}

func TestBFS_EmptyGraph(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)
	rg.AddNode(2)
	// No edges

	result := BFS(rg, 1, 2)

	if result.Found {
		t.Error("Path should not be found in graph without edges")
	}
}

func TestBFSLevel_EmptyGraph(t *testing.T) {
	rg := NewResidualGraph()

	levels := BFSLevel(rg, 1)

	// Source not in graph, but it should still get level 0
	if levels[1] != 0 {
		t.Errorf("Source level should be 0, got %d", levels[1])
	}
}
