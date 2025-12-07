package algorithms

import (
	"logistics/services/solver-svc/internal/graph"
)

// BellmanFordResult результат алгоритма Беллмана-Форда
type BellmanFordResult struct {
	Distances        map[int64]float64
	Parent           map[int64]int64
	HasNegativeCycle bool
}

// BellmanFord реализует алгоритм Беллмана-Форда
func BellmanFord(g *graph.ResidualGraph, source int64) *BellmanFordResult {
	nodes := g.GetNodes()
	n := len(nodes)

	dist := make(map[int64]float64)
	parent := make(map[int64]int64)

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	for i := 0; i < n-1; i++ {
		updated := relaxAllEdges(g, dist, parent)
		if !updated {
			break
		}
	}

	hasNegativeCycle := checkNegativeCycle(g, dist)

	return &BellmanFordResult{
		Distances:        dist,
		Parent:           parent,
		HasNegativeCycle: hasNegativeCycle,
	}
}

// BellmanFordWithPotentials Беллман-Форд с потенциалами
func BellmanFordWithPotentials(g *graph.ResidualGraph, source int64, potentials map[int64]float64) *BellmanFordResult {
	nodes := g.GetNodes()
	n := len(nodes)

	dist := make(map[int64]float64)
	parent := make(map[int64]int64)

	for _, node := range nodes {
		dist[node] = graph.Infinity
		parent[node] = -1
	}
	dist[source] = 0

	for i := 0; i < n-1; i++ {
		updated := false

		for u, edges := range g.Edges {
			if dist[u] == graph.Infinity {
				continue
			}

			for v, edge := range edges {
				if edge.Capacity > graph.Epsilon {
					reducedCost := edge.Cost + potentials[u] - potentials[v]
					newDist := dist[u] + reducedCost

					if newDist < dist[v]-graph.Epsilon {
						dist[v] = newDist
						parent[v] = u
						updated = true
					}
				}
			}
		}

		if !updated {
			break
		}
	}

	hasNegativeCycle := checkNegativeCycleWithPotentials(g, dist, potentials)

	return &BellmanFordResult{
		Distances:        dist,
		Parent:           parent,
		HasNegativeCycle: hasNegativeCycle,
	}
}

func relaxAllEdges(g *graph.ResidualGraph, dist map[int64]float64, parent map[int64]int64) bool {
	updated := false

	for u, edges := range g.Edges {
		if dist[u] == graph.Infinity {
			continue
		}

		for v, edge := range edges {
			if edge.Capacity > graph.Epsilon {
				newDist := dist[u] + edge.Cost
				if newDist < dist[v]-graph.Epsilon {
					dist[v] = newDist
					parent[v] = u
					updated = true
				}
			}
		}
	}

	return updated
}

func checkNegativeCycle(g *graph.ResidualGraph, dist map[int64]float64) bool {
	for u, edges := range g.Edges {
		if dist[u] == graph.Infinity {
			continue
		}

		for v, edge := range edges {
			if edge.Capacity > graph.Epsilon {
				if dist[u]+edge.Cost < dist[v]-graph.Epsilon {
					return true
				}
			}
		}
	}
	return false
}

func checkNegativeCycleWithPotentials(g *graph.ResidualGraph, dist map[int64]float64, potentials map[int64]float64) bool {
	for u, edges := range g.Edges {
		if dist[u] == graph.Infinity {
			continue
		}

		for v, edge := range edges {
			if edge.Capacity > graph.Epsilon {
				reducedCost := edge.Cost + potentials[u] - potentials[v]
				if dist[u]+reducedCost < dist[v]-graph.Epsilon {
					return true
				}
			}
		}
	}
	return false
}

// FindShortestPath находит кратчайший путь
func FindShortestPath(g *graph.ResidualGraph, source, sink int64) ([]int64, float64, bool) {
	result := BellmanFord(g, source)

	if result.HasNegativeCycle {
		return nil, 0, false
	}

	if result.Distances[sink] == graph.Infinity {
		return nil, 0, false
	}

	path := graph.ReconstructPath(result.Parent, source, sink)
	return path, result.Distances[sink], len(path) > 0
}
