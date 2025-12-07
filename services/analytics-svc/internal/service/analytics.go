package service

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/metrics"
	"logistics/pkg/telemetry"
	"logistics/services/analytics-svc/internal/analysis"
)

type AnalyticsService struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	metrics *metrics.Metrics
}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{
		metrics: metrics.Get(),
	}
}

func (s *AnalyticsService) CalculateCost(
	ctx context.Context,
	req *analyticsv1.CalculateCostRequest,
) (*analyticsv1.CalculateCostResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AnalyticsService.CalculateCost")
	defer span.End()

	// Валидация входных данных
	if err := s.validateGraph(ctx, req.Graph); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(err)
	}

	// Добавляем атрибуты
	telemetry.SetAttributes(ctx, telemetry.GraphAttributes(
		len(req.Graph.Nodes),
		len(req.Graph.Edges),
		req.Graph.SourceId,
		req.Graph.SinkId,
	)...)

	result := analysis.CalculateCost(req.Graph, req.Options)

	// Логируем результат
	telemetry.AddEvent(ctx, "cost_calculated",
		attribute.Float64("total_cost", result.TotalCost),
		attribute.String("currency", result.Currency),
	)

	span.SetAttributes(
		attribute.Float64("total_cost", result.TotalCost),
		attribute.String("currency", result.Currency),
	)

	return result, nil
}

func (s *AnalyticsService) FindBottlenecks(
	ctx context.Context,
	req *analyticsv1.FindBottlenecksRequest,
) (*analyticsv1.FindBottlenecksResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AnalyticsService.FindBottlenecks",
		trace.WithAttributes(
			attribute.Float64("threshold", req.UtilizationThreshold),
			attribute.Int("top_n", int(req.TopN)),
		),
	)
	defer span.End()

	if err := s.validateGraph(ctx, req.Graph); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(err)
	}

	threshold := req.UtilizationThreshold
	if threshold <= 0 {
		threshold = 0.9
	}

	result := analysis.FindBottlenecks(req.Graph, threshold, req.TopN)

	// Записываем метрики и телеметрию
	bottleneckCount := len(result.Bottlenecks)
	span.SetAttributes(attribute.Int("bottlenecks_found", bottleneckCount))

	telemetry.AddEvent(ctx, "bottlenecks_found",
		attribute.Int("count", bottleneckCount),
		attribute.Int("recommendations", len(result.Recommendations)),
	)

	if s.metrics != nil && bottleneckCount > 0 {
		severityCounts := make(map[string]int)
		for _, b := range result.Bottlenecks {
			severityCounts[b.Severity.String()]++
		}
		for severity, count := range severityCounts {
			s.metrics.RecordBottlenecks(severity, count)
		}
	}

	return result, nil
}

func (s *AnalyticsService) AnalyzeFlow(
	ctx context.Context,
	req *analyticsv1.AnalyzeFlowRequest,
) (*analyticsv1.AnalyzeFlowResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AnalyticsService.AnalyzeFlow")
	defer span.End()

	if err := s.validateGraph(ctx, req.Graph); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(err)
	}

	response := &analyticsv1.AnalyzeFlowResponse{}

	if req.Options == nil {
		req.Options = &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
		}
	}

	// Статистика
	if req.Options.CalculateStatistics {
		_, statsSpan := telemetry.StartSpan(ctx, "CalculateStatistics")
		response.FlowStats = analysis.CalculateFlowStatistics(req.Graph)
		response.GraphStats = analysis.CalculateGraphStatistics(req.Graph)
		statsSpan.End()
	}

	// Стоимость
	if req.Options.AnalyzeCosts {
		_, costSpan := telemetry.StartSpan(ctx, "CalculateCost")
		response.Cost = analysis.CalculateCost(req.Graph, nil)
		costSpan.End()
	}

	// Bottlenecks
	if req.Options.FindBottlenecks {
		_, bnSpan := telemetry.StartSpan(ctx, "FindBottlenecks")
		threshold := req.Options.BottleneckThreshold
		if threshold <= 0 {
			threshold = 0.9
		}
		response.Bottlenecks = analysis.FindBottlenecks(req.Graph, threshold, 0)
		bnSpan.End()
	}

	// Эффективность
	response.Efficiency = calculateEfficiency(req.Graph, response.FlowStats)

	// Добавляем результаты в span
	if response.FlowStats != nil {
		span.SetAttributes(
			attribute.Float64("total_flow", response.FlowStats.TotalFlow),
			attribute.Float64("avg_utilization", response.FlowStats.AverageUtilization),
		)
	}
	if response.Efficiency != nil {
		span.SetAttributes(
			attribute.String("efficiency_grade", response.Efficiency.Grade),
		)
	}

	return response, nil
}

