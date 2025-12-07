package domain

// Path представляет путь в графе
type Path struct {
	Nodes  []int64
	Flow   float64
	Cost   float64
	Length float64
}

// ReconstructPath восстанавливает путь из parent map
func ReconstructPath(parent map[int64]int64, source, sink int64) []int64 {
	if _, exists := parent[sink]; !exists {
		return nil
	}

	path := []int64{}
	current := sink

	for current != source {
		path = append([]int64{current}, path...)
		p, exists := parent[current]
		if !exists || p == -1 {
			if current == source {
				break
			}
			return nil
		}
		current = p
	}
	path = append([]int64{source}, path...)

	return path
}

// FindMinCapacityOnPath находит минимальную остаточную пропускную способность на пути
func FindMinCapacityOnPath(g *Graph, path []int64) float64 {
	if len(path) < 2 {
		return 0
	}

	minCapacity := Infinity

	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]

		edge, ok := g.GetEdge(from, to)
		if !ok {
			return 0
		}

		residual := edge.ResidualCapacity()
		if residual < minCapacity {
			minCapacity = residual
		}
	}

	if minCapacity == Infinity {
		return 0
	}

	return minCapacity
}

// CalculatePathCost вычисляет стоимость пути
func CalculatePathCost(g *Graph, path []int64) float64 {
	if len(path) < 2 {
		return 0
	}

	var cost float64
	for i := 0; i < len(path)-1; i++ {
		edge, ok := g.GetEdge(path[i], path[i+1])
		if ok {
			cost += edge.Cost
		}
	}
	return cost
}

// CalculatePathLength вычисляет длину пути
func CalculatePathLength(g *Graph, path []int64) float64 {
	if len(path) < 2 {
		return 0
	}

	var length float64
	for i := 0; i < len(path)-1; i++ {
		edge, ok := g.GetEdge(path[i], path[i+1])
		if ok {
			length += edge.Length
		}
	}
	return length
}

// AugmentPath увеличивает поток вдоль пути
func AugmentPath(g *Graph, path []int64, flow float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]

		// Прямое ребро
		if edge, ok := g.Edges[EdgeKey{From: from, To: to}]; ok {
			edge.CurrentFlow += flow
		}

		// Обратное ребро (для остаточного графа)
		reverseKey := EdgeKey{From: to, To: from}
		if reverseEdge, ok := g.Edges[reverseKey]; ok {
			reverseEdge.CurrentFlow -= flow
		}
	}
}

// CreatePath создаёт объект пути с вычисленными метриками
func CreatePath(g *Graph, nodes []int64, flow float64) *Path {
	return &Path{
		Nodes:  nodes,
		Flow:   flow,
		Cost:   CalculatePathCost(g, nodes) * flow,
		Length: CalculatePathLength(g, nodes),
	}
}
