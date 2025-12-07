package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"logistics/pkg/database"
	"logistics/pkg/telemetry"
)

// PostgresAuditRepository PostgreSQL реализация
type PostgresAuditRepository struct {
	db database.DB
}

// NewPostgresAuditRepository создаёт новый репозиторий
func NewPostgresAuditRepository(db database.DB) *PostgresAuditRepository {
	return &PostgresAuditRepository{db: db}
}

func (r *PostgresAuditRepository) Create(ctx context.Context, entry *AuditEntry) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.Create")
	defer span.End()

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (
			timestamp, service, method, request_id,
			action, outcome,
			user_id, username, user_role,
			client_ip, user_agent,
			resource_type, resource_id, resource_name,
			duration_ms, error_code, error_message,
			changes_before, changes_after, changed_fields,
			metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id
	`

	var clientIP any
	if entry.ClientIP != "" {
		clientIP = entry.ClientIP
	}

	err = r.db.QueryRow(ctx, query,
		entry.Timestamp,
		entry.Service,
		entry.Method,
		nullString(entry.RequestID),
		entry.Action,
		entry.Outcome,
		nullString(entry.UserID),
		nullString(entry.Username),
		nullString(entry.UserRole),
		clientIP,
		nullString(entry.UserAgent),
		nullString(entry.ResourceType),
		nullString(entry.ResourceID),
		nullString(entry.ResourceName),
		entry.DurationMs,
		nullString(entry.ErrorCode),
		nullString(entry.ErrorMessage),
		entry.ChangesBefore,
		entry.ChangesAfter,
		entry.ChangedFields,
		metadataJSON,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}

	return nil
}

func (r *PostgresAuditRepository) CreateBatch(ctx context.Context, entries []*AuditEntry) (int, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.CreateBatch")
	defer span.End()

	if len(entries) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			telemetry.RecordError(ctx, err)
		}
	}()

	count := 0
	for _, entry := range entries {
		if err := r.insertEntry(ctx, tx, entry); err == nil {
			count++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *PostgresAuditRepository) insertEntry(ctx context.Context, tx pgx.Tx, entry *AuditEntry) error {
	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (
			timestamp, service, method, request_id,
			action, outcome,
			user_id, username, user_role,
			client_ip, user_agent,
			resource_type, resource_id, resource_name,
			duration_ms, error_code, error_message,
			changes_before, changes_after, changed_fields,
			metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	var clientIP any
	if entry.ClientIP != "" {
		clientIP = entry.ClientIP
	}

	_, err = tx.Exec(ctx, query,
		entry.Timestamp,
		entry.Service,
		entry.Method,
		nullString(entry.RequestID),
		entry.Action,
		entry.Outcome,
		nullString(entry.UserID),
		nullString(entry.Username),
		nullString(entry.UserRole),
		clientIP,
		nullString(entry.UserAgent),
		nullString(entry.ResourceType),
		nullString(entry.ResourceID),
		nullString(entry.ResourceName),
		entry.DurationMs,
		nullString(entry.ErrorCode),
		nullString(entry.ErrorMessage),
		entry.ChangesBefore,
		entry.ChangesAfter,
		entry.ChangedFields,
		metadataJSON,
	)
	return err
}

func (r *PostgresAuditRepository) GetByID(ctx context.Context, id string) (*AuditEntry, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.GetByID")
	defer span.End()

	query := `
		SELECT
			id, timestamp, service, method, request_id,
			action, outcome,
			user_id, username, user_role,
			client_ip, user_agent,
			resource_type, resource_id, resource_name,
			duration_ms, error_code, error_message,
			changes_before, changes_after, changed_fields,
			metadata
		FROM audit_logs
		WHERE id = $1
	`

	entry := &AuditEntry{}
	var (
		requestID, userID, username, userRole pgtype.Text
		clientIP                              pgtype.Text
		userAgent, resourceType, resourceID   pgtype.Text
		resourceName, errorCode, errorMessage pgtype.Text
		changesBefore, changesAfter           []byte
		changedFields                         pgtype.Array[string]
		metadata                              []byte
	)

	err := r.db.QueryRow(ctx, query, id).Scan(
		&entry.ID,
		&entry.Timestamp,
		&entry.Service,
		&entry.Method,
		&requestID,
		&entry.Action,
		&entry.Outcome,
		&userID,
		&username,
		&userRole,
		&clientIP,
		&userAgent,
		&resourceType,
		&resourceID,
		&resourceName,
		&entry.DurationMs,
		&errorCode,
		&errorMessage,
		&changesBefore,
		&changesAfter,
		&changedFields,
		&metadata,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuditNotFound
		}
		return nil, fmt.Errorf("failed to get audit entry: %w", err)
	}

	r.populateEntryFromScan(entry, requestID, userID, username, userRole, clientIP,
		userAgent, resourceType, resourceID, resourceName, errorCode, errorMessage,
		changesBefore, changesAfter, changedFields, metadata)

	return entry, nil
}

func (r *PostgresAuditRepository) populateEntryFromScan(
	entry *AuditEntry,
	requestID, userID, username, userRole, clientIP pgtype.Text,
	userAgent, resourceType, resourceID, resourceName pgtype.Text,
	errorCode, errorMessage pgtype.Text,
	changesBefore, changesAfter []byte,
	changedFields pgtype.Array[string],
	metadata []byte,
) {
	entry.RequestID = requestID.String
	entry.UserID = userID.String
	entry.Username = username.String
	entry.UserRole = userRole.String
	entry.ClientIP = clientIP.String
	entry.UserAgent = userAgent.String
	entry.ResourceType = resourceType.String
	entry.ResourceID = resourceID.String
	entry.ResourceName = resourceName.String
	entry.ErrorCode = errorCode.String
	entry.ErrorMessage = errorMessage.String
	entry.ChangesBefore = changesBefore
	entry.ChangesAfter = changesAfter
	entry.ChangedFields = changedFields.Elements

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			entry.Metadata = make(map[string]string)
		}
	}
}

func (r *PostgresAuditRepository) List(ctx context.Context, filter *AuditFilter, opts *ListOptions) ([]*AuditEntry, int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.List")
	defer span.End()

	opts = normalizeListOptions(opts)
	where, args := r.buildWhereClause(filter)

	total, err := r.countEntries(ctx, where, args)
	if err != nil {
		return nil, 0, err
	}

	entries, err := r.fetchEntries(ctx, where, args, opts)
	if err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func normalizeListOptions(opts *ListOptions) *ListOptions {
	if opts == nil {
		return &ListOptions{Limit: 50, Offset: 0, SortOrder: "timestamp_desc"}
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 1000 {
		opts.Limit = 1000
	}
	return opts
}

func (r *PostgresAuditRepository) countEntries(ctx context.Context, where string, args []any) (int64, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", where)
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return total, nil
}

func (r *PostgresAuditRepository) fetchEntries(ctx context.Context, where string, args []any, opts *ListOptions) ([]*AuditEntry, error) {
	orderBy := "timestamp DESC"
	if opts.SortOrder == "timestamp_asc" {
		orderBy = "timestamp ASC"
	}

	selectQuery := fmt.Sprintf(`
		SELECT
			id, timestamp, service, method, request_id,
			action, outcome,
			user_id, username, user_role,
			client_ip, user_agent,
			resource_type, resource_id, resource_name,
			duration_ms, error_code, error_message,
			metadata
		FROM audit_logs
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, len(args)+1, len(args)+2)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		entry, err := r.scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (r *PostgresAuditRepository) GetResourceHistory(
	ctx context.Context,
	resourceType, resourceID string,
	opts *ListOptions,
) ([]*AuditEntry, *ResourceSummary, int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.GetResourceHistory")
	defer span.End()

	filter := &AuditFilter{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	entries, total, err := r.List(ctx, filter, opts)
	if err != nil {
		return nil, nil, 0, err
	}

	summary, err := r.getResourceSummary(ctx, resourceType, resourceID)
	if err != nil {
		return nil, nil, 0, err
	}

	return entries, summary, total, nil
}

func (r *PostgresAuditRepository) getResourceSummary(ctx context.Context, resourceType, resourceID string) (*ResourceSummary, error) {
	summaryQuery := `
		SELECT
			MIN(timestamp) as created_at,
			(SELECT user_id FROM audit_logs WHERE resource_type = $1 AND resource_id = $2 AND action = 'CREATE' ORDER BY timestamp LIMIT 1) as created_by,
			MAX(timestamp) as last_modified_at,
			(SELECT user_id FROM audit_logs WHERE resource_type = $1 AND resource_id = $2 ORDER BY timestamp DESC LIMIT 1) as last_modified_by,
			COUNT(*) as total_changes
		FROM audit_logs
		WHERE resource_type = $1 AND resource_id = $2
	`

	summary := &ResourceSummary{}
	var createdBy, lastModifiedBy pgtype.Text
	err := r.db.QueryRow(ctx, summaryQuery, resourceType, resourceID).Scan(
		&summary.CreatedAt,
		&createdBy,
		&summary.LastModifiedAt,
		&lastModifiedBy,
		&summary.TotalChanges,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	summary.CreatedBy = createdBy.String
	summary.LastModifiedBy = lastModifiedBy.String

	return summary, nil
}

func (r *PostgresAuditRepository) GetUserActivity(
	ctx context.Context,
	userID string,
	timeRange *TimeRange,
	opts *ListOptions,
) ([]*AuditEntry, *UserActivitySummary, int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.GetUserActivity")
	defer span.End()

	filter := &AuditFilter{
		UserID:    userID,
		TimeRange: timeRange,
	}

	entries, total, err := r.List(ctx, filter, opts)
	if err != nil {
		return nil, nil, 0, err
	}

	summary, err := r.getUserActivitySummary(ctx, userID, timeRange)
	if err != nil {
		return nil, nil, 0, err
	}

	return entries, summary, total, nil
}

func (r *PostgresAuditRepository) getUserActivitySummary(ctx context.Context, userID string, timeRange *TimeRange) (*UserActivitySummary, error) {
	whereClause, args := r.buildUserActivityWhere(userID, timeRange)

	summary := &UserActivitySummary{
		ActionsByType:    make(map[string]int),
		ActionsByService: make(map[string]int),
	}

	if err := r.fetchActivityCounts(ctx, whereClause, args, summary); err != nil {
		return nil, err
	}

	r.fetchActionsByType(ctx, whereClause, args, summary)
	r.fetchActionsByService(ctx, whereClause, args, summary)

	return summary, nil
}

func (r *PostgresAuditRepository) buildUserActivityWhere(userID string, timeRange *TimeRange) (string, []any) {
	whereClause := "user_id = $1"
	args := []any{userID}
	argNum := 2

	if timeRange != nil {
		if !timeRange.Start.IsZero() {
			whereClause += fmt.Sprintf(" AND timestamp >= $%d", argNum)
			args = append(args, timeRange.Start)
			argNum++
		}
		if !timeRange.End.IsZero() {
			whereClause += fmt.Sprintf(" AND timestamp <= $%d", argNum)
			args = append(args, timeRange.End)
		}
	}

	return whereClause, args
}

func (r *PostgresAuditRepository) fetchActivityCounts(ctx context.Context, whereClause string, args []any, summary *UserActivitySummary) error {
	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE outcome = 'SUCCESS') as success,
			COUNT(*) FILTER (WHERE outcome = 'FAILURE') as failure,
			COUNT(*) FILTER (WHERE outcome = 'DENIED') as denied,
			MIN(timestamp) as first_activity,
			MAX(timestamp) as last_activity
		FROM audit_logs
		WHERE %s
	`, whereClause)

	err := r.db.QueryRow(ctx, summaryQuery, args...).Scan(
		&summary.TotalActions,
		&summary.SuccessfulActions,
		&summary.FailedActions,
		&summary.DeniedActions,
		&summary.FirstActivity,
		&summary.LastActivity,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

func (r *PostgresAuditRepository) fetchActionsByType(ctx context.Context, whereClause string, args []any, summary *UserActivitySummary) {
	typeQuery := fmt.Sprintf(`
		SELECT action, COUNT(*) FROM audit_logs WHERE %s GROUP BY action
	`, whereClause)
	typeRows, err := r.db.Query(ctx, typeQuery, args...)
	if err != nil || typeRows == nil {
		return
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var action string
		var count int
		if err := typeRows.Scan(&action, &count); err == nil {
			summary.ActionsByType[action] = count
		}
	}
}

func (r *PostgresAuditRepository) fetchActionsByService(ctx context.Context, whereClause string, args []any, summary *UserActivitySummary) {
	svcQuery := fmt.Sprintf(`
		SELECT service, COUNT(*) FROM audit_logs WHERE %s GROUP BY service
	`, whereClause)
	svcRows, err := r.db.Query(ctx, svcQuery, args...)
	if err != nil || svcRows == nil {
		return
	}
	defer svcRows.Close()

	for svcRows.Next() {
		var service string
		var count int
		if err := svcRows.Scan(&service, &count); err == nil {
			summary.ActionsByService[service] = count
		}
	}
}

func (r *PostgresAuditRepository) GetStats(ctx context.Context, timeRange *TimeRange, _ string) (*AuditStats, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresAuditRepository.GetStats")
	defer span.End()

	whereClause, args := r.buildTimeRangeWhere(timeRange)

	stats := &AuditStats{
		ByService:    make(map[string]int64),
		ByAction:     make(map[string]int64),
		ByOutcome:    make(map[string]int64),
		Timeline:     make([]TimelinePoint, 0),
		TopUsers:     make([]TopUser, 0),
		TopResources: make([]TopResource, 0),
	}

	if err := r.fetchStatsSummary(ctx, whereClause, args, stats); err != nil {
		return nil, err
	}

	r.fetchStatsByService(ctx, whereClause, args, stats)
	r.fetchStatsByAction(ctx, whereClause, args, stats)
	r.fetchStatsByOutcome(ctx, whereClause, args, stats)
	r.fetchTopUsers(ctx, whereClause, args, stats)
	r.fetchTopResources(ctx, whereClause, args, stats)

	return stats, nil
}

func (r *PostgresAuditRepository) buildTimeRangeWhere(timeRange *TimeRange) (string, []any) {
	whereClause := "1=1"
	args := []any{}
	argNum := 1

	if timeRange != nil {
		if !timeRange.Start.IsZero() {
			whereClause += fmt.Sprintf(" AND timestamp >= $%d", argNum)
			args = append(args, timeRange.Start)
			argNum++
		}
		if !timeRange.End.IsZero() {
			whereClause += fmt.Sprintf(" AND timestamp <= $%d", argNum)
			args = append(args, timeRange.End)
		}
	}

	return whereClause, args
}

func (r *PostgresAuditRepository) fetchStatsSummary(ctx context.Context, whereClause string, args []any, stats *AuditStats) error {
	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE outcome = 'SUCCESS'),
			COUNT(*) FILTER (WHERE outcome = 'FAILURE'),
			COUNT(*) FILTER (WHERE outcome = 'DENIED'),
			COUNT(DISTINCT user_id),
			COUNT(DISTINCT resource_id),
			COALESCE(AVG(duration_ms), 0)
		FROM audit_logs
		WHERE %s
	`, whereClause)

	err := r.db.QueryRow(ctx, summaryQuery, args...).Scan(
		&stats.TotalEvents,
		&stats.SuccessfulEvents,
		&stats.FailedEvents,
		&stats.DeniedEvents,
		&stats.UniqueUsers,
		&stats.UniqueResources,
		&stats.AvgDurationMs,
	)
	if err != nil {
		return fmt.Errorf("failed to get stats summary: %w", err)
	}
	return nil
}

