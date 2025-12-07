package validators

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
)

func TestValidateStructure(t *testing.T) {
	tests := []struct {
		name       string
		graph      *commonv1.Graph
		wantErrors int
		wantCodes  []string
	}{
		{
			name:       "valid_graph",
			graph:      createValidGraph(),
			wantErrors: 0,
			wantCodes:  nil,
		},
		{
			name: "empty_graph",
			graph: &commonv1.Graph{
				Nodes:    []*commonv1.Node{},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeEmptyGraph)},
		},
		{
			name: "duplicate_node_id",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeDuplicateNode)},
		},
		{
			name: "invalid_source",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
				SourceId: 999,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeInvalidSource)},
		},
		{
			name: "invalid_sink",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
				SourceId: 1,
				SinkId:   999,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeInvalidSink)},
		},
		{
			name: "source_equals_sink",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
				},
				SourceId: 1,
				SinkId:   1,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeSourceEqualsSink)},
		},
		{
			name: "dangling_edge_from",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 999, To: 2, Capacity: 10}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeDanglingEdge)},
		},
		{
			name: "dangling_edge_to",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 999, Capacity: 10}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeDanglingEdge)},
		},
		{
			name: "self_loop",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 1, Capacity: 10}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeSelfLoop)},
		},
		{
			name: "invalid_capacity",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 0}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeInvalidCapacity)},
		},
		{
			name: "negative_cost",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10, Cost: -5}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeNegativeCost)},
		},
		{
			name: "negative_length",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10, Length: -5}},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 1,
			wantCodes:  []string{string(pkgerrors.CodeNegativeLength)},
		},
		{
			name: "multiple_errors",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
				},
				Edges: []*commonv1.Edge{
					{From: 1, To: 1, Capacity: -10, Cost: -5}, // self loop + invalid capacity + negative cost
				},
				SourceId: 1,
				SinkId:   2,
			},
			wantErrors: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateStructure(tt.graph)

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

func TestValidateStructure_LargeGraph(t *testing.T) {
	// Create a large valid graph
	nodes := make([]*commonv1.Node, 1000)
	edges := make([]*commonv1.Edge, 999)

	nodes[0] = &commonv1.Node{Id: 0, Type: commonv1.NodeType_NODE_TYPE_SOURCE}
	for i := 1; i < 999; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i), Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE}
		edges[i-1] = &commonv1.Edge{From: int64(i - 1), To: int64(i), Capacity: 10}
	}
	nodes[999] = &commonv1.Node{Id: 999, Type: commonv1.NodeType_NODE_TYPE_SINK}
	edges[998] = &commonv1.Edge{From: 998, To: 999, Capacity: 10}

	graph := &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   999,
	}

	errors := ValidateStructure(graph)

	if len(errors) != 0 {
		t.Errorf("expected no errors for large valid graph, got: %+v", errors)
	}
}
