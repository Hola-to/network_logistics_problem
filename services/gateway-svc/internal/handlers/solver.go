package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/pkg/config"
	"logistics/pkg/logger"
	"logistics/services/gateway-svc/internal/clients"
	"logistics/services/gateway-svc/internal/middleware"
)

// SolverHandler обработчики оптимизации
type SolverHandler struct {
	clients *clients.Manager
	config  *config.Config
}

// NewSolverHandler создаёт handler
func NewSolverHandler(clients *clients.Manager, cfg *config.Config) *SolverHandler {
	return &SolverHandler{
		clients: clients,
		config:  cfg,
	}
}

func (h *SolverHandler) GetAlgorithms(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.AlgorithmsResponse], error) {
	algorithms, err := h.clients.Solver().GetAlgorithms(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := make([]*gatewayv1.AlgorithmInfo, 0, len(algorithms))
	for _, alg := range algorithms {
		result = append(result, &gatewayv1.AlgorithmInfo{
			Algorithm:             alg.Algorithm,
			Name:                  alg.Name,
			Description:           alg.Description,
			TimeComplexity:        alg.TimeComplexity,
			SpaceComplexity:       alg.SpaceComplexity,
			SupportsMinCost:       alg.SupportsMinCost,
			SupportsNegativeCosts: alg.SupportsNegativeCosts,
			BestFor:               alg.BestFor,
		})
	}

	return connect.NewResponse(&gatewayv1.AlgorithmsResponse{
		Algorithms: result,
	}), nil
}

func (h *SolverHandler) SolveGraph(
	ctx context.Context,
	req *connect.Request[gatewayv1.SolveGraphRequest],
) (*connect.Response[gatewayv1.SolveGraphResponse], error) {
	msg := req.Msg

	algorithm := msg.Algorithm
	if algorithm == commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
		algorithm = commonv1.Algorithm_ALGORITHM_DINIC
	}

	var opts *optimizationv1.SolveOptions
	if msg.Options != nil {
		opts = &optimizationv1.SolveOptions{
			TimeoutSeconds: msg.Options.TimeoutSeconds,
			ReturnPaths:    msg.Options.ReturnPaths,
			MaxIterations:  msg.Options.MaxIterations,
			Epsilon:        msg.Options.Epsilon,
		}
	}

	result, err := h.clients.Solver().Solve(ctx, msg.Graph, algorithm, opts)
	if err != nil {
		logger.Log.Error("Solve failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.SolveGraphResponse{
		Success:      result.Success,
		Result:       result.Result,
		SolvedGraph:  result.SolvedGraph,
		Metrics:      h.convertMetrics(result.Metrics),
		ErrorMessage: result.ErrorMessage,
	}), nil
}

