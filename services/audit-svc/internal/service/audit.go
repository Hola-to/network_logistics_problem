package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/telemetry"
	"logistics/services/audit-svc/internal/repository"
)

var startTime = time.Now()

// AuditService реализация gRPC сервиса аудита
type AuditService struct {
	auditv1.UnimplementedAuditServiceServer
	repo    repository.AuditRepository
	version string
}

// NewAuditService создаёт новый сервис
func NewAuditService(repo repository.AuditRepository, version string) *AuditService {
	return &AuditService{
		repo:    repo,
		version: version,
	}
}

// LogEvent записывает одно событие
func (s *AuditService) LogEvent(ctx context.Context, req *auditv1.LogEventRequest) (*auditv1.LogEventResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuditService.LogEvent")
	defer span.End()

	if req.Entry == nil {
		return &auditv1.LogEventResponse{Success: false}, nil
	}

	entry := s.protoToEntry(req.Entry)

	if err := s.repo.Create(ctx, entry); err != nil {
		telemetry.SetError(ctx, err)
		return &auditv1.LogEventResponse{
			Success: false,
		}, nil
	}

	return &auditv1.LogEventResponse{
		EventId: entry.ID,
		Success: true,
	}, nil
}

// LogEventBatch записывает batch событий
func (s *AuditService) LogEventBatch(ctx context.Context, req *auditv1.LogEventBatchRequest) (*auditv1.LogEventBatchResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuditService.LogEventBatch",
		telemetry.WithAttributes(attribute.Int("batch_size", len(req.Entries))),
	)
	defer span.End()

	entries := make([]*repository.AuditEntry, 0, len(req.Entries))
	for _, protoEntry := range req.Entries {
		entries = append(entries, s.protoToEntry(protoEntry))
	}

	logged, err := s.repo.CreateBatch(ctx, entries)
	if err != nil {
		telemetry.SetError(ctx, err)
	}

	return &auditv1.LogEventBatchResponse{
		LoggedCount: int32(logged),
		FailedCount: int32(len(entries) - logged),
	}, nil
}

// GetAuditLogs получает аудит логи
func (s *AuditService) GetAuditLogs(ctx context.Context, req *auditv1.GetAuditLogsRequest) (*auditv1.GetAuditLogsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuditService.GetAuditLogs")
	defer span.End()

	filter := s.protoToFilter(req.Filter)
	opts := s.protoToListOptions(req.Pagination, req.Sort)

	entries, total, err := s.repo.List(ctx, filter, opts)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get audit logs"),
		)
	}

	protoEntries := make([]*auditv1.AuditEntry, 0, len(entries))
	for _, entry := range entries {
		protoEntries = append(protoEntries, s.entryToProto(entry))
	}

	pageSize := int32(opts.Limit)
	currentPage := int32(opts.Offset/opts.Limit) + 1
	totalPages := int32((total + int64(opts.Limit) - 1) / int64(opts.Limit))

	return &auditv1.GetAuditLogsResponse{
		Entries: protoEntries,
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

// GetResourceHistory получает историю ресурса
func (s *AuditService) GetResourceHistory(ctx context.Context, req *auditv1.GetResourceHistoryRequest) (*auditv1.GetResourceHistoryResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuditService.GetResourceHistory",
		telemetry.WithAttributes(
			attribute.String("resource_type", req.ResourceType),
			attribute.String("resource_id", req.ResourceId),
		),
	)
	defer span.End()

	if req.ResourceType == "" || req.ResourceId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "resource_type and resource_id are required"),
		)
	}

	opts := s.protoToListOptions(req.Pagination, auditv1.AuditSortOrder_AUDIT_SORT_ORDER_TIMESTAMP_DESC)

	entries, summary, total, err := s.repo.GetResourceHistory(ctx, req.ResourceType, req.ResourceId, opts)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get resource history"),
		)
	}

	protoEntries := make([]*auditv1.AuditEntry, 0, len(entries))
	for _, entry := range entries {
		protoEntries = append(protoEntries, s.entryToProto(entry))
	}

	pageSize := int32(opts.Limit)
	currentPage := int32(opts.Offset/opts.Limit) + 1
	totalPages := int32((total + int64(opts.Limit) - 1) / int64(opts.Limit))

	return &auditv1.GetResourceHistoryResponse{
		Entries: protoEntries,
		Pagination: &commonv1.PaginationResponse{
			CurrentPage: currentPage,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			TotalItems:  total,
			HasNext:     int64(opts.Offset+opts.Limit) < total,
			HasPrevious: opts.Offset > 0,
		},
		Summary: &auditv1.ResourceSummary{
			CreatedAt:      timestamppb.New(summary.CreatedAt),
			CreatedBy:      summary.CreatedBy,
			LastModifiedAt: timestamppb.New(summary.LastModifiedAt),
			LastModifiedBy: summary.LastModifiedBy,
			TotalChanges:   int32(summary.TotalChanges),
		},
	}, nil
}