func (r *PostgresAuditRepository) fetchStatsByService(ctx context.Context, whereClause string, args []any, stats *AuditStats) {
	svcQuery := fmt.Sprintf(`
		SELECT service, COUNT(*) FROM audit_logs WHERE %s GROUP BY service ORDER BY COUNT(*) DESC
	`, whereClause)
	svcRows, err := r.db.Query(ctx, svcQuery, args...)
	if err != nil || svcRows == nil {
		return
	}
	defer svcRows.Close()

	for svcRows.Next() {
		var service string
		var count int64
		if err := svcRows.Scan(&service, &count); err == nil {
			stats.ByService[service] = count
		}
	}
}

func (r *PostgresAuditRepository) fetchStatsByAction(ctx context.Context, whereClause string, args []any, stats *AuditStats) {
	actionQuery := fmt.Sprintf(`
		SELECT action, COUNT(*) FROM audit_logs WHERE %s GROUP BY action ORDER BY COUNT(*) DESC
	`, whereClause)
	actionRows, err := r.db.Query(ctx, actionQuery, args...)
	if err != nil || actionRows == nil {
		return
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var action string
		var count int64
		if err := actionRows.Scan(&action, &count); err == nil {
			stats.ByAction[action] = count
		}
	}
}

func (r *PostgresAuditRepository) fetchStatsByOutcome(ctx context.Context, whereClause string, args []any, stats *AuditStats) {
	outcomeQuery := fmt.Sprintf(`
		SELECT outcome, COUNT(*) FROM audit_logs WHERE %s GROUP BY outcome
	`, whereClause)
	outcomeRows, err := r.db.Query(ctx, outcomeQuery, args...)
	if err != nil || outcomeRows == nil {
		return
	}
	defer outcomeRows.Close()

	for outcomeRows.Next() {
		var outcome string
		var count int64
		if err := outcomeRows.Scan(&outcome, &count); err == nil {
			stats.ByOutcome[outcome] = count
		}
	}
}

