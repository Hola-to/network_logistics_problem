package handlers

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/services/gateway-svc/internal/clients"
	"logistics/services/gateway-svc/internal/middleware"
)

// SimulationHandler обработчики симуляций
type SimulationHandler struct {
	clients *clients.Manager
}

// NewSimulationHandler создаёт handler
func NewSimulationHandler(clients *clients.Manager) *SimulationHandler {
	return &SimulationHandler{clients: clients}
}

func (h *SimulationHandler) RunWhatIf(
	ctx context.Context,
	req *connect.Request[gatewayv1.WhatIfRequest],
) (*connect.Response[gatewayv1.WhatIfResponse], error) {
	msg := req.Msg

	modifications := make([]*simulationv1.Modification, 0, len(msg.Modifications))
	for _, m := range msg.Modifications {
		mod := &simulationv1.Modification{
			Type:        simulationv1.ModificationType(m.Type),
			EdgeKey:     m.EdgeKey,
			NodeId:      m.NodeId,
			Target:      simulationv1.ModificationTarget(m.Target),
			Description: m.Description,
		}
		if m.IsRelative {
			mod.Change = &simulationv1.Modification_RelativeChange{RelativeChange: m.Value}
		} else {
			mod.Change = &simulationv1.Modification_AbsoluteValue{AbsoluteValue: m.Value}
		}
		modifications = append(modifications, mod)
	}

	var opts *simulationv1.WhatIfOptions
	if msg.Options != nil {
		opts = &simulationv1.WhatIfOptions{
			CompareWithBaseline: msg.Options.CompareWithBaseline,
			CalculateCostImpact: msg.Options.CalculateCostImpact,
			FindNewBottlenecks:  msg.Options.FindNewBottlenecks,
			ReturnModifiedGraph: msg.Options.ReturnModifiedGraph,
		}
	}

	resp, err := h.clients.Simulation().RunWhatIf(ctx, &simulationv1.RunWhatIfRequest{
		BaselineGraph: msg.BaselineGraph,
		Modifications: modifications,
		Algorithm:     msg.Algorithm,
		Options:       opts,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.WhatIfResponse{
		Success:       resp.Success,
		Baseline:      h.convertScenarioResult(resp.Baseline),
		Modified:      h.convertScenarioResult(resp.Modified),
		Comparison:    h.convertComparison(resp.Comparison),
		ModifiedGraph: resp.ModifiedGraph,
		Metadata:      h.convertMetadata(resp.Metadata),
		ErrorMessage:  "",
	}), nil
}

func (h *SimulationHandler) RunMonteCarlo(
	ctx context.Context,
	req *connect.Request[gatewayv1.MonteCarloRequest],
) (*connect.Response[gatewayv1.MonteCarloResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Simulation().RunMonteCarlo(ctx, &simulationv1.RunMonteCarloRequest{
		Graph:         msg.Graph,
		Config:        h.convertMonteCarloConfig(msg.Config),
		Uncertainties: h.convertUncertainties(msg.Uncertainties),
		Algorithm:     msg.Algorithm,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.MonteCarloResponse{
		Success:      resp.Success,
		FlowStats:    h.convertMonteCarloStats(resp.FlowStats),
		CostStats:    h.convertMonteCarloStats(resp.CostStats),
		RiskAnalysis: h.convertRiskAnalysis(resp.RiskAnalysis),
		Metadata:     h.convertMetadata(resp.Metadata),
		ErrorMessage: "",
	}), nil
}

func (h *SimulationHandler) RunMonteCarloStream(
	ctx context.Context,
	req *connect.Request[gatewayv1.MonteCarloRequest],
	stream *connect.ServerStream[gatewayv1.MonteCarloProgressEvent],
) error {
	msg := req.Msg

	progressCh, errCh := h.clients.Simulation().RunMonteCarloStream(ctx, &simulationv1.RunMonteCarloRequest{
		Graph:         msg.Graph,
		Config:        h.convertMonteCarloConfig(msg.Config),
		Uncertainties: h.convertUncertainties(msg.Uncertainties),
		Algorithm:     msg.Algorithm,
	})

	for {
		select {
		case progress, ok := <-progressCh:
			if !ok {
				return nil
			}

			event := &gatewayv1.MonteCarloProgressEvent{
				Iteration:       progress.Iteration,
				TotalIterations: progress.TotalIterations,
				ProgressPercent: progress.ProgressPercent,
				CurrentMeanFlow: progress.CurrentMeanFlow,
				CurrentStdDev:   progress.CurrentStdDev,
				Status:          progress.Status,
				IsFinal:         progress.Status == "completed",
			}

			if err := stream.Send(event); err != nil {
				return err
			}

		case err := <-errCh:
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (h *SimulationHandler) AnalyzeSensitivity(
	ctx context.Context,
	req *connect.Request[gatewayv1.SensitivityRequest],
) (*connect.Response[gatewayv1.SensitivityResponse], error) {
	msg := req.Msg

	params := make([]*simulationv1.SensitivityParameter, 0, len(msg.Parameters))
	for _, p := range msg.Parameters {
		params = append(params, &simulationv1.SensitivityParameter{
			Edge:          p.Edge,
			NodeId:        p.NodeId,
			Target:        simulationv1.ModificationTarget(p.Target),
			MinMultiplier: p.MinMultiplier,
			MaxMultiplier: p.MaxMultiplier,
			NumSteps:      p.NumSteps,
		})
	}

	resp, err := h.clients.Simulation().AnalyzeSensitivity(ctx, &simulationv1.AnalyzeSensitivityRequest{
		Graph:      msg.Graph,
		Parameters: params,
		Algorithm:  msg.Algorithm,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	results := make([]*gatewayv1.SensitivityResult, 0, len(resp.ParameterResults))
	for _, r := range resp.ParameterResults {
		curve := make([]*gatewayv1.SensitivityPoint, 0, len(r.Curve))
		for _, p := range r.Curve {
			curve = append(curve, &gatewayv1.SensitivityPoint{
				ParameterValue: p.ParameterValue,
				FlowValue:      p.FlowValue,
				CostValue:      p.CostValue,
			})
		}
		results = append(results, &gatewayv1.SensitivityResult{
			ParameterId:      r.ParameterId,
			Curve:            curve,
			Elasticity:       r.Elasticity,
			SensitivityIndex: r.SensitivityIndex,
		})
	}

	rankings := make([]*gatewayv1.ParameterRanking, 0, len(resp.Rankings))
	for _, r := range resp.Rankings {
		rankings = append(rankings, &gatewayv1.ParameterRanking{
			ParameterId:      r.ParameterId,
			Rank:             r.Rank,
			SensitivityIndex: r.SensitivityIndex,
			Description:      r.Description,
		})
	}

	return connect.NewResponse(&gatewayv1.SensitivityResponse{
		Success:  resp.Success,
		Results:  results,
		Rankings: rankings,
		Metadata: h.convertMetadata(resp.Metadata),
	}), nil
}

func (h *SimulationHandler) AnalyzeResilience(
	ctx context.Context,
	req *connect.Request[gatewayv1.ResilienceRequest],
) (*connect.Response[gatewayv1.ResilienceResponse], error) {
	msg := req.Msg

	var cfg *simulationv1.ResilienceConfig
	if msg.Config != nil {
		cfg = &simulationv1.ResilienceConfig{
			MaxFailuresToTest:     msg.Config.MaxFailuresToTest,
			TestCascadingFailures: msg.Config.TestCascadingFailures,
			LoadFactor:            msg.Config.LoadFactor,
		}
	}

	resp, err := h.clients.Simulation().AnalyzeResilience(ctx, &simulationv1.AnalyzeResilienceRequest{
		Graph:     msg.Graph,
		Config:    cfg,
		Algorithm: msg.Algorithm,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	weaknesses := make([]*gatewayv1.ResilienceWeakness, 0, len(resp.Weaknesses))
	for _, w := range resp.Weaknesses {
		weaknesses = append(weaknesses, &gatewayv1.ResilienceWeakness{
			Description:          w.Description,
			Type:                 string(w.Type),
			Severity:             w.Severity,
			AffectedEdges:        w.AffectedEdges,
			MitigationSuggestion: w.MitigationSuggestion,
		})
	}

	return connect.NewResponse(&gatewayv1.ResilienceResponse{
		Success: resp.Success,
		Metrics: &gatewayv1.ResilienceMetrics{
			OverallScore:           resp.Metrics.OverallScore,
			ConnectivityRobustness: resp.Metrics.ConnectivityRobustness,
			FlowRobustness:         resp.Metrics.FlowRobustness,
			RedundancyLevel:        resp.Metrics.RedundancyLevel,
			MinCutSize:             resp.Metrics.MinCutSize,
		},
		Weaknesses: weaknesses,
		Metadata:   h.convertMetadata(resp.Metadata),
	}), nil
}

func (h *SimulationHandler) SimulateFailures(
	ctx context.Context,
	req *connect.Request[gatewayv1.FailureSimulationRequest],
) (*connect.Response[gatewayv1.FailureSimulationResponse], error) {
	msg := req.Msg

	scenarios := make([]*simulationv1.FailureScenario, 0, len(msg.Scenarios))
	for _, s := range msg.Scenarios {
		scenarios = append(scenarios, &simulationv1.FailureScenario{
			Name:        s.Name,
			FailedEdges: s.FailedEdges,
			FailedNodes: s.FailedNodes,
			Probability: s.Probability,
		})
	}

	resp, err := h.clients.Simulation().SimulateFailures(ctx, &simulationv1.SimulateFailuresRequest{
		Graph:            msg.Graph,
		FailureScenarios: scenarios,
		Algorithm:        msg.Algorithm,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	results := make([]*gatewayv1.FailureScenarioResult, 0, len(resp.ScenarioResults))
	for _, r := range resp.ScenarioResults {
		results = append(results, &gatewayv1.FailureScenarioResult{
			ScenarioName:        r.ScenarioName,
			Probability:         r.Probability,
			Result:              h.convertScenarioResult(r.Result),
			VsBaseline:          h.convertComparison(r.VsBaseline),
			NetworkDisconnected: r.NetworkDisconnected,
		})
	}

	return connect.NewResponse(&gatewayv1.FailureSimulationResponse{
		Success:         resp.Success,
		Baseline:        h.convertScenarioResult(resp.Baseline),
		ScenarioResults: results,
		Stats: &gatewayv1.FailureStats{
			ExpectedFlowLoss:           resp.Stats.ExpectedFlowLoss,
			MaxFlowLoss:                resp.Stats.MaxFlowLoss,
			ProbabilityOfDisconnection: resp.Stats.ProbabilityOfDisconnection,
		},
		Metadata: h.convertMetadata(resp.Metadata),
	}), nil
}

func (h *SimulationHandler) FindCriticalElements(
	ctx context.Context,
	req *connect.Request[gatewayv1.CriticalElementsRequest],
) (*connect.Response[gatewayv1.CriticalElementsResponse], error) {
	msg := req.Msg

	var cfg *simulationv1.CriticalElementsConfig
	if msg.Config != nil {
		cfg = &simulationv1.CriticalElementsConfig{
			AnalyzeEdges:     msg.Config.AnalyzeEdges,
			AnalyzeNodes:     msg.Config.AnalyzeNodes,
			TopN:             msg.Config.TopN,
			FailureThreshold: msg.Config.FailureThreshold,
		}
	}

	resp, err := h.clients.Simulation().FindCriticalElements(ctx, &simulationv1.FindCriticalElementsRequest{
		Graph:     msg.Graph,
		Config:    cfg,
		Algorithm: msg.Algorithm,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	edges := make([]*gatewayv1.CriticalEdge, 0, len(resp.CriticalEdges))
	for _, e := range resp.CriticalEdges {
		edges = append(edges, &gatewayv1.CriticalEdge{
			Edge:                   e.Edge,
			CriticalityScore:       e.CriticalityScore,
			FlowImpactIfRemoved:    e.FlowImpactIfRemoved,
			Rank:                   e.Rank,
			IsSinglePointOfFailure: e.IsSinglePointOfFailure,
		})
	}

	nodes := make([]*gatewayv1.CriticalNode, 0, len(resp.CriticalNodes))
	for _, n := range resp.CriticalNodes {
		nodes = append(nodes, &gatewayv1.CriticalNode{
			NodeId:              n.NodeId,
			CriticalityScore:    n.CriticalityScore,
			FlowImpactIfRemoved: n.FlowImpactIfRemoved,
			AffectedEdges:       n.AffectedEdges,
			Rank:                n.Rank,
		})
	}

	return connect.NewResponse(&gatewayv1.CriticalElementsResponse{
		Success:               resp.Success,
		CriticalEdges:         edges,
		CriticalNodes:         nodes,
		SinglePointsOfFailure: resp.SinglePointsOfFailure,
		ResilienceScore:       resp.ResilienceScore,
		Metadata:              h.convertMetadata(resp.Metadata),
	}), nil
}

func (h *SimulationHandler) GetSimulation(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetSimulationRequest],
) (*connect.Response[gatewayv1.SimulationRecord], error) {
	userID := middleware.GetUserID(ctx)

	resp, err := h.clients.Simulation().Raw().GetSimulation(ctx, &simulationv1.GetSimulationRequest{
		SimulationId: req.Msg.SimulationId,
		UserId:       userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.SimulationRecord{
		Id:         resp.Record.Id,
		Name:       resp.Record.Name,
		Type:       string(resp.Record.Type),
		CreatedAt:  resp.Record.CreatedAt,
		Tags:       resp.Record.Tags,
		ResultData: resp.Record.ResponseData,
	}), nil
}

func (h *SimulationHandler) ListSimulations(
	ctx context.Context,
	req *connect.Request[gatewayv1.ListSimulationsRequest],
) (*connect.Response[gatewayv1.ListSimulationsResponse], error) {
	userID := middleware.GetUserID(ctx)
	msg := req.Msg

	var simType simulationv1.SimulationType
	switch msg.SimulationType {
	case "whatif":
		simType = simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF
	case "montecarlo":
		simType = simulationv1.SimulationType_SIMULATION_TYPE_MONTE_CARLO
	case "sensitivity":
		simType = simulationv1.SimulationType_SIMULATION_TYPE_SENSITIVITY
	}

	page := int32(1)
	if msg.Limit > 0 {
		page = msg.Offset/msg.Limit + 1
	}

	resp, err := h.clients.Simulation().ListSimulations(ctx, &simulationv1.ListSimulationsRequest{
		UserId: userID,
		Type:   simType,
		Pagination: &commonv1.PaginationRequest{
			Page:     page,
			PageSize: msg.Limit,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	simulations := make([]*gatewayv1.SimulationRecord, 0, len(resp.Simulations))
	for _, s := range resp.Simulations {
		simulations = append(simulations, &gatewayv1.SimulationRecord{
			Id:        s.Id,
			Name:      s.Name,
			Type:      string(s.Type),
			CreatedAt: s.CreatedAt,
			Tags:      s.Tags,
		})
	}

	return connect.NewResponse(&gatewayv1.ListSimulationsResponse{
		Simulations: simulations,
		TotalCount:  resp.Pagination.TotalItems,
		HasMore:     resp.Pagination.HasNext,
	}), nil
}

func (h *SimulationHandler) DeleteSimulation(
	ctx context.Context,
	req *connect.Request[gatewayv1.DeleteSimulationRequest],
) (*connect.Response[emptypb.Empty], error) {
	// Прокси к simulation-svc
	// TODO: добавить проверку прав
	_ = req // suppress unused warning
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// Conversion helpers
func (h *SimulationHandler) convertScenarioResult(r *simulationv1.ScenarioResult) *gatewayv1.ScenarioResult {
	if r == nil {
		return nil
	}
	return &gatewayv1.ScenarioResult{
		Name:                  r.Name,
		MaxFlow:               r.MaxFlow,
		TotalCost:             r.TotalCost,
		Efficiency:            r.AverageUtilization,
		ImprovementVsBaseline: 0,
	}
}

func (h *SimulationHandler) convertComparison(c *simulationv1.ScenarioComparison) *gatewayv1.ScenarioComparison {
	if c == nil {
		return nil
	}
	return &gatewayv1.ScenarioComparison{
		FlowChange:        c.FlowChange,
		FlowChangePercent: c.FlowChangePercent,
		CostChange:        c.CostChange,
		CostChangePercent: c.CostChangePercent,
		ImpactSummary:     c.ImpactSummary,
		ImpactLevel:       gatewayv1.ImpactLevel(c.ImpactLevel),
	}
}

func (h *SimulationHandler) convertMetadata(m *simulationv1.SimulationMetadata) *gatewayv1.SimulationMetadata {
	if m == nil {
		return nil
	}
	return &gatewayv1.SimulationMetadata{
		SimulationId:      m.SimulationId,
		ComputationTimeMs: m.ComputationTimeMs,
		Iterations:        m.Iterations,
		MemoryUsedBytes:   m.MemoryUsedBytes,
		AlgorithmUsed:     m.AlgorithmUsed,
		CompletedAt:       m.CompletedAt,
	}
}

func (h *SimulationHandler) convertMonteCarloConfig(c *gatewayv1.MonteCarloConfig) *simulationv1.MonteCarloConfig {
	if c == nil {
		return nil
	}
	return &simulationv1.MonteCarloConfig{
		NumIterations:   c.NumIterations,
		RandomSeed:      c.RandomSeed,
		ConfidenceLevel: c.ConfidenceLevel,
		Parallel:        c.Parallel,
	}
}

func (h *SimulationHandler) convertUncertainties(specs []*gatewayv1.UncertaintySpec) []*simulationv1.UncertaintySpec {
	result := make([]*simulationv1.UncertaintySpec, 0, len(specs))
	for _, s := range specs {
		result = append(result, &simulationv1.UncertaintySpec{
			Edge:   s.Edge,
			NodeId: s.NodeId,
			Target: simulationv1.ModificationTarget(s.Target),
			Distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType(s.Distribution.Type),
				Param1: s.Distribution.Param1,
				Param2: s.Distribution.Param2,
				Param3: s.Distribution.Param3,
			},
		})
	}
	return result
}

func (h *SimulationHandler) convertMonteCarloStats(s *simulationv1.MonteCarloStats) *gatewayv1.MonteCarloStats {
	if s == nil {
		return nil
	}
	return &gatewayv1.MonteCarloStats{
		Mean:                   s.Mean,
		StdDev:                 s.StdDev,
		Min:                    s.Min,
		Max:                    s.Max,
		Median:                 s.Median,
		ConfidenceIntervalLow:  s.ConfidenceIntervalLow,
		ConfidenceIntervalHigh: s.ConfidenceIntervalHigh,
	}
}

func (h *SimulationHandler) convertRiskAnalysis(r *simulationv1.RiskAnalysis) *gatewayv1.RiskAnalysis {
	if r == nil {
		return nil
	}
	return &gatewayv1.RiskAnalysis{
		ProbabilityBelowThreshold: r.ProbabilityBelowThreshold,
		ValueAtRisk:               r.ValueAtRisk,
		WorstCaseFlow:             r.WorstCaseFlow,
		BestCaseFlow:              r.BestCaseFlow,
	}
}
