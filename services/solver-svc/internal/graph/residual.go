package graph

import "logistics/pkg/domain"

const (
	Epsilon  = domain.Epsilon
	Infinity = domain.Infinity
)

// ResidualEdge - ребро в остаточном графе
type ResidualEdge struct {
	To               int64
	Capacity         float64 // Остаточная пропускная способность
	Cost             float64 // Стоимость
	Flow             float64 // Текущий поток
	OriginalCapacity float64 // Исходная пропускная способность
	IsReverse        bool    // Является ли обратным ребром
}

// ResidualGraph - остаточный граф для алгоритмов потока
type ResidualGraph struct {
	Nodes map[int64]bool
	Edges map[int64]map[int64]*ResidualEdge
}

// NewResidualGraph создаёт новый остаточный граф
func NewResidualGraph() *ResidualGraph {
	return &ResidualGraph{
		Nodes: make(map[int64]bool),
		Edges: make(map[int64]map[int64]*ResidualEdge),
	}
}

// AddNode добавляет узел в граф
func (rg *ResidualGraph) AddNode(id int64) {
	rg.Nodes[id] = true
}

// AddEdge добавляет прямое ребро
// Если ребро уже существует - увеличивает capacity (для параллельных рёбер)
func (rg *ResidualGraph) AddEdge(from, to int64, capacity, cost float64) {
	rg.ensureNode(from)
	rg.ensureNode(to)

	if rg.Edges[from] == nil {
		rg.Edges[from] = make(map[int64]*ResidualEdge)
	}

	if existing := rg.Edges[from][to]; existing != nil {
		// Ребро уже существует
		if existing.IsReverse {
			// Было reverse ребро - превращаем в прямое
			rg.Edges[from][to] = &ResidualEdge{
				To:               to,
				Capacity:         capacity,
				Cost:             cost,
				Flow:             0,
				OriginalCapacity: capacity,
				IsReverse:        false,
			}
		} else {
			// Уже есть прямое ребро - суммируем capacity (параллельные рёбра)
			existing.Capacity += capacity
			existing.OriginalCapacity += capacity
			// Cost берём средневзвешенный или оставляем первый
		}
		return
	}

	rg.Edges[from][to] = &ResidualEdge{
		To:               to,
		Capacity:         capacity,
		Cost:             cost,
		Flow:             0,
		OriginalCapacity: capacity,
		IsReverse:        false,
	}
}

// AddReverseEdge добавляет обратное ребро
// НЕ перезаписывает существующее прямое ребро (для anti-parallel случаев)
func (rg *ResidualGraph) AddReverseEdge(from, to int64, cost float64) {
	rg.ensureNode(from)
	rg.ensureNode(to)

	if rg.Edges[from] == nil {
		rg.Edges[from] = make(map[int64]*ResidualEdge)
	}

	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: проверяем существование
	if existing := rg.Edges[from][to]; existing != nil {
		// Если уже есть ПРЯМОЕ ребро (anti-parallel case) - НЕ перезаписываем
		// Прямое ребро само управляет своей residual capacity
		if !existing.IsReverse {
			return
		}
		// Если есть reverse ребро - оставляем как есть
		return
	}

	rg.Edges[from][to] = &ResidualEdge{
		To:               to,
		Capacity:         0,
		Cost:             -cost,
		Flow:             0,
		OriginalCapacity: 0,
		IsReverse:        true,
	}
}

// AddEdgeWithReverse добавляет прямое и обратное ребро
func (rg *ResidualGraph) AddEdgeWithReverse(from, to int64, capacity, cost float64) {
	rg.AddEdge(from, to, capacity, cost)
	rg.AddReverseEdge(to, from, cost)
}

// GetEdge возвращает ребро между вершинами
func (rg *ResidualGraph) GetEdge(from, to int64) *ResidualEdge {
	if rg.Edges[from] == nil {
		return nil
	}
	return rg.Edges[from][to]
}

// GetNeighbors возвращает соседей вершины
func (rg *ResidualGraph) GetNeighbors(node int64) map[int64]*ResidualEdge {
	return rg.Edges[node]
}

