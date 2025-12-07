// services/report-svc/internal/generator/types.go
package generator

import (
	"time"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
)

// =====================================================
// Внутренние типы для генераторов отчётов
// =====================================================

// FlowReportData данные для отчёта по потоку
type FlowReportData struct {
	AlgorithmUsed string
	GraphStats    *commonv1.GraphStatistics
	FlowStats     *commonv1.FlowStatistics
	Metrics       *optimizationv1.SolveMetrics
}

// AnalyticsReportData данные для аналитического отчёта
type AnalyticsReportData struct {
	TotalCost       float64
	Currency        string
	CostBreakdown   *CostBreakdownData
	Bottlenecks     []*BottleneckData
	Recommendations []*RecommendationData
	Efficiency      *EfficiencyData
	FlowStats       *commonv1.FlowStatistics
	GraphStats      *commonv1.GraphStatistics
}

// CostBreakdownData разбивка стоимости
type CostBreakdownData struct {
	TransportCost  float64
	FixedCost      float64
	HandlingCost   float64
	RoadBaseCost   float64
	DiscountAmount float64
	MarkupAmount   float64
	CostByRoadType map[string]float64
	CostByNodeType map[string]float64
	ActiveEdges    int32
	TotalFlow      float64
}

// BottleneckData данные об узком месте
type BottleneckData struct {
	From        int64
	To          int64
	Utilization float64
	ImpactScore float64
	Severity    string
}

// RecommendationData рекомендация
type RecommendationData struct {
	Type                 string
	Description          string
	AffectedEdgeFrom     int64
	AffectedEdgeTo       int64
	EstimatedImprovement float64
	EstimatedCost        float64
}

// EfficiencyData данные эффективности
type EfficiencyData struct {
	OverallEfficiency   float64
	CapacityUtilization float64
	UnusedEdges         int32
	SaturatedEdges      int32
	Grade               string
}

// SimulationReportData данные для отчёта симуляции
type SimulationReportData struct {
	SimulationType string
	BaselineFlow   float64
	BaselineCost   float64
	Scenarios      []*ScenarioData
	MonteCarlo     *MonteCarloData
	Sensitivity    []*SensitivityData
	Resilience     *ResilienceData
	TimeSteps      []*TimeStepData
}

// ScenarioData результат сценария
type ScenarioData struct {
	Name              string
	MaxFlow           float64
	TotalCost         float64
	FlowChangePercent float64
	ImpactLevel       string
}

// MonteCarloData результаты Monte Carlo
type MonteCarloData struct {
	Iterations      int32
	MeanFlow        float64
	StdDev          float64
	MinFlow         float64
	MaxFlow         float64
	P5              float64
	P50             float64
	P95             float64
	ConfidenceLevel float64
	CiLow           float64
	CiHigh          float64
}

// SensitivityData данные чувствительности
type SensitivityData struct {
	ParameterId      string
	Elasticity       float64
	SensitivityIndex float64
	Level            string
}

// ResilienceData данные устойчивости
type ResilienceData struct {
	OverallScore           float64
	SinglePointsOfFailure  int32
	WorstCaseFlowReduction float64
	NMinusOneFeasible      bool
}

// TimeStepData результат временного шага
type TimeStepData struct {
	Step               int32
	Timestamp          time.Time
	MaxFlow            float64
	TotalCost          float64
	AverageUtilization float64
	SaturatedEdges     int32
}

// ComparisonItemData элемент сравнения
type ComparisonItemData struct {
	Name       string
	MaxFlow    float64
	TotalCost  float64
	Efficiency float64
	Metrics    map[string]float64
}

// EdgeFlowData поток по ребру (для PDF/Excel)
type EdgeFlowData struct {
	From        int64
	To          int64
	Flow        float64
	Capacity    float64
	Cost        float64
	Utilization float64
}

// =====================================================
// Конвертеры из proto типов
// =====================================================

// ConvertBottlenecks конвертирует bottlenecks из proto
func ConvertBottlenecks(bottlenecks []*analyticsv1.Bottleneck) []*BottleneckData {
	result := make([]*BottleneckData, 0, len(bottlenecks))
	for _, b := range bottlenecks {
		if b == nil || b.Edge == nil {
			continue
		}
		result = append(result, &BottleneckData{
			From:        b.Edge.From,
			To:          b.Edge.To,
			Utilization: b.Utilization,
			ImpactScore: b.ImpactScore,
			Severity:    b.Severity.String(),
		})
	}
	return result
}

// ConvertRecommendations конвертирует recommendations из proto
func ConvertRecommendations(recs []*analyticsv1.Recommendation) []*RecommendationData {
	result := make([]*RecommendationData, 0, len(recs))
	for _, r := range recs {
		if r == nil {
			continue
		}
		data := &RecommendationData{
			Type:                 r.Type,
			Description:          r.Description,
			EstimatedImprovement: r.EstimatedImprovement,
			EstimatedCost:        r.EstimatedCost,
		}
		if r.AffectedEdge != nil {
			data.AffectedEdgeFrom = r.AffectedEdge.From
			data.AffectedEdgeTo = r.AffectedEdge.To
		}
		result = append(result, data)
	}
	return result
}

