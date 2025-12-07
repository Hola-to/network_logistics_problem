package validators

import (
	"fmt"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
)

// ValidateStructure проверяет базовую структуру графа
func ValidateStructure(graph *commonv1.Graph) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	// 1. Пустой граф
	if len(graph.Nodes) == 0 {
		return append(errors, &commonv1.ValidationError{
			Field:   "nodes",
			Message: "Граф пуст",
			Code:    string(pkgerrors.CodeEmptyGraph),
		})
	}

	// 2. Индексация узлов для O(1) проверок
	nodeMap := make(map[int64]*commonv1.Node)
	for _, node := range graph.Nodes {
		if _, exists := nodeMap[node.Id]; exists {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("nodes[%d]", node.Id),
				Message: fmt.Sprintf("Дубликат ID узла: %d", node.Id),
				Code:    string(pkgerrors.CodeDuplicateNode),
			})
		}
		nodeMap[node.Id] = node
	}

	// 3. Проверка Source и Sink
	if _, ok := nodeMap[graph.SourceId]; !ok {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "source_id",
			Message: "ID Истока не найден в списке узлов",
			Code:    string(pkgerrors.CodeInvalidSource),
		})
	}
	if _, ok := nodeMap[graph.SinkId]; !ok {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "sink_id",
			Message: "ID Стока не найден в списке узлов",
			Code:    string(pkgerrors.CodeInvalidSink),
		})
	}
	if graph.SourceId == graph.SinkId {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "sink_id",
			Message: "Исток и Сток не могут совпадать",
			Code:    string(pkgerrors.CodeSourceEqualsSink),
		})
	}

	// 4. Проверка рёбер
	for i, edge := range graph.Edges {
		// Существование концов
		if _, ok := nodeMap[edge.From]; !ok {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d].from", i),
				Message: fmt.Sprintf("Ребро ссылается на несуществующий узел From: %d", edge.From),
				Code:    string(pkgerrors.CodeDanglingEdge),
			})
		}
		if _, ok := nodeMap[edge.To]; !ok {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d].to", i),
				Message: fmt.Sprintf("Ребро ссылается на несуществующий узел To: %d", edge.To),
				Code:    string(pkgerrors.CodeDanglingEdge),
			})
		}

		// Петли
		if edge.From == edge.To {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d]", i),
				Message: "Обнаружена петля (ребро в себя)",
				Code:    string(pkgerrors.CodeSelfLoop),
			})
		}

		// Неотрицательные параметры
		if edge.Capacity <= 0 {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d].capacity", i),
				Message: "Пропускная способность должна быть > 0",
				Code:    string(pkgerrors.CodeInvalidCapacity),
			})
		}
		if edge.Cost < 0 {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d].cost", i),
				Message: "Стоимость не может быть отрицательной",
				Code:    string(pkgerrors.CodeNegativeCost),
			})
		}
		if edge.Length < 0 {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d].length", i),
				Message: "Длина не может быть отрицательной",
				Code:    string(pkgerrors.CodeNegativeLength),
			})
		}
	}

	return errors
}
