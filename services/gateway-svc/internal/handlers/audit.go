package handlers

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	"logistics/pkg/logger"
	"logistics/services/gateway-svc/internal/clients"
	"logistics/services/gateway-svc/internal/middleware"
)

// AuditHandler обработчики для аудита
type AuditHandler struct {
	clients *clients.Manager
}

// NewAuditHandler создаёт обработчик
func NewAuditHandler(clients *clients.Manager) *AuditHandler {
	return &AuditHandler{clients: clients}
}

// GetAuditLogs получает аудит логи (только для админов)
func (h *AuditHandler) GetAuditLogs(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetAuditLogsRequest],
) (*connect.Response[gatewayv1.AuditLogsResponse], error) {
	// Проверяем права доступа
	if err := h.checkAdminAccess(ctx); err != nil {
		return nil, err
	}

	msg := req.Msg
	requestID := middleware.GetRequestID(ctx)

	logger.Log.Info("Getting audit logs",
		"request_id", requestID,
		"services", msg.Services,
		"actions", msg.Actions,
		"user_id", msg.UserId,
	)

	// Конвертируем actions в enum
	actions := make([]auditv1.AuditAction, 0, len(msg.Actions))
	for _, a := range msg.Actions {
		actions = append(actions, h.parseAction(a))
	}

	// Вызываем audit-svc
	resp, err := h.clients.Audit().Raw().GetAuditLogs(ctx, &auditv1.GetAuditLogsRequest{
		Filter: &auditv1.AuditFilter{
			TimeRange: &commonv1.TimeRange{
				StartTimestamp: msg.StartTime.AsTime().Unix(),
				EndTimestamp:   msg.EndTime.AsTime().Unix(),
			},
			Services:     msg.Services,
			Actions:      actions,
			UserId:       msg.UserId,
			ResourceType: msg.ResourceType,
		},
		Pagination: &commonv1.PaginationRequest{
			Page:     h.calculatePage(msg.Offset, msg.Limit),
			PageSize: msg.Limit,
		},
	})
	if err != nil {
		logger.Log.Error("Failed to get audit logs",
			"request_id", requestID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Конвертируем ответ
	entries := make([]*gatewayv1.AuditEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, h.convertEntry(e))
	}

	return connect.NewResponse(&gatewayv1.AuditLogsResponse{
		Entries:    entries,
		TotalCount: resp.Pagination.TotalItems,
		HasMore:    resp.Pagination.HasNext,
	}), nil
}

// GetUserActivity получает активность пользователя
func (h *AuditHandler) GetUserActivity(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetUserActivityRequest],
) (*connect.Response[gatewayv1.UserActivityResponse], error) {
	msg := req.Msg
	requestID := middleware.GetRequestID(ctx)
	currentUserID := middleware.GetUserID(ctx)

	// Определяем целевого пользователя
	targetUserID := msg.UserId
	if targetUserID == "" {
		targetUserID = currentUserID
	}

	// Пользователь может видеть только свою активность, админ - любую
	if targetUserID != currentUserID {
		if err := h.checkAdminAccess(ctx); err != nil {
			return nil, err
		}
	}

	logger.Log.Info("Getting user activity",
		"request_id", requestID,
		"target_user_id", targetUserID,
		"requester_id", currentUserID,
	)

	// Вызываем audit-svc
	resp, err := h.clients.Audit().Raw().GetUserActivity(ctx, &auditv1.GetUserActivityRequest{
		UserId: targetUserID,
		TimeRange: &commonv1.TimeRange{
			StartTimestamp: msg.StartTime.AsTime().Unix(),
			EndTimestamp:   msg.EndTime.AsTime().Unix(),
		},
		Pagination: &commonv1.PaginationRequest{
			Page:     h.calculatePage(msg.Offset, msg.Limit),
			PageSize: msg.Limit,
		},
	})
	if err != nil {
		logger.Log.Error("Failed to get user activity",
			"request_id", requestID,
			"target_user_id", targetUserID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Конвертируем entries
	entries := make([]*gatewayv1.AuditEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, h.convertEntry(e))
	}

	return connect.NewResponse(&gatewayv1.UserActivityResponse{
		Entries:    entries,
		Summary:    h.convertActivitySummary(resp.Summary),
		TotalCount: resp.Pagination.TotalItems,
	}), nil
}

// GetAuditStats получает статистику аудита (только для админов)
func (h *AuditHandler) GetAuditStats(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetAuditStatsRequest],
) (*connect.Response[gatewayv1.AuditStatsResponse], error) {
	if err := h.checkAdminAccess(ctx); err != nil {
		return nil, err
	}

	msg := req.Msg
	requestID := middleware.GetRequestID(ctx)

	logger.Log.Info("Getting audit stats",
		"request_id", requestID,
		"group_by", msg.GroupBy,
	)

	// Вызываем audit-svc
	resp, err := h.clients.Audit().Raw().GetAuditStats(ctx, &auditv1.GetAuditStatsRequest{
		TimeRange: &commonv1.TimeRange{
			StartTimestamp: msg.StartTime.AsTime().Unix(),
			EndTimestamp:   msg.EndTime.AsTime().Unix(),
		},
		GroupBy: msg.GroupBy,
	})
	if err != nil {
		logger.Log.Error("Failed to get audit stats",
			"request_id", requestID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Конвертируем timeline
	timeline := make([]*gatewayv1.AuditStatsPoint, 0, len(resp.Timeline))
	for _, t := range resp.Timeline {
		timeline = append(timeline, &gatewayv1.AuditStatsPoint{
			Timestamp:    t.Timestamp,
			Count:        t.Count,
			SuccessCount: t.SuccessCount,
			FailureCount: t.FailureCount,
		})
	}

	return connect.NewResponse(&gatewayv1.AuditStatsResponse{
		TotalEvents:      resp.Summary.TotalEvents,
		SuccessfulEvents: resp.Summary.SuccessfulEvents,
		FailedEvents:     resp.Summary.FailedEvents,
		UniqueUsers:      resp.Summary.UniqueUsers,
		ByService:        resp.ByService,
		ByAction:         resp.ByAction,
		Timeline:         timeline,
	}), nil
}

// ============================================================
// Helper methods
// ============================================================

func (h *AuditHandler) checkAdminAccess(ctx context.Context) error {
	userInfo := middleware.GetUserInfo(ctx)
	if userInfo == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Прямое сравнение поля Role
	if userInfo.Role != "admin" {
		logger.Log.Warn("Access denied to audit",
			"user_id", userInfo.UserId,
			"role", userInfo.Role,
		)
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin access required"))
	}

	return nil
}

func (h *AuditHandler) calculatePage(offset, limit int32) int32 {
	if limit <= 0 {
		limit = 20
	}
	return (offset / limit) + 1
}

func (h *AuditHandler) parseAction(action string) auditv1.AuditAction {
	switch action {
	case "CREATE":
		return auditv1.AuditAction_AUDIT_ACTION_CREATE
	case "READ":
		return auditv1.AuditAction_AUDIT_ACTION_READ
	case "UPDATE":
		return auditv1.AuditAction_AUDIT_ACTION_UPDATE
	case "DELETE":
		return auditv1.AuditAction_AUDIT_ACTION_DELETE
	case "LOGIN":
		return auditv1.AuditAction_AUDIT_ACTION_LOGIN
	case "LOGOUT":
		return auditv1.AuditAction_AUDIT_ACTION_LOGOUT
	case "SOLVE":
		return auditv1.AuditAction_AUDIT_ACTION_SOLVE
	case "ANALYZE":
		return auditv1.AuditAction_AUDIT_ACTION_ANALYZE
	case "VALIDATE":
		return auditv1.AuditAction_AUDIT_ACTION_VALIDATE
	case "EXPORT":
		return auditv1.AuditAction_AUDIT_ACTION_EXPORT
	default:
		return auditv1.AuditAction_AUDIT_ACTION_UNSPECIFIED
	}
}

func (h *AuditHandler) formatAction(action auditv1.AuditAction) string {
	switch action {
	case auditv1.AuditAction_AUDIT_ACTION_CREATE:
		return "CREATE"
	case auditv1.AuditAction_AUDIT_ACTION_READ:
		return "READ"
	case auditv1.AuditAction_AUDIT_ACTION_UPDATE:
		return "UPDATE"
	case auditv1.AuditAction_AUDIT_ACTION_DELETE:
		return "DELETE"
	case auditv1.AuditAction_AUDIT_ACTION_LOGIN:
		return "LOGIN"
	case auditv1.AuditAction_AUDIT_ACTION_LOGOUT:
		return "LOGOUT"
	case auditv1.AuditAction_AUDIT_ACTION_SOLVE:
		return "SOLVE"
	case auditv1.AuditAction_AUDIT_ACTION_ANALYZE:
		return "ANALYZE"
	case auditv1.AuditAction_AUDIT_ACTION_VALIDATE:
		return "VALIDATE"
	case auditv1.AuditAction_AUDIT_ACTION_EXPORT:
		return "EXPORT"
	default:
		return "UNKNOWN"
	}
}

func (h *AuditHandler) formatOutcome(outcome auditv1.AuditOutcome) string {
	switch outcome {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS:
		return "SUCCESS"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FAILURE:
		return "FAILURE"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_DENIED:
		return "DENIED"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (h *AuditHandler) convertEntry(e *auditv1.AuditEntry) *gatewayv1.AuditEntry {
	if e == nil {
		return nil
	}

	return &gatewayv1.AuditEntry{
		Id:           e.Id,
		Timestamp:    e.Timestamp,
		Service:      e.Service,
		Method:       e.Method,
		Action:       h.formatAction(e.Action),
		Outcome:      h.formatOutcome(e.Outcome),
		UserId:       e.UserId,
		Username:     e.Username,
		ClientIp:     e.ClientIp,
		ResourceType: e.ResourceType,
		ResourceId:   e.ResourceId,
		DurationMs:   e.DurationMs,
		ErrorMessage: e.ErrorMessage,
		Metadata:     e.Metadata,
	}
}

func (h *AuditHandler) convertActivitySummary(s *auditv1.UserActivitySummary) *gatewayv1.UserActivitySummary {
	if s == nil {
		return nil
	}

	return &gatewayv1.UserActivitySummary{
		TotalActions:      s.TotalActions,
		SuccessfulActions: s.SuccessfulActions,
		FailedActions:     s.FailedActions,
		ActionsByType:     s.ActionsByType,
		ActionsByService:  s.ActionsByService,
		FirstActivity:     s.FirstActivity,
		LastActivity:      s.LastActivity,
	}
}