// GetUserActivity получает активность пользователя
func (s *AuditService) GetUserActivity(ctx context.Context, req *auditv1.GetUserActivityRequest) (*auditv1.GetUserActivityResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuditService.GetUserActivity",
		telemetry.WithAttributes(attribute.String("user_id", req.UserId)),
	)
	defer span.End()

	if req.UserId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "user_id is required"),
		)
	}

	var timeRange *repository.TimeRange
	if req.TimeRange != nil {
		timeRange = &repository.TimeRange{
			Start: time.Unix(req.TimeRange.StartTimestamp, 0),
			End:   time.Unix(req.TimeRange.EndTimestamp, 0),
		}
	}

	opts := s.protoToListOptions(req.Pagination, auditv1.AuditSortOrder_AUDIT_SORT_ORDER_TIMESTAMP_DESC)

	entries, summary, total, err := s.repo.GetUserActivity(ctx, req.UserId, timeRange, opts)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get user activity"),
		)
	}

	protoEntries := make([]*auditv1.AuditEntry, 0, len(entries))
	for _, entry := range entries {
		protoEntries = append(protoEntries, s.entryToProto(entry))
	}

	actionsByType := make(map[string]int32)
	for k, v := range summary.ActionsByType {
		actionsByType[k] = int32(v)
	}
	actionsByService := make(map[string]int32)
	for k, v := range summary.ActionsByService {
		actionsByService[k] = int32(v)
	}

	pageSize := int32(opts.Limit)
	currentPage := int32(opts.Offset/opts.Limit) + 1
	totalPages := int32((total + int64(opts.Limit) - 1) / int64(opts.Limit))

	return &auditv1.GetUserActivityResponse{
		Entries: protoEntries,
		Pagination: &commonv1.PaginationResponse{
			CurrentPage: currentPage,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			TotalItems:  total,
			HasNext:     int64(opts.Offset+opts.Limit) < total,
			HasPrevious: opts.Offset > 0,
		},
		Summary: &auditv1.UserActivitySummary{
			TotalActions:      int32(summary.TotalActions),
			SuccessfulActions: int32(summary.SuccessfulActions),
			FailedActions:     int32(summary.FailedActions),
			DeniedActions:     int32(summary.DeniedActions),
			ActionsByType:     actionsByType,
			ActionsByService:  actionsByService,
			FirstActivity:     timestamppb.New(summary.FirstActivity),
			LastActivity:      timestamppb.New(summary.LastActivity),
		},
	}, nil
}

// GetAuditStats получает статистику
func (s *AuditService) GetAuditStats(ctx context.Context, req *auditv1.GetAuditStatsRequest) (*auditv1.GetAuditStatsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuditService.GetAuditStats")
	defer span.End()

	var timeRange *repository.TimeRange
	if req.TimeRange != nil {
		timeRange = &repository.TimeRange{
			Start: time.Unix(req.TimeRange.StartTimestamp, 0),
			End:   time.Unix(req.TimeRange.EndTimestamp, 0),
		}
	}

	stats, err := s.repo.GetStats(ctx, timeRange, req.GroupBy)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get audit stats"),
		)
	}

	topUsers := make([]*auditv1.TopUser, 0, len(stats.TopUsers))
	for _, u := range stats.TopUsers {
		topUsers = append(topUsers, &auditv1.TopUser{
			UserId:      u.UserID,
			Username:    u.Username,
			ActionCount: u.ActionCount,
		})
	}

	topResources := make([]*auditv1.TopResource, 0, len(stats.TopResources))
	for _, r := range stats.TopResources {
		topResources = append(topResources, &auditv1.TopResource{
			ResourceType: r.ResourceType,
			ResourceId:   r.ResourceID,
			ActionCount:  r.ActionCount,
		})
	}

	timeline := make([]*auditv1.AuditStatsPoint, 0, len(stats.Timeline))
	for _, p := range stats.Timeline {
		timeline = append(timeline, &auditv1.AuditStatsPoint{
			Timestamp:    timestamppb.New(p.Timestamp),
			Count:        p.Count,
			SuccessCount: p.SuccessCount,
			FailureCount: p.FailureCount,
		})
	}

	return &auditv1.GetAuditStatsResponse{
		Summary: &auditv1.AuditStatsSummary{
			TotalEvents:      stats.TotalEvents,
			SuccessfulEvents: stats.SuccessfulEvents,
			FailedEvents:     stats.FailedEvents,
			DeniedEvents:     stats.DeniedEvents,
			UniqueUsers:      stats.UniqueUsers,
			UniqueResources:  stats.UniqueResources,
			AvgDurationMs:    stats.AvgDurationMs,
		},
		Timeline:     timeline,
		ByService:    stats.ByService,
		ByAction:     stats.ByAction,
		ByOutcome:    stats.ByOutcome,
		TopUsers:     topUsers,
		TopResources: topResources,
	}, nil
}

