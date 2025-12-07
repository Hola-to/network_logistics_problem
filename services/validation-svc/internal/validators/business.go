package validators

import (
	"fmt"

	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
)

// ValidateBusinessRules проверяет логистические ограничения
func ValidateBusinessRules(graph *commonv1.Graph) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	inDegree := make(map[int64]int)
	outDegree := make(map[int64]int)

	for _, edge := range graph.Edges {
		outDegree[edge.From]++
		inDegree[edge.To]++
	}

	for _, node := range graph.Nodes {
		// Правило 1: Склады не могут быть "тупиками"
		if node.Type == commonv1.NodeType_NODE_TYPE_WAREHOUSE {
			if outDegree[node.Id] == 0 && node.Id != graph.SinkId {
				errors = append(errors, &commonv1.ValidationError{
					Field:   fmt.Sprintf("nodes[%d]", node.Id),
					Code:    string(pkgerrors.CodeIsolatedWarehouse),
					Message: "Склад не имеет исходящих путей (тупик)",
				})
			}
		}

		// Правило 2: Точки доставки должны быть достижимы
		if node.Type == commonv1.NodeType_NODE_TYPE_DELIVERY_POINT {
			if inDegree[node.Id] == 0 && node.Id != graph.SourceId {
				errors = append(errors, &commonv1.ValidationError{
					Field:   fmt.Sprintf("nodes[%d]", node.Id),
					Code:    string(pkgerrors.CodeUnreachableDelivery),
					Message: "В точку доставки не ведёт ни одна дорога",
				})
			}
		}
	}

	return errors
}