func (r *PostgresAuditRepository) fetchTopUsers(ctx context.Context, whereClause string, args []any, stats *AuditStats) {
	topUsersQuery := fmt.Sprintf(`
		SELECT user_id, username, COUNT(*) as cnt
		FROM audit_logs
		WHERE %s AND user_id IS NOT NULL
		GROUP BY user_id, username
		ORDER BY cnt DESC
		LIMIT 10
	`, whereClause)
	userRows, err := r.db.Query(ctx, topUsersQuery, args...)
	if err != nil || userRows == nil {
		return
	}
	defer userRows.Close()

	for userRows.Next() {
		var tu TopUser
		var username pgtype.Text
		if err := userRows.Scan(&tu.UserID, &username, &tu.ActionCount); err == nil {
			tu.Username = username.String
			stats.TopUsers = append(stats.TopUsers, tu)
		}
	}
}

func (r *PostgresAuditRepository) fetchTopResources(ctx context.Context, whereClause string, args []any, stats *AuditStats) {
	topResQuery := fmt.Sprintf(`
		SELECT resource_type, resource_id, COUNT(*) as cnt
		FROM audit_logs
		WHERE %s AND resource_id IS NOT NULL
		GROUP BY resource_type, resource_id
		ORDER BY cnt DESC
		LIMIT 10
	`, whereClause)
	resRows, err := r.db.Query(ctx, topResQuery, args...)
	if err != nil || resRows == nil {
		return
	}
	defer resRows.Close()

	for resRows.Next() {
		var tr TopResource
		if err := resRows.Scan(&tr.ResourceType, &tr.ResourceID, &tr.ActionCount); err == nil {
			stats.TopResources = append(stats.TopResources, tr)
		}
	}
}