// ExportAuditLogs экспортирует логи (streaming)
func (s *AuditService) ExportAuditLogs(req *auditv1.ExportAuditLogsRequest, stream auditv1.AuditService_ExportAuditLogsServer) error {
	ctx := stream.Context()
	ctx, span := telemetry.StartSpan(ctx, "AuditService.ExportAuditLogs")
	defer span.End()

	filter := s.protoToFilter(req.Filter)

	// Экспортируем постранично
	offset := 0
	batchSize := 100

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		opts := &repository.ListOptions{
			Limit:     batchSize,
			Offset:    offset,
			SortOrder: "timestamp_desc",
		}

		entries, _, err := s.repo.List(ctx, filter, opts)
		if err != nil {
			return pkgerrors.ToGRPC(
				pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to export audit logs"),
			)
		}

		if len(entries) == 0 {
			break
		}

		for _, entry := range entries {
			if err := stream.Send(s.entryToProto(entry)); err != nil {
				return err
			}
		}

		offset += len(entries)

		if len(entries) < batchSize {
			break
		}
	}

	return nil
}

// Health проверка здоровья
func (s *AuditService) Health(ctx context.Context, _ *auditv1.HealthRequest) (*auditv1.HealthResponse, error) {
	total, err := s.repo.Count(ctx)
	if err != nil {
		// Логируем ошибку, но возвращаем ответ с нулевым total
		telemetry.RecordError(ctx, err)
		total = 0
	}

	return &auditv1.HealthResponse{
		Status:            "SERVING",
		Version:           s.version,
		UptimeSeconds:     int64(time.Since(startTime).Seconds()),
		TotalEventsStored: total,
	}, nil
}

// Конвертация proto -> repository

func (s *AuditService) protoToEntry(p *auditv1.AuditEntry) *repository.AuditEntry {
	entry := &repository.AuditEntry{
		ID:           p.Id,
		Service:      p.Service,
		Method:       p.Method,
		RequestID:    p.RequestId,
		Action:       p.Action.String(),
		Outcome:      p.Outcome.String(),
		UserID:       p.UserId,
		Username:     p.Username,
		UserRole:     p.UserRole,
		ClientIP:     p.ClientIp,
		UserAgent:    p.UserAgent,
		ResourceType: p.ResourceType,
		ResourceID:   p.ResourceId,
		ResourceName: p.ResourceName,
		DurationMs:   p.DurationMs,
		ErrorCode:    p.ErrorCode,
		ErrorMessage: p.ErrorMessage,
		Metadata:     p.Metadata,
	}

	if p.Timestamp != nil {
		entry.Timestamp = p.Timestamp.AsTime()
	} else {
		entry.Timestamp = time.Now()
	}

	if p.Changes != nil {
		entry.ChangesBefore = []byte(p.Changes.BeforeJson)
		entry.ChangesAfter = []byte(p.Changes.AfterJson)
		entry.ChangedFields = p.Changes.ChangedFields
	}

	return entry
}

func (s *AuditService) protoToFilter(p *auditv1.AuditFilter) *repository.AuditFilter {
	if p == nil {
		return nil
	}

	filter := &repository.AuditFilter{
		Services:     p.Services,
		Methods:      p.Methods,
		UserID:       p.UserId,
		ResourceType: p.ResourceType,
		ResourceID:   p.ResourceId,
		ClientIP:     p.ClientIp,
		SearchQuery:  p.SearchQuery,
	}

	for _, a := range p.Actions {
		filter.Actions = append(filter.Actions, a.String())
	}
	for _, o := range p.Outcomes {
		filter.Outcomes = append(filter.Outcomes, o.String())
	}

	if p.TimeRange != nil {
		filter.TimeRange = &repository.TimeRange{
			Start: time.Unix(p.TimeRange.StartTimestamp, 0),
			End:   time.Unix(p.TimeRange.EndTimestamp, 0),
		}
	}

	return filter
}

