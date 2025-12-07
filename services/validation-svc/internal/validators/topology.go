package validators

import (
	"fmt"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/domain"
)

// TopologyResult результат топологической валидации
type TopologyResult struct {
	Errors   []*commonv1.ValidationError
	Warnings []string
}

// ValidateTopology проверяет топологию графа
func ValidateTopology(graph *commonv1.Graph) *TopologyResult {
	result := &TopologyResult{
		Errors:   []*commonv1.ValidationError{},
		Warnings: []string{},
	}

	// Конвертируем в domain.Graph
	domainGraph := convertToDomainGraph(graph)

	// 1. Проверка изолированных узлов
	result.checkIsolatedNodes(graph)

	// 2. Проверка обратной достижимости
	result.checkReverseReachability(domainGraph, graph)

	// 3. Проверка на отрицательные циклы
	result.checkNegativeCycles(graph)

	// 4. Проверка компонент связности
	result.checkConnectedComponents(domainGraph)

	return result
}

func (r *TopologyResult) checkIsolatedNodes(graph *commonv1.Graph) {
	hasEdge := make(map[int64]bool)
	for _, edge := range graph.Edges {
		hasEdge[edge.From] = true
		hasEdge[edge.To] = true
	}

	for _, node := range graph.Nodes {
		if !hasEdge[node.Id] {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("Изолированный узел %d (%s) без рёбер", node.Id, node.Name))
		}
	}
}

func (r *TopologyResult) checkReverseReachability(domainGraph *domain.Graph, protoGraph *commonv1.Graph) {
	// Используем BFSReverse из pkg/domain
	reachableFromSink := domain.BFSReverse(domainGraph, domainGraph.SinkID)

	if !reachableFromSink[domainGraph.SourceID] {
		r.Errors = append(r.Errors, &commonv1.ValidationError{
			Code:    string(pkgerrors.CodeNoPath),
			Message: "Источник не может достичь стока (нет пути)",
			Field:   "graph",
		})
	}

	// Проверяем склады
	for _, node := range protoGraph.Nodes {
		if node.Type == commonv1.NodeType_NODE_TYPE_WAREHOUSE && !reachableFromSink[node.Id] {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("Склад %d не может достичь стока", node.Id))
		}
	}
}

func (r *TopologyResult) checkNegativeCycles(graph *commonv1.Graph) {
	n := int64(len(graph.Nodes))
	dist := make(map[int64]float64)

	for _, node := range graph.Nodes {
		dist[node.Id] = domain.Infinity
	}
	dist[graph.SourceId] = 0

	// Релаксация n-1 раз
	for i := int64(0); i < n-1; i++ {
		for _, edge := range graph.Edges {
			if dist[edge.From]+edge.Cost < dist[edge.To] {
				dist[edge.To] = dist[edge.From] + edge.Cost
			}
		}
	}

	// Проверка на отрицательный цикл
	for _, edge := range graph.Edges {
		if dist[edge.From]+edge.Cost < dist[edge.To]-domain.Epsilon {
			r.Errors = append(r.Errors, &commonv1.ValidationError{
				Code:    string(pkgerrors.CodeNegativeCycle),
				Message: fmt.Sprintf("Обнаружен отрицательный цикл (ребро %d→%d)", edge.From, edge.To),
				Field:   "edges",
			})
			return
		}
	}
}

func (r *TopologyResult) checkConnectedComponents(domainGraph *domain.Graph) {
	// Используем FindConnectedComponents из pkg/domain
	components := domain.FindConnectedComponents(domainGraph)

	if len(components) > 1 {
		r.Warnings = append(r.Warnings,
			fmt.Sprintf("Граф состоит из %d несвязных компонент", len(components)))
	}
}
