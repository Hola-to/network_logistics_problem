package algorithms

import (
	"logistics/services/solver-svc/internal/graph"
)

// DinicResult результат алгоритма Диница
type DinicResult struct {
	MaxFlow    float64
	Iterations int
	Paths      [][]int64
}

// Dinic реализует алгоритм Диница
func Dinic(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *DinicResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	maxFlow := 0.0
	iterations := 0
	var paths [][]int64

	for options.MaxIterations <= 0 || iterations < options.MaxIterations {

		level := graph.BFSLevel(g, source)

		if _, exists := level[sink]; !exists {
			break
		}

		for {
			flow, path := dinicDFS(g, source, sink, graph.Infinity, level, options)
			if flow <= options.Epsilon {
				break
			}

			maxFlow += flow

			if options.ReturnPaths && len(path) > 0 {
				paths = append(paths, path)
			}
		}

		iterations++
	}

	return &DinicResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
	}
}

func dinicDFS(g *graph.ResidualGraph, u, sink int64, pushed float64, level map[int64]int, options *SolverOptions) (float64, []int64) {
	if u == sink {
		return pushed, []int64{sink}
	}

	neighbors := g.GetNeighbors(u)
	if neighbors == nil {
		return 0, nil
	}

	for v, edge := range neighbors {
		if level[v] != level[u]+1 || edge.Capacity <= options.Epsilon {
			continue
		}

		canPush := min(pushed, edge.Capacity)
		flow, path := dinicDFS(g, v, sink, canPush, level, options)

		if flow > options.Epsilon {
			g.UpdateFlow(u, v, flow)
			return flow, append([]int64{u}, path...)
		}
	}

	return 0, nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
