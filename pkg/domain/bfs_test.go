package domain

import (
	"testing"
)

func createTestGraph() *Graph {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 5

	// Nodes
	for i := int64(1); i <= 5; i++ {
		g.AddNode(&Node{ID: i})
	}

	// Edges: 1->2->3->5 and 1->4->5
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10})
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10})
	g.AddEdge(&Edge{From: 3, To: 5, Capacity: 10})
	g.AddEdge(&Edge{From: 1, To: 4, Capacity: 5})
	g.AddEdge(&Edge{From: 4, To: 5, Capacity: 5})

	return g
}

func TestBFS_PathExists(t *testing.T) {
	g := createTestGraph()

	result := BFS(g, 1, 5)

	if !result.Found {
		t.Error("expected path to be found")
	}

	// Check parent map
	if result.Parent[5] == -1 {
		t.Error("expected sink to have parent")
	}

	// All nodes should be visited
	if !result.Visited[1] {
		t.Error("source should be visited")
	}
	if !result.Visited[5] {
		t.Error("sink should be visited")
	}
}

func TestBFS_NoPath(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 3

	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10})
	// No edge to sink

	result := BFS(g, 1, 3)

	if result.Found {
		t.Error("expected no path")
	}
}

func TestBFS_SaturatedEdge(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 2

	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, CurrentFlow: 10}) // Saturated

	result := BFS(g, 1, 2)

	if result.Found {
		t.Error("expected no path through saturated edge")
	}
}

func TestBFSLevel(t *testing.T) {
	g := createTestGraph()

	level := BFSLevel(g, 1)

	if level[1] != 0 {
		t.Errorf("source level should be 0, got %d", level[1])
	}
	if level[2] != 1 && level[4] != 1 {
		t.Error("nodes 2 and 4 should be at level 1")
	}
}

func TestBFSReachable(t *testing.T) {
	g := createTestGraph()

	reachable := BFSReachable(g, 1)

	// All nodes should be reachable from source
	for i := int64(1); i <= 5; i++ {
		if !reachable[i] {
			t.Errorf("node %d should be reachable", i)
		}
	}
}

func TestBFSReverse(t *testing.T) {
	g := createTestGraph()

	reachable := BFSReverse(g, 5)

	// All nodes should reach sink
	for i := int64(1); i <= 5; i++ {
		if !reachable[i] {
			t.Errorf("node %d should reach sink", i)
		}
	}
}

func TestIsConnected(t *testing.T) {
	g := createTestGraph()

	if !IsConnected(g) {
		t.Error("graph should be connected")
	}

	// Create disconnected graph
	g2 := NewGraph()
	g2.SourceID = 1
	g2.SinkID = 3
	g2.AddNode(&Node{ID: 1})
	g2.AddNode(&Node{ID: 2})
	g2.AddNode(&Node{ID: 3})
	g2.AddEdge(&Edge{From: 1, To: 2, Capacity: 10})

	if IsConnected(g2) {
		t.Error("disconnected graph should return false")
	}
}

func TestFindConnectedComponents(t *testing.T) {
	g := NewGraph()

	// Component 1: 1-2
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddEdge(&Edge{From: 1, To: 2})

	// Component 2: 3-4
	g.AddNode(&Node{ID: 3})
	g.AddNode(&Node{ID: 4})
	g.AddEdge(&Edge{From: 3, To: 4})

	// Component 3: 5 (isolated)
	g.AddNode(&Node{ID: 5})

	components := FindConnectedComponents(g)

	if len(components) != 3 {
		t.Errorf("expected 3 components, got %d", len(components))
	}
}
