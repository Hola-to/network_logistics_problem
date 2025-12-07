package service

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	historyv1 "logistics/gen/go/logistics/history/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/telemetry"
	"logistics/services/history-svc/internal/repository"
)

// HistoryService реализация gRPC сервиса истории
type HistoryService struct {
	historyv1.UnimplementedHistoryServiceServer
	repo repository.CalculationRepository
}

// NewHistoryService создаёт новый сервис
func NewHistoryService(repo repository.CalculationRepository) *HistoryService {
	return &HistoryService{repo: repo}
}

// SaveCalculation сохраняет расчёт
func (s *HistoryService) SaveCalculation(
	ctx context.Context,
	req *historyv1.SaveCalculationRequest,
) (*historyv1.SaveCalculationResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "HistoryService.SaveCalculation")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserId),
		attribute.String("name", req.Name),
	)

	// Валидация
	if req.UserId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.NewWithField(pkgerrors.CodeInvalidArgument, "user_id is required", "user_id"),
		)
	}
	if req.Request == nil || req.Response == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.NewWithField(pkgerrors.CodeInvalidArgument, "request and response are required", "request"),
		)
	}

	// Сериализуем request и response в JSON
	requestData, err := protojson.Marshal(req.Request)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to serialize request"),
		)
	}

	responseData, err := protojson.Marshal(req.Response)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to serialize response"),
		)
	}

	// Извлекаем метрики
	algorithm := ""
	if req.Request.Algorithm != commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
		algorithm = req.Request.Algorithm.String()
	}

	var maxFlow, totalCost, computationTimeMs float64
	nodeCount, edgeCount := 0, 0

	if req.Response.Result != nil {
		maxFlow = req.Response.Result.MaxFlow
		totalCost = req.Response.Result.TotalCost
		computationTimeMs = req.Response.Result.ComputationTimeMs
	}

	if req.Request.Graph != nil {
		nodeCount = len(req.Request.Graph.Nodes)
		edgeCount = len(req.Request.Graph.Edges)
	}

	// Конвертируем теги
	tags := make([]string, 0, len(req.Tags))
	for k, v := range req.Tags {
		tags = append(tags, k+":"+v)
	}

	// Создаём запись
	calc := &repository.Calculation{
		UserID:            req.UserId,
		Name:              req.Name,
		Algorithm:         algorithm,
		MaxFlow:           maxFlow,
		TotalCost:         totalCost,
		ComputationTimeMs: computationTimeMs,
		NodeCount:         nodeCount,
		EdgeCount:         edgeCount,
		RequestData:       requestData,
		ResponseData:      responseData,
		Tags:              tags,
	}

	if err := s.repo.Create(ctx, calc); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to save calculation"),
		)
	}

	telemetry.AddEvent(ctx, "calculation_saved",
		attribute.String("calculation_id", calc.ID),
		attribute.Float64("max_flow", maxFlow),
	)

	return &historyv1.SaveCalculationResponse{
		CalculationId: calc.ID,
		CreatedAt:     timestamppb.New(calc.CreatedAt),
	}, nil
}

// GetCalculation получает расчёт по ID
func (s *HistoryService) GetCalculation(
	ctx context.Context,
	req *historyv1.GetCalculationRequest,
) (*historyv1.GetCalculationResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "HistoryService.GetCalculation")
	defer span.End()

	span.SetAttributes(
		attribute.String("calculation_id", req.CalculationId),
		attribute.String("user_id", req.UserId),
	)

	if req.CalculationId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.NewWithField(pkgerrors.CodeInvalidArgument, "calculation_id is required", "calculation_id"),
		)
	}

	calc, err := s.repo.GetByID(ctx, req.CalculationId)
	if err != nil {
		if errors.Is(err, repository.ErrCalculationNotFound) {
			return nil, pkgerrors.ToGRPC(
				pkgerrors.New(pkgerrors.CodeNotFound, "calculation not found"),
			)
		}
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get calculation"),
		)
	}

	// Проверяем владельца
	if req.UserId != "" && calc.UserID != req.UserId {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodePermissionDenied, "access denied"),
		)
	}

	record, err := s.toCalculationRecord(calc)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to convert calculation"),
		)
	}

	return &historyv1.GetCalculationResponse{
		Record: record,
	}, nil
}

