package validators

import (
	"math"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestCalculateGraphStatistics(t *testing.T) {
	tests := []struct {
		name          string
		graph         *commonv1.Graph
		wantNodeCount int64
		wantEdgeCount int64
		wantConnected bool
	}{
		{
			name:          "nil_graph",
			graph:         nil,
			wantNodeCount: 0,
			wantEdgeCount: 0,
		},
		{
			name:          "simple_graph",
			graph:         createStatsTestGraph(),
			wantNodeCount: 4,
			wantEdgeCount: 3,
			wantConnected: true,
		},
		{
			name:          "empty_graph",
			graph:         &commonv1.Graph{},
			wantNodeCount: 0,
			wantEdgeCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := CalculateGraphStatistics(tt.graph)

			if tt.graph == nil {
				if stats != nil {
					t.Error("expected nil stats for nil graph")
				}
				return
			}

			if stats.NodeCount != tt.wantNodeCount {
				t.Errorf("NodeCount = %d, want %d", stats.NodeCount, tt.wantNodeCount)
			}

			if stats.EdgeCount != tt.wantEdgeCount {
				t.Errorf("EdgeCount = %d, want %d", stats.EdgeCount, tt.wantEdgeCount)
			}
		})
	}
}

func TestCalculateGraphStatistics_NodeTypeCounts(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 5, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 6, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 7, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   7,
	}

	stats := CalculateGraphStatistics(graph)

	if stats.WarehouseCount != 2 {
		t.Errorf("WarehouseCount = %d, want 2", stats.WarehouseCount)
	}

	if stats.DeliveryPointCount != 3 {
		t.Errorf("DeliveryPointCount = %d, want 3", stats.DeliveryPointCount)
	}
}

func TestCalculateGraphStatistics_Capacity(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Length: 100},
		},
		SourceId: 1,
		SinkId:   2,
	}

	stats := CalculateGraphStatistics(graph)

	if stats.TotalCapacity != 10 {
		t.Errorf("TotalCapacity = %f, want 10", stats.TotalCapacity)
	}

	if stats.AverageEdgeLength != 100 {
		t.Errorf("AverageEdgeLength = %f, want 100", stats.AverageEdgeLength)
	}
}

func TestCalculateGraphStatistics_Density(t *testing.T) {
	// Complete graph with 3 nodes (3 edges possible, 3 actual)
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 1, To: 3, Capacity: 10},
			{From: 2, To: 3, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   3,
	}

	stats := CalculateGraphStatistics(graph)

	// Density = edges / (nodes * (nodes - 1)) = 3 / 6 = 0.5
	expectedDensity := 0.5
	if math.Abs(stats.Density-expectedDensity) > 0.001 {
		t.Errorf("Density = %f, want %f", stats.Density, expectedDensity)
	}
}

func TestCalculateFlowSummary(t *testing.T) {
	tests := []struct {
		name              string
		graph             *commonv1.Graph
		wantTotalFlow     float64
		wantEdgesWithFlow int32
		wantSaturated     int32
	}{
		{
			name:              "nil_graph",
			graph:             nil,
			wantTotalFlow:     0,
			wantEdgesWithFlow: 0,
		},
		{
			name:              "simple_flow",
			graph:             createSimpleFlowGraph(),
			wantTotalFlow:     5,
			wantEdgesWithFlow: 2,
			wantSaturated:     0,
		},
		{
			name:              "saturated_edges",
			graph:             createSaturatedFlowGraph(),
			wantTotalFlow:     10,
			wantEdgesWithFlow: 2,
			wantSaturated:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := CalculateFlowSummary(tt.graph)

			if tt.graph == nil {
				if summary != nil {
					t.Error("expected nil summary for nil graph")
				}
				return
			}

			if math.Abs(summary.TotalFlow-tt.wantTotalFlow) > Epsilon {
				t.Errorf("TotalFlow = %f, want %f", summary.TotalFlow, tt.wantTotalFlow)
			}

			if summary.EdgesWithFlow != tt.wantEdgesWithFlow {
				t.Errorf("EdgesWithFlow = %d, want %d", summary.EdgesWithFlow, tt.wantEdgesWithFlow)
			}

			if summary.SaturatedEdges != tt.wantSaturated {
				t.Errorf("SaturatedEdges = %d, want %d", summary.SaturatedEdges, tt.wantSaturated)
			}
		})
	}
}

func TestCalculateFlowSummary_SourceAndSink(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 7, Cost: 2},
			{From: 2, To: 3, Capacity: 10, CurrentFlow: 7, Cost: 3},
		},
		SourceId: 1,
		SinkId:   3,
	}

	summary := CalculateFlowSummary(graph)

	if summary.SourceOutflow != 7 {
		t.Errorf("SourceOutflow = %f, want 7", summary.SourceOutflow)
	}

	if summary.SinkInflow != 7 {
		t.Errorf("SinkInflow = %f, want 7", summary.SinkInflow)
	}

	// TotalCost = 7*2 + 7*3 = 14 + 21 = 35
	expectedCost := 35.0
	if math.Abs(summary.TotalCost-expectedCost) > Epsilon {
		t.Errorf("TotalCost = %f, want %f", summary.TotalCost, expectedCost)
	}
}

// Helper functions
func createStatsTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Length: 50},
			{From: 2, To: 3, Capacity: 10, Length: 30},
			{From: 3, To: 4, Capacity: 10, Length: 20},
		},
		SourceId: 1,
		SinkId:   4,
	}
}

func createSimpleFlowGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 5},
			{From: 2, To: 3, Capacity: 10, CurrentFlow: 5},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createSaturatedFlowGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 10}, // 100% utilization
			{From: 2, To: 3, Capacity: 10, CurrentFlow: 10}, // 100% utilization
		},
		SourceId: 1,
		SinkId:   3,
	}
}
