package analysis

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestIsVirtualNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeID   int64
		expected bool
	}{
		{
			name:     "positive node is not virtual",
			nodeID:   1,
			expected: false,
		},
		{
			name:     "zero node is not virtual",
			nodeID:   0,
			expected: false,
		},
		{
			name:     "negative node is virtual",
			nodeID:   -1,
			expected: true,
		},
		{
			name:     "large negative node is virtual",
			nodeID:   -1000000,
			expected: true,
		},
		{
			name:     "large positive node is not virtual",
			nodeID:   1000000,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVirtualNode(tt.nodeID)
			if result != tt.expected {
				t.Errorf("IsVirtualNode(%d) = %v, want %v", tt.nodeID, result, tt.expected)
			}
		})
	}
}

func TestCalculateUtilization(t *testing.T) {
	tests := []struct {
		name     string
		flow     float64
		capacity float64
		expected float64
	}{
		{
			name:     "zero capacity returns zero",
			flow:     100,
			capacity: 0,
			expected: 0,
		},
		{
			name:     "very small capacity returns zero",
			flow:     100,
			capacity: 1e-10,
			expected: 0,
		},
		{
			name:     "half utilization",
			flow:     50,
			capacity: 100,
			expected: 0.5,
		},
		{
			name:     "full utilization",
			flow:     100,
			capacity: 100,
			expected: 1.0,
		},
		{
			name:     "over utilization",
			flow:     150,
			capacity: 100,
			expected: 1.5,
		},
		{
			name:     "zero flow",
			flow:     0,
			capacity: 100,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateUtilization(tt.flow, tt.capacity)
			if !floatEquals(result, tt.expected, 0.0001) {
				t.Errorf("CalculateUtilization(%v, %v) = %v, want %v",
					tt.flow, tt.capacity, result, tt.expected)
			}
		})
	}
}

func TestBuildEdgeMap(t *testing.T) {
	graph := &commonv1.Graph{
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 100},
			{From: 1, To: 3, Capacity: 50},
			{From: 2, To: 3, Capacity: 75},
		},
	}

	edgeMap := BuildEdgeMap(graph)

	// Проверяем наличие рёбер
	if edgeMap[1][2] == nil {
		t.Error("Expected edge 1->2 to exist")
	}
	if edgeMap[1][3] == nil {
		t.Error("Expected edge 1->3 to exist")
	}
	if edgeMap[2][3] == nil {
		t.Error("Expected edge 2->3 to exist")
	}

	// Проверяем отсутствие несуществующих рёбер
	if edgeMap[3][1] != nil {
		t.Error("Expected edge 3->1 to not exist")
	}

	// Проверяем значения
	if edgeMap[1][2].Capacity != 100 {
		t.Errorf("Edge 1->2 capacity = %v, want 100", edgeMap[1][2].Capacity)
	}
}

func TestGetEdge(t *testing.T) {
	edgeMap := map[int64]map[int64]*commonv1.Edge{
		1: {
			2: {From: 1, To: 2, Capacity: 100},
			3: {From: 1, To: 3, Capacity: 50},
		},
	}

	tests := []struct {
		name     string
		from     int64
		to       int64
		expected bool
	}{
		{
			name:     "existing edge",
			from:     1,
			to:       2,
			expected: true,
		},
		{
			name:     "non-existing edge - wrong to",
			from:     1,
			to:       4,
			expected: false,
		},
		{
			name:     "non-existing edge - wrong from",
			from:     2,
			to:       3,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := GetEdge(edgeMap, tt.from, tt.to)
			if (edge != nil) != tt.expected {
				t.Errorf("GetEdge(%d, %d) exists = %v, want %v",
					tt.from, tt.to, edge != nil, tt.expected)
			}
		})
	}
}

func TestGetNodesByType(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION},
			{Id: 5, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
		},
	}

	tests := []struct {
		name        string
		nodeType    commonv1.NodeType
		expectedIDs []int64
		expectedLen int
	}{
		{
			name:        "warehouses",
			nodeType:    commonv1.NodeType_NODE_TYPE_WAREHOUSE,
			expectedIDs: []int64{1, 3},
			expectedLen: 2,
		},
		{
			name:        "delivery points",
			nodeType:    commonv1.NodeType_NODE_TYPE_DELIVERY_POINT,
			expectedIDs: []int64{2, 5},
			expectedLen: 2,
		},
		{
			name:        "intersections",
			nodeType:    commonv1.NodeType_NODE_TYPE_INTERSECTION,
			expectedIDs: []int64{4},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNodesByType(graph, tt.nodeType)
			if len(result) != tt.expectedLen {
				t.Errorf("GetNodesByType() returned %d nodes, want %d",
					len(result), tt.expectedLen)
			}
		})
	}
}

// Helper function
func floatEquals(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}
