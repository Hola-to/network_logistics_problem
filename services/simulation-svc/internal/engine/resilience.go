// services/simulation-svc/internal/engine/resilience.go
package engine

import (
	"context"
	"math"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
)

// ResilienceEngine движок анализа устойчивости
type ResilienceEngine struct {
	solverClient SolverClientInterface // Изменено на интерфейс
}

// NewResilienceEngine создаёт новый движок
func NewResilienceEngine(solverClient SolverClientInterface) *ResilienceEngine {
	return &ResilienceEngine{
		solverClient: solverClient,
	}
}

// AnalyzeResilience выполняет полный анализ устойчивости
func (e *ResilienceEngine) AnalyzeResilience(
	ctx context.Context,
	graph *commonv1.Graph,
	config *simulationv1.ResilienceConfig,
	algorithm commonv1.Algorithm,
) (*simulationv1.AnalyzeResilienceResponse, error) {
	// Базовый результат
	baseResult, err := e.solverClient.Solve(ctx, graph, algorithm, nil)
	if err != nil {
		return nil, err
	}

	response := &simulationv1.AnalyzeResilienceResponse{
		Success: true,
	}

	// N-1 анализ
	n1 := e.performN1Analysis(ctx, graph, algorithm, baseResult)
	response.NMinusOne = n1.analysis

	// Метрики
	response.Metrics = &simulationv1.ResilienceMetrics{
		OverallScore:           n1.overallScore,
		ConnectivityRobustness: n1.connectivityRobustness,
		FlowRobustness:         n1.flowRobustness,
		RedundancyLevel:        n1.redundancyLevel,
		MinCutSize:             n1.minCutSize,
	}

	// Слабости
	response.Weaknesses = e.identifyWeaknesses(n1, graph)

	return response, nil
}

type n1Result struct {
	analysis               *simulationv1.NMinusOneAnalysis
	overallScore           float64
	connectivityRobustness float64
	flowRobustness         float64
	redundancyLevel        float64
	minCutSize             int32
	spofEdges              []*commonv1.EdgeKey
	spofNodes              []int64
}

func (e *ResilienceEngine) performN1Analysis(
	ctx context.Context,
	graph *commonv1.Graph,
	algorithm commonv1.Algorithm,
	baseResult *client.SolveResult,
) *n1Result {
	result := &n1Result{
		analysis: &simulationv1.NMinusOneAnalysis{
			AllScenariosFeasible: true,
		},
		spofEdges: make([]*commonv1.EdgeKey, 0),
		spofNodes: make([]int64, 0),
	}

	var worstFlowReduction float64
	var scenariosTested, scenariosFailed int

	// Тестируем удаление каждого ребра
	for _, edge := range graph.Edges {
		modGraph := e.removeEdge(graph, edge.From, edge.To)
		modResult, err := e.solverClient.Solve(ctx, modGraph, algorithm, nil)
		scenariosTested++

		if err != nil {
			scenariosFailed++
			result.analysis.AllScenariosFeasible = false
			result.spofEdges = append(result.spofEdges, &commonv1.EdgeKey{
				From: edge.From,
				To:   edge.To,
			})
			continue
		}

		if modResult.MaxFlow == 0 && baseResult.MaxFlow > 0 {
			scenariosFailed++
			result.analysis.AllScenariosFeasible = false
			result.spofEdges = append(result.spofEdges, &commonv1.EdgeKey{
				From: edge.From,
				To:   edge.To,
			})
			continue
		}

		reduction := baseResult.MaxFlow - modResult.MaxFlow
		if reduction > worstFlowReduction {
			worstFlowReduction = reduction
			result.analysis.MostCriticalEdge = &commonv1.EdgeKey{
				From: edge.From,
				To:   edge.To,
			}
		}
	}

	result.analysis.ScenariosTested = int32(scenariosTested)
	result.analysis.ScenariosFailed = int32(scenariosFailed)
	result.analysis.WorstCaseFlowReduction = worstFlowReduction

	// Вычисляем метрики
	if scenariosTested > 0 {
		result.connectivityRobustness = float64(scenariosTested-scenariosFailed) / float64(scenariosTested)
	}

	if baseResult.MaxFlow > 0 {
		result.flowRobustness = 1.0 - (worstFlowReduction / baseResult.MaxFlow)
	} else {
		result.flowRobustness = 0
	}

	if len(graph.Nodes) > 0 {
		result.redundancyLevel = float64(len(graph.Edges)) / float64(len(graph.Nodes))
	}

	result.minCutSize = e.estimateMinCut(graph, scenariosFailed)
	result.overallScore = (result.connectivityRobustness + result.flowRobustness) / 2

	return result
}