func (s *AuditService) protoToListOptions(p *commonv1.PaginationRequest, sort auditv1.AuditSortOrder) *repository.ListOptions {
	opts := &repository.ListOptions{
		Limit:     50,
		Offset:    0,
		SortOrder: "timestamp_desc",
	}

	if p != nil {
		if p.PageSize > 0 {
			opts.Limit = int(p.PageSize)
		}
		if p.Page > 0 {
			opts.Offset = int((p.Page - 1) * p.PageSize)
		}
	}

	if sort == auditv1.AuditSortOrder_AUDIT_SORT_ORDER_TIMESTAMP_ASC {
		opts.SortOrder = "timestamp_asc"
	}

	return opts
}

// Конвертация repository -> proto

func (s *AuditService) entryToProto(e *repository.AuditEntry) *auditv1.AuditEntry {
	entry := &auditv1.AuditEntry{
		Id:           e.ID,
		Timestamp:    timestamppb.New(e.Timestamp),
		Service:      e.Service,
		Method:       e.Method,
		RequestId:    e.RequestID,
		Action:       s.parseAction(e.Action),
		Outcome:      s.parseOutcome(e.Outcome),
		UserId:       e.UserID,
		Username:     e.Username,
		UserRole:     e.UserRole,
		ClientIp:     e.ClientIP,
		UserAgent:    e.UserAgent,
		ResourceType: e.ResourceType,
		ResourceId:   e.ResourceID,
		ResourceName: e.ResourceName,
		DurationMs:   e.DurationMs,
		ErrorCode:    e.ErrorCode,
		ErrorMessage: e.ErrorMessage,
		Metadata:     e.Metadata,
	}

	if len(e.ChangesBefore) > 0 || len(e.ChangesAfter) > 0 {
		entry.Changes = &auditv1.ChangeSet{
			BeforeJson:    string(e.ChangesBefore),
			AfterJson:     string(e.ChangesAfter),
			ChangedFields: e.ChangedFields,
		}
	}

	return entry
}

func (s *AuditService) parseAction(action string) auditv1.AuditAction {
	switch action {
	case "AUDIT_ACTION_CREATE", "CREATE":
		return auditv1.AuditAction_AUDIT_ACTION_CREATE
	case "AUDIT_ACTION_READ", "READ":
		return auditv1.AuditAction_AUDIT_ACTION_READ
	case "AUDIT_ACTION_UPDATE", "UPDATE":
		return auditv1.AuditAction_AUDIT_ACTION_UPDATE
	case "AUDIT_ACTION_DELETE", "DELETE":
		return auditv1.AuditAction_AUDIT_ACTION_DELETE
	case "AUDIT_ACTION_LOGIN", "LOGIN":
		return auditv1.AuditAction_AUDIT_ACTION_LOGIN
	case "AUDIT_ACTION_LOGOUT", "LOGOUT":
		return auditv1.AuditAction_AUDIT_ACTION_LOGOUT
	case "AUDIT_ACTION_SOLVE", "SOLVE":
		return auditv1.AuditAction_AUDIT_ACTION_SOLVE
	case "AUDIT_ACTION_ANALYZE", "ANALYZE":
		return auditv1.AuditAction_AUDIT_ACTION_ANALYZE
	default:
		return auditv1.AuditAction_AUDIT_ACTION_UNSPECIFIED
	}
}

func (s *AuditService) parseOutcome(outcome string) auditv1.AuditOutcome {
	switch outcome {
	case "AUDIT_OUTCOME_SUCCESS", "SUCCESS":
		return auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS
	case "AUDIT_OUTCOME_FAILURE", "FAILURE":
		return auditv1.AuditOutcome_AUDIT_OUTCOME_FAILURE
	case "AUDIT_OUTCOME_DENIED", "DENIED":
		return auditv1.AuditOutcome_AUDIT_OUTCOME_DENIED
	case "AUDIT_OUTCOME_ERROR", "ERROR":
		return auditv1.AuditOutcome_AUDIT_OUTCOME_ERROR
	default:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_UNSPECIFIED
	}
}
