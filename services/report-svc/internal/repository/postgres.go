// services/report-svc/internal/repository/postgres.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"

	reportv1 "logistics/gen/go/logistics/report/v1"
	"logistics/pkg/database"
)

// PostgresRepository реализация хранилища на PostgreSQL
type PostgresRepository struct {
	db database.DB
}

// NewPostgresRepository создаёт новый репозиторий
func NewPostgresRepository(db database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create сохраняет новый отчёт
func (r *PostgresRepository) Create(ctx context.Context, params *CreateParams) (*Report, error) {
	report := &Report{
		ID:               uuid.New(),
		Title:            params.Title,
		Description:      params.Description,
		Author:           params.Author,
		ReportType:       params.ReportType,
		Format:           params.Format,
		Content:          params.Content,
		ContentType:      params.ContentType,
		Filename:         params.Filename,
		SizeBytes:        int64(len(params.Content)),
		CalculationID:    params.CalculationID,
		GraphID:          params.GraphID,
		UserID:           params.UserID,
		GenerationTimeMs: params.GenerationTimeMs,
		Version:          params.Version,
		Tags:             params.Tags,
		CustomFields:     params.CustomFields,
		CreatedAt:        time.Now().UTC(),
	}

	if params.TTL > 0 {
		expiresAt := report.CreatedAt.Add(params.TTL)
		report.ExpiresAt = &expiresAt
	}

	query := `
		INSERT INTO reports (
			id, title, description, author,
			report_type, format,
			content, content_type, filename, size_bytes,
			calculation_id, graph_id, user_id,
			generation_time_ms, version,
			tags, custom_fields,
			created_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19
		)`

	_, err := r.db.Exec(ctx, query,
		report.ID, report.Title, report.Description, report.Author,
		report.ReportType.String(), report.Format.String(),
		report.Content, report.ContentType, report.Filename, report.SizeBytes,
		nullString(report.CalculationID), nullString(report.GraphID), nullString(report.UserID),
		report.GenerationTimeMs, report.Version,
		report.Tags, report.CustomFields,
		report.CreatedAt, report.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert report: %w", err)
	}

	return report, nil
}

// Get возвращает отчёт по ID
func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (*Report, error) {
	query := `
		SELECT
			id, title, description, author,
			report_type, format,
			content, content_type, filename, size_bytes,
			calculation_id, graph_id, user_id,
			generation_time_ms, version,
			tags, custom_fields,
			created_at, expires_at, deleted_at
		FROM reports
		WHERE id = $1 AND deleted_at IS NULL`

	row := r.db.QueryRow(ctx, query, id)
	return r.scanReport(row)
}

// GetContent возвращает только контент
func (r *PostgresRepository) GetContent(ctx context.Context, id uuid.UUID) ([]byte, error) {
	query := `SELECT content FROM reports WHERE id = $1 AND deleted_at IS NULL`

	var content []byte
	err := r.db.QueryRow(ctx, query, id).Scan(&content)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return content, nil
}

// List возвращает список отчётов
func (r *PostgresRepository) List(ctx context.Context, params *ListParams) (*ListResult, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.OrderBy == "" {
		params.OrderBy = "created_at"
	}

	conditions, args := r.buildListConditions(params)
	whereClause := strings.Join(conditions, " AND ")

	validOrderBy := map[string]bool{"created_at": true, "size_bytes": true, "title": true}
	if !validOrderBy[params.OrderBy] {
		params.OrderBy = "created_at"
	}

	orderDir := "ASC"
	if params.OrderDesc {
		orderDir = "DESC"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reports WHERE %s", whereClause)
	var totalCount int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count reports: %w", err)
	}

	argIdx := len(args) + 1
	query := fmt.Sprintf(`
		SELECT
			id, title, description, author, report_type, format,
			content_type, filename, size_bytes, calculation_id, graph_id, user_id,
			generation_time_ms, version, tags, custom_fields, created_at, expires_at
		FROM reports
		WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		whereClause, params.OrderBy, orderDir, argIdx, argIdx+1)

	args = append(args, params.Limit+1, params.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	defer rows.Close()

	var reports []*Report
	for rows.Next() {
		report, err := r.scanReportWithoutContent(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	hasMore := len(reports) > int(params.Limit)
	if hasMore {
		reports = reports[:params.Limit]
	}

	return &ListResult{Reports: reports, TotalCount: totalCount, HasMore: hasMore}, nil
}

// Delete мягко удаляет отчёт
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE reports SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// HardDelete физически удаляет отчёт
func (r *PostgresRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reports WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to hard delete report: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteExpired удаляет устаревшие отчёты
func (r *PostgresRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM reports WHERE expires_at < NOW() AND deleted_at IS NULL`
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired: %w", err)
	}
	return result.RowsAffected(), nil
}

// UpdateTags обновляет теги отчёта
func (r *PostgresRepository) UpdateTags(ctx context.Context, id uuid.UUID, tags []string, replace bool) ([]string, error) {
	var query string
	if replace {
		query = `UPDATE reports SET tags = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING tags`
	} else {
		query = `UPDATE reports SET tags = array_cat(tags, $2) WHERE id = $1 AND deleted_at IS NULL RETURNING tags`
	}

	var resultTags []string
	err := r.db.QueryRow(ctx, query, id, tags).Scan(&resultTags)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update tags: %w", err)
	}
	return resultTags, nil
}

