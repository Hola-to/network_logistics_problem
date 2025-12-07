package analysis

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/pkg/domain"
)

// Используем константы из pkg/domain
const Epsilon = domain.Epsilon

// BuildEdgeMap создаёт карту рёбер для быстрого доступа
func BuildEdgeMap(graph *commonv1.Graph) map[int64]map[int64]*commonv1.Edge {
	edgeMap := make(map[int64]map[int64]*commonv1.Edge)

	for _, edge := range graph.Edges {
		if edgeMap[edge.From] == nil {
			edgeMap[edge.From] = make(map[int64]*commonv1.Edge)
		}
		edgeMap[edge.From][edge.To] = edge
	}

	return edgeMap
}

// GetEdge возвращает ребро между вершинами
func GetEdge(edgeMap map[int64]map[int64]*commonv1.Edge, from, to int64) *commonv1.Edge {
	if edgeMap[from] == nil {
		return nil
	}
	return edgeMap[from][to]
}

// IsVirtualNode проверяет, является ли узел виртуальным
// Делегирует в pkg/domain
func IsVirtualNode(nodeID int64) bool {
	return domain.IsVirtualNode(nodeID)
}

// GetNodesByType возвращает узлы определённого типа
func GetNodesByType(graph *commonv1.Graph, nodeType commonv1.NodeType) []int64 {
	var result []int64
	for _, node := range graph.Nodes {
		if node.Type == nodeType {
			result = append(result, node.Id)
		}
	}
	return result
}

// CalculateUtilization вычисляет коэффициент использования
func CalculateUtilization(flow, capacity float64) float64 {
	if capacity <= Epsilon {
		return 0.0
	}
	return flow / capacity
}
