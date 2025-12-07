package analysis

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestCalculateFlowStatistics(t *testing.T) {
	tests := []struct {
		name         string
		graph        *commonv1.Graph
		expectedFlow float64
		expectedCost float64
		expectedSat  int64
		expectedZero int64
	}{
		{
			name: "basic statistics",
			graph: &commonv1.Graph{
				SourceId: 1,
				SinkId:   3,
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 50, Cost: 2, Capacity: 100},
					{From: 2, To: 3, CurrentFlow: 50, Cost: 3, Capacity: 50}, // saturated
				},
			},
			expectedFlow: 50,  // flow from source
			expectedCost: 250, // 50*2 + 50*3
			expectedSat:  1,
			expectedZero: 0,
		},
		{
			name: "with zero flow edges",
			graph: &commonv1.Graph{
				SourceId: 1,
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 50, Cost: 2, Capacity: 100},
					{From: 3, To: 4, CurrentFlow: 0, Cost: 3, Capacity: 50},
				},
			},
			expectedFlow: 50,
			expectedZero: 1,
		},
		{
			name: "ignores virtual nodes",
			graph: &commonv1.Graph{
				SourceId: 1,
				Edges: []*commonv1.Edge{
					{From: -1, To: 1, CurrentFlow: 100, Cost: 1, Capacity: 100},
					{From: 1, To: 2, CurrentFlow: 50, Cost: 2, Capacity: 100},
				},
			},
			expectedFlow: 50, // only non-virtual source edges
			expectedCost: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFlowStatistics(tt.graph)

			if !floatEquals(result.TotalFlow, tt.expectedFlow, 0.01) {
				t.Errorf("TotalFlow = %v, want %v", result.TotalFlow, tt.expectedFlow)
			}

			if tt.expectedCost > 0 && !floatEquals(result.TotalCost, tt.expectedCost, 0.01) {
				t.Errorf("TotalCost = %v, want %v", result.TotalCost, tt.expectedCost)
			}

			if result.SaturatedEdges != tt.expectedSat {
				t.Errorf("SaturatedEdges = %v, want %v", result.SaturatedEdges, tt.expectedSat)
			}

			if result.ZeroFlowEdges != tt.expectedZero {
				t.Errorf("ZeroFlowEdges = %v, want %v", result.ZeroFlowEdges, tt.expectedZero)
			}
		})
	}
}

func TestCalculateGraphStatistics(t *testing.T) {
	tests := []struct {
		name              string
		graph             *commonv1.Graph
		expectedNodes     int64
		expectedEdges     int64
		expectedWarehouse int64
		expectedDelivery  int64
	}{
		{
			name: "basic graph statistics",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION},
					{Id: 3, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
					{Id: 4, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
				},
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 100, Length: 10},
					{From: 2, To: 3, Capacity: 50, Length: 20},
					{From: 2, To: 4, Capacity: 50, Length: 15},
				},
			},
			expectedNodes:     4,
			expectedEdges:     3,
			expectedWarehouse: 1,
			expectedDelivery:  2,
		},
		{
			name: "excludes virtual edges",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
				},
				Edges: []*commonv1.Edge{
					{From: -1, To: 1, Capacity: 100},
					{From: 1, To: 2, Capacity: 50},
					{From: 2, To: -2, Capacity: 100},
				},
			},
			expectedNodes:     2,
			expectedEdges:     1,
			expectedWarehouse: 1,
			expectedDelivery:  1,
		},
		{
			name: "empty graph",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{},
				Edges: []*commonv1.Edge{},
			},
			expectedNodes:     0,
			expectedEdges:     0,
			expectedWarehouse: 0,
			expectedDelivery:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateGraphStatistics(tt.graph)

			if result.NodeCount != tt.expectedNodes {
				t.Errorf("NodeCount = %v, want %v", result.NodeCount, tt.expectedNodes)
			}

			if result.EdgeCount != tt.expectedEdges {
				t.Errorf("EdgeCount = %v, want %v", result.EdgeCount, tt.expectedEdges)
			}

			if result.WarehouseCount != tt.expectedWarehouse {
				t.Errorf("WarehouseCount = %v, want %v",
					result.WarehouseCount, tt.expectedWarehouse)
			}

			if result.DeliveryPointCount != tt.expectedDelivery {
				t.Errorf("DeliveryPointCount = %v, want %v",
					result.DeliveryPointCount, tt.expectedDelivery)
			}
		})
	}
}

