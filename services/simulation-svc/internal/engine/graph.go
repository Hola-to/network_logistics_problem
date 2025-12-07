package engine

import (
	"fmt"
	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
)

// CloneGraph создаёт глубокую копию графа
func CloneGraph(g *commonv1.Graph) *commonv1.Graph {
	if g == nil {
		return nil
	}

	clone := &commonv1.Graph{
		SourceId: g.SourceId,
		SinkId:   g.SinkId,
		Name:     g.Name,
		Metadata: make(map[string]string),
		Nodes:    make([]*commonv1.Node, len(g.Nodes)),
		Edges:    make([]*commonv1.Edge, len(g.Edges)),
	}

	for k, v := range g.Metadata {
		clone.Metadata[k] = v
	}

	for i, node := range g.Nodes {
		clone.Nodes[i] = &commonv1.Node{
			Id:       node.Id,
			X:        node.X,
			Y:        node.Y,
			Type:     node.Type,
			Name:     node.Name,
			Supply:   node.Supply,
			Demand:   node.Demand,
			Metadata: make(map[string]string),
		}
		for k, v := range node.Metadata {
			clone.Nodes[i].Metadata[k] = v
		}
	}

	for i, edge := range g.Edges {
		clone.Edges[i] = &commonv1.Edge{
			From:          edge.From,
			To:            edge.To,
			Capacity:      edge.Capacity,
			Cost:          edge.Cost,
			Length:        edge.Length,
			RoadType:      edge.RoadType,
			CurrentFlow:   edge.CurrentFlow,
			Bidirectional: edge.Bidirectional,
		}
	}

	return clone
}

// ApplyModifications применяет модификации к графу
func ApplyModifications(g *commonv1.Graph, mods []*simulationv1.Modification) *commonv1.Graph {
	modified := CloneGraph(g)

	// Индексы для быстрого доступа
	nodeIndex := make(map[int64]int)
	for i, node := range modified.Nodes {
		nodeIndex[node.Id] = i
	}

	edgeIndex := make(map[string]int)
	for i, edge := range modified.Edges {
		key := edgeKey(edge.From, edge.To)
		edgeIndex[key] = i
	}

	for _, mod := range mods {
		switch mod.Type {
		case simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE:
			applyEdgeModification(modified, edgeIndex, mod)

		case simulationv1.ModificationType_MODIFICATION_TYPE_REMOVE_EDGE:
			modified = removeEdge(modified, mod.EdgeKey)
			// Перестраиваем индекс
			edgeIndex = make(map[string]int)
			for i, edge := range modified.Edges {
				edgeIndex[edgeKey(edge.From, edge.To)] = i
			}

		case simulationv1.ModificationType_MODIFICATION_TYPE_ADD_EDGE:
			newEdge := &commonv1.Edge{
				From:     mod.EdgeKey.From,
				To:       mod.EdgeKey.To,
				Capacity: getModValue(mod, 0),
				Cost:     0,
			}
			modified.Edges = append(modified.Edges, newEdge)
			edgeIndex[edgeKey(newEdge.From, newEdge.To)] = len(modified.Edges) - 1

		case simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_NODE:
			applyNodeModification(modified, nodeIndex, mod)

		case simulationv1.ModificationType_MODIFICATION_TYPE_REMOVE_NODE:
			modified = removeNode(modified, mod.NodeId)
			// Перестраиваем индексы
			nodeIndex = make(map[int64]int)
			for i, node := range modified.Nodes {
				nodeIndex[node.Id] = i
			}
			edgeIndex = make(map[string]int)
			for i, edge := range modified.Edges {
				edgeIndex[edgeKey(edge.From, edge.To)] = i
			}

		case simulationv1.ModificationType_MODIFICATION_TYPE_DISABLE_NODE:
			// Устанавливаем capacity всех связанных рёбер в 0
			disableNode(modified, mod.NodeId)
		}
	}

	return modified
}

func applyEdgeModification(g *commonv1.Graph, index map[string]int, mod *simulationv1.Modification) {
	key := edgeKey(mod.EdgeKey.From, mod.EdgeKey.To)
	idx, ok := index[key]
	if !ok {
		return
	}

	edge := g.Edges[idx]
	value := getTargetValue(edge, mod.Target)
	newValue := calculateNewValue(value, mod)
	setTargetValue(edge, mod.Target, newValue)
}

func applyNodeModification(g *commonv1.Graph, index map[int64]int, mod *simulationv1.Modification) {
	idx, ok := index[mod.NodeId]
	if !ok {
		return
	}

	node := g.Nodes[idx]
	switch mod.Target {
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY:
		node.Supply = calculateNewValue(node.Supply, mod)
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_DEMAND:
		node.Demand = calculateNewValue(node.Demand, mod)
	}
}

func removeEdge(g *commonv1.Graph, key *commonv1.EdgeKey) *commonv1.Graph {
	newEdges := make([]*commonv1.Edge, 0, len(g.Edges)-1)
	for _, edge := range g.Edges {
		if edge.From != key.From || edge.To != key.To {
			newEdges = append(newEdges, edge)
		}
	}
	g.Edges = newEdges
	return g
}

func removeNode(g *commonv1.Graph, nodeID int64) *commonv1.Graph {
	// Удаляем узел
	newNodes := make([]*commonv1.Node, 0, len(g.Nodes)-1)
	for _, node := range g.Nodes {
		if node.Id != nodeID {
			newNodes = append(newNodes, node)
		}
	}
	g.Nodes = newNodes

	// Удаляем связанные рёбра
	newEdges := make([]*commonv1.Edge, 0)
	for _, edge := range g.Edges {
		if edge.From != nodeID && edge.To != nodeID {
			newEdges = append(newEdges, edge)
		}
	}
	g.Edges = newEdges

	return g
}

func disableNode(g *commonv1.Graph, nodeID int64) {
	for _, edge := range g.Edges {
		if edge.From == nodeID || edge.To == nodeID {
			edge.Capacity = 0
		}
	}
}

func getTargetValue(edge *commonv1.Edge, target simulationv1.ModificationTarget) float64 {
	switch target {
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY:
		return edge.Capacity
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_COST:
		return edge.Cost
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_LENGTH:
		return edge.Length
	default:
		return edge.Capacity
	}
}

func setTargetValue(edge *commonv1.Edge, target simulationv1.ModificationTarget, value float64) {
	switch target {
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY:
		edge.Capacity = value
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_COST:
		edge.Cost = value
	case simulationv1.ModificationTarget_MODIFICATION_TARGET_LENGTH:
		edge.Length = value
	}
}

func calculateNewValue(current float64, mod *simulationv1.Modification) float64 {
	switch v := mod.Change.(type) {
	case *simulationv1.Modification_AbsoluteValue:
		return v.AbsoluteValue
	case *simulationv1.Modification_RelativeChange:
		return current * v.RelativeChange
	case *simulationv1.Modification_Delta:
		return current + v.Delta
	default:
		return current
	}
}

func getModValue(mod *simulationv1.Modification, defaultValue float64) float64 {
	switch v := mod.Change.(type) {
	case *simulationv1.Modification_AbsoluteValue:
		return v.AbsoluteValue
	default:
		return defaultValue
	}
}

func edgeKey(from, to int64) string {
	return fmt.Sprintf("%d->%d", from, to)
}

// ResetFlow сбрасывает потоки на всех рёбрах
func ResetFlow(g *commonv1.Graph) {
	if g == nil {
		return
	}
	for _, edge := range g.Edges {
		edge.CurrentFlow = 0
	}
}
