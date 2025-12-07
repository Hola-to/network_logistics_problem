package analysis

import (
	"sort"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
)

// FindBottlenecks находит узкие места в сети
func FindBottlenecks(graph *commonv1.Graph, threshold float64, topN int32) *analyticsv1.FindBottlenecksResponse {
	var bottlenecks []*analyticsv1.Bottleneck

	for _, edge := range graph.Edges {
		// Пропускаем виртуальные узлы
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}

		// Пропускаем рёбра без потока
		if edge.CurrentFlow <= Epsilon {
			continue
		}

		utilization := CalculateUtilization(edge.CurrentFlow, edge.Capacity)

		if utilization >= threshold {
			severity := calculateSeverity(utilization)
			impact := calculateImpactScore(edge, graph)

			bottlenecks = append(bottlenecks, &analyticsv1.Bottleneck{
				Edge:        edge,
				Utilization: utilization,
				ImpactScore: impact,
				Severity:    severity,
			})
		}
	}

	// Сортируем по utilization (убывание)
	sort.Slice(bottlenecks, func(i, j int) bool {
		return bottlenecks[i].Utilization > bottlenecks[j].Utilization
	})

	// Ограничиваем количество
	if topN > 0 && int(topN) < len(bottlenecks) {
		bottlenecks = bottlenecks[:topN]
	}

	// Генерируем рекомендации
	recommendations := generateRecommendations(bottlenecks)

	return &analyticsv1.FindBottlenecksResponse{
		Bottlenecks:     bottlenecks,
		Recommendations: recommendations,
	}
}

func calculateSeverity(utilization float64) analyticsv1.BottleneckSeverity {
	switch {
	case utilization >= 0.99:
		return analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL
	case utilization >= 0.95:
		return analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH
	case utilization >= 0.90:
		return analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_MEDIUM
	default:
		return analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW
	}
}

func calculateImpactScore(edge *commonv1.Edge, graph *commonv1.Graph) float64 {
	// Упрощённая оценка влияния: доля потока через это ребро
	totalFlow := 0.0
	for _, e := range graph.Edges {
		if !IsVirtualNode(e.From) && !IsVirtualNode(e.To) {
			totalFlow += e.CurrentFlow
		}
	}

	if totalFlow <= Epsilon {
		return 0.0
	}

	return edge.CurrentFlow / totalFlow
}

func generateRecommendations(bottlenecks []*analyticsv1.Bottleneck) []*analyticsv1.Recommendation {
	var recommendations []*analyticsv1.Recommendation

	for _, b := range bottlenecks {
		if b.Severity >= analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH {
			recommendations = append(recommendations, &analyticsv1.Recommendation{
				Type:        "increase_capacity",
				Description: "Увеличьте пропускную способность ребра для устранения узкого места",
				AffectedEdge: &commonv1.EdgeKey{
					From: b.Edge.From,
					To:   b.Edge.To,
				},
				EstimatedImprovement: (1.0 - b.Utilization) * 100, // Потенциальное улучшение в %
				EstimatedCost:        b.Edge.Capacity * 0.5,       // Примерная оценка
			})
		}
	}

	return recommendations
}
