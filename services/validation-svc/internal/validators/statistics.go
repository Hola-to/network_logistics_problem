package validators

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/pkg/domain"
)

// Используем константы из pkg/domain
const Epsilon = domain.Epsilon

// CalculateGraphStatistics вычисляет статистику графа
func CalculateGraphStatistics(graph *commonv1.Graph) *commonv1.GraphStatistics {
	if graph == nil {
		return nil
	}

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
		// Пропускаем виртуальные узлы
		if domain.IsVirtualNode(edge.From) || domain.IsVirtualNode(edge.To) {
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

	isConnected := checkIsConnected(graph)

	return &commonv1.GraphStatistics{
		NodeCount:          nodeCount,
		EdgeCount:          edgeCount,
		WarehouseCount:     warehouseCount,
		DeliveryPointCount: deliveryPointCount,
		TotalCapacity:      totalCapacity,
		AverageEdgeLength:  avgLength,
		IsConnected:        isConnected,
		Density:            density,
	}
}

// CalculateFlowSummary вычисляет сводку по потоку
func CalculateFlowSummary(graph *commonv1.Graph) *validationv1.FlowSummary {
	if graph == nil {
		return nil
	}

	var (
		totalFlow      float64
		totalCost      float64
		edgesWithFlow  int32
		saturatedEdges int32
		sourceOutflow  float64
		sinkInflow     float64
	)

	for _, edge := range graph.Edges {
		// Пропускаем виртуальные узлы
		if domain.IsVirtualNode(edge.From) || domain.IsVirtualNode(edge.To) {
			continue
		}

		if edge.CurrentFlow > Epsilon {
			edgesWithFlow++
			totalCost += edge.CurrentFlow * edge.Cost

			utilization := edge.CurrentFlow / edge.Capacity
			if utilization >= 1.0-Epsilon {
				saturatedEdges++
			}
		}

		if edge.From == graph.SourceId {
			sourceOutflow += edge.CurrentFlow
		}
		if edge.To == graph.SinkId {
			sinkInflow += edge.CurrentFlow
		}
	}

	totalFlow = sinkInflow

	return &validationv1.FlowSummary{
		TotalFlow:      totalFlow,
		TotalCost:      totalCost,
		EdgesWithFlow:  edgesWithFlow,
		SaturatedEdges: saturatedEdges,
		SourceOutflow:  sourceOutflow,
		SinkInflow:     sinkInflow,
	}
}

func checkIsConnected(graph *commonv1.Graph) bool {
	if len(graph.Nodes) == 0 {
		return true
	}

	adj := make(map[int64][]int64)
	for _, edge := range graph.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		adj[edge.To] = append(adj[edge.To], edge.From)
	}

	visited := make(map[int64]bool)
	queue := []int64{graph.Nodes[0].Id}
	visited[graph.Nodes[0].Id] = true

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	return len(visited) == len(graph.Nodes)
}
