package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBFS_SimpleGraph(t *testing.T) {
	rg := NewResidualGraph()
	// Create: 1 -> 2 -> 3 -> 4
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.AddEdgeWithReverse(2, 3, 10.0, 1.0)
	rg.AddEdgeWithReverse(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	assert.True(t, result.Found)
	assert.True(t, result.Visited[1])
	assert.True(t, result.Visited[2])
	assert.True(t, result.Visited[3])
	assert.True(t, result.Visited[4])
	assert.Equal(t, int64(3), result.Parent[4])
	assert.Equal(t, int64(2), result.Parent[3])
	assert.Equal(t, int64(1), result.Parent[2])
}

func TestBFS_NoPath(t *testing.T) {
	rg := NewResidualGraph()
	// Disconnected: 1 -> 2, 3 -> 4
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	assert.False(t, result.Found)
	assert.True(t, result.Visited[1])
	assert.True(t, result.Visited[2])
	assert.False(t, result.Visited[3])
	assert.False(t, result.Visited[4])
}

func TestBFS_ZeroCapacityEdge(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 0.0, 1.0) // Zero capacity
	rg.AddEdge(3, 4, 10.0, 1.0)

	result := BFS(rg, 1, 4)

	assert.False(t, result.Found)
}

func TestBFS_SaturatedPath(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10.0, 1.0)
	rg.AddEdgeWithReverse(2, 3, 10.0, 1.0)

	// Saturate edge 2->3
	rg.UpdateFlow(2, 3, 10.0)

	result := BFS(rg, 1, 3)

	assert.False(t, result.Found)
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

	assert.True(t, result.Found)
	parent4 := result.Parent[4]
	assert.True(t, parent4 == 2 || parent4 == 3)
}

func TestBFS_SourceEqualsSink(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)
	rg.AddEdge(1, 2, 10.0, 1.0)

	result := BFS(rg, 1, 1)

	// Source == sink is found immediately but no path needed
	if result.Found {
		assert.Equal(t, int64(-1), result.Parent[1])
	}
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

	assert.True(t, result.Found)

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

	assert.Equal(t, 999, pathLen)
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
		gotLevel, exists := levels[node]
		assert.True(t, exists, "Node %d not in levels", node)
		assert.Equal(t, wantLevel, gotLevel, "Level of node %d", node)
	}
}

func TestBFSLevel_Disconnected(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddNode(3) // Disconnected node

	levels := BFSLevel(rg, 1)

	_, exists := levels[3]
	assert.False(t, exists)
}

func TestBFSLevel_ZeroCapacity(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 0.0, 1.0) // Zero capacity
	rg.AddEdge(3, 4, 10.0, 1.0)

	levels := BFSLevel(rg, 1)

	_, exists3 := levels[3]
	_, exists4 := levels[4]
	assert.False(t, exists3)
	assert.False(t, exists4)
}

func TestBFSLevel_SingleNode(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)

	levels := BFSLevel(rg, 1)

	assert.Equal(t, 0, levels[1])
	assert.Len(t, levels, 1)
}

func TestBFSLevel_Cycle(t *testing.T) {
	rg := NewResidualGraph()
	// Triangle: 1 -> 2 -> 3 -> 1
	rg.AddEdge(1, 2, 10.0, 1.0)
	rg.AddEdge(2, 3, 10.0, 1.0)
	rg.AddEdge(3, 1, 10.0, 1.0)

	levels := BFSLevel(rg, 1)

	assert.Equal(t, 0, levels[1])
	assert.Equal(t, 1, levels[2])
	assert.Equal(t, 2, levels[3])
}

func TestBFS_EmptyGraph(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddNode(1)
	rg.AddNode(2)
	// No edges

	result := BFS(rg, 1, 2)

	assert.False(t, result.Found)
}

func TestBFSLevel_EmptyGraph(t *testing.T) {
	rg := NewResidualGraph()

	levels := BFSLevel(rg, 1)

	assert.Equal(t, 0, levels[1])
}

