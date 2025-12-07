package domain

import (
	"fmt"
	"sync"
)

// NodeType тип узла графа
type NodeType int

const (
	NodeTypeUnspecified NodeType = iota
	NodeTypeWarehouse
	NodeTypeDeliveryPoint
	NodeTypeIntersection
	NodeTypeSource
	NodeTypeSink
)

// String возвращает строковое представление типа узла
func (n NodeType) String() string {
	switch n {
	case NodeTypeWarehouse:
		return "warehouse"
	case NodeTypeDeliveryPoint:
		return "delivery_point"
	case NodeTypeIntersection:
		return "intersection"
	case NodeTypeSource:
		return "source"
	case NodeTypeSink:
		return "sink"
	default:
		return "unspecified"
	}
}

// RoadType тип дороги
type RoadType int

const (
	RoadTypeUnspecified RoadType = iota
	RoadTypeHighway
	RoadTypePrimary
	RoadTypeSecondary
	RoadTypeLocal
	RoadTypeUrban
)

// String возвращает строковое представление типа дороги
func (r RoadType) String() string {
	switch r {
	case RoadTypeHighway:
		return "highway"
	case RoadTypePrimary:
		return "primary"
	case RoadTypeSecondary:
		return "secondary"
	case RoadTypeLocal:
		return "local"
	case RoadTypeUrban:
		return "urban"
	default:
		return "unspecified"
	}
}

// EdgeKey уникальный ключ ребра
type EdgeKey struct {
	From int64
	To   int64
}

// String возвращает строковое представление ключа ребра
func (e EdgeKey) String() string {
	return fmt.Sprintf("%d->%d", e.From, e.To)
}

// Node представляет узел графа
type Node struct {
	ID       int64
	X        float64
	Y        float64
	Type     NodeType
	Name     string
	Metadata map[string]string
	Supply   float64 // > 0 для источников (min-cost flow)
	Demand   float64 // > 0 для стоков (min-cost flow)
}

// Clone создаёт глубокую копию узла
func (n *Node) Clone() *Node {
	clone := &Node{
		ID:       n.ID,
		X:        n.X,
		Y:        n.Y,
		Type:     n.Type,
		Name:     n.Name,
		Supply:   n.Supply,
		Demand:   n.Demand,
		Metadata: make(map[string]string, len(n.Metadata)),
	}
	for k, v := range n.Metadata {
		clone.Metadata[k] = v
	}
	return clone
}

// Edge представляет ребро графа
type Edge struct {
	From          int64
	To            int64
	Capacity      float64
	Cost          float64
	Length        float64
	RoadType      RoadType
	CurrentFlow   float64
	Bidirectional bool
}

// Clone создаёт глубокую копию ребра
func (e *Edge) Clone() *Edge {
	return &Edge{
		From:          e.From,
		To:            e.To,
		Capacity:      e.Capacity,
		Cost:          e.Cost,
		Length:        e.Length,
		RoadType:      e.RoadType,
		CurrentFlow:   e.CurrentFlow,
		Bidirectional: e.Bidirectional,
	}
}

// Utilization возвращает коэффициент использования ребра
func (e *Edge) Utilization() float64 {
	if e.Capacity <= Epsilon {
		return 0
	}
	return e.CurrentFlow / e.Capacity
}

// IsSaturated проверяет, насыщено ли ребро
func (e *Edge) IsSaturated() bool {
	return e.Utilization() >= 1.0-Epsilon
}

// HasFlow проверяет, есть ли поток на ребре
func (e *Edge) HasFlow() bool {
	return e.CurrentFlow > Epsilon
}

// ResidualCapacity возвращает остаточную пропускную способность
func (e *Edge) ResidualCapacity() float64 {
	return e.Capacity - e.CurrentFlow
}

// Key возвращает ключ ребра
func (e *Edge) Key() EdgeKey {
	return EdgeKey{From: e.From, To: e.To}
}

// Graph представляет граф логистической сети
type Graph struct {
	Nodes    map[int64]*Node
	Edges    map[EdgeKey]*Edge
	SourceID int64
	SinkID   int64
	Name     string
	Metadata map[string]string

	// Индексы для быстрого доступа
	outgoing map[int64][]int64 // node -> list of neighbor nodes
	incoming map[int64][]int64 // node -> list of predecessor nodes

	mu sync.RWMutex
}

// NewGraph создаёт новый пустой граф
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[int64]*Node),
		Edges:    make(map[EdgeKey]*Edge),
		Metadata: make(map[string]string),
		outgoing: make(map[int64][]int64),
		incoming: make(map[int64][]int64),
	}
}

// AddNode добавляет узел в граф
func (g *Graph) AddNode(node *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Nodes[node.ID] = node
}

