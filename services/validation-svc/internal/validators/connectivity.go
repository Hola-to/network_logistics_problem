package validators

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/domain"
)

// ValidateConnectivity проверяет достижимость Стока из Истока
func ValidateConnectivity(graph *commonv1.Graph) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	// Конвертируем в domain.Graph для использования BFS
	domainGraph := convertToDomainGraph(graph)

	// Проверяем связность используя pkg/domain
	if !domain.IsConnected(domainGraph) {
		errors = append(errors, &commonv1.ValidationError{
			Code:    string(pkgerrors.CodeNoPath),
			Message: "Невозможно добраться от Истока до Стока (граф несвязный)",
		})
	}

	return errors
}

// convertToDomainGraph конвертирует proto граф в domain.Graph
func convertToDomainGraph(protoGraph *commonv1.Graph) *domain.Graph {
	g := domain.NewGraph()
	g.SourceID = protoGraph.SourceId
	g.SinkID = protoGraph.SinkId
	g.Name = protoGraph.Name

	for _, pNode := range protoGraph.Nodes {
		node := &domain.Node{
			ID:       pNode.Id,
			X:        pNode.X,
			Y:        pNode.Y,
			Type:     convertNodeType(pNode.Type),
			Name:     pNode.Name,
			Metadata: pNode.Metadata,
			Supply:   pNode.Supply,
			Demand:   pNode.Demand,
		}
		g.AddNode(node)
	}

	for _, pEdge := range protoGraph.Edges {
		edge := &domain.Edge{
			From:          pEdge.From,
			To:            pEdge.To,
			Capacity:      pEdge.Capacity,
			Cost:          pEdge.Cost,
			Length:        pEdge.Length,
			RoadType:      convertRoadType(pEdge.RoadType),
			CurrentFlow:   pEdge.CurrentFlow,
			Bidirectional: pEdge.Bidirectional,
		}
		g.AddEdge(edge)
	}

	return g
}

func convertNodeType(pt commonv1.NodeType) domain.NodeType {
	switch pt {
	case commonv1.NodeType_NODE_TYPE_WAREHOUSE:
		return domain.NodeTypeWarehouse
	case commonv1.NodeType_NODE_TYPE_DELIVERY_POINT:
		return domain.NodeTypeDeliveryPoint
	case commonv1.NodeType_NODE_TYPE_INTERSECTION:
		return domain.NodeTypeIntersection
	case commonv1.NodeType_NODE_TYPE_SOURCE:
		return domain.NodeTypeSource
	case commonv1.NodeType_NODE_TYPE_SINK:
		return domain.NodeTypeSink
	default:
		return domain.NodeTypeUnspecified
	}
}

func convertRoadType(pt commonv1.RoadType) domain.RoadType {
	switch pt {
	case commonv1.RoadType_ROAD_TYPE_HIGHWAY:
		return domain.RoadTypeHighway
	case commonv1.RoadType_ROAD_TYPE_PRIMARY:
		return domain.RoadTypePrimary
	case commonv1.RoadType_ROAD_TYPE_SECONDARY:
		return domain.RoadTypeSecondary
	case commonv1.RoadType_ROAD_TYPE_LOCAL:
		return domain.RoadTypeLocal
	case commonv1.RoadType_ROAD_TYPE_URBAN:
		return domain.RoadTypeUrban
	default:
		return domain.RoadTypeUnspecified
	}
}
