package validators

import (
	"fmt"

	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
)

// ValidateForAlgorithm проверяет совместимость графа с алгоритмом
func ValidateForAlgorithm(graph *commonv1.Graph, algo commonv1.Algorithm) *validationv1.ValidateForAlgorithmResponse {
	response := &validationv1.ValidateForAlgorithmResponse{
		IsCompatible:    true,
		Issues:          []string{},
		Recommendations: []string{},
	}

	n := len(graph.Nodes)
	m := len(graph.Edges)

	switch algo {
	case commonv1.Algorithm_ALGORITHM_EDMONDS_KARP:
		response.Complexity = &validationv1.AlgorithmComplexity{
			TimeComplexity:      fmt.Sprintf("O(VE²) ≈ O(%d)", n*m*m),
			SpaceComplexity:     fmt.Sprintf("O(V+E) ≈ O(%d)", n+m),
			EstimatedIterations: int64(n * m),
			Recommendation:      "Хорошо для небольших и средних графов",
		}
		checkNonNegativeCapacity(graph, response)

	case commonv1.Algorithm_ALGORITHM_DINIC:
		response.Complexity = &validationv1.AlgorithmComplexity{
			TimeComplexity:      fmt.Sprintf("O(V²E) ≈ O(%d)", n*n*m),
			SpaceComplexity:     fmt.Sprintf("O(V+E) ≈ O(%d)", n+m),
			EstimatedIterations: int64(n * n),
			Recommendation:      "Оптимален для большинства задач",
		}
		checkNonNegativeCapacity(graph, response)

		if n > 10000 || m > 100000 {
			response.Recommendations = append(response.Recommendations,
				"Для очень больших графов рассмотрите Push-Relabel")
		}

	case commonv1.Algorithm_ALGORITHM_MIN_COST:
		response.Complexity = &validationv1.AlgorithmComplexity{
			TimeComplexity:      fmt.Sprintf("O(V²E + VE·log(V)) ≈ O(%d)", n*n*m),
			SpaceComplexity:     fmt.Sprintf("O(V+E) ≈ O(%d)", n+m),
			EstimatedIterations: int64(n * m),
			Recommendation:      "Используйте при необходимости минимизации стоимости",
		}
		checkCostsExist(graph, response)
		checkNoNegativeCycles(graph, response)

	case commonv1.Algorithm_ALGORITHM_PUSH_RELABEL:
		response.Complexity = &validationv1.AlgorithmComplexity{
			TimeComplexity:      fmt.Sprintf("O(V²E) или O(V³) ≈ O(%d)", n*n*m),
			SpaceComplexity:     fmt.Sprintf("O(V+E) ≈ O(%d)", n+m),
			EstimatedIterations: int64(n * n),
			Recommendation:      "Лучший выбор для плотных графов",
		}
		checkNonNegativeCapacity(graph, response)

		// Проверка плотности
		if n > 1 {
			density := float64(m) / float64(n*(n-1))
			if density < 0.1 {
				response.Recommendations = append(response.Recommendations,
					fmt.Sprintf("Граф разреженный (плотность %.1f%%), рассмотрите Dinic", density*100))
			}
		}

	case commonv1.Algorithm_ALGORITHM_FORD_FULKERSON:
		response.Complexity = &validationv1.AlgorithmComplexity{
			TimeComplexity:      "O(E·max_flow) — может не сходиться",
			SpaceComplexity:     fmt.Sprintf("O(V+E) ≈ O(%d)", n+m),
			EstimatedIterations: -1, // Неопределено
			Recommendation:      "НЕ рекомендуется для production",
		}
		checkIntegerCapacity(graph, response)
		response.Recommendations = append(response.Recommendations,
			"Используйте Edmonds-Karp вместо Ford-Fulkerson")

	default:
		response.Issues = append(response.Issues,
			fmt.Sprintf("Неизвестный алгоритм: %s", algo))
		response.IsCompatible = false
	}

	return response
}

func checkNonNegativeCapacity(graph *commonv1.Graph, resp *validationv1.ValidateForAlgorithmResponse) {
	for _, edge := range graph.Edges {
		if edge.Capacity < 0 {
			resp.Issues = append(resp.Issues,
				fmt.Sprintf("Отрицательная capacity на ребре %d→%d", edge.From, edge.To))
			resp.IsCompatible = false
		}
	}
}

func checkCostsExist(graph *commonv1.Graph, resp *validationv1.ValidateForAlgorithmResponse) {
	hasCost := false
	for _, edge := range graph.Edges {
		if edge.Cost != 0 {
			hasCost = true
			break
		}
	}
	if !hasCost {
		resp.Recommendations = append(resp.Recommendations,
			"Все рёбра имеют нулевую стоимость — Min-Cost бессмысленен")
	}
}

func checkNoNegativeCycles(graph *commonv1.Graph, resp *validationv1.ValidateForAlgorithmResponse) {
	topoResult := ValidateTopology(graph)
	for _, err := range topoResult.Errors {
		if err.Code == "NEGATIVE_CYCLE" {
			resp.Issues = append(resp.Issues, err.Message)
			resp.IsCompatible = false
		}
	}
}

func checkIntegerCapacity(graph *commonv1.Graph, resp *validationv1.ValidateForAlgorithmResponse) {
	for _, edge := range graph.Edges {
		if edge.Capacity != float64(int64(edge.Capacity)) {
			resp.Issues = append(resp.Issues,
				"Ford-Fulkerson с нецелыми capacity может не сходиться")
			resp.Recommendations = append(resp.Recommendations,
				"Используйте Edmonds-Karp для нецелых capacity")
			return
		}
	}
}