// AddEdge добавляет ребро в граф
func (g *Graph) AddEdge(edge *Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := edge.Key()
	g.Edges[key] = edge

	// Обновляем индексы
	g.outgoing[edge.From] = append(g.outgoing[edge.From], edge.To)
	g.incoming[edge.To] = append(g.incoming[edge.To], edge.From)
}

// GetNode возвращает узел по ID
func (g *Graph) GetNode(id int64) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, ok := g.Nodes[id]
	return node, ok
}

// GetEdge возвращает ребро между двумя узлами
func (g *Graph) GetEdge(from, to int64) (*Edge, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edge, ok := g.Edges[EdgeKey{From: from, To: to}]
	return edge, ok
}

// GetOutgoing возвращает исходящие соседи узла
func (g *Graph) GetOutgoing(nodeID int64) []int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.outgoing[nodeID]
}

// GetIncoming возвращает входящие соседи узла
func (g *Graph) GetIncoming(nodeID int64) []int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.incoming[nodeID]
}

// NodeCount возвращает количество узлов
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.Nodes)
}

// EdgeCount возвращает количество рёбер
func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.Edges)
}

// Clone создаёт глубокую копию графа
func (g *Graph) Clone() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := NewGraph()
	clone.SourceID = g.SourceID
	clone.SinkID = g.SinkID
	clone.Name = g.Name

	for k, v := range g.Metadata {
		clone.Metadata[k] = v
	}

	for _, node := range g.Nodes {
		clone.Nodes[node.ID] = node.Clone()
	}

	for key, edge := range g.Edges {
		clone.Edges[key] = edge.Clone()
		clone.outgoing[edge.From] = append(clone.outgoing[edge.From], edge.To)
		clone.incoming[edge.To] = append(clone.incoming[edge.To], edge.From)
	}

	return clone
}

// GetNodesByType возвращает узлы определённого типа
func (g *Graph) GetNodesByType(nodeType NodeType) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []*Node
	for _, node := range g.Nodes {
		if node.Type == nodeType {
			result = append(result, node)
		}
	}
	return result
}

// GetActiveEdges возвращает рёбра с потоком
func (g *Graph) GetActiveEdges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []*Edge
	for _, edge := range g.Edges {
		if edge.HasFlow() {
			result = append(result, edge)
		}
	}
	return result
}

// GetSaturatedEdges возвращает насыщенные рёбра
func (g *Graph) GetSaturatedEdges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []*Edge
	for _, edge := range g.Edges {
		if edge.IsSaturated() {
			result = append(result, edge)
		}
	}
	return result
}

// ResetFlow сбрасывает поток на всех рёбрах
func (g *Graph) ResetFlow() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, edge := range g.Edges {
		edge.CurrentFlow = 0
	}
}

// TotalFlow возвращает общий поток из источника
func (g *Graph) TotalFlow() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var total float64
	for _, to := range g.outgoing[g.SourceID] {
		if edge, ok := g.Edges[EdgeKey{From: g.SourceID, To: to}]; ok {
			total += edge.CurrentFlow
		}
	}
	return total
}

// TotalCost возвращает общую стоимость потока
func (g *Graph) TotalCost() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var total float64
	for _, edge := range g.Edges {
		if edge.HasFlow() && !IsVirtualNode(edge.From) && !IsVirtualNode(edge.To) {
			total += edge.CurrentFlow * edge.Cost
		}
	}
	return total
}

// Validate проверяет корректность графа
func (g *Graph) Validate() []error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var errs []error

	// Проверка source и sink
	if _, ok := g.Nodes[g.SourceID]; !ok {
		errs = append(errs, fmt.Errorf("source node %d not found", g.SourceID))
	}
	if _, ok := g.Nodes[g.SinkID]; !ok {
		errs = append(errs, fmt.Errorf("sink node %d not found", g.SinkID))
	}
	if g.SourceID == g.SinkID {
		errs = append(errs, fmt.Errorf("source and sink cannot be the same node"))
	}

	// Проверка рёбер
	for key, edge := range g.Edges {
		if _, ok := g.Nodes[edge.From]; !ok {
			errs = append(errs, fmt.Errorf("edge %s references non-existent node %d", key, edge.From))
		}
		if _, ok := g.Nodes[edge.To]; !ok {
			errs = append(errs, fmt.Errorf("edge %s references non-existent node %d", key, edge.To))
		}
		if edge.From == edge.To {
			errs = append(errs, fmt.Errorf("self-loop detected at node %d", edge.From))
		}
		if edge.Capacity < 0 {
			errs = append(errs, fmt.Errorf("edge %s has negative capacity", key))
		}
	}

	return errs
}
