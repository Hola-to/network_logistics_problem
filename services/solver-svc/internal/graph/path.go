package graph

import (
	"logistics/pkg/domain"
)

// ReconstructPath восстанавливает путь из parent map
// Делегирует в pkg/domain
func ReconstructPath(parent map[int64]int64, source, sink int64) []int64 {
	return domain.ReconstructPath(parent, source, sink)
}

// FindMinCapacityOnPath находит минимальную пропускную способность на пути
func FindMinCapacityOnPath(g *ResidualGraph, path []int64) float64 {
	if len(path) < 2 {
		return 0
	}

	minCapacity := Infinity

	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]

		edge := g.GetEdge(from, to)
		if edge == nil {
			return 0
		}

		if edge.Capacity < minCapacity {
			minCapacity = edge.Capacity
		}
	}

	if minCapacity == Infinity {
		return 0
	}

	return minCapacity
}

// AugmentPath увеличивает поток вдоль пути
func AugmentPath(g *ResidualGraph, path []int64, flow float64) {
	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]
		g.UpdateFlow(from, to, flow)
	}
}