func TestCalculateGraphDensity(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2},
			{From: 2, To: 3},
			{From: 3, To: 4},
		},
	}

	result := CalculateGraphStatistics(graph)

	// 4 nodes, max edges = 4*3 = 12, actual = 3
	// density = 3/12 = 0.25
	expectedDensity := 0.25
	if !floatEquals(result.Density, expectedDensity, 0.01) {
		t.Errorf("Density = %v, want %v", result.Density, expectedDensity)
	}
}

func TestAverageEdgeLength(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Length: 10},
			{From: 2, To: 3, Length: 20},
		},
	}

	result := CalculateGraphStatistics(graph)

	expectedAvg := 15.0
	if !floatEquals(result.AverageEdgeLength, expectedAvg, 0.01) {
		t.Errorf("AverageEdgeLength = %v, want %v",
			result.AverageEdgeLength, expectedAvg)
	}
}

func TestAverageUtilization(t *testing.T) {
	graph := &commonv1.Graph{
		SourceId: 1,
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 50, Capacity: 100}, // 50%
			{From: 2, To: 3, CurrentFlow: 80, Capacity: 100}, // 80%
		},
	}

	result := CalculateFlowStatistics(graph)

	expectedAvg := 0.65 // (0.5 + 0.8) / 2
	if !floatEquals(result.AverageUtilization, expectedAvg, 0.01) {
		t.Errorf("AverageUtilization = %v, want %v",
			result.AverageUtilization, expectedAvg)
	}
}

func TestCalculateFlowStatistics_MultipleSources(t *testing.T) {
	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 30, Cost: 1, Capacity: 50},
			{From: 1, To: 3, CurrentFlow: 20, Cost: 1, Capacity: 50},
			{From: 2, To: 4, CurrentFlow: 30, Cost: 1, Capacity: 50},
			{From: 3, To: 4, CurrentFlow: 20, Cost: 1, Capacity: 50},
		},
	}

	result := CalculateFlowStatistics(graph)

	// Total flow from source: 30 + 20 = 50
	if result.TotalFlow != 50 {
		t.Errorf("TotalFlow = %v, want 50", result.TotalFlow)
	}
}

func TestCalculateGraphStatistics_OnlyVirtualEdges(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
		},
		Edges: []*commonv1.Edge{
			{From: -1, To: 1, Capacity: 100},
			{From: 1, To: -2, Capacity: 100},
		},
	}

	result := CalculateGraphStatistics(graph)

	if result.EdgeCount != 0 {
		t.Errorf("EdgeCount = %v, want 0 (virtual edges excluded)", result.EdgeCount)
	}
	if result.NodeCount != 1 {
		t.Errorf("NodeCount = %v, want 1", result.NodeCount)
	}
}

func TestCalculateFlowStatistics_SaturatedEdgesWithBottlenecks(t *testing.T) {
	graph := &commonv1.Graph{
		SourceId: 1,
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 100, Cost: 1, Capacity: 100}, // saturated
			{From: 2, To: 3, CurrentFlow: 100, Cost: 1, Capacity: 100}, // saturated
			{From: 3, To: 4, CurrentFlow: 50, Cost: 1, Capacity: 100},  // not saturated
		},
	}

	result := CalculateFlowStatistics(graph)

	if result.SaturatedEdges != 2 {
		t.Errorf("SaturatedEdges = %v, want 2", result.SaturatedEdges)
	}

	if len(result.Bottlenecks) != 2 {
		t.Errorf("Bottlenecks count = %v, want 2", len(result.Bottlenecks))
	}
}