func (h *SolverHandler) SolveGraphStream(
	ctx context.Context,
	req *connect.Request[gatewayv1.SolveGraphRequest],
	stream *connect.ServerStream[gatewayv1.SolveProgressEvent],
) error {
	msg := req.Msg

	algorithm := msg.Algorithm
	if algorithm == commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
		algorithm = commonv1.Algorithm_ALGORITHM_DINIC
	}

	var opts *optimizationv1.SolveOptions
	if msg.Options != nil {
		opts = &optimizationv1.SolveOptions{
			TimeoutSeconds: msg.Options.TimeoutSeconds,
			ReturnPaths:    msg.Options.ReturnPaths,
			MaxIterations:  msg.Options.MaxIterations,
			Epsilon:        msg.Options.Epsilon,
		}
	}

	progressCh, errCh := h.clients.Solver().SolveStream(ctx, msg.Graph, algorithm, opts)

	for {
		select {
		case progress, ok := <-progressCh:
			if !ok {
				return nil
			}

			event := &gatewayv1.SolveProgressEvent{
				Iteration:         progress.Iteration,
				CurrentFlow:       progress.CurrentFlow,
				ProgressPercent:   progress.ProgressPercent,
				Status:            progress.Status,
				LastPath:          progress.LastPath,
				ComputationTimeMs: progress.ComputationTimeMs,
				MemoryUsedBytes:   progress.MemoryUsedBytes,
				IsFinal:           progress.Status == "completed",
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

func (h *SolverHandler) BatchSolve(
	ctx context.Context,
	req *connect.Request[gatewayv1.BatchSolveRequest],
) (*connect.Response[gatewayv1.BatchSolveResponse], error) {
	msg := req.Msg
	startTime := time.Now()

	defaultAlg := msg.DefaultAlgorithm
	if defaultAlg == commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
		defaultAlg = commonv1.Algorithm_ALGORITHM_DINIC
	}

	results := make([]*gatewayv1.BatchSolveResult, len(msg.Items))
	var successful, failed int32

	// TODO: параллельное выполнение если msg.Parallel
	for i, item := range msg.Items {
		algorithm := item.Algorithm
		if algorithm == commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
			algorithm = defaultAlg
		}

		itemStart := time.Now()
		solveResult, err := h.clients.Solver().Solve(ctx, item.Graph, algorithm, nil)

		result := &gatewayv1.BatchSolveResult{
			Id:                item.Id,
			ComputationTimeMs: float64(time.Since(itemStart).Milliseconds()),
		}

		if err != nil {
			result.Success = false
			result.ErrorMessage = err.Error()
			failed++
		} else if !solveResult.Success {
			result.Success = false
			result.ErrorMessage = solveResult.ErrorMessage
			failed++
		} else {
			result.Success = true
			result.MaxFlow = solveResult.Result.MaxFlow
			result.TotalCost = solveResult.Result.TotalCost
			result.ComputationTimeMs = solveResult.Metrics.ComputationTimeMs
			successful++
		}

		results[i] = result
	}

	return connect.NewResponse(&gatewayv1.BatchSolveResponse{
		Results:     results,
		TotalTimeMs: float64(time.Since(startTime).Milliseconds()),
		Successful:  successful,
		Failed:      failed,
	}), nil
}

func (h *SolverHandler) CalculateLogistics(
	ctx context.Context,
	req *connect.Request[gatewayv1.CalculateLogisticsRequest],
) (*connect.Response[gatewayv1.CalculateLogisticsResponse], error) {
	msg := req.Msg
	startTime := time.Now()
	requestID := generateRequestID()

	resp := &gatewayv1.CalculateLogisticsResponse{
		Metadata: &gatewayv1.RequestMetadata{
			RequestId:      requestID,
			ProcessedAt:    timestamppb.Now(),
			AlgorithmUsed:  msg.Algorithm.String(),
			ServiceVersion: h.config.App.Version,
		},
	}

	var validationTime, solveTime, analyticsTime time.Duration

	// 1. Валидация
	if !msg.SkipValidation {
		valStart := time.Now()

		level := validationv1.ValidationLevel(msg.ValidationLevel)
		valResp, err := h.clients.Validation().ValidateGraph(ctx, msg.Graph, level)
		validationTime = time.Since(valStart)

		if err != nil {
			resp.Errors = append(resp.Errors, &commonv1.ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			})
			h.setMetadata(resp.Metadata, validationTime, 0, 0, startTime)
			return connect.NewResponse(resp), nil
		}

		resp.Validation = &gatewayv1.ValidationResult{
			IsValid:       valResp.Result.IsValid,
			ErrorsCount:   int32(len(valResp.Result.Errors)),
			WarningsCount: int32(len(valResp.Warnings)),
			Errors:        valResp.Result.Errors,
			Warnings:      valResp.Warnings,
			GraphStats:    valResp.Statistics,
		}

		if !valResp.Result.IsValid {
			h.setMetadata(resp.Metadata, validationTime, 0, 0, startTime)
			return connect.NewResponse(resp), nil
		}
	}

	// 2. Решение
	solveStart := time.Now()

	algorithm := msg.Algorithm
	if algorithm == commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
		algorithm = commonv1.Algorithm_ALGORITHM_DINIC
	}

	var solveOpts *optimizationv1.SolveOptions
	if msg.SolveOptions != nil {
		solveOpts = &optimizationv1.SolveOptions{
			TimeoutSeconds: msg.SolveOptions.TimeoutSeconds,
			ReturnPaths:    msg.SolveOptions.ReturnPaths,
			MaxIterations:  msg.SolveOptions.MaxIterations,
			Epsilon:        msg.SolveOptions.Epsilon,
		}
	}

	solveResult, err := h.clients.Solver().Solve(ctx, msg.Graph, algorithm, solveOpts)
	solveTime = time.Since(solveStart)

	if err != nil {
		resp.Errors = append(resp.Errors, &commonv1.ErrorDetail{
			Code:    "SOLVE_ERROR",
			Message: err.Error(),
		})
		h.setMetadata(resp.Metadata, validationTime, solveTime, 0, startTime)
		return connect.NewResponse(resp), nil
	}

	if !solveResult.Success {
		resp.Errors = append(resp.Errors, &commonv1.ErrorDetail{
			Code:    "SOLVE_FAILED",
			Message: solveResult.ErrorMessage,
		})
		h.setMetadata(resp.Metadata, validationTime, solveTime, 0, startTime)
		return connect.NewResponse(resp), nil
	}

	resp.Optimization = &gatewayv1.SolveResult{
		SolvedGraph:       solveResult.SolvedGraph,
		MaxFlow:           solveResult.Result.MaxFlow,
		TotalCost:         solveResult.Result.TotalCost,
		Status:            solveResult.Result.Status,
		Iterations:        solveResult.Result.Iterations,
		ComputationTimeMs: solveResult.Metrics.ComputationTimeMs,
		PathsFound:        int32(len(solveResult.Result.Paths)),
		Paths:             solveResult.Result.Paths,
	}

	// 3. Аналитика
	if msg.CalculateCost || msg.FindBottlenecks || msg.CalculateStatistics {
		analyticsStart := time.Now()

		threshold := msg.BottleneckThreshold
		if threshold == 0 {
			threshold = 0.9
		}

		analyzeResp, err := h.clients.Analytics().AnalyzeFlow(ctx, solveResult.SolvedGraph, &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        msg.CalculateCost,
			FindBottlenecks:     msg.FindBottlenecks,
			CalculateStatistics: msg.CalculateStatistics,
			SuggestImprovements: msg.SuggestImprovements,
			BottleneckThreshold: threshold,
		})
		analyticsTime = time.Since(analyticsStart)

		if err != nil {
			resp.Warnings = append(resp.Warnings, "Analytics failed: "+err.Error())
		} else {
			resp.Analytics = h.convertAnalytics(analyzeResp)
		}
	}

	// 4. Сохранение в историю
	if msg.SaveToHistory {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			userID = "anonymous"
		}

		saveResp, err := h.clients.History().SaveCalculation(
			ctx, userID, msg.CalculationName,
			&optimizationv1.SolveRequest{
				Graph:     msg.Graph,
				Algorithm: algorithm,
				Options:   solveOpts,
			},
			&optimizationv1.SolveResponse{
				Success:     true,
				Result:      solveResult.Result,
				SolvedGraph: solveResult.SolvedGraph,
				Metrics:     solveResult.Metrics,
			},
			msg.Tags,
		)

		if err != nil {
			resp.Warnings = append(resp.Warnings, "Failed to save: "+err.Error())
		} else {
			resp.CalculationId = saveResp.CalculationId
		}
	}

	resp.Success = true
	h.setMetadata(resp.Metadata, validationTime, solveTime, analyticsTime, startTime)

	logger.Log.Info("CalculateLogistics completed",
		"request_id", requestID,
		"max_flow", resp.Optimization.MaxFlow,
		"total_time_ms", resp.Metadata.TotalTimeMs,
	)

	return connect.NewResponse(resp), nil
}

