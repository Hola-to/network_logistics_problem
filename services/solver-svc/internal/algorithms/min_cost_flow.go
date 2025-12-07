package algorithms

import (
	"logistics/services/solver-svc/internal/graph"
)

// MinCostFlowResult результат алгоритма минимальной стоимости
type MinCostFlowResult struct {
	Flow       float64
	Cost       float64
	Iterations int
	Paths      [][]int64
}

// MinCostMaxFlow находит поток минимальной стоимости
func MinCostMaxFlow(g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	totalFlow := 0.0
	totalCost := 0.0
	iterations := 0
	var paths [][]int64

	potentials := make(map[int64]float64)
	for node := range g.Nodes {
		potentials[node] = 0
	}

	initResult := BellmanFord(g, source)
	if !initResult.HasNegativeCycle {
		for node, dist := range initResult.Distances {
			if dist < graph.Infinity {
				potentials[node] = dist
			}
		}
	}

	for totalFlow < requiredFlow {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		bfResult := BellmanFordWithPotentials(g, source, potentials)

		if bfResult.Distances[sink] == graph.Infinity {
			break
		}

		for node := range g.Nodes {
			if bfResult.Distances[node] < graph.Infinity {
				potentials[node] += bfResult.Distances[node]
			}
		}

		path := graph.ReconstructPath(bfResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := requiredFlow - totalFlow
		pathFlow = min(pathFlow, graph.FindMinCapacityOnPath(g, path))

		if pathFlow <= options.Epsilon {
			break
		}

		pathCost := 0.0
		for i := 0; i < len(path)-1; i++ {
			edge := g.GetEdge(path[i], path[i+1])
			if edge != nil {
				pathCost += edge.Cost * pathFlow
			}
		}

		graph.AugmentPath(g, path, pathFlow)

		totalFlow += pathFlow
		totalCost += pathCost
		iterations++

		if options.ReturnPaths {
			paths = append(paths, path)
		}
	}

	return &MinCostFlowResult{
		Flow:       totalFlow,
		Cost:       totalCost,
		Iterations: iterations,
		Paths:      paths,
	}
}

// SuccessiveShortestPath алиас для MinCostMaxFlow
func SuccessiveShortestPath(g *graph.ResidualGraph, source, sink int64, requiredFlow float64, options *SolverOptions) *MinCostFlowResult {
	return MinCostMaxFlow(g, source, sink, requiredFlow, options)
}
