package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"logistics/pkg/database"
	"logistics/pkg/telemetry"
)

// PostgresCalculationRepository PostgreSQL реализация
type PostgresCalculationRepository struct {
	db database.DB
}

// NewPostgresCalculationRepository создаёт новый репозиторий
func NewPostgresCalculationRepository(db database.DB) *PostgresCalculationRepository {
	return &PostgresCalculationRepository{db: db}
}

func (r *PostgresCalculationRepository) Create(ctx context.Context, calc *Calculation) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.Create")
	defer span.End()

	query := `
		INSERT INTO calculations (
			user_id, name, algorithm, max_flow, total_cost,
			computation_time_ms, node_count, edge_count,
			request_data, response_data, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		calc.UserID,
		calc.Name,
		calc.Algorithm,
		calc.MaxFlow,
		calc.TotalCost,
		calc.ComputationTimeMs,
		calc.NodeCount,
		calc.EdgeCount,
		calc.RequestData,
		calc.ResponseData,
		calc.Tags,
	).Scan(&calc.ID, &calc.CreatedAt, &calc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create calculation: %w", err)
	}

	return nil
}

func (r *PostgresCalculationRepository) GetByID(ctx context.Context, id string) (*Calculation, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.GetByID")
	defer span.End()

	query := `
		SELECT
			id, user_id, name, algorithm, max_flow, total_cost,
			computation_time_ms, node_count, edge_count,
			request_data, response_data, tags, created_at, updated_at
		FROM calculations
		WHERE id = $1
	`

	calc := &Calculation{}
	var tags pgtype.Array[string]

	err := r.db.QueryRow(ctx, query, id).Scan(
		&calc.ID,
		&calc.UserID,
		&calc.Name,
		&calc.Algorithm,
		&calc.MaxFlow,
		&calc.TotalCost,
		&calc.ComputationTimeMs,
		&calc.NodeCount,
		&calc.EdgeCount,
		&calc.RequestData,
		&calc.ResponseData,
		&tags,
		&calc.CreatedAt,
		&calc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCalculationNotFound
		}
		return nil, fmt.Errorf("failed to get calculation: %w", err)
	}

	calc.Tags = tags.Elements

	return calc, nil
}

func (r *PostgresCalculationRepository) Delete(ctx context.Context, id string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.Delete")
	defer span.End()

	query := `DELETE FROM calculations WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete calculation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCalculationNotFound
	}

	return nil
}

func (r *PostgresCalculationRepository) List(
	ctx context.Context,
	userID string,
	opts *ListOptions,
) ([]*CalculationSummary, int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.List")
	defer span.End()

	if opts == nil {
		opts = &ListOptions{Limit: 20, Offset: 0, Sort: SortByCreatedDesc}
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Строим WHERE clause
	where, args := r.buildWhereClause(userID, opts.Filter)

	// Получаем общее количество
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM calculations WHERE %s`, where)
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count calculations: %w", err)
	}

	// Получаем записи
	orderBy := r.buildOrderBy(opts.Sort)

	selectQuery := fmt.Sprintf(`
		SELECT
			id, name, algorithm, max_flow, total_cost,
			computation_time_ms, node_count, edge_count, tags, created_at
		FROM calculations
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, len(args)+1, len(args)+2)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list calculations: %w", err)
	}
	defer rows.Close()

	var results []*CalculationSummary
	for rows.Next() {
		summary := &CalculationSummary{}
		var tags pgtype.Array[string]

		err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.Algorithm,
			&summary.MaxFlow,
			&summary.TotalCost,
			&summary.ComputationTimeMs,
			&summary.NodeCount,
			&summary.EdgeCount,
			&tags,
			&summary.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan calculation: %w", err)
		}

		summary.Tags = tags.Elements
		results = append(results, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return results, total, nil
}

func (r *PostgresCalculationRepository) buildWhereClause(userID string, filter *ListFilter) (string, []any) {
	conditions := []string{"user_id = $1"}
	args := []any{userID}
	argNum := 2

	if filter != nil {
		if filter.Algorithm != "" {
			conditions = append(conditions, fmt.Sprintf("algorithm = $%d", argNum))
			args = append(args, filter.Algorithm)
			argNum++
		}

		if len(filter.Tags) > 0 {
			conditions = append(conditions, fmt.Sprintf("tags && $%d", argNum))
			args = append(args, filter.Tags)
			argNum++
		}

		if filter.MinFlow != nil {
			conditions = append(conditions, fmt.Sprintf("max_flow >= $%d", argNum))
			args = append(args, *filter.MinFlow)
			argNum++
		}

		if filter.MaxFlow != nil {
			conditions = append(conditions, fmt.Sprintf("max_flow <= $%d", argNum))
			args = append(args, *filter.MaxFlow)
			argNum++
		}

		if filter.StartTime != nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
			args = append(args, *filter.StartTime)
			argNum++
		}

		if filter.EndTime != nil {
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argNum))
			args = append(args, *filter.EndTime)
		}
	}

	return strings.Join(conditions, " AND "), args
}