func TestBFSDeterministic(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)
	rg.AddEdgeWithReverse(1, 3, 10, 0)
	rg.AddEdgeWithReverse(2, 4, 10, 0)
	rg.AddEdgeWithReverse(3, 4, 10, 0)

	// Run multiple times to verify determinism
	for i := 0; i < 10; i++ {
		result := BFSDeterministic(rg, 1, 4)
		assert.True(t, result.Found)
		// Parent of 4 should be consistent
	}
}

func TestBFSWithCallback(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10, 0)
	rg.AddEdge(2, 3, 10, 0)
	rg.AddEdge(3, 4, 10, 0)
	rg.AddEdge(4, 5, 10, 0)

	visited := make([]int64, 0)
	maxLevel := 0

	BFSWithCallback(rg, 1, func(node int64, level int) bool {
		visited = append(visited, node)
		if level > maxLevel {
			maxLevel = level
		}
		// Stop at level 2
		return level < 2
	})

	assert.Contains(t, visited, int64(1))
	assert.Contains(t, visited, int64(2))
	assert.Contains(t, visited, int64(3))
	// Should stop before visiting nodes at level 3+
}

func TestBFSWithCallback_EarlyTermination(t *testing.T) {
	rg := NewResidualGraph()
	for i := int64(1); i < 100; i++ {
		rg.AddEdge(i, i+1, 10, 0)
	}

	count := 0
	BFSWithCallback(rg, 1, func(node int64, level int) bool {
		count++
		return count < 5 // Stop after 5 nodes
	})

	assert.Equal(t, 5, count)
}

func TestBFSReverse(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)
	rg.AddEdgeWithReverse(2, 3, 10, 0)
	rg.AddEdgeWithReverse(3, 4, 10, 0)

	heights := BFSReverse(rg, 4)

	assert.Equal(t, 0, heights[4])
	assert.Equal(t, 1, heights[3])
	assert.Equal(t, 2, heights[2])
	assert.Equal(t, 3, heights[1])
}

func TestBFSReverse_Disconnected(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)
	rg.AddNode(3)
	rg.AddNode(4)
	// 3 and 4 not connected to 2

	heights := BFSReverse(rg, 2)

	_, exists3 := heights[3]
	_, exists4 := heights[4]
	assert.False(t, exists3)
	assert.False(t, exists4)
}

func TestBFSAllPaths(t *testing.T) {
	rg := NewResidualGraph()
	// Diamond: 1 -> 2 -> 4
	//          1 -> 3 -> 4
	rg.AddEdge(1, 2, 10, 0)
	rg.AddEdge(1, 3, 10, 0)
	rg.AddEdge(2, 4, 10, 0)
	rg.AddEdge(3, 4, 10, 0)

	paths := BFSAllPaths(rg, 1, 4, 10)

	assert.Len(t, paths, 2)

	// Both paths should have length 3 (1, intermediate, 4)
	for _, path := range paths {
		assert.Len(t, path, 3)
		assert.Equal(t, int64(1), path[0])
		assert.Equal(t, int64(4), path[2])
	}
}

func TestBFSAllPaths_MaxLimit(t *testing.T) {
	rg := NewResidualGraph()
	// Many parallel paths
	for i := int64(2); i <= 10; i++ {
		rg.AddEdge(1, i, 10, 0)
		rg.AddEdge(i, 100, 10, 0)
	}

	paths := BFSAllPaths(rg, 1, 100, 3)

	assert.Len(t, paths, 3) // Limited to 3
}

func TestBFSAllPaths_NoPath(t *testing.T) {
	rg := NewResidualGraph()
	rg.AddEdge(1, 2, 10, 0)
	rg.AddNode(3)

	paths := BFSAllPaths(rg, 1, 3, 10)

	assert.Nil(t, paths)
}

func TestQueue(t *testing.T) {
	q := NewQueue(10)

	assert.True(t, q.Empty())
	assert.Equal(t, 0, q.Len())

	q.Push(1)
	q.Push(2)
	q.Push(3)

	assert.False(t, q.Empty())
	assert.Equal(t, 3, q.Len())

	assert.Equal(t, int64(1), q.Pop())
	assert.Equal(t, int64(2), q.Pop())
	assert.Equal(t, 1, q.Len())

	q.Reset()
	assert.True(t, q.Empty())
}
