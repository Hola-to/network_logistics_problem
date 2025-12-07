package validators

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
)

func TestValidateConnectivity(t *testing.T) {
	tests := []struct {
		name       string
		graph      *commonv1.Graph
		wantErrors int
	}{
		{
			name:       "connected_graph",
			graph:      createConnectedGraph(),
			wantErrors: 0,
		},
		{
			name:       "disconnected_graph",
			graph:      createDisconnectedGraph(),
			wantErrors: 1,
		},
		{
			name:       "single_edge",
			graph:      createSingleEdgeGraph(),
			wantErrors: 0,
		},
		{
			name:       "linear_path",
			graph:      createLinearPathGraph(),
			wantErrors: 0,
		},
		{
			name:       "source_isolated",
			graph:      createSourceIsolatedGraph(),
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateConnectivity(tt.graph)

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d: %+v", len(errors), tt.wantErrors, errors)
			}

			if tt.wantErrors > 0 {
				found := false
				for _, err := range errors {
					if err.Code == string(pkgerrors.CodeNoPath) {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected NoPath error code")
				}
			}
		})
	}
}

func createConnectedGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 1, To: 3, Capacity: 10},
			{From: 2, To: 4, Capacity: 10},
			{From: 3, To: 4, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   4,
	}
}

func createDisconnectedGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10}, // Component 1
			{From: 3, To: 4, Capacity: 10}, // Component 2 - disconnected
		},
		SourceId: 1,
		SinkId:   4,
	}
}

func createSingleEdgeGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   2,
	}
}

func createLinearPathGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 5, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 2, To: 3, Capacity: 10},
			{From: 3, To: 4, Capacity: 10},
			{From: 4, To: 5, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   5,
	}
}

func createSourceIsolatedGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE}, // No edges
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 2, To: 3, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func TestConvertToDomainGraph(t *testing.T) {
	protoGraph := &commonv1.Graph{
		Name: "TestGraph",
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE, Name: "Source", X: 0, Y: 0},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Warehouse", X: 1, Y: 1},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK, Name: "Sink", X: 2, Y: 2},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 5, Length: 100, RoadType: commonv1.RoadType_ROAD_TYPE_HIGHWAY},
			{From: 2, To: 3, Capacity: 15, Cost: 3, Length: 50, RoadType: commonv1.RoadType_ROAD_TYPE_PRIMARY},
		},
		SourceId: 1,
		SinkId:   3,
	}

	domainGraph := convertToDomainGraph(protoGraph)

	if domainGraph.Name != protoGraph.Name {
		t.Errorf("Name mismatch: got %s, want %s", domainGraph.Name, protoGraph.Name)
	}

	if domainGraph.SourceID != protoGraph.SourceId {
		t.Errorf("SourceID mismatch: got %d, want %d", domainGraph.SourceID, protoGraph.SourceId)
	}

	if domainGraph.SinkID != protoGraph.SinkId {
		t.Errorf("SinkID mismatch: got %d, want %d", domainGraph.SinkID, protoGraph.SinkId)
	}

	if len(domainGraph.Nodes) != len(protoGraph.Nodes) {
		t.Errorf("Node count mismatch: got %d, want %d", len(domainGraph.Nodes), len(protoGraph.Nodes))
	}

	if len(domainGraph.Edges) != len(protoGraph.Edges) {
		t.Errorf("Edge count mismatch: got %d, want %d", len(domainGraph.Edges), len(protoGraph.Edges))
	}
}