func (r *PostgresCalculationRepository) buildOrderBy(sort SortOrder) string {
	switch sort {
	case SortByCreatedAsc:
		return "created_at ASC"
	case SortByMaxFlowDesc:
		return "max_flow DESC"
	case SortByTotalCostDesc:
		return "total_cost DESC"
	default:
		return "created_at DESC"
	}
}

func (r *PostgresCalculationRepository) GetUserStatistics(
	ctx context.Context,
	userID string,
	startTime, endTime *time.Time,
) (*UserStatistics, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.GetUserStatistics")
	defer span.End()

	stats := &UserStatistics{
		CalculationsByAlgorithm: make(map[string]int),
		DailyStats:              []DailyStats{},
	}

	// Базовые условия
	where := "user_id = $1"
	args := []any{userID}
	argNum := 2

	if startTime != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, *startTime)
		argNum++
	}
	if endTime != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, *endTime)
	}

	// Общая статистика
	statsQuery := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(AVG(max_flow), 0),
			COALESCE(AVG(total_cost), 0),
			COALESCE(AVG(computation_time_ms), 0)
		FROM calculations
		WHERE %s
	`, where)

	err := r.db.QueryRow(ctx, statsQuery, args...).Scan(
		&stats.TotalCalculations,
		&stats.AverageMaxFlow,
		&stats.AverageTotalCost,
		&stats.AverageComputationTimeMs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Статистика по алгоритмам
	algoQuery := fmt.Sprintf(`
		SELECT algorithm, COUNT(*)
		FROM calculations
		WHERE %s
		GROUP BY algorithm
	`, where)

	algoRows, err := r.db.Query(ctx, algoQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm stats: %w", err)
	}
	defer algoRows.Close()

	for algoRows.Next() {
		var algorithm string
		var count int
		if err := algoRows.Scan(&algorithm, &count); err != nil {
			return nil, fmt.Errorf("failed to scan algorithm stats: %w", err)
		}
		stats.CalculationsByAlgorithm[algorithm] = count
	}

	// Дневная статистика
	dailyQuery := fmt.Sprintf(`
		SELECT
			DATE(created_at) as date,
			COUNT(*) as count,
			COALESCE(SUM(max_flow), 0) as total_flow
		FROM calculations
		WHERE %s
		GROUP BY DATE(created_at)
		ORDER BY date DESC
		LIMIT 30
	`, where)

	dailyRows, err := r.db.Query(ctx, dailyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}
	defer dailyRows.Close()

	for dailyRows.Next() {
		var ds DailyStats
		var date time.Time
		if err := dailyRows.Scan(&date, &ds.Count, &ds.TotalFlow); err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}
		ds.Date = date.Format("2006-01-02")
		stats.DailyStats = append(stats.DailyStats, ds)
	}

	return stats, nil
}

func (r *PostgresCalculationRepository) Search(
	ctx context.Context,
	userID string,
	query string,
	limit int,
) ([]*CalculationSummary, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.Search")
	defer span.End()

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	searchQuery := `
		SELECT
			id, name, algorithm, max_flow, total_cost,
			computation_time_ms, node_count, edge_count, tags, created_at
		FROM calculations
		WHERE user_id = $1
			AND to_tsvector('russian', name) @@ plainto_tsquery('russian', $2)
		ORDER BY ts_rank(to_tsvector('russian', name), plainto_tsquery('russian', $2)) DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, searchQuery, userID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search calculations: %w", err)
	}
	defer rows.Close()

	var results []*CalculationSummary
	for rows.Next() {
		summary := &CalculationSummary{}
		var tags pgtype.Array[string]

		err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.Algorithm,
			&summary.MaxFlow,
			&summary.TotalCost,
			&summary.ComputationTimeMs,
			&summary.NodeCount,
			&summary.EdgeCount,
			&tags,
			&summary.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan calculation: %w", err)
		}

		summary.Tags = tags.Elements
		results = append(results, summary)
	}

	return results, nil
}

// DeleteByUserID удаляет все расчёты пользователя
func (r *PostgresCalculationRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.DeleteByUserID")
	defer span.End()

	query := `DELETE FROM calculations WHERE user_id = $1`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user calculations: %w", err)
	}

	return result.RowsAffected(), nil
}

// GetByUserAndID получает расчёт с проверкой владельца
func (r *PostgresCalculationRepository) GetByUserAndID(ctx context.Context, userID, id string) (*Calculation, error) {
	calc, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if calc.UserID != userID {
		return nil, ErrAccessDenied
	}

	return calc, nil
}

// DeleteByUserAndID удаляет расчёт с проверкой владельца
func (r *PostgresCalculationRepository) DeleteByUserAndID(ctx context.Context, userID, id string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresCalculationRepository.DeleteByUserAndID")
	defer span.End()

	query := `DELETE FROM calculations WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete calculation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCalculationNotFound
	}

	return nil
}
