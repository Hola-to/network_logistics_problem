package handlers

import (
	"context"

	"connectrpc.com/connect"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	"logistics/services/gateway-svc/internal/clients"
)

// AnalyticsHandler обработчики аналитики
type AnalyticsHandler struct {
	clients *clients.Manager
}

// NewAnalyticsHandler создаёт handler
func NewAnalyticsHandler(clients *clients.Manager) *AnalyticsHandler {
	return &AnalyticsHandler{clients: clients}
}

func (h *AnalyticsHandler) AnalyzeGraph(
	ctx context.Context,
	req *connect.Request[gatewayv1.AnalyzeGraphRequest],
) (*connect.Response[gatewayv1.AnalyzeGraphResponse], error) {
	msg := req.Msg

	var opts *analyticsv1.AnalysisOptions
	if msg.Options != nil {
		opts = &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        msg.Options.AnalyzeCosts,
			FindBottlenecks:     msg.Options.FindBottlenecks,
			CalculateStatistics: msg.Options.CalculateStatistics,
			SuggestImprovements: msg.Options.SuggestImprovements,
			BottleneckThreshold: msg.Options.BottleneckThreshold,
		}
	}

	resp, err := h.clients.Analytics().AnalyzeFlow(ctx, msg.Graph, opts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.AnalyzeGraphResponse{
		FlowStats:   resp.FlowStats,
		GraphStats:  resp.GraphStats,
		Cost:        h.convertCostAnalysis(resp.Cost),
		Bottlenecks: h.convertBottleneckAnalysis(resp.Bottlenecks),
		Efficiency:  h.convertEfficiency(resp.Efficiency),
	}), nil
}

func (h *AnalyticsHandler) CalculateCost(
	ctx context.Context,
	req *connect.Request[gatewayv1.CalculateCostRequest],
) (*connect.Response[gatewayv1.CalculateCostResponse], error) {
	msg := req.Msg

	var opts *analyticsv1.CostOptions
	if msg.Options != nil {
		opts = &analyticsv1.CostOptions{
			Currency:          msg.Options.Currency,
			IncludeFixedCosts: msg.Options.IncludeFixedCosts,
			CostMultipliers:   msg.Options.CostMultipliers,
			DiscountPercent:   msg.Options.DiscountPercent,
			MarkupPercent:     msg.Options.MarkupPercent,
		}
	}

	resp, err := h.clients.Analytics().CalculateCost(ctx, msg.Graph, opts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.CalculateCostResponse{
		TotalCost: resp.TotalCost,
		Currency:  resp.Currency,
		Breakdown: h.convertCostBreakdown(resp.Breakdown),
	}), nil
}

