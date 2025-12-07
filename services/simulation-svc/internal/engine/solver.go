// services/simulation-svc/internal/engine/solver.go
package engine

import (
	"context"
	"fmt"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
)

// SolverEngine обёртка над solver клиентом
type SolverEngine struct {
	client *client.SolverClient
}

// NewSolverEngine создаёт новый движок
func NewSolverEngine(solverClient *client.SolverClient) *SolverEngine {
	return &SolverEngine{
		client: solverClient,
	}
}

// SolveResult результат решения (локальный алиас)
type SolveResult = client.SolveResult

// Solve решает задачу потока
func (e *SolverEngine) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm) (*SolveResult, error) {
	if e.client == nil {
		return nil, fmt.Errorf("solver client not initialized")
	}

	// Сбрасываем потоки
	ResetFlow(graph)

	return e.client.Solve(ctx, graph, algorithm, nil)
}

// ToScenarioResult конвертирует результат в proto
func ToScenarioResult(r *SolveResult, name string) *simulationv1.ScenarioResult {
	if r == nil {
		return &simulationv1.ScenarioResult{Name: name}
	}

	return &simulationv1.ScenarioResult{
		Name:               name,
		MaxFlow:            r.MaxFlow,
		TotalCost:          r.TotalCost,
		AverageUtilization: r.AverageUtilization,
		SaturatedEdges:     r.SaturatedEdges,
		ActivePaths:        r.ActivePaths,
		Status:             r.Status,
	}
}

// CompareResults сравнивает два результата
func CompareResults(baseline, modified *SolveResult) *simulationv1.ScenarioComparison {
	if baseline == nil || modified == nil {
		return &simulationv1.ScenarioComparison{}
	}

	flowChange := modified.MaxFlow - baseline.MaxFlow
	flowChangePercent := 0.0
	if baseline.MaxFlow > 0 {
		flowChangePercent = (flowChange / baseline.MaxFlow) * 100
	}

	costChange := modified.TotalCost - baseline.TotalCost
	costChangePercent := 0.0
	if baseline.TotalCost > 0 {
		costChangePercent = (costChange / baseline.TotalCost) * 100
	}

	utilChange := modified.AverageUtilization - baseline.AverageUtilization

	impactLevel := determineImpactLevel(flowChangePercent)
	summary := generateImpactSummary(flowChangePercent, costChangePercent, impactLevel)

	return &simulationv1.ScenarioComparison{
		FlowChange:        flowChange,
		FlowChangePercent: flowChangePercent,
		CostChange:        costChange,
		CostChangePercent: costChangePercent,
		UtilizationChange: utilChange,
		ImpactLevel:       impactLevel,
		ImpactSummary:     summary,
	}
}

func determineImpactLevel(changePercent float64) simulationv1.ImpactLevel {
	absChange := changePercent
	if absChange < 0 {
		absChange = -absChange
	}

	switch {
	case absChange < 1:
		return simulationv1.ImpactLevel_IMPACT_LEVEL_NONE
	case absChange < 5:
		return simulationv1.ImpactLevel_IMPACT_LEVEL_LOW
	case absChange < 15:
		return simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM
	case absChange < 30:
		return simulationv1.ImpactLevel_IMPACT_LEVEL_HIGH
	default:
		return simulationv1.ImpactLevel_IMPACT_LEVEL_CRITICAL
	}
}

func generateImpactSummary(flowChange, costChange float64, level simulationv1.ImpactLevel) string {
	flowDir := "увеличился"
	if flowChange < 0 {
		flowDir = "уменьшился"
	}

	costDir := "увеличилась"
	if costChange < 0 {
		costDir = "уменьшилась"
	}

	switch level {
	case simulationv1.ImpactLevel_IMPACT_LEVEL_NONE:
		return "Изменения практически не влияют на производительность сети"
	case simulationv1.ImpactLevel_IMPACT_LEVEL_LOW:
		return fmt.Sprintf("Незначительное влияние: поток %s на %.1f%%, стоимость %s на %.1f%%",
			flowDir, abs(flowChange), costDir, abs(costChange))
	case simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM:
		return fmt.Sprintf("Умеренное влияние: поток %s на %.1f%%, стоимость %s на %.1f%%",
			flowDir, abs(flowChange), costDir, abs(costChange))
	case simulationv1.ImpactLevel_IMPACT_LEVEL_HIGH:
		return fmt.Sprintf("Значительное влияние: поток %s на %.1f%%, стоимость %s на %.1f%%",
			flowDir, abs(flowChange), costDir, abs(costChange))
	case simulationv1.ImpactLevel_IMPACT_LEVEL_CRITICAL:
		return fmt.Sprintf("КРИТИЧЕСКОЕ влияние: поток %s на %.1f%%, стоимость %s на %.1f%%",
			flowDir, abs(flowChange), costDir, abs(costChange))
	default:
		return "Влияние не определено"
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
