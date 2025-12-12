package handlers

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	historyv1 "logistics/gen/go/logistics/history/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/services/gateway-svc/internal/clients"
	"logistics/services/gateway-svc/internal/middleware"
)

// Константы
const (
	anonymousUserID = "anonymous"
)

// HistoryHandler обработчики истории
type HistoryHandler struct {
	clients *clients.Manager
}

// NewHistoryHandler создаёт handler
func NewHistoryHandler(clients *clients.Manager) *HistoryHandler {
	return &HistoryHandler{clients: clients}
}

func (h *HistoryHandler) SaveCalculation(
	ctx context.Context,
	req *connect.Request[gatewayv1.SaveCalculationRequest],
) (*connect.Response[gatewayv1.SaveCalculationResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = anonymousUserID
	}
	msg := req.Msg

	// ========================================================================
	// 1. Конвертация Request (Gateway -> Optimization)
	// ========================================================================

	var optOptions *optimizationv1.SolveOptions

	solveReq := &optimizationv1.SolveRequest{
		Graph:   msg.Graph,
		Options: optOptions,
	}

	// ========================================================================
	// 2. Конвертация Response (Gateway -> Optimization)
	// ========================================================================
	var solveResp *optimizationv1.SolveResponse

	if msg.Result != nil {
		// 2.1 Конвертация Метрик (Gateway Metrics -> Optimization Metrics)
		var optMetrics *optimizationv1.SolveMetrics
		if msg.Result.Metrics != nil {
			optMetrics = &optimizationv1.SolveMetrics{
				ComputationTimeMs:    msg.Result.Metrics.ComputationTimeMs,
				Iterations:           msg.Result.Metrics.Iterations,
				AugmentingPathsFound: msg.Result.Metrics.AugmentingPathsFound,
				MemoryUsedBytes:      msg.Result.Metrics.MemoryUsedBytes,
			}
		}

		// 2.2 Сборка основного ответа
		solveResp = &optimizationv1.SolveResponse{
			Success:      msg.Result.Success,
			Result:       msg.Result.Result,
			SolvedGraph:  msg.Result.SolvedGraph,
			ErrorMessage: msg.Result.ErrorMessage,
			Metrics:      optMetrics,
		}
	}

	// ========================================================================
	// 3. Отправка в сервис истории
	// ========================================================================

	resp, err := h.clients.History().Raw().SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
		UserId:   userID,
		Name:     msg.Name,
		Tags:     msg.Tags,
		Request:  solveReq,
		Response: solveResp,
	})

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.SaveCalculationResponse{
		CalculationId: resp.CalculationId,
		CreatedAt:     resp.CreatedAt,
	}), nil
}

func (h *HistoryHandler) GetCalculation(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetCalculationRequest],
) (*connect.Response[gatewayv1.CalculationRecord], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = anonymousUserID
	}

	resp, err := h.clients.History().GetCalculation(ctx, userID, req.Msg.CalculationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	record := resp.Record
	return connect.NewResponse(&gatewayv1.CalculationRecord{
		CalculationId: record.CalculationId,
		Name:          record.Name,
		CreatedAt:     record.CreatedAt,
		Graph:         record.Request.Graph,
		Tags:          record.Tags,
	}), nil
}

