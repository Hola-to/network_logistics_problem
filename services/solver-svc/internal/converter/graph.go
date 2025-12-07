package converter

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"
)

// ToResidualGraph конвертирует proto Graph в ResidualGraph
func ToResidualGraph(protoGraph *commonv1.Graph) *graph.ResidualGraph {
	rg := graph.NewResidualGraph()

	for _, node := range protoGraph.Nodes {
		rg.AddNode(node.Id)
	}

	for _, edge := range protoGraph.Edges {
		rg.AddEdgeWithReverse(edge.From, edge.To, edge.Capacity, edge.Cost)

		if edge.Bidirectional {
			rg.AddEdgeWithReverse(edge.To, edge.From, edge.Capacity, edge.Cost)
		}
	}

	return rg
}

// ToFlowEdges конвертирует результат в proto FlowEdge
func ToFlowEdges(rg *graph.ResidualGraph) []*commonv1.FlowEdge {
	var result []*commonv1.FlowEdge

	for from, edges := range rg.Edges {
		for _, edge := range edges {
			if edge.IsReverse || edge.Flow < graph.Epsilon {
				continue
			}

			utilization := 0.0
			if edge.OriginalCapacity > 0 {
				utilization = edge.Flow / edge.OriginalCapacity
			}

			result = append(result, &commonv1.FlowEdge{
				From:        from,
				To:          edge.To,
				Flow:        edge.Flow,
				Capacity:    edge.OriginalCapacity,
				Cost:        edge.Cost,
				Utilization: utilization,
			})
		}
	}

	return result
}

// ToPaths конвертирует пути в proto Path
func ToPaths(paths [][]int64, rg *graph.ResidualGraph) []*commonv1.Path {
	result := make([]*commonv1.Path, 0, len(paths))

	for _, path := range paths {
		if len(path) < 2 {
			continue
		}

		var totalCost, flow float64
		flow = graph.Infinity

		for i := 0; i < len(path)-1; i++ {
			edge := rg.GetEdge(path[i], path[i+1])
			if edge != nil {
				totalCost += edge.Cost
				if edge.Flow < flow {
					flow = edge.Flow
				}
			}
		}

		if flow == graph.Infinity {
			flow = 0
		}

		result = append(result, &commonv1.Path{
			NodeIds: path,
			Flow:    flow,
			Cost:    totalCost * flow,
		})
	}

	return result
}

// UpdateGraphWithFlow обновляет proto Graph результатами расчёта
func UpdateGraphWithFlow(protoGraph *commonv1.Graph, rg *graph.ResidualGraph) *commonv1.Graph {
	result := &commonv1.Graph{
		Nodes:    protoGraph.Nodes,
		Edges:    make([]*commonv1.Edge, len(protoGraph.Edges)),
		SourceId: protoGraph.SourceId,
		SinkId:   protoGraph.SinkId,
		Name:     protoGraph.Name,
		Metadata: protoGraph.Metadata,
	}

	for i, edge := range protoGraph.Edges {
		newEdge := &commonv1.Edge{
			From:          edge.From,
			To:            edge.To,
			Capacity:      edge.Capacity,
			Cost:          edge.Cost,
			Length:        edge.Length,
			RoadType:      edge.RoadType,
			Bidirectional: edge.Bidirectional,
		}

		if re := rg.GetEdge(edge.From, edge.To); re != nil {
			newEdge.CurrentFlow = re.Flow
		}

		result.Edges[i] = newEdge
	}

	return result
}

// CalculateGraphStatistics вычисляет статистику графа
func CalculateGraphStatistics(protoGraph *commonv1.Graph) *commonv1.GraphStatistics {
	var warehouseCount, deliveryPointCount int64
	var totalCapacity, totalLength float64

	for _, node := range protoGraph.Nodes {
		switch node.Type {
		case commonv1.NodeType_NODE_TYPE_WAREHOUSE:
			warehouseCount++
		case commonv1.NodeType_NODE_TYPE_DELIVERY_POINT:
			deliveryPointCount++
		}
	}

	for _, edge := range protoGraph.Edges {
		totalCapacity += edge.Capacity
		totalLength += edge.Length
	}

	nodeCount := int64(len(protoGraph.Nodes))
	edgeCount := int64(len(protoGraph.Edges))

	avgLength := 0.0
	if edgeCount > 0 {
		avgLength = totalLength / float64(edgeCount)
	}

	density := 0.0
	if nodeCount > 1 {
		maxEdges := nodeCount * (nodeCount - 1)
		density = float64(edgeCount) / float64(maxEdges)
	}

	return &commonv1.GraphStatistics{
		NodeCount:          nodeCount,
		EdgeCount:          edgeCount,
		WarehouseCount:     warehouseCount,
		DeliveryPointCount: deliveryPointCount,
		TotalCapacity:      totalCapacity,
		AverageEdgeLength:  avgLength,
		IsConnected:        true,
		Density:            density,
	}
}
