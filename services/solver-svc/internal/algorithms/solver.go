package algorithms

import (
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"
)

// SolverOptions опции для алгоритмов
type SolverOptions struct {
	Epsilon       float64
	MaxIterations int
	Timeout       time.Duration
	ReturnPaths   bool
}

// DefaultSolverOptions возвращает опции по умолчанию
func DefaultSolverOptions() *SolverOptions {
	return &SolverOptions{
		Epsilon:       graph.Epsilon,
		MaxIterations: 0,
		Timeout:       0,
		ReturnPaths:   false,
	}
}

// SolverResult общий результат решения
type SolverResult struct {
	MaxFlow    float64
	TotalCost  float64
	Iterations int
	Paths      [][]int64
	Status     commonv1.FlowStatus
	Error      error
}

// Solve решает задачу потока с выбранным алгоритмом
func Solve(g *graph.ResidualGraph, source, sink int64, algorithm commonv1.Algorithm, options *SolverOptions) *SolverResult {
	if options == nil {
		options = DefaultSolverOptions()
	}

	switch algorithm {
	case commonv1.Algorithm_ALGORITHM_EDMONDS_KARP:
		result := EdmondsKarp(g, source, sink, options)
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			TotalCost:  g.GetTotalCost(),
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		}

	case commonv1.Algorithm_ALGORITHM_DINIC:
		result := Dinic(g, source, sink, options)
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			TotalCost:  g.GetTotalCost(),
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		}

	case commonv1.Algorithm_ALGORITHM_PUSH_RELABEL:
		result := PushRelabel(g, source, sink, options)
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			TotalCost:  g.GetTotalCost(),
			Iterations: result.Iterations,
			Paths:      nil,
			Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		}

	case commonv1.Algorithm_ALGORITHM_MIN_COST:
		ekResult := EdmondsKarp(g.Clone(), source, sink, options)
		requiredFlow := ekResult.MaxFlow

		result := MinCostMaxFlow(g, source, sink, requiredFlow, options)
		return &SolverResult{
			MaxFlow:    result.Flow,
			TotalCost:  result.Cost,
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		}

	case commonv1.Algorithm_ALGORITHM_FORD_FULKERSON:
		result := EdmondsKarp(g, source, sink, options)
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			TotalCost:  g.GetTotalCost(),
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		}

	default:
		result := EdmondsKarp(g, source, sink, options)
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			TotalCost:  g.GetTotalCost(),
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		}
	}
}

// AlgorithmInfo информация об алгоритме
type AlgorithmInfo struct {
	Algorithm             commonv1.Algorithm
	Name                  string
	Description           string
	TimeComplexity        string
	SpaceComplexity       string
	SupportsMinCost       bool
	SupportsNegativeCosts bool
	BestFor               []string
}

// GetAlgorithmInfo возвращает информацию об алгоритме
func GetAlgorithmInfo(algo commonv1.Algorithm) *AlgorithmInfo {
	infos := map[commonv1.Algorithm]*AlgorithmInfo{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			Name:            "Edmonds-Karp",
			Description:     "BFS-based Ford-Fulkerson implementation",
			TimeComplexity:  "O(VE²)",
			SpaceComplexity: "O(V+E)",
			BestFor:         []string{"general_graphs", "small_to_medium_size"},
		},
		commonv1.Algorithm_ALGORITHM_DINIC: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_DINIC,
			Name:            "Dinic",
			Description:     "Level graphs and blocking flows",
			TimeComplexity:  "O(V²E)",
			SpaceComplexity: "O(V+E)",
			BestFor:         []string{"large_graphs", "unit_capacity_graphs"},
		},
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
			Name:            "Push-Relabel",
			Description:     "Preflow-push algorithm",
			TimeComplexity:  "O(V²E)",
			SpaceComplexity: "O(V+E)",
			BestFor:         []string{"dense_graphs"},
		},
		commonv1.Algorithm_ALGORITHM_MIN_COST: {
			Algorithm:             commonv1.Algorithm_ALGORITHM_MIN_COST,
			Name:                  "Min-Cost Max-Flow",
			Description:           "Successive Shortest Paths",
			TimeComplexity:        "O(V²E + VE·F)",
			SpaceComplexity:       "O(V+E)",
			SupportsMinCost:       true,
			SupportsNegativeCosts: true,
			BestFor:               []string{"cost_optimization"},
		},
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
			Name:            "Ford-Fulkerson",
			Description:     "Classic augmenting path",
			TimeComplexity:  "O(E·max_flow)",
			SpaceComplexity: "O(V+E)",
			BestFor:         []string{"integer_capacities"},
		},
	}

	return infos[algo]
}

// GetAllAlgorithms возвращает все алгоритмы
func GetAllAlgorithms() []*AlgorithmInfo {
	algorithms := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
		commonv1.Algorithm_ALGORITHM_MIN_COST,
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
	}

	var infos []*AlgorithmInfo
	for _, algo := range algorithms {
		if info := GetAlgorithmInfo(algo); info != nil {
			infos = append(infos, info)
		}
	}
	return infos
}
