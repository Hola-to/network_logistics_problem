package validators

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
)

func TestValidateFlowLogic(t *testing.T) {
	tests := []struct {
		name       string
		graph      *commonv1.Graph
		wantErrors int
		wantCodes  []string
	}{
		{
			name:       "valid_flow",
			graph:      createValidFlowGraph(),
			wantErrors: 0,
		},
		{
			name:       "negative_flow",
			graph:      createNegativeFlowGraph(),
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeNegativeFlow)},
		},
		{
			name:       "capacity_overflow",
			graph:      createCapacityOverflowGraph(),
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeCapacityOverflow)},
		},
		{
			name:       "conservation_violation",
			graph:      createConservationViolationGraph(),
			wantErrors: 2, // Node imbalance triggers both conservation violation and global flow mismatch
			wantCodes: []string{
				string(pkgerrors.CodeConservationViolation),
				string(pkgerrors.CodeFlowImbalance),
			},
		},
		{
			name:       "flow_imbalance_only",
			graph:      createFlowImbalanceOnlyGraph(),
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeFlowImbalance)},
		},
		{
			name:       "zero_flow",
			graph:      createZeroFlowGraph(),
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateFlowLogic(tt.graph)

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
						t.Errorf("expected error code %s not found in errors: %+v", wantCode, errors)
					}
				}
			}
		})
	}
}

// createFlowImbalanceOnlyGraph creates a graph where flow out of source != flow into sink,
// but intermediate conservation checks are bypassed (using a secondary sink node).
func createFlowImbalanceOnlyGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
			// "Leak" node: Typed as SINK so conservation check skips it,
			// but it absorbs flow, causing global imbalance.
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 5},
			{From: 1, To: 3, Capacity: 10, CurrentFlow: 5},
		},
		SourceId: 1,
		SinkId:   2,
	}
}

func createValidFlowGraph() *commonv1.Graph {
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

func createNegativeFlowGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: -5},
		},
		SourceId: 1,
		SinkId:   2,
	}
}

func createCapacityOverflowGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 15},
		},
		SourceId: 1,
		SinkId:   2,
	}
}

func createConservationViolationGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE}, // intermediate node
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 10}, // 10 in
			{From: 2, To: 3, Capacity: 10, CurrentFlow: 5},  // 5 out - imbalance at node 2
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createZeroFlowGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 0},
			{From: 2, To: 3, Capacity: 10, CurrentFlow: 0},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func TestValidateFlowLogic_ComplexGraph(t *testing.T) {
	// Diamond-shaped graph with valid flow
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 5},
			{From: 1, To: 3, Capacity: 10, CurrentFlow: 5},
			{From: 2, To: 4, Capacity: 10, CurrentFlow: 5},
			{From: 3, To: 4, Capacity: 10, CurrentFlow: 5},
		},
		SourceId: 1,
		SinkId:   4,
	}

	errors := ValidateFlowLogic(graph)

	if len(errors) != 0 {
		t.Errorf("expected no errors for valid complex flow, got: %+v", errors)
	}
}