// ListCalculations возвращает список расчётов
func (s *HistoryService) ListCalculations(
	ctx context.Context,
	req *historyv1.ListCalculationsRequest,
) (*historyv1.ListCalculationsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "HistoryService.ListCalculations")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", req.UserId))

	if req.UserId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.NewWithField(pkgerrors.CodeInvalidArgument, "user_id is required", "user_id"),
		)
	}

	opts := s.toListOptions(req)

	calculations, total, err := s.repo.List(ctx, req.UserId, opts)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to list calculations"),
		)
	}

	summaries := make([]*historyv1.CalculationSummary, len(calculations))
	for i, calc := range calculations {
		summaries[i] = s.toCalculationSummary(calc)
	}

	pageSize := int32(opts.Limit)
	currentPage := int32(opts.Offset/opts.Limit) + 1
	totalPages := int32((total + int64(opts.Limit) - 1) / int64(opts.Limit))

	return &historyv1.ListCalculationsResponse{
		Calculations: summaries,
		Pagination: &commonv1.PaginationResponse{
			CurrentPage: currentPage,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			TotalItems:  total,
			HasNext:     int64(opts.Offset+opts.Limit) < total,
			HasPrevious: opts.Offset > 0,
		},
	}, nil
}

// DeleteCalculation удаляет расчёт
func (s *HistoryService) DeleteCalculation(
	ctx context.Context,
	req *historyv1.DeleteCalculationRequest,
) (*historyv1.DeleteCalculationResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "HistoryService.DeleteCalculation")
	defer span.End()

	span.SetAttributes(
		attribute.String("calculation_id", req.CalculationId),
		attribute.String("user_id", req.UserId),
	)

	if req.CalculationId == "" || req.UserId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "calculation_id and user_id are required"),
		)
	}

	calc, err := s.repo.GetByID(ctx, req.CalculationId)
	if err != nil {
		if errors.Is(err, repository.ErrCalculationNotFound) {
			return &historyv1.DeleteCalculationResponse{Success: true}, nil
		}
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get calculation"),
		)
	}

	if calc.UserID != req.UserId {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodePermissionDenied, "access denied"),
		)
	}

	if err := s.repo.Delete(ctx, req.CalculationId); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to delete calculation"),
		)
	}

	telemetry.AddEvent(ctx, "calculation_deleted",
		attribute.String("calculation_id", req.CalculationId),
	)

	return &historyv1.DeleteCalculationResponse{Success: true}, nil
}

// GetStatistics возвращает статистику пользователя
func (s *HistoryService) GetStatistics(
	ctx context.Context,
	req *historyv1.GetStatisticsRequest,
) (*historyv1.GetStatisticsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "HistoryService.GetStatistics")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", req.UserId))

	if req.UserId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.NewWithField(pkgerrors.CodeInvalidArgument, "user_id is required", "user_id"),
		)
	}

	var startTime, endTime *time.Time
	if req.TimeRange != nil {
		if req.TimeRange.StartTimestamp > 0 {
			t := time.Unix(req.TimeRange.StartTimestamp, 0)
			startTime = &t
		}
		if req.TimeRange.EndTimestamp > 0 {
			t := time.Unix(req.TimeRange.EndTimestamp, 0)
			endTime = &t
		}
	}

	stats, err := s.repo.GetUserStatistics(ctx, req.UserId, startTime, endTime)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get statistics"),
		)
	}

	dailyStats := make([]*historyv1.DailyStats, len(stats.DailyStats))
	for i, ds := range stats.DailyStats {
		dailyStats[i] = &historyv1.DailyStats{
			Date:      ds.Date,
			Count:     int32(ds.Count),
			TotalFlow: ds.TotalFlow,
		}
	}

	return &historyv1.GetStatisticsResponse{
		TotalCalculations:        int32(stats.TotalCalculations),
		AverageMaxFlow:           stats.AverageMaxFlow,
		AverageCost:              stats.AverageTotalCost,
		AverageComputationTimeMs: stats.AverageComputationTimeMs,
		CalculationsByAlgorithm:  toInt32Map(stats.CalculationsByAlgorithm),
		DailyStats:               dailyStats,
	}, nil
}

// Вспомогательные методы

