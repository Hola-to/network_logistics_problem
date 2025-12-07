package graph

import (
	"logistics/pkg/domain"
)

// BFSResult результат BFS
type BFSResult = domain.BFSResult

// BFS выполняет поиск в ширину (делегирует в domain с адаптером)
func BFS(g *ResidualGraph, source, sink int64) *BFSResult {
	parent := make(map[int64]int64)
	visited := make(map[int64]bool)

	queue := []int64{source}
	visited[source] = true
	parent[source] = -1

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		neighbors := g.GetNeighbors(u)
		if neighbors == nil {
			continue
		}

		for v, edge := range neighbors {
			if !visited[v] && edge.Capacity > Epsilon {
				parent[v] = u
				visited[v] = true
				queue = append(queue, v)

				if v == sink {
					return &BFSResult{
						Found:   true,
						Parent:  parent,
						Visited: visited,
					}
				}
			}
		}
	}

	return &BFSResult{
		Found:   false,
		Parent:  parent,
		Visited: visited,
	}
}

// BFSLevel строит граф уровней для Dinic
func BFSLevel(g *ResidualGraph, source int64) map[int64]int {
	level := make(map[int64]int)
	level[source] = 0

	queue := []int64{source}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		neighbors := g.GetNeighbors(u)
		if neighbors == nil {
			continue
		}

		for v, edge := range neighbors {
			if _, exists := level[v]; !exists && edge.Capacity > Epsilon {
				level[v] = level[u] + 1
				queue = append(queue, v)
			}
		}
	}

	return level
}