// Stats возвращает статистику
func (r *PostgresRepository) Stats(ctx context.Context, userID string) (*Stats, error) {
	stats := &Stats{
		ReportsByType:   make(map[string]int64),
		ReportsByFormat: make(map[string]int64),
		SizeByType:      make(map[string]int64),
	}

	// Базовое условие
	whereClause := "deleted_at IS NULL"
	var args []any
	if userID != "" {
		whereClause += " AND user_id = $1"
		args = append(args, userID)
	}

	// Общая статистика
	query := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(size_bytes), 0),
			COALESCE(AVG(size_bytes), 0),
			MIN(created_at),
			MAX(created_at),
			COUNT(*) FILTER (WHERE expires_at < NOW())
		FROM reports
		WHERE %s`, whereClause)

	var oldest, newest sql.NullTime
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalReports,
		&stats.TotalSizeBytes,
		&stats.AvgSizeBytes,
		&oldest,
		&newest,
		&stats.ExpiredReports,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	if oldest.Valid {
		stats.OldestReportAt = &oldest.Time
	}
	if newest.Valid {
		stats.NewestReportAt = &newest.Time
	}

	// По типам
	typeQuery := fmt.Sprintf(`
		SELECT report_type, COUNT(*), COALESCE(SUM(size_bytes), 0)
		FROM reports
		WHERE %s
		GROUP BY report_type`, whereClause)

	rows, err := r.db.Query(ctx, typeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reportType string
		var count, size int64
		if err := rows.Scan(&reportType, &count, &size); err != nil {
			return nil, err
		}
		stats.ReportsByType[reportType] = count
		stats.SizeByType[reportType] = size
	}

	// По форматам
	formatQuery := fmt.Sprintf(`
		SELECT format, COUNT(*)
		FROM reports
		WHERE %s
		GROUP BY format`, whereClause)

	rows, err = r.db.Query(ctx, formatQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get format stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var format string
		var count int64
		if err := rows.Scan(&format, &count); err != nil {
			return nil, err
		}
		stats.ReportsByFormat[format] = count
	}

	return stats, nil
}

// Close закрывает пул
func (r *PostgresRepository) Close() error {
	r.db.Close()
	return nil
}

// Ping проверяет соединение
func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// === Вспомогательные методы ===

func (r *PostgresRepository) buildListConditions(params *ListParams) ([]string, []any) {
	conditions := []string{"deleted_at IS NULL"}
	var args []any
	argIdx := 1

	if params.ReportType != nil && *params.ReportType != reportv1.ReportType_REPORT_TYPE_UNSPECIFIED {
		conditions = append(conditions, fmt.Sprintf("report_type = $%d", argIdx))
		args = append(args, params.ReportType.String())
		argIdx++
	}

	if params.Format != nil && *params.Format != reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED {
		conditions = append(conditions, fmt.Sprintf("format = $%d", argIdx))
		args = append(args, params.Format.String())
		argIdx++
	}

	if params.CalculationID != "" {
		conditions = append(conditions, fmt.Sprintf("calculation_id = $%d", argIdx))
		args = append(args, params.CalculationID)
		argIdx++
	}

	if params.GraphID != "" {
		conditions = append(conditions, fmt.Sprintf("graph_id = $%d", argIdx))
		args = append(args, params.GraphID)
		argIdx++
	}

	if params.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, params.UserID)
		argIdx++
	}

	if len(params.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, pq.Array(params.Tags))
		argIdx++
	}

	if params.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, params.CreatedAfter)
		argIdx++
	}

	if params.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, params.CreatedBefore)
		// Не инкрементируем - последнее использование
	}

	return conditions, args
}

func (r *PostgresRepository) scanReport(row pgx.Row) (*Report, error) {
	var report Report
	var description, author sql.NullString
	var calculationID, graphID, userID sql.NullString
	var reportType, format string
	var expiresAt, deletedAt sql.NullTime
	var customFields map[string]string

	err := row.Scan(
		&report.ID, &report.Title, &description, &author,
		&reportType, &format,
		&report.Content, &report.ContentType, &report.Filename, &report.SizeBytes,
		&calculationID, &graphID, &userID,
		&report.GenerationTimeMs, &report.Version,
		&report.Tags, &customFields,
		&report.CreatedAt, &expiresAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan report: %w", err)
	}

	report.Description = description.String
	report.Author = author.String
	report.CalculationID = calculationID.String
	report.GraphID = graphID.String
	report.UserID = userID.String
	report.ReportType = parseReportType(reportType)
	report.Format = parseReportFormat(format)
	report.CustomFields = customFields

	if expiresAt.Valid {
		report.ExpiresAt = &expiresAt.Time
	}
	if deletedAt.Valid {
		report.DeletedAt = &deletedAt.Time
	}

	return &report, nil
}

func (r *PostgresRepository) scanReportWithoutContent(rows pgx.Rows) (*Report, error) {
	var report Report
	var description, author sql.NullString
	var calculationID, graphID, userID sql.NullString
	var reportType, format string
	var expiresAt sql.NullTime
	var customFields map[string]string

	err := rows.Scan(
		&report.ID, &report.Title, &description, &author,
		&reportType, &format,
		&report.ContentType, &report.Filename, &report.SizeBytes,
		&calculationID, &graphID, &userID,
		&report.GenerationTimeMs, &report.Version,
		&report.Tags, &customFields,
		&report.CreatedAt, &expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan report: %w", err)
	}

	report.Description = description.String
	report.Author = author.String
	report.CalculationID = calculationID.String
	report.GraphID = graphID.String
	report.UserID = userID.String
	report.ReportType = parseReportType(reportType)
	report.Format = parseReportFormat(format)
	report.CustomFields = customFields

	if expiresAt.Valid {
		report.ExpiresAt = &expiresAt.Time
	}

	return &report, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func parseReportType(s string) reportv1.ReportType {
	if v, ok := reportv1.ReportType_value[s]; ok {
		return reportv1.ReportType(v)
	}
	return reportv1.ReportType_REPORT_TYPE_UNSPECIFIED
}

func parseReportFormat(s string) reportv1.ReportFormat {
	if v, ok := reportv1.ReportFormat_value[s]; ok {
		return reportv1.ReportFormat(v)
	}
	return reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED
}