// ConvertEfficiency конвертирует efficiency из proto
func ConvertEfficiency(e *analyticsv1.EfficiencyReport) *EfficiencyData {
	if e == nil {
		return nil
	}
	return &EfficiencyData{
		OverallEfficiency:   e.OverallEfficiency,
		CapacityUtilization: e.CapacityUtilization,
		UnusedEdges:         e.UnusedEdgesCount,
		SaturatedEdges:      e.SaturatedEdgesCount,
		Grade:               e.Grade,
	}
}

// ConvertCostBreakdown конвертирует cost breakdown из proto
func ConvertCostBreakdown(cb *analyticsv1.CostBreakdown) *CostBreakdownData {
	if cb == nil {
		return nil
	}
	return &CostBreakdownData{
		TransportCost:  cb.TransportCost,
		FixedCost:      cb.FixedCost,
		HandlingCost:   cb.HandlingCost,
		RoadBaseCost:   cb.RoadBaseCost,
		DiscountAmount: cb.DiscountAmount,
		MarkupAmount:   cb.MarkupAmount,
		CostByRoadType: cb.CostByRoadType,
		CostByNodeType: cb.CostByNodeType,
		ActiveEdges:    cb.ActiveEdges,
		TotalFlow:      cb.TotalFlow,
	}
}

// ConvertScenarioResults конвертирует результаты сценариев
func ConvertScenarioResults(results []*simulationv1.ScenarioResultWithRank) []*ScenarioData {
	data := make([]*ScenarioData, 0, len(results))
	for _, r := range results {
		if r == nil || r.Result == nil {
			continue
		}
		sd := &ScenarioData{
			Name:      r.Result.Name,
			MaxFlow:   r.Result.MaxFlow,
			TotalCost: r.Result.TotalCost,
		}
		if r.VsBaseline != nil {
			sd.FlowChangePercent = r.VsBaseline.FlowChangePercent
			sd.ImpactLevel = r.VsBaseline.ImpactLevel.String()
		}
		data = append(data, sd)
	}
	return data
}

// ConvertMonteCarloStats конвертирует статистику Monte Carlo
func ConvertMonteCarloStats(resp *simulationv1.RunMonteCarloResponse) *MonteCarloData {
	if resp == nil || resp.FlowStats == nil {
		return nil
	}
	fs := resp.FlowStats
	data := &MonteCarloData{
		MeanFlow:        fs.Mean,
		StdDev:          fs.StdDev,
		MinFlow:         fs.Min,
		MaxFlow:         fs.Max,
		ConfidenceLevel: fs.ConfidenceIntervalLow, // Предполагаем что это уровень
		CiLow:           fs.ConfidenceIntervalLow,
		CiHigh:          fs.ConfidenceIntervalHigh,
	}
	// Percentiles
	if p, ok := resp.FlowPercentiles["p5"]; ok {
		data.P5 = p
	}
	if p, ok := resp.FlowPercentiles["p50"]; ok {
		data.P50 = p
	}
	if p, ok := resp.FlowPercentiles["p95"]; ok {
		data.P95 = p
	}
	return data
}

// ConvertSensitivityResults конвертирует результаты анализа чувствительности
func ConvertSensitivityResults(results []*simulationv1.SensitivityResult) []*SensitivityData {
	data := make([]*SensitivityData, 0, len(results))
	for _, r := range results {
		if r == nil {
			continue
		}
		data = append(data, &SensitivityData{
			ParameterId:      r.ParameterId,
			Elasticity:       r.Elasticity,
			SensitivityIndex: r.SensitivityIndex,
			Level:            r.Level.String(),
		})
	}
	return data
}

// ConvertResilienceMetrics конвертирует метрики устойчивости
func ConvertResilienceMetrics(resp *simulationv1.AnalyzeResilienceResponse) *ResilienceData {
	if resp == nil || resp.Metrics == nil {
		return nil
	}
	data := &ResilienceData{
		OverallScore: resp.Metrics.OverallScore,
	}
	if resp.NMinusOne != nil {
		data.NMinusOneFeasible = resp.NMinusOne.AllScenariosFeasible
		data.WorstCaseFlowReduction = resp.NMinusOne.WorstCaseFlowReduction
	}
	// Считаем single points of failure из weaknesses
	for _, w := range resp.Weaknesses {
		if w != nil && w.Type == simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE {
			data.SinglePointsOfFailure++
		}
	}
	return data
}

// ConvertFlowEdges конвертирует рёбра с потоком
func ConvertFlowEdges(edges []*commonv1.FlowEdge) []*EdgeFlowData {
	result := make([]*EdgeFlowData, 0, len(edges))
	for _, e := range edges {
		if e == nil {
			continue
		}
		result = append(result, &EdgeFlowData{
			From:        e.From,
			To:          e.To,
			Flow:        e.Flow,
			Capacity:    e.Capacity,
			Cost:        e.Cost,
			Utilization: e.Utilization,
		})
	}
	return result
}
