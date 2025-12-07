package validators

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
)

func TestValidateBusinessRules(t *testing.T) {
	tests := []struct {
		name       string
		graph      *commonv1.Graph
		wantErrors int
		wantCodes  []string
	}{
		{
			name:       "valid_graph",
			graph:      createValidBusinessGraph(),
			wantErrors: 0,
		},
		{
			name:       "warehouse_deadend",
			graph:      createWarehouseDeadendGraph(),
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeIsolatedWarehouse)},
		},
		{
			name:       "unreachable_delivery_point",
			graph:      createUnreachableDeliveryGraph(),
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeUnreachableDelivery)},
		},
		{
			name:       "warehouse_as_sink_is_ok",
			graph:      createWarehouseAsSinkGraph(),
			wantErrors: 0,
		},
		{
			name:       "delivery_point_as_source_is_ok",
			graph:      createDeliveryAsSourceGraph(),
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateBusinessRules(tt.graph)

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d: %+v", len(errors), tt.wantErrors, errors)
			}

			if tt.wantCodes != nil {
				for _, wantCode := range tt.wantCodes {
					found := false
					for _, err := range errors {
						if err.Code == wantCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error code %s not found", wantCode)
					}
				}
			}
		})
	}
}

func createValidBusinessGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 2, To: 3, Capacity: 10},
			{From: 3, To: 4, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   4,
	}
}

func createWarehouseDeadendGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE}, // deadend - no outgoing edges
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 1, To: 3, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createUnreachableDeliveryGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT}, // no incoming edges
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 3, Capacity: 10},
			{From: 2, To: 3, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createWarehouseAsSinkGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
		},
		SourceId: 1,
		SinkId:   2, // Warehouse is sink - should be OK
	}
}

func createDeliveryAsSourceGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
		},
		SourceId: 1, // Delivery point is source - should be OK
		SinkId:   2,
	}
}