func (s *AnalyticsService) CompareScenarios(
	ctx context.Context,
	req *analyticsv1.CompareScenariosRequest,
) (*analyticsv1.CompareScenariosResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AnalyticsService.CompareScenarios",
		trace.WithAttributes(
			attribute.Int("scenarios_count", len(req.Scenarios)),
		),
	)
	defer span.End()

	if req.Baseline == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.NewWithField(pkgerrors.CodeInvalidArgument, "baseline graph is required", "baseline"),
		)
	}

	results := make([]*analyticsv1.ScenarioResult, 0, len(req.Scenarios))

	// Базовый сценарий
	baselineStats := analysis.CalculateFlowStatistics(req.Baseline)
	baselineCost := analysis.CalculateCost(req.Baseline, nil)

	telemetry.AddEvent(ctx, "baseline_analyzed",
		attribute.Float64("flow", baselineStats.TotalFlow),
		attribute.Float64("cost", baselineCost.TotalCost),
	)

	// Сравниваем каждый сценарий
	for i, scenario := range req.Scenarios {
		name := ""
		if i < len(req.ScenarioNames) {
			name = req.ScenarioNames[i]
		} else {
			name = fmt.Sprintf("Сценарий %c", 'A'+i)
		}

		stats := analysis.CalculateFlowStatistics(scenario)
		cost := analysis.CalculateCost(scenario, nil)

		improvement := 0.0
		if baselineStats.TotalFlow > 0 {
			improvement = ((stats.TotalFlow - baselineStats.TotalFlow) / baselineStats.TotalFlow) * 100
		}

		results = append(results, &analyticsv1.ScenarioResult{
			Name:                  name,
			MaxFlow:               stats.TotalFlow,
			TotalCost:             cost.TotalCost,
			Efficiency:            stats.AverageUtilization,
			ImprovementVsBaseline: improvement,
		})

		telemetry.AddEvent(ctx, "scenario_analyzed",
			attribute.String("name", name),
			attribute.Float64("flow", stats.TotalFlow),
			attribute.Float64("improvement", improvement),
		)
	}

	// Находим лучший сценарий
	bestScenario := ""
	bestFlow := baselineStats.TotalFlow
	for _, r := range results {
		if r.MaxFlow > bestFlow {
			bestFlow = r.MaxFlow
			bestScenario = r.Name
		}
	}

	span.SetAttributes(
		attribute.String("best_scenario", bestScenario),
		attribute.Float64("best_flow", bestFlow),
	)

	return &analyticsv1.CompareScenariosResponse{
		Results:           results,
		BestScenario:      bestScenario,
		ComparisonSummary: generateComparisonSummary(results, baselineStats, baselineCost),
	}, nil
}

// validateGraph валидирует граф
func (s *AnalyticsService) validateGraph(ctx context.Context, graph *commonv1.Graph) error {
	if graph == nil {
		return pkgerrors.ErrNilGraph
	}

	if len(graph.Nodes) == 0 {
		return pkgerrors.ErrEmptyGraph
	}

	return nil
}

func calculateEfficiency(graph *commonv1.Graph, stats *commonv1.FlowStatistics) *analyticsv1.EfficiencyReport {
	if stats == nil {
		stats = analysis.CalculateFlowStatistics(graph)
	}

	grade := "F"
	switch {
	case stats.AverageUtilization >= 0.8:
		grade = "A"
	case stats.AverageUtilization >= 0.6:
		grade = "B"
	case stats.AverageUtilization >= 0.4:
		grade = "C"
	case stats.AverageUtilization >= 0.2:
		grade = "D"
	}

	return &analyticsv1.EfficiencyReport{
		OverallEfficiency:   stats.AverageUtilization,
		CapacityUtilization: stats.AverageUtilization,
		UnusedEdgesCount:    int32(stats.ZeroFlowEdges),
		SaturatedEdgesCount: int32(stats.SaturatedEdges),
		Grade:               grade,
	}
}

func generateComparisonSummary(
	results []*analyticsv1.ScenarioResult,
	baseStats *commonv1.FlowStatistics,
	_ *analyticsv1.CalculateCostResponse,
) string {
	if len(results) == 0 {
		return "Нет сценариев для сравнения"
	}

	best := ""
	maxImprovement := 0.0
	for _, r := range results {
		if r.ImprovementVsBaseline > maxImprovement {
			maxImprovement = r.ImprovementVsBaseline
			best = r.Name
		}
	}

	if best != "" {
		return fmt.Sprintf("Лучший сценарий: %s (улучшение %.1f%%). Базовый поток: %.2f",
			best, maxImprovement, baseStats.TotalFlow)
	}

	return fmt.Sprintf("Все сценарии хуже базового. Базовый поток: %.2f", baseStats.TotalFlow)
}
