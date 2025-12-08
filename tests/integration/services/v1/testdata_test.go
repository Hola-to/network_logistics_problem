package v1_test

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
)

// CreateSimpleGraph creates a simple test graph for testing
func CreateSimpleGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 0,
		SinkId:   3,
		Nodes: []*commonv1.Node{
			{Id: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Source"},
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, Name: "Node1"},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, Name: "Node2"},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT, Name: "Sink"},
		},
		Edges: []*commonv1.Edge{
			{From: 0, To: 1, Capacity: 10, Cost: 1, RoadType: commonv1.RoadType_ROAD_TYPE_HIGHWAY},
			{From: 0, To: 2, Capacity: 10, Cost: 2, RoadType: commonv1.RoadType_ROAD_TYPE_PRIMARY},
			{From: 1, To: 2, Capacity: 5, Cost: 1, RoadType: commonv1.RoadType_ROAD_TYPE_LOCAL},
			{From: 1, To: 3, Capacity: 10, Cost: 3, RoadType: commonv1.RoadType_ROAD_TYPE_HIGHWAY},
			{From: 2, To: 3, Capacity: 10, Cost: 2, RoadType: commonv1.RoadType_ROAD_TYPE_PRIMARY},
		},
	}
}

// CreateLargeGraph creates a larger test graph
func CreateLargeGraph(nodeCount int) *commonv1.Graph {
	nodes := make([]*commonv1.Node, nodeCount)
	edges := make([]*commonv1.Edge, 0)

	// Source
	nodes[0] = &commonv1.Node{Id: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Source"}

	// Intermediate nodes
	for i := 1; i < nodeCount-1; i++ {
		nodes[i] = &commonv1.Node{
			Id:   int64(i),
			Type: commonv1.NodeType_NODE_TYPE_INTERSECTION,
			Name: "Node" + string(rune('A'+i-1)),
		}
	}

	// Sink
	nodes[nodeCount-1] = &commonv1.Node{
		Id:   int64(nodeCount - 1),
		Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT,
		Name: "Sink",
	}

	// Create edges in a layered pattern
	for i := 0; i < nodeCount-1; i++ {
		// Connect to next node
		edges = append(edges, &commonv1.Edge{
			From:     int64(i),
			To:       int64(i + 1),
			Capacity: 10,
			Cost:     1,
			RoadType: commonv1.RoadType_ROAD_TYPE_HIGHWAY,
		})

		// Some skip connections
		if i+2 < nodeCount {
			edges = append(edges, &commonv1.Edge{
				From:     int64(i),
				To:       int64(i + 2),
				Capacity: 5,
				Cost:     2,
				RoadType: commonv1.RoadType_ROAD_TYPE_PRIMARY,
			})
		}
	}

	return &commonv1.Graph{
		SourceId: 0,
		SinkId:   int64(nodeCount - 1),
		Nodes:    nodes,
		Edges:    edges,
	}
}

// CreateSolvedGraph creates a graph with flow already set
func CreateSolvedGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 0,
		SinkId:   3,
		Nodes: []*commonv1.Node{
			{Id: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Source"},
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, Name: "Node1"},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, Name: "Node2"},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT, Name: "Sink"},
		},
		Edges: []*commonv1.Edge{
			{From: 0, To: 1, Capacity: 10, Cost: 1, CurrentFlow: 10},
			{From: 0, To: 2, Capacity: 10, Cost: 2, CurrentFlow: 5},
			{From: 1, To: 2, Capacity: 5, Cost: 1, CurrentFlow: 0},
			{From: 1, To: 3, Capacity: 10, Cost: 3, CurrentFlow: 10},
			{From: 2, To: 3, Capacity: 10, Cost: 2, CurrentFlow: 5},
		},
	}
}

// CreateInvalidGraph creates an invalid graph for testing error cases
func CreateInvalidGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 0,
		SinkId:   99, // Non-existent sink
		Nodes: []*commonv1.Node{
			{Id: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Source"},
		},
		Edges: []*commonv1.Edge{},
	}
}

// CreateDisconnectedGraph creates a graph with disconnected components
func CreateDisconnectedGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 0,
		SinkId:   3,
		Nodes: []*commonv1.Node{
			{Id: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Source"},
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, Name: "Node1"},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, Name: "Node2"},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT, Name: "Sink"},
		},
		Edges: []*commonv1.Edge{
			{From: 0, To: 1, Capacity: 10, Cost: 1},
			// Missing connection to sink
		},
	}
}

// CreateFlowResult creates a sample flow result
func CreateFlowResult() *commonv1.FlowResult {
	return &commonv1.FlowResult{
		MaxFlow:   15,
		TotalCost: 45,
		Status:    commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		Edges: []*commonv1.FlowEdge{
			{From: 0, To: 1, Flow: 10, Capacity: 10},
			{From: 0, To: 2, Flow: 5, Capacity: 10},
			{From: 1, To: 3, Flow: 10, Capacity: 10},
			{From: 2, To: 3, Flow: 5, Capacity: 10},
		},
	}
}