func (h *SolverHandler) setMetadata(meta *gatewayv1.RequestMetadata, validation, solve, analytics time.Duration, start time.Time) {
	meta.ValidationTimeMs = float64(validation.Milliseconds())
	meta.SolveTimeMs = float64(solve.Milliseconds())
	meta.AnalyticsTimeMs = float64(analytics.Milliseconds())
	meta.TotalTimeMs = float64(time.Since(start).Milliseconds())
}

func (h *SolverHandler) convertMetrics(m *optimizationv1.SolveMetrics) *gatewayv1.SolveMetrics {
	if m == nil {
		return nil
	}
	return &gatewayv1.SolveMetrics{
		ComputationTimeMs:    m.ComputationTimeMs,
		Iterations:           m.Iterations,
		AugmentingPathsFound: m.AugmentingPathsFound,
		MemoryUsedBytes:      m.MemoryUsedBytes,
	}
}

func (h *SolverHandler) convertAnalytics(resp *analyticsv1.AnalyzeFlowResponse) *gatewayv1.AnalyticsResult {
	result := &gatewayv1.AnalyticsResult{
		FlowStats: resp.FlowStats,
	}

	if resp.Cost != nil {
		result.TotalCost = resp.Cost.TotalCost
		result.Currency = resp.Cost.Currency
		if resp.Cost.Breakdown != nil {
			result.CostBreakdown = &gatewayv1.CostBreakdown{
				TransportCost:  resp.Cost.Breakdown.TransportCost,
				FixedCost:      resp.Cost.Breakdown.FixedCost,
				HandlingCost:   resp.Cost.Breakdown.HandlingCost,
				DiscountAmount: resp.Cost.Breakdown.DiscountAmount,
				MarkupAmount:   resp.Cost.Breakdown.MarkupAmount,
				CostByRoadType: resp.Cost.Breakdown.CostByRoadType,
				CostByNodeType: resp.Cost.Breakdown.CostByNodeType,
			}
		}
	}

	if resp.Bottlenecks != nil {
		result.BottlenecksCount = int32(len(resp.Bottlenecks.Bottlenecks))
		for _, b := range resp.Bottlenecks.Bottlenecks {
			result.Bottlenecks = append(result.Bottlenecks, &gatewayv1.Bottleneck{
				Edge: &commonv1.EdgeKey{
					From: b.Edge.From,
					To:   b.Edge.To,
				},
				Utilization: b.Utilization,
				ImpactScore: b.ImpactScore,
				Severity:    gatewayv1.BottleneckSeverity(b.Severity),
			})
		}
		for _, r := range resp.Bottlenecks.Recommendations {
			result.Recommendations = append(result.Recommendations, &gatewayv1.Recommendation{
				Type:                 r.Type,
				Description:          r.Description,
				AffectedEdge:         r.AffectedEdge,
				EstimatedImprovement: r.EstimatedImprovement,
				EstimatedCost:        r.EstimatedCost,
			})
		}
	}

	if resp.Efficiency != nil {
		result.Efficiency = &gatewayv1.EfficiencyReport{
			OverallEfficiency:   resp.Efficiency.OverallEfficiency,
			CapacityUtilization: resp.Efficiency.CapacityUtilization,
			UnusedEdgesCount:    resp.Efficiency.UnusedEdgesCount,
			SaturatedEdgesCount: resp.Efficiency.SaturatedEdgesCount,
			Grade:               resp.Efficiency.Grade,
		}
	}

	return result
}

func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return time.Now().Format("20060102150405") + "-fallback"
	}
	return time.Now().Format("20060102150405") + "-" + hex.EncodeToString(bytes)
}
