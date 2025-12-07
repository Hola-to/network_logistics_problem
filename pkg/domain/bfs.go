package domain

// BFSResult результат BFS обхода
type BFSResult struct {
	Found   bool
	Parent  map[int64]int64
	Visited map[int64]bool
	Level   map[int64]int
}

// BFS выполняет поиск в ширину от source до sink
func BFS(g *Graph, source, sink int64) *BFSResult {
	parent := make(map[int64]int64)
	visited := make(map[int64]bool)
	level := make(map[int64]int)

	queue := []int64{source}
	visited[source] = true
	parent[source] = -1
	level[source] = 0

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		for _, v := range g.GetOutgoing(u) {
			if visited[v] {
				continue
			}

			edge, ok := g.GetEdge(u, v)
			if !ok || edge.ResidualCapacity() <= Epsilon {
				continue
			}

			parent[v] = u
			visited[v] = true
			level[v] = level[u] + 1
			queue = append(queue, v)

			if v == sink {
				return &BFSResult{
					Found:   true,
					Parent:  parent,
					Visited: visited,
					Level:   level,
				}
			}
		}
	}

	return &BFSResult{
		Found:   false,
		Parent:  parent,
		Visited: visited,
		Level:   level,
	}
}

// BFSLevel строит граф уровней (для алгоритма Диница)
func BFSLevel(g *Graph, source int64) map[int64]int {
	level := make(map[int64]int)
	level[source] = 0

	queue := []int64{source}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		for _, v := range g.GetOutgoing(u) {
			if _, exists := level[v]; exists {
				continue
			}

			edge, ok := g.GetEdge(u, v)
			if !ok || edge.ResidualCapacity() <= Epsilon {
				continue
			}

			level[v] = level[u] + 1
			queue = append(queue, v)
		}
	}

	return level
}

// BFSReachable возвращает все достижимые вершины из source
func BFSReachable(g *Graph, source int64) map[int64]bool {
	visited := make(map[int64]bool)
	queue := []int64{source}
	visited[source] = true

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		for _, v := range g.GetOutgoing(u) {
			if visited[v] {
				continue
			}

			edge, ok := g.GetEdge(u, v)
			if !ok || edge.Capacity <= Epsilon {
				continue
			}

			visited[v] = true
			queue = append(queue, v)
		}
	}

	return visited
}

// BFSReverse выполняет обратный BFS (от sink к source)
func BFSReverse(g *Graph, sink int64) map[int64]bool {
	visited := make(map[int64]bool)
	queue := []int64{sink}
	visited[sink] = true

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		for _, v := range g.GetIncoming(u) {
			if visited[v] {
				continue
			}

			edge, ok := g.GetEdge(v, u)
			if !ok || edge.Capacity <= Epsilon {
				continue
			}

			visited[v] = true
			queue = append(queue, v)
		}
	}

	return visited
}

// IsConnected проверяет, существует ли путь от source к sink
func IsConnected(g *Graph) bool {
	reachable := BFSReachable(g, g.SourceID)
	return reachable[g.SinkID]
}

// FindConnectedComponents находит компоненты связности
func FindConnectedComponents(g *Graph) [][]int64 {
	visited := make(map[int64]bool)

	// Pre-allocate с примерным размером
	components := make([][]int64, 0, len(g.Nodes)/10+1)

	// Строим неориентированный граф смежности
	adj := make(map[int64][]int64)
	for _, edge := range g.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		adj[edge.To] = append(adj[edge.To], edge.From)
	}

	for nodeID := range g.Nodes {
		if visited[nodeID] {
			continue
		}

		var component []int64
		queue := []int64{nodeID}
		visited[nodeID] = true

		for len(queue) > 0 {
			u := queue[0]
			queue = queue[1:]
			component = append(component, u)

			for _, v := range adj[u] {
				if !visited[v] {
					visited[v] = true
					queue = append(queue, v)
				}
			}
		}

		components = append(components, component)
	}

	return components
}