func (e *ResilienceEngine) removeEdge(g *commonv1.Graph, from, to int64) *commonv1.Graph {
	clone := CloneGraph(g)
	newEdges := make([]*commonv1.Edge, 0, len(clone.Edges)-1)
	for _, edge := range clone.Edges {
		if edge.From != from || edge.To != to {
			newEdges = append(newEdges, edge)
		}
	}
	clone.Edges = newEdges
	return clone
}

func (e *ResilienceEngine) estimateMinCut(graph *commonv1.Graph, failedScenarios int) int32 {
	if failedScenarios > 0 {
		return 1
	}

	minDegree := len(graph.Edges)
	degree := make(map[int64]int)

	for _, edge := range graph.Edges {
		degree[edge.From]++
		degree[edge.To]++
	}

	for _, node := range graph.Nodes {
		if node.Id == graph.SourceId || node.Id == graph.SinkId {
			continue
		}
		if d := degree[node.Id]; d < minDegree && d > 0 {
			minDegree = d
		}
	}

	return int32(minDegree)
}

func (e *ResilienceEngine) identifyWeaknesses(n1 *n1Result, graph *commonv1.Graph) []*simulationv1.ResilienceWeakness {
	var weaknesses []*simulationv1.ResilienceWeakness

	if len(n1.spofEdges) > 0 {
		edgeKeys := make([]*commonv1.EdgeKey, len(n1.spofEdges))
		copy(edgeKeys, n1.spofEdges)

		weaknesses = append(weaknesses, &simulationv1.ResilienceWeakness{
			Type:                 simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE,
			Description:          "Обнаружены критические рёбра, удаление которых разрывает сеть",
			Severity:             1.0,
			AffectedEdges:        edgeKeys,
			MitigationSuggestion: "Добавьте альтернативные маршруты для критических рёбер",
		})
	}

	if n1.flowRobustness < 0.7 {
		weaknesses = append(weaknesses, &simulationv1.ResilienceWeakness{
			Type:                 simulationv1.WeaknessType_WEAKNESS_TYPE_CAPACITY_BOTTLENECK,
			Description:          "Низкая устойчивость потока к единичным отказам",
			Severity:             1.0 - n1.flowRobustness,
			MitigationSuggestion: "Увеличьте пропускную способность резервных маршрутов",
		})
	}

	if n1.redundancyLevel < 1.5 {
		weaknesses = append(weaknesses, &simulationv1.ResilienceWeakness{
			Type:                 simulationv1.WeaknessType_WEAKNESS_TYPE_NO_REDUNDANCY,
			Description:          "Низкий уровень резервирования сети",
			Severity:             0.5,
			MitigationSuggestion: "Добавьте дополнительные связи между узлами",
		})
	}

	if e.hasGeographicConcentration(graph) {
		weaknesses = append(weaknesses, &simulationv1.ResilienceWeakness{
			Type:                 simulationv1.WeaknessType_WEAKNESS_TYPE_GEOGRAPHIC_CONCENTRATION,
			Description:          "Высокая географическая концентрация узлов",
			Severity:             0.3,
			MitigationSuggestion: "Рассмотрите распределение узлов по разным географическим зонам",
		})
	}

	return weaknesses
}

func (e *ResilienceEngine) hasGeographicConcentration(graph *commonv1.Graph) bool {
	if graph == nil || len(graph.Nodes) < 3 {
		return false
	}

	var sumX, sumY float64
	var count int
	for _, node := range graph.Nodes {
		if node.X != 0 || node.Y != 0 {
			sumX += node.X
			sumY += node.Y
			count++
		}
	}

	if count < 3 {
		return false
	}

	centerX := sumX / float64(count)
	centerY := sumY / float64(count)

	var totalDist float64
	for _, node := range graph.Nodes {
		if node.X != 0 || node.Y != 0 {
			dx := node.X - centerX
			dy := node.Y - centerY
			totalDist += math.Sqrt(dx*dx + dy*dy)
		}
	}
	avgDist := totalDist / float64(count)

	return avgDist < 10
}