func (h *AnalyticsHandler) GetBottlenecks(
	ctx context.Context,
	req *connect.Request[gatewayv1.BottlenecksRequest],
) (*connect.Response[gatewayv1.BottlenecksResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Analytics().FindBottlenecks(ctx, msg.Graph, msg.UtilizationThreshold, msg.TopN)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	bottlenecks := make([]*gatewayv1.Bottleneck, 0, len(resp.Bottlenecks))
	for _, b := range resp.Bottlenecks {
		bottlenecks = append(bottlenecks, &gatewayv1.Bottleneck{
			Edge: &commonv1.EdgeKey{
				From: b.Edge.From,
				To:   b.Edge.To,
			},
			Utilization: b.Utilization,
			ImpactScore: b.ImpactScore,
			Severity:    gatewayv1.BottleneckSeverity(b.Severity),
		})
	}

	recommendations := make([]*gatewayv1.Recommendation, 0, len(resp.Recommendations))
	for _, r := range resp.Recommendations {
		recommendations = append(recommendations, &gatewayv1.Recommendation{
			Type:                 r.Type,
			Description:          r.Description,
			AffectedEdge:         r.AffectedEdge,
			EstimatedImprovement: r.EstimatedImprovement,
			EstimatedCost:        r.EstimatedCost,
		})
	}

	return connect.NewResponse(&gatewayv1.BottlenecksResponse{
		Bottlenecks:     bottlenecks,
		Recommendations: recommendations,
	}), nil
}

func (h *AnalyticsHandler) CompareScenarios(
	ctx context.Context,
	req *connect.Request[gatewayv1.CompareScenariosRequest],
) (*connect.Response[gatewayv1.CompareScenariosResponse], error) {
	msg := req.Msg

	graphs := make([]*commonv1.Graph, 0, len(msg.Scenarios))
	names := make([]string, 0, len(msg.Scenarios))
	for _, s := range msg.Scenarios {
		graphs = append(graphs, s.Graph)
		names = append(names, s.Name)
	}

	resp, err := h.clients.Analytics().CompareScenarios(ctx, msg.Baseline, graphs, names)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	scenarios := make([]*gatewayv1.ScenarioResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		scenarios = append(scenarios, &gatewayv1.ScenarioResult{
			Name:                  r.Name,
			MaxFlow:               r.MaxFlow,
			TotalCost:             r.TotalCost,
			Efficiency:            r.Efficiency,
			ImprovementVsBaseline: r.ImprovementVsBaseline,
		})
	}

	var baseline *gatewayv1.ScenarioResult
	if len(scenarios) > 0 {
		baseline = scenarios[0]
		scenarios = scenarios[1:]
	}

	return connect.NewResponse(&gatewayv1.CompareScenariosResponse{
		Baseline:       baseline,
		Scenarios:      scenarios,
		BestScenario:   resp.BestScenario,
		Recommendation: resp.ComparisonSummary,
	}), nil
}

// Conversion helpers
func (h *AnalyticsHandler) convertCostAnalysis(c *analyticsv1.CalculateCostResponse) *gatewayv1.CostAnalysis {
	if c == nil {
		return nil
	}
	return &gatewayv1.CostAnalysis{
		TotalCost: c.TotalCost,
		Currency:  c.Currency,
		Breakdown: h.convertCostBreakdown(c.Breakdown),
	}
}

func (h *AnalyticsHandler) convertCostBreakdown(b *analyticsv1.CostBreakdown) *gatewayv1.CostBreakdown {
	if b == nil {
		return nil
	}
	return &gatewayv1.CostBreakdown{
		TransportCost:  b.TransportCost,
		FixedCost:      b.FixedCost,
		HandlingCost:   b.HandlingCost,
		DiscountAmount: b.DiscountAmount,
		MarkupAmount:   b.MarkupAmount,
		CostByRoadType: b.CostByRoadType,
		CostByNodeType: b.CostByNodeType,
	}
}

func (h *AnalyticsHandler) convertBottleneckAnalysis(b *analyticsv1.FindBottlenecksResponse) *gatewayv1.BottleneckAnalysis {
	if b == nil {
		return nil
	}

	bottlenecks := make([]*gatewayv1.Bottleneck, 0, len(b.Bottlenecks))
	for _, bn := range b.Bottlenecks {
		bottlenecks = append(bottlenecks, &gatewayv1.Bottleneck{
			Edge: &commonv1.EdgeKey{
				From: bn.Edge.From,
				To:   bn.Edge.To,
			},
			Utilization: bn.Utilization,
			ImpactScore: bn.ImpactScore,
			Severity:    gatewayv1.BottleneckSeverity(bn.Severity),
		})
	}

	recommendations := make([]*gatewayv1.Recommendation, 0, len(b.Recommendations))
	for _, r := range b.Recommendations {
		recommendations = append(recommendations, &gatewayv1.Recommendation{
			Type:                 r.Type,
			Description:          r.Description,
			AffectedEdge:         r.AffectedEdge,
			EstimatedImprovement: r.EstimatedImprovement,
			EstimatedCost:        r.EstimatedCost,
		})
	}

	return &gatewayv1.BottleneckAnalysis{
		Bottlenecks:      bottlenecks,
		Recommendations:  recommendations,
		TotalBottlenecks: int32(len(bottlenecks)),
	}
}

func (h *AnalyticsHandler) convertEfficiency(e *analyticsv1.EfficiencyReport) *gatewayv1.EfficiencyReport {
	if e == nil {
		return nil
	}
	return &gatewayv1.EfficiencyReport{
		OverallEfficiency:   e.OverallEfficiency,
		CapacityUtilization: e.CapacityUtilization,
		UnusedEdgesCount:    e.UnusedEdgesCount,
		SaturatedEdgesCount: e.SaturatedEdgesCount,
		Grade:               e.Grade,
	}
}
