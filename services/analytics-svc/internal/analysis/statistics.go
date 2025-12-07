package analysis

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
)

// CalculateFlowStatistics вычисляет статистику потока
func CalculateFlowStatistics(graph *commonv1.Graph) *commonv1.FlowStatistics {
	var (
		totalFlow        float64
		totalCost        float64
		totalUtilization float64
		saturatedEdges   int64
		zeroFlowEdges    int64
		activeEdgesCount int
		bottlenecks      []*commonv1.EdgeKey
	)

	for _, edge := range graph.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}

		if edge.CurrentFlow <= Epsilon {
			zeroFlowEdges++
			continue
		}

		if edge.From == graph.SourceId {
			totalFlow += edge.CurrentFlow
		}

		activeEdgesCount++
		totalCost += edge.CurrentFlow * edge.Cost

		utilization := CalculateUtilization(edge.CurrentFlow, edge.Capacity)
		totalUtilization += utilization

		if utilization >= 1.0-Epsilon {
			saturatedEdges++
			bottlenecks = append(bottlenecks, &commonv1.EdgeKey{
				From: edge.From,
				To:   edge.To,
			})
		}
	}

	avgUtilization := 0.0
	if activeEdgesCount > 0 {
		avgUtilization = totalUtilization / float64(activeEdgesCount)
	}

	return &commonv1.FlowStatistics{
		TotalFlow:          totalFlow,
		TotalCost:          totalCost,
		AverageUtilization: avgUtilization,
		SaturatedEdges:     saturatedEdges,
		ZeroFlowEdges:      zeroFlowEdges,
		Bottlenecks:        bottlenecks,
	}
}

// CalculateGraphStatistics вычисляет статистику графа
func CalculateGraphStatistics(graph *commonv1.Graph) *commonv1.GraphStatistics {
	var (
		warehouseCount     int64
		deliveryPointCount int64
		totalCapacity      float64
		totalLength        float64
		edgeCount          int64
	)

	for _, node := range graph.Nodes {
		switch node.Type {
		case commonv1.NodeType_NODE_TYPE_WAREHOUSE:
			warehouseCount++
		case commonv1.NodeType_NODE_TYPE_DELIVERY_POINT:
			deliveryPointCount++
		}
	}

	for _, edge := range graph.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}
		edgeCount++
		totalCapacity += edge.Capacity
		totalLength += edge.Length
	}

	avgLength := 0.0
	if edgeCount > 0 {
		avgLength = totalLength / float64(edgeCount)
	}

	nodeCount := int64(len(graph.Nodes))
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