func (s *HistoryService) toListOptions(req *historyv1.ListCalculationsRequest) *repository.ListOptions {
	opts := &repository.ListOptions{
		Limit:  20,
		Offset: 0,
		Sort:   repository.SortByCreatedDesc,
	}

	if req.Pagination != nil {
		if req.Pagination.PageSize > 0 {
			opts.Limit = int(req.Pagination.PageSize)
		}
		if req.Pagination.Page > 0 {
			opts.Offset = int((req.Pagination.Page - 1) * req.Pagination.PageSize)
		}
	}

	if req.Filter != nil {
		opts.Filter = &repository.ListFilter{}

		if req.Filter.Algorithm != commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
			opts.Filter.Algorithm = req.Filter.Algorithm.String()
		}

		if len(req.Filter.Tags) > 0 {
			opts.Filter.Tags = req.Filter.Tags
		}

		if req.Filter.MinFlow > 0 {
			opts.Filter.MinFlow = &req.Filter.MinFlow
		}
		if req.Filter.MaxFlow > 0 {
			opts.Filter.MaxFlow = &req.Filter.MaxFlow
		}

		if req.Filter.TimeRange != nil {
			if req.Filter.TimeRange.StartTimestamp > 0 {
				t := time.Unix(req.Filter.TimeRange.StartTimestamp, 0)
				opts.Filter.StartTime = &t
			}
			if req.Filter.TimeRange.EndTimestamp > 0 {
				t := time.Unix(req.Filter.TimeRange.EndTimestamp, 0)
				opts.Filter.EndTime = &t
			}
		}
	}

	switch req.Sort {
	case historyv1.HistorySortOrder_HISTORY_SORT_ORDER_CREATED_ASC:
		opts.Sort = repository.SortByCreatedAsc
	case historyv1.HistorySortOrder_HISTORY_SORT_ORDER_MAX_FLOW_DESC:
		opts.Sort = repository.SortByMaxFlowDesc
	case historyv1.HistorySortOrder_HISTORY_SORT_ORDER_COST_DESC:
		opts.Sort = repository.SortByTotalCostDesc
	default:
		opts.Sort = repository.SortByCreatedDesc
	}

	return opts
}

func (s *HistoryService) toCalculationRecord(calc *repository.Calculation) (*historyv1.CalculationRecord, error) {
	// Десериализуем в правильные типы из optimization/v1
	var solveRequest optimizationv1.SolveRequest
	var solveResponse optimizationv1.SolveResponse

	if err := protojson.Unmarshal(calc.RequestData, &solveRequest); err != nil {
		// Логируем, но не падаем — данные могут быть повреждены
		solveRequest = optimizationv1.SolveRequest{}
	}

	if err := protojson.Unmarshal(calc.ResponseData, &solveResponse); err != nil {
		solveResponse = optimizationv1.SolveResponse{}
	}

	// Конвертируем теги обратно в map
	tags := make(map[string]string)
	for _, tag := range calc.Tags {
		parts := splitOnce(tag, ":")
		if len(parts) == 2 {
			tags[parts[0]] = parts[1]
		}
	}

	return &historyv1.CalculationRecord{
		CalculationId: calc.ID,
		UserId:        calc.UserID,
		Name:          calc.Name,
		CreatedAt:     timestamppb.New(calc.CreatedAt),
		Request:       &solveRequest,
		Response:      &solveResponse,
		Tags:          tags,
	}, nil
}

func (s *HistoryService) toCalculationSummary(calc *repository.CalculationSummary) *historyv1.CalculationSummary {
	algo := commonv1.Algorithm_ALGORITHM_UNSPECIFIED
	if v, ok := commonv1.Algorithm_value[calc.Algorithm]; ok {
		algo = commonv1.Algorithm(v)
	}

	return &historyv1.CalculationSummary{
		CalculationId:     calc.ID,
		Name:              calc.Name,
		CreatedAt:         timestamppb.New(calc.CreatedAt),
		MaxFlow:           calc.MaxFlow,
		TotalCost:         calc.TotalCost,
		Algorithm:         algo,
		ComputationTimeMs: calc.ComputationTimeMs,
		NodeCount:         int32(calc.NodeCount),
		EdgeCount:         int32(calc.EdgeCount),
		Tags:              calc.Tags,
	}
}

func splitOnce(s, sep string) []string {
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}

func toInt32Map(m map[string]int) map[string]int32 {
	result := make(map[string]int32, len(m))
	for k, v := range m {
		result[k] = int32(v)
	}
	return result
}