func (r *PostgresAuditRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs").Scan(&count)
	return count, err
}

func (r *PostgresAuditRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, "DELETE FROM audit_logs WHERE timestamp < $1", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Вспомогательные методы

func (r *PostgresAuditRepository) buildWhereClause(filter *AuditFilter) (string, []any) {
	if filter == nil {
		return "1=1", nil
	}

	conditions := []string{"1=1"}
	args := []any{}
	argNum := 1

	argNum = r.addTimeRangeConditions(filter.TimeRange, &conditions, &args, argNum)
	argNum = r.addArrayCondition(filter.Services, "service", &conditions, &args, argNum)
	argNum = r.addArrayCondition(filter.Methods, "method", &conditions, &args, argNum)
	argNum = r.addArrayCondition(filter.Actions, "action", &conditions, &args, argNum)
	argNum = r.addArrayCondition(filter.Outcomes, "outcome", &conditions, &args, argNum)
	argNum = r.addStringCondition(filter.UserID, "user_id", &conditions, &args, argNum)
	argNum = r.addStringCondition(filter.ResourceType, "resource_type", &conditions, &args, argNum)
	argNum = r.addStringCondition(filter.ResourceID, "resource_id", &conditions, &args, argNum)
	argNum = r.addStringCondition(filter.ClientIP, "client_ip", &conditions, &args, argNum)

	if filter.SearchQuery != "" {
		conditions = append(conditions, fmt.Sprintf(
			"to_tsvector('russian', COALESCE(method, '') || ' ' || COALESCE(error_message, '')) @@ plainto_tsquery('russian', $%d)",
			argNum,
		))
		args = append(args, filter.SearchQuery)
	}

	return strings.Join(conditions, " AND "), args
}

func (r *PostgresAuditRepository) addTimeRangeConditions(tr *TimeRange, conditions *[]string, args *[]any, argNum int) int {
	if tr == nil {
		return argNum
	}
	if !tr.Start.IsZero() {
		*conditions = append(*conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		*args = append(*args, tr.Start)
		argNum++
	}
	if !tr.End.IsZero() {
		*conditions = append(*conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		*args = append(*args, tr.End)
		argNum++
	}
	return argNum
}

func (r *PostgresAuditRepository) addArrayCondition(values []string, column string, conditions *[]string, args *[]any, argNum int) int {
	if len(values) > 0 {
		*conditions = append(*conditions, fmt.Sprintf("%s = ANY($%d)", column, argNum))
		*args = append(*args, values)
		argNum++
	}
	return argNum
}

func (r *PostgresAuditRepository) addStringCondition(value, column string, conditions *[]string, args *[]any, argNum int) int {
	if value != "" {
		*conditions = append(*conditions, fmt.Sprintf("%s = $%d", column, argNum))
		*args = append(*args, value)
		argNum++
	}
	return argNum
}

func (r *PostgresAuditRepository) scanEntry(rows pgx.Rows) (*AuditEntry, error) {
	entry := &AuditEntry{}
	var (
		requestID, userID, username, userRole pgtype.Text
		clientIP                              pgtype.Text
		userAgent, resourceType, resourceID   pgtype.Text
		resourceName, errorCode, errorMessage pgtype.Text
		metadata                              []byte
	)

	err := rows.Scan(
		&entry.ID,
		&entry.Timestamp,
		&entry.Service,
		&entry.Method,
		&requestID,
		&entry.Action,
		&entry.Outcome,
		&userID,
		&username,
		&userRole,
		&clientIP,
		&userAgent,
		&resourceType,
		&resourceID,
		&resourceName,
		&entry.DurationMs,
		&errorCode,
		&errorMessage,
		&metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan audit entry: %w", err)
	}

	entry.RequestID = requestID.String
	entry.UserID = userID.String
	entry.Username = username.String
	entry.UserRole = userRole.String
	entry.ClientIP = clientIP.String
	entry.UserAgent = userAgent.String
	entry.ResourceType = resourceType.String
	entry.ResourceID = resourceID.String
	entry.ResourceName = resourceName.String
	entry.ErrorCode = errorCode.String
	entry.ErrorMessage = errorMessage.String

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			entry.Metadata = make(map[string]string)
		}
	}

	return entry, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
