package algorithms

import (
	"math"

	"logistics/services/solver-svc/internal/graph"
)

// PushRelabelResult результат работы алгоритма
type PushRelabelResult struct {
	MaxFlow    float64
	Iterations int
}

// PushRelabel реализует алгоритм проталкивания предпотока
func PushRelabel(g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *PushRelabelResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	nodes := g.GetNodes()
	height := make(map[int64]int)
	excess := make(map[int64]float64)

	activeQueue := make([]int64, 0)
	inQueue := make(map[int64]bool)

	height[source] = len(nodes)

	sourceNeighbors := g.GetNeighbors(source)
	for v, edge := range sourceNeighbors {
		if edge.Capacity > options.Epsilon {
			flow := edge.Capacity
			edge.Capacity = 0
			edge.Flow += flow

			updateBackwardEdgePR(g, v, source, flow, edge.Cost)

			excess[v] += flow
			excess[source] -= flow

			if v != sink && !inQueue[v] {
				activeQueue = append(activeQueue, v)
				inQueue[v] = true
			}
		}
	}

	iterations := 0

	for len(activeQueue) > 0 {
		if options.MaxIterations > 0 && iterations >= options.MaxIterations {
			break
		}

		u := activeQueue[0]
		activeQueue = activeQueue[1:]
		inQueue[u] = false

		discharge(g, u, source, sink, height, excess, &activeQueue, inQueue, options)

		iterations++
	}

	return &PushRelabelResult{
		MaxFlow:    excess[sink],
		Iterations: iterations,
	}
}

func discharge(
	g *graph.ResidualGraph,
	u, source, sink int64,
	height map[int64]int,
	excess map[int64]float64,
	activeQueue *[]int64,
	inQueue map[int64]bool,
	options *SolverOptions,
) {
	for excess[u] > options.Epsilon {
		if pushed := tryPush(g, u, source, sink, height, excess, activeQueue, inQueue, options); pushed {
			if excess[u] <= options.Epsilon {
				return
			}
			continue
		}

		if !relabel(g, u, height, options) {
			return
		}
	}
}

func tryPush(
	g *graph.ResidualGraph,
	u, source, sink int64,
	height map[int64]int,
	excess map[int64]float64,
	activeQueue *[]int64,
	inQueue map[int64]bool,
	options *SolverOptions,
) bool {
	neighbors := g.GetNeighbors(u)
	pushed := false

	for v, edge := range neighbors {
		if edge.Capacity <= options.Epsilon || height[u] != height[v]+1 {
			continue
		}

		delta := min(excess[u], edge.Capacity)
		edge.Capacity -= delta
		edge.Flow += delta
		updateBackwardEdgePR(g, v, u, delta, edge.Cost)

		excess[u] -= delta
		excess[v] += delta

		if v != source && v != sink && !inQueue[v] {
			*activeQueue = append(*activeQueue, v)
			inQueue[v] = true
		}

		pushed = true
		if excess[u] <= options.Epsilon {
			break
		}
	}

	return pushed
}

func relabel(
	g *graph.ResidualGraph,
	u int64,
	height map[int64]int,
	options *SolverOptions,
) bool {
	minH := math.MaxInt32

	for _, edge := range g.GetNeighbors(u) {
		if edge.Capacity > options.Epsilon && height[edge.To] < minH {
			minH = height[edge.To]
		}
	}

	if minH == math.MaxInt32 {
		return false
	}

	height[u] = minH + 1
	return true
}

func updateBackwardEdgePR(g *graph.ResidualGraph, from, to int64, flow, cost float64) {
	if g.Edges[from] == nil {
		g.Edges[from] = make(map[int64]*graph.ResidualEdge)
	}

	if backEdge, exists := g.Edges[from][to]; exists {
		backEdge.Capacity += flow
	} else {
		g.Edges[from][to] = &graph.ResidualEdge{
			To:               to,
			Capacity:         flow,
			Cost:             -cost,
			Flow:             0,
			OriginalCapacity: 0,
			IsReverse:        true,
		}
	}
}