func (h *HistoryHandler) ListCalculations(
	ctx context.Context,
	req *connect.Request[gatewayv1.ListCalculationsRequest],
) (*connect.Response[gatewayv1.ListCalculationsResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = anonymousUserID
	}
	msg := req.Msg

	var filter *historyv1.HistoryFilter
	if msg.Algorithm != commonv1.Algorithm_ALGORITHM_UNSPECIFIED || len(msg.Tags) > 0 {
		filter = &historyv1.HistoryFilter{
			Algorithm: msg.Algorithm,
			Tags:      msg.Tags,
		}
		if msg.CreatedAfter != nil {
			filter.TimeRange = &commonv1.TimeRange{
				StartTimestamp: msg.CreatedAfter.AsTime().Unix(),
			}
		}
		if msg.CreatedBefore != nil {
			if filter.TimeRange == nil {
				filter.TimeRange = &commonv1.TimeRange{}
			}
			filter.TimeRange.EndTimestamp = msg.CreatedBefore.AsTime().Unix()
		}
	}

	var sort historyv1.HistorySortOrder
	switch msg.SortBy {
	case "created_at":
		if msg.SortDesc {
			sort = historyv1.HistorySortOrder_HISTORY_SORT_ORDER_CREATED_DESC
		} else {
			sort = historyv1.HistorySortOrder_HISTORY_SORT_ORDER_CREATED_ASC
		}
	case "max_flow":
		sort = historyv1.HistorySortOrder_HISTORY_SORT_ORDER_MAX_FLOW_DESC
	}

	// Вычисляем page без лишней конверсии
	page := int32(1)
	if msg.Limit > 0 {
		page = msg.Offset/msg.Limit + 1
	}

	resp, err := h.clients.History().ListCalculations(ctx, &historyv1.ListCalculationsRequest{
		UserId: userID,
		Pagination: &commonv1.PaginationRequest{
			Page:     page,
			PageSize: msg.Limit,
		},
		Filter: filter,
		Sort:   sort,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	calculations := make([]*gatewayv1.CalculationSummary, 0, len(resp.Calculations))
	for _, c := range resp.Calculations {
		tags := make([]string, 0, len(c.Tags))
		tags = append(tags, c.Tags...)
		calculations = append(calculations, &gatewayv1.CalculationSummary{
			CalculationId:     c.CalculationId,
			Name:              c.Name,
			CreatedAt:         c.CreatedAt,
			MaxFlow:           c.MaxFlow,
			TotalCost:         c.TotalCost,
			Algorithm:         c.Algorithm,
			ComputationTimeMs: c.ComputationTimeMs,
			NodeCount:         c.NodeCount,
			EdgeCount:         c.EdgeCount,
			Tags:              tags,
		})
	}

	return connect.NewResponse(&gatewayv1.ListCalculationsResponse{
		Calculations: calculations,
		TotalCount:   resp.Pagination.TotalItems,
		HasMore:      resp.Pagination.HasNext,
	}), nil
}

func (h *HistoryHandler) DeleteCalculation(
	ctx context.Context,
	req *connect.Request[gatewayv1.DeleteCalculationRequest],
) (*connect.Response[emptypb.Empty], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = anonymousUserID
	}

	_, err := h.clients.History().DeleteCalculation(ctx, userID, req.Msg.CalculationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (h *HistoryHandler) GetStatistics(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetStatisticsRequest],
) (*connect.Response[gatewayv1.StatisticsResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = anonymousUserID
	}
	msg := req.Msg

	var timeRange *commonv1.TimeRange
	if msg.StartTime != nil || msg.EndTime != nil {
		timeRange = &commonv1.TimeRange{}
		if msg.StartTime != nil {
			timeRange.StartTimestamp = msg.StartTime.AsTime().Unix()
		}
		if msg.EndTime != nil {
			timeRange.EndTimestamp = msg.EndTime.AsTime().Unix()
		}
	}

	resp, err := h.clients.History().GetStatistics(ctx, &historyv1.GetStatisticsRequest{
		UserId:    userID,
		TimeRange: timeRange,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	dailyStats := make([]*gatewayv1.DailyStats, 0, len(resp.DailyStats))
	for _, d := range resp.DailyStats {
		dailyStats = append(dailyStats, &gatewayv1.DailyStats{
			Date:      d.Date,
			Count:     d.Count,
			TotalFlow: d.TotalFlow,
		})
	}

	return connect.NewResponse(&gatewayv1.StatisticsResponse{
		TotalCalculations:        resp.TotalCalculations,
		AverageMaxFlow:           resp.AverageMaxFlow,
		AverageCost:              resp.AverageCost,
		AverageComputationTimeMs: resp.AverageComputationTimeMs,
		CalculationsByAlgorithm:  resp.CalculationsByAlgorithm,
		DailyStats:               dailyStats,
	}), nil
}