// GetNodes возвращает все вершины графа
func (rg *ResidualGraph) GetNodes() []int64 {
	nodes := make([]int64, 0, len(rg.Nodes))
	for node := range rg.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// NodeCount возвращает количество узлов
func (rg *ResidualGraph) NodeCount() int {
	return len(rg.Nodes)
}

// EdgeCount возвращает количество рёбер
func (rg *ResidualGraph) EdgeCount() int {
	count := 0
	for _, edges := range rg.Edges {
		count += len(edges)
	}
	return count
}

// Clone создаёт глубокую копию графа
func (rg *ResidualGraph) Clone() *ResidualGraph {
	clone := NewResidualGraph()

	for node := range rg.Nodes {
		clone.Nodes[node] = true
	}

	for from, edges := range rg.Edges {
		clone.Edges[from] = make(map[int64]*ResidualEdge)
		for to, edge := range edges {
			clone.Edges[from][to] = &ResidualEdge{
				To:               edge.To,
				Capacity:         edge.Capacity,
				Cost:             edge.Cost,
				Flow:             edge.Flow,
				OriginalCapacity: edge.OriginalCapacity,
				IsReverse:        edge.IsReverse,
			}
		}
	}

	return clone
}

// Reset сбрасывает потоки
func (rg *ResidualGraph) Reset() {
	for _, edges := range rg.Edges {
		for _, edge := range edges {
			if edge.IsReverse {
				edge.Capacity = 0
			} else {
				edge.Capacity = edge.OriginalCapacity
			}
			edge.Flow = 0
		}
	}
}

func (rg *ResidualGraph) ensureNode(id int64) {
	rg.Nodes[id] = true
}

// ResidualCapacity возвращает остаточную пропускную способность
func (e *ResidualEdge) ResidualCapacity() float64 {
	return e.Capacity - e.Flow
}

// HasCapacity проверяет наличие остаточной пропускной способности
func (e *ResidualEdge) HasCapacity() bool {
	return e.ResidualCapacity() > Epsilon
}

// UpdateFlow обновляет поток по ребру
// Корректно обрабатывает anti-parallel рёбра
func (rg *ResidualGraph) UpdateFlow(from, to int64, flow float64) {
	// Прямое ребро: уменьшаем capacity, увеличиваем flow
	if edge := rg.GetEdge(from, to); edge != nil {
		edge.Flow += flow
		edge.Capacity -= flow
	}

	// Обратное ребро (reverse или anti-parallel)
	if backEdge := rg.GetEdge(to, from); backEdge != nil {
		// Для ЛЮБОГО типа обратного ребра увеличиваем capacity
		// Это даёт возможность "отменить" поток:
		// - Для reverse ребра: стандартное поведение residual graph
		// - Для anti-parallel ребра: добавляем к его capacity возможность
		//   использовать "отмену" потока в обратном направлении
		//
		// При Reset() для прямого ребра capacity вернётся к OriginalCapacity,
		// а для reverse - к 0, что корректно сбрасывает добавленную capacity.
		backEdge.Capacity += flow
	} else {
		// Обратного ребра нет - создаём reverse
		if rg.Edges[to] == nil {
			rg.Edges[to] = make(map[int64]*ResidualEdge)
		}
		forwardEdge := rg.GetEdge(from, to)
		cost := 0.0
		if forwardEdge != nil {
			cost = -forwardEdge.Cost
		}
		rg.Edges[to][from] = &ResidualEdge{
			To:               from,
			Capacity:         flow,
			Cost:             cost,
			Flow:             0,
			OriginalCapacity: 0,
			IsReverse:        true,
		}
	}
}

// GetFlowOnEdge возвращает поток по ребру
func (rg *ResidualGraph) GetFlowOnEdge(from, to int64) float64 {
	if edge := rg.GetEdge(from, to); edge != nil {
		return edge.Flow
	}
	return 0
}

// GetTotalFlow вычисляет общий поток из источника
func (rg *ResidualGraph) GetTotalFlow(source int64) float64 {
	totalFlow := 0.0
	if edges := rg.GetNeighbors(source); edges != nil {
		for _, edge := range edges {
			if !edge.IsReverse && edge.Flow > 0 {
				totalFlow += edge.Flow
			}
		}
	}
	return totalFlow
}

// GetTotalCost вычисляет общую стоимость потока
func (rg *ResidualGraph) GetTotalCost() float64 {
	totalCost := 0.0
	for _, edges := range rg.Edges {
		for _, edge := range edges {
			if !edge.IsReverse && edge.Flow > 0 {
				totalCost += edge.Flow * edge.Cost
			}
		}
	}
	return totalCost
}
