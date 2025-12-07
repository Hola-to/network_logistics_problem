// services/simulation-svc/internal/repository/postgres.go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"logistics/pkg/database"
	"logistics/pkg/telemetry"
)

// PostgresSimulationRepository PostgreSQL реализация
type PostgresSimulationRepository struct {
	db database.DB
}

// NewPostgresSimulationRepository создаёт новый репозиторий
func NewPostgresSimulationRepository(db database.DB) *PostgresSimulationRepository {
	return &PostgresSimulationRepository{db: db}
}

func (r *PostgresSimulationRepository) Create(ctx context.Context, sim *Simulation) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresSimulationRepository.Create")
	defer span.End()

	query := `
		INSERT INTO simulations (
			user_id, name, description, simulation_type,
			node_count, edge_count, computation_time_ms,
			baseline_flow, result_flow, flow_change_percent,
			graph_data, request_data, response_data, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		sim.UserID,
		sim.Name,
		sim.Description,
		sim.SimulationType,
		sim.NodeCount,
		sim.EdgeCount,
		sim.ComputationTimeMs,
		sim.BaselineFlow,
		sim.ResultFlow,
		sim.FlowChangePercent,
		sim.GraphData,
		sim.RequestData,
		sim.ResponseData,
		sim.Tags,
	).Scan(&sim.ID, &sim.CreatedAt, &sim.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create simulation: %w", err)
	}

	return nil
}

func (r *PostgresSimulationRepository) GetByID(ctx context.Context, id string) (*Simulation, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresSimulationRepository.GetByID")
	defer span.End()

	query := `
		SELECT
			id, user_id, name, description, simulation_type,
			node_count, edge_count, computation_time_ms,
			baseline_flow, result_flow, flow_change_percent,
			graph_data, request_data, response_data, tags,
			created_at, updated_at
		FROM simulations
		WHERE id = $1
	`

	sim := &Simulation{}
	var (
		description       pgtype.Text
		baselineFlow      pgtype.Float8
		resultFlow        pgtype.Float8
		flowChangePercent pgtype.Float8
		graphData         []byte
		tags              pgtype.Array[string]
	)

	err := r.db.QueryRow(ctx, query, id).Scan(
		&sim.ID,
		&sim.UserID,
		&sim.Name,
		&description,
		&sim.SimulationType,
		&sim.NodeCount,
		&sim.EdgeCount,
		&sim.ComputationTimeMs,
		&baselineFlow,
		&resultFlow,
		&flowChangePercent,
		&graphData,
		&sim.RequestData,
		&sim.ResponseData,
		&tags,
		&sim.CreatedAt,
		&sim.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSimulationNotFound
		}
		return nil, fmt.Errorf("failed to get simulation: %w", err)
	}

	sim.Description = description.String
	sim.GraphData = graphData
	sim.Tags = tags.Elements

	if baselineFlow.Valid {
		sim.BaselineFlow = &baselineFlow.Float64
	}
	if resultFlow.Valid {
		sim.ResultFlow = &resultFlow.Float64
	}
	if flowChangePercent.Valid {
		sim.FlowChangePercent = &flowChangePercent.Float64
	}

	return sim, nil
}

func (r *PostgresSimulationRepository) Delete(ctx context.Context, id string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresSimulationRepository.Delete")
	defer span.End()

	query := `DELETE FROM simulations WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete simulation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSimulationNotFound
	}

	return nil
}

func (r *PostgresSimulationRepository) List(
	ctx context.Context,
	userID string,
	simType string,
	opts *ListOptions,
) ([]*SimulationSummary, int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresSimulationRepository.List")
	defer span.End()

	if opts == nil {
		opts = &ListOptions{Limit: 20, Offset: 0}
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Строим WHERE
	where := "user_id = $1"
	args := []any{userID}
	argNum := 2

	if simType != "" {
		where += fmt.Sprintf(" AND simulation_type = $%d", argNum)
		args = append(args, simType)
		argNum++
	}

	// Подсчёт
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM simulations WHERE %s", where)
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count simulations: %w", err)
	}

	// Данные
	selectQuery := fmt.Sprintf(`
		SELECT id, name, simulation_type, tags, created_at
		FROM simulations
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argNum, argNum+1)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list simulations: %w", err)
	}
	defer rows.Close()

	var results []*SimulationSummary
	for rows.Next() {
		summary := &SimulationSummary{}
		var tags pgtype.Array[string]

		err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.SimulationType,
			&tags,
			&summary.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan simulation: %w", err)
		}

		summary.Tags = tags.Elements
		results = append(results, summary)
	}

	return results, total, nil
}

func (r *PostgresSimulationRepository) ListByUser(
	ctx context.Context,
	userID string,
	opts *ListOptions,
) ([]*SimulationSummary, int64, error) {
	return r.List(ctx, userID, "", opts)
}

func (r *PostgresSimulationRepository) GetByUserAndID(
	ctx context.Context,
	userID, id string,
) (*Simulation, error) {
	sim, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if sim.UserID != userID {
		return nil, ErrAccessDenied
	}

	return sim, nil
}
