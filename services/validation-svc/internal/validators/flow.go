package validators

import (
	"fmt"
	"math"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/domain"
)

// ValidateFlowLogic проверяет корректность рассчитанного потока
func ValidateFlowLogic(graph *commonv1.Graph) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	// Карта баланса (Входящий - Исходящий)
	balance := make(map[int64]float64)

	for i, edge := range graph.Edges {
		// Пропускаем виртуальные узлы в проверках
		if domain.IsVirtualNode(edge.From) || domain.IsVirtualNode(edge.To) {
			continue
		}

		// 1. Ограничение пропускной способности (0 <= Flow <= Capacity)
		if edge.CurrentFlow < -domain.Epsilon {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d]", i),
				Code:    string(pkgerrors.CodeNegativeFlow),
				Message: fmt.Sprintf("Отрицательный поток: %.2f", edge.CurrentFlow),
			})
		}
		if edge.CurrentFlow > edge.Capacity+domain.Epsilon {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("edges[%d]", i),
				Code:    string(pkgerrors.CodeCapacityOverflow),
				Message: fmt.Sprintf("Поток %.2f превышает capacity %.2f", edge.CurrentFlow, edge.Capacity),
			})
		}

		balance[edge.From] -= edge.CurrentFlow
		balance[edge.To] += edge.CurrentFlow
	}

	// 2. Закон сохранения потока (Kirchhoff's Law)
	for _, node := range graph.Nodes {
		if node.Id == graph.SourceId || node.Id == graph.SinkId {
			continue
		}

		if node.Type == commonv1.NodeType_NODE_TYPE_SOURCE || node.Type == commonv1.NodeType_NODE_TYPE_SINK {
			continue
		}

		// Пропускаем виртуальные узлы
		if domain.IsVirtualNode(node.Id) {
			continue
		}

		diff := balance[node.Id]
		if math.Abs(diff) > domain.Epsilon {
			errors = append(errors, &commonv1.ValidationError{
				Field:   fmt.Sprintf("nodes[%d]", node.Id),
				Code:    string(pkgerrors.CodeConservationViolation),
				Message: fmt.Sprintf("Нарушение сохранения потока: imbalance %.4f", diff),
			})
		}
	}

	// 3. Проверка Total Flow (Выход из Source == Вход в Sink)
	sourceOut := -balance[graph.SourceId]
	sinkIn := balance[graph.SinkId]

	if math.Abs(sourceOut-sinkIn) > domain.Epsilon {
		errors = append(errors, &commonv1.ValidationError{
			Code:    string(pkgerrors.CodeFlowImbalance),
			Message: fmt.Sprintf("Поток из источника (%.2f) не равен потоку в сток (%.2f)", sourceOut, sinkIn),
		})
	}

	return errors
}
