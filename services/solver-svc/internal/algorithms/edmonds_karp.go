package algorithms

import (
	"logistics/services/solver-svc/internal/graph"
)

// EdmondsKarpResult результат алгоритма Эдмондса-Карпа
type EdmondsKarpResult struct {
	MaxFlow    float64
	Iterations int
	Paths      [][]int64
}

// EdmondsKarp реализует алгоритм Эдмондса-Карпа
func EdmondsKarp(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *EdmondsKarpResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	maxFlow := 0.0
	iterations := 0
	var paths [][]int64

	for options.MaxIterations <= 0 || iterations < options.MaxIterations {

		bfsResult := graph.BFS(g, source, sink)
		if !bfsResult.Found {
			break
		}

		path := graph.ReconstructPath(bfsResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := graph.FindMinCapacityOnPath(g, path)
		if pathFlow <= options.Epsilon {
			break
		}

		graph.AugmentPath(g, path, pathFlow)

		maxFlow += pathFlow
		iterations++

		if options.ReturnPaths {
			paths = append(paths, path)
		}
	}

	return &EdmondsKarpResult{
		MaxFlow:    maxFlow,
		Iterations: iterations,
		Paths:      paths,
	}
}
