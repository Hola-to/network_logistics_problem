// services/simulation-svc/internal/engine/sensitivity.go
package engine

import (
	"context"
	"fmt"
	"math"
	"sort"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
	"logistics/pkg/logger"
)

// SensitivityEngine движок анализа чувствительности
type SensitivityEngine struct {
	solverClient SolverClientInterface
}

// NewSensitivityEngine создаёт новый движок
func NewSensitivityEngine(solverClient SolverClientInterface) *SensitivityEngine {
	return &SensitivityEngine{
		solverClient: solverClient,
	}
}

// AnalyzeSensitivity выполняет анализ чувствительности
func (e *SensitivityEngine) AnalyzeSensitivity(
	ctx context.Context,
	graph *commonv1.Graph,
	params []*simulationv1.SensitivityParameter,
	config *simulationv1.SensitivityConfig,
	algorithm commonv1.Algorithm,
) (*simulationv1.AnalyzeSensitivityResponse, error) {
	// Базовый результат
	baseResult, err := e.solverClient.Solve(ctx, graph, algorithm, nil)
	if err != nil {
		return nil, err
	}

	results := make([]*simulationv1.SensitivityResult, 0, len(params))
	rankings := make([]*simulationv1.ParameterRanking, 0, len(params))
	var thresholds []*simulationv1.ThresholdPoint

	for _, param := range params {
		paramID := e.buildParamID(param)
		paramResult, paramThresholds := e.analyzeParameter(ctx, graph, param, baseResult, algorithm, config)
		paramResult.ParameterId = paramID
		results = append(results, paramResult)
		thresholds = append(thresholds, paramThresholds...)

		rankings = append(rankings, &simulationv1.ParameterRanking{
			ParameterId:      paramID,
			SensitivityIndex: paramResult.SensitivityIndex,
		})
	}

	// Сортируем rankings
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].SensitivityIndex > rankings[j].SensitivityIndex
	})
	for i := range rankings {
		rankings[i].Rank = int32(i + 1)
		rankings[i].Description = e.describeRank(rankings[i])
	}

	return &simulationv1.AnalyzeSensitivityResponse{
		Success:          true,
		ParameterResults: results,
		Rankings:         rankings,
		Thresholds:       thresholds,
	}, nil
}

func (e *SensitivityEngine) buildParamID(param *simulationv1.SensitivityParameter) string {
	if param.Edge != nil {
		return fmt.Sprintf("edge_%d_%d_%s", param.Edge.From, param.Edge.To, param.Target.String())
	}
	if param.NodeId > 0 {
		return fmt.Sprintf("node_%d_%s", param.NodeId, param.Target.String())
	}
	return fmt.Sprintf("param_%s", param.Target.String())
}

func (e *SensitivityEngine) analyzeParameter(
	ctx context.Context,
	graph *commonv1.Graph,
	param *simulationv1.SensitivityParameter,
	baseResult *client.SolveResult,
	algorithm commonv1.Algorithm,
	config *simulationv1.SensitivityConfig,
) (*simulationv1.SensitivityResult, []*simulationv1.ThresholdPoint) {
	numSteps := int(param.NumSteps)
	if numSteps <= 0 {
		numSteps = 10
	}

	minMult := param.MinMultiplier
	maxMult := param.MaxMultiplier
	if minMult == 0 {
		minMult = 0.5
	}
	if maxMult == 0 {
		maxMult = 1.5
	}

	step := (maxMult - minMult) / float64(numSteps-1)
	curve := make([]*simulationv1.SensitivityPoint, 0, numSteps)

	var minFlow, maxFlow float64 = math.MaxFloat64, 0
	var thresholds []*simulationv1.ThresholdPoint
	var prevFlow float64

	for i := 0; i < numSteps; i++ {
		multiplier := minMult + float64(i)*step

		// Создаём модификацию
		mod := &simulationv1.Modification{
			Target: param.Target,
		}
		mod.Change = &simulationv1.Modification_RelativeChange{
			RelativeChange: multiplier,
		}

		if param.Edge != nil {
			mod.Type = simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE
			mod.EdgeKey = param.Edge
		} else if param.NodeId > 0 {
			mod.Type = simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_NODE
			mod.NodeId = param.NodeId
		}

		modifiedGraph := ApplyModifications(graph, []*simulationv1.Modification{mod})
		result, err := e.solverClient.Solve(ctx, modifiedGraph, algorithm, nil)

		flow := 0.0
		cost := 0.0
		if err != nil {
			// Логируем ошибку, но продолжаем с нулевыми значениями
			logger.Log.Warn("Failed to solve for sensitivity step",
				"step", i,
				"multiplier", multiplier,
				"error", err,
			)
		} else if result != nil {
			flow = result.MaxFlow
			cost = result.TotalCost
		}

		point := &simulationv1.SensitivityPoint{
			ParameterValue: multiplier,
			FlowValue:      flow,
			CostValue:      cost,
		}
		curve = append(curve, point)

		if flow < minFlow {
			minFlow = flow
		}
		if flow > maxFlow {
			maxFlow = flow
		}

		// Обнаружение порогов
		if config != nil && config.FindThresholds && i > 0 && prevFlow > 0 {
			flowDrop := (prevFlow - flow) / prevFlow * 100
			if flowDrop > 10 {
				thresholds = append(thresholds, &simulationv1.ThresholdPoint{
					ParameterId:    e.buildParamID(param),
					ThresholdValue: multiplier,
					Type:           simulationv1.ThresholdType_THRESHOLD_TYPE_FLOW_DROPS,
					Description:    fmt.Sprintf("Поток падает на %.1f%% при множителе %.2f", flowDrop, multiplier),
				})
			}
		}
		prevFlow = flow
	}

	// Расчёт эластичности
	elasticity := 0.0
	if len(curve) >= 2 && baseResult.MaxFlow > 0 {
		mid := len(curve) / 2
		if mid > 0 && mid < len(curve)-1 {
			dFlow := (curve[mid+1].FlowValue - curve[mid-1].FlowValue) / baseResult.MaxFlow
			dParam := (curve[mid+1].ParameterValue - curve[mid-1].ParameterValue)
			if dParam != 0 {
				elasticity = dFlow / dParam
			}
		}
	}

	// Нормализованный индекс чувствительности
	impactRange := maxFlow - minFlow
	sensitivityIndex := 0.0
	if baseResult.MaxFlow > 0 {
		sensitivityIndex = impactRange / baseResult.MaxFlow
	}

	level := determineSensitivityLevel(sensitivityIndex)

	return &simulationv1.SensitivityResult{
		Curve:            curve,
		Elasticity:       elasticity,
		SensitivityIndex: sensitivityIndex,
		ImpactRange:      impactRange,
		Level:            level,
	}, thresholds
}

func determineSensitivityLevel(index float64) simulationv1.SensitivityLevel {
	switch {
	case index < 0.01:
		return simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_NEGLIGIBLE
	case index < 0.05:
		return simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_LOW
	case index < 0.15:
		return simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM
	case index < 0.30:
		return simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH
	default:
		return simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_CRITICAL
	}
}

func (e *SensitivityEngine) describeRank(r *simulationv1.ParameterRanking) string {
	switch {
	case r.Rank == 1:
		return "Наиболее критичный параметр"
	case r.Rank <= 3:
		return "Высокоприоритетный параметр"
	case r.Rank <= 5:
		return "Параметр средней важности"
	default:
		return "Параметр низкой важности"
	}
}
