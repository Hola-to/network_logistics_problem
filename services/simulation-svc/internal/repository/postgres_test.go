package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// MOCK DB ADAPTER
// ============================================================

type pgxMockAdapter struct {
	mock pgxmock.PgxPoolIface
}

func (a *pgxMockAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return a.mock.Exec(ctx, sql, args...)
}

func (a *pgxMockAdapter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return a.mock.Query(ctx, sql, args...)
}

func (a *pgxMockAdapter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return a.mock.QueryRow(ctx, sql, args...)
}

func (a *pgxMockAdapter) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return a.mock.BeginTx(ctx, txOptions)
}

func (a *pgxMockAdapter) Close() {
	a.mock.Close()
}

func (a *pgxMockAdapter) Ping(ctx context.Context) error {
	return a.mock.Ping(ctx)
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func setupMockDB(t *testing.T) (pgxmock.PgxPoolIface, *PostgresSimulationRepository) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	adapter := &pgxMockAdapter{mock: mock}
	repo := NewPostgresSimulationRepository(adapter)

	return mock, repo
}

// createTagsArray создаёт pgtype.Array[string] для тестов
func createTagsArray(tags []string) pgtype.Array[string] {
	if tags == nil {
		return pgtype.Array[string]{Valid: false}
	}
	return pgtype.Array[string]{
		Elements: tags,
		Valid:    true,
		Dims:     []pgtype.ArrayDimension{{Length: int32(len(tags)), LowerBound: 1}},
	}
}

// ============================================================
// CREATE TESTS
// ============================================================

func TestPostgresSimulationRepository_Create_Success(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	sim := &Simulation{
		UserID:            "user-123",
		Name:              "Test Simulation",
		Description:       "Test description",
		SimulationType:    "WHAT_IF",
		NodeCount:         10,
		EdgeCount:         15,
		ComputationTimeMs: 123.45,
		GraphData:         []byte(`{"test": true}`),
		RequestData:       []byte(`{"request": "data"}`),
		ResponseData:      []byte(`{"response": "data"}`),
		Tags:              []string{"env:test"},
	}

	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("sim-123", now, now)

	mock.ExpectQuery(`INSERT INTO simulations`).
		WithArgs(
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
		).
		WillReturnRows(rows)

	err := repo.Create(ctx, sim)

	require.NoError(t, err)
	assert.Equal(t, "sim-123", sim.ID)
	assert.Equal(t, now, sim.CreatedAt)
	assert.Equal(t, now, sim.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_Create_WithOptionalFields(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()
	baselineFlow := 100.0
	resultFlow := 120.0
	flowChange := 20.0

	sim := &Simulation{
		UserID:            "user-123",
		Name:              "Test Simulation",
		SimulationType:    "WHAT_IF",
		BaselineFlow:      &baselineFlow,
		ResultFlow:        &resultFlow,
		FlowChangePercent: &flowChange,
	}

	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("sim-456", now, now)

	mock.ExpectQuery(`INSERT INTO simulations`).
		WithArgs(
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
		).
		WillReturnRows(rows)

	err := repo.Create(ctx, sim)

	require.NoError(t, err)
	assert.Equal(t, "sim-456", sim.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_Create_Error(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	sim := &Simulation{
		UserID:         "user-123",
		Name:           "Test",
		SimulationType: "WHAT_IF",
	}

	mock.ExpectQuery(`INSERT INTO simulations`).
		WithArgs(
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
		).
		WillReturnError(errors.New("database error"))

	err := repo.Create(ctx, sim)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create simulation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// GET BY ID TESTS
// ============================================================

func TestPostgresSimulationRepository_GetByID_Success(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	// Используем pgtype для корректного сканирования
	description := pgtype.Text{String: "Description", Valid: true}
	baselineFlow := pgtype.Float8{Float64: 100.0, Valid: true}
	resultFlow := pgtype.Float8{Float64: 120.0, Valid: true}
	flowChangePercent := pgtype.Float8{Float64: 20.0, Valid: true}
	tags := createTagsArray([]string{"env:test"})

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "name", "description", "simulation_type",
		"node_count", "edge_count", "computation_time_ms",
		"baseline_flow", "result_flow", "flow_change_percent",
		"graph_data", "request_data", "response_data", "tags",
		"created_at", "updated_at",
	}).AddRow(
		"sim-123", "user-123", "Test Sim", description, "WHAT_IF",
		10, 15, 100.5,
		baselineFlow, resultFlow, flowChangePercent,
		[]byte(`{}`), []byte(`{}`), []byte(`{}`), tags,
		now, now,
	)

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnRows(rows)

	sim, err := repo.GetByID(ctx, "sim-123")

	require.NoError(t, err)
	assert.Equal(t, "sim-123", sim.ID)
	assert.Equal(t, "user-123", sim.UserID)
	assert.Equal(t, "Test Sim", sim.Name)
	assert.Equal(t, "Description", sim.Description)
	assert.Equal(t, "WHAT_IF", sim.SimulationType)
	assert.Equal(t, 10, sim.NodeCount)
	assert.Equal(t, 15, sim.EdgeCount)
	assert.NotNil(t, sim.BaselineFlow)
	assert.Equal(t, 100.0, *sim.BaselineFlow)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_GetByID_NotFound(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	sim, err := repo.GetByID(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, sim)
	assert.Equal(t, ErrSimulationNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_GetByID_DatabaseError(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnError(errors.New("connection lost"))

	sim, err := repo.GetByID(ctx, "sim-123")

	assert.Error(t, err)
	assert.Nil(t, sim)
	assert.Contains(t, err.Error(), "failed to get simulation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_GetByID_NullOptionalFields(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	// NULL значения
	description := pgtype.Text{Valid: false}
	baselineFlow := pgtype.Float8{Valid: false}
	resultFlow := pgtype.Float8{Valid: false}
	flowChangePercent := pgtype.Float8{Valid: false}
	tags := pgtype.Array[string]{Valid: false}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "name", "description", "simulation_type",
		"node_count", "edge_count", "computation_time_ms",
		"baseline_flow", "result_flow", "flow_change_percent",
		"graph_data", "request_data", "response_data", "tags",
		"created_at", "updated_at",
	}).AddRow(
		"sim-123", "user-123", "Test Sim", description, "WHAT_IF",
		10, 15, 100.5,
		baselineFlow, resultFlow, flowChangePercent,
		nil, []byte(`{}`), []byte(`{}`), tags,
		now, now,
	)

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnRows(rows)

	sim, err := repo.GetByID(ctx, "sim-123")

	require.NoError(t, err)
	assert.Equal(t, "sim-123", sim.ID)
	assert.Empty(t, sim.Description)
	assert.Nil(t, sim.BaselineFlow)
	assert.Nil(t, sim.ResultFlow)
	assert.Nil(t, sim.FlowChangePercent)
	assert.Nil(t, sim.GraphData)
	assert.Empty(t, sim.Tags)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// DELETE TESTS
// ============================================================

func TestPostgresSimulationRepository_Delete_Success(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(ctx, "sim-123")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_Delete_NotFound(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM simulations WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.Delete(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, ErrSimulationNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_Delete_Error(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnError(errors.New("database error"))

	err := repo.Delete(ctx, "sim-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete simulation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// LIST TESTS
// ============================================================

func TestPostgresSimulationRepository_List_Success(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	// Count query
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(2))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	// Select query с pgtype.Array
	tags1 := createTagsArray([]string{"env:test"})
	tags2 := createTagsArray([]string{"env:prod"})

	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"}).
		AddRow("sim-1", "Sim 1", "WHAT_IF", tags1, now).
		AddRow("sim-2", "Sim 2", "MONTE_CARLO", tags2, now)

	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 20, 0).
		WillReturnRows(selectRows)

	opts := &ListOptions{Limit: 20, Offset: 0}
	sims, total, err := repo.List(ctx, "user-123", "", opts)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, sims, 2)
	assert.Equal(t, "sim-1", sims[0].ID)
	assert.Equal(t, "sim-2", sims[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_List_WithTypeFilter(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	// Count query with type filter
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1 AND simulation_type = \$2`).
		WithArgs("user-123", "WHAT_IF").
		WillReturnRows(countRows)

	// Select query with type filter
	tags := createTagsArray([]string{})
	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"}).
		AddRow("sim-1", "Sim 1", "WHAT_IF", tags, now)

	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", "WHAT_IF", 20, 0).
		WillReturnRows(selectRows)

	opts := &ListOptions{Limit: 20, Offset: 0}
	sims, total, err := repo.List(ctx, "user-123", "WHAT_IF", opts)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, sims, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_List_DefaultOptions(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"})
	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 20, 0).
		WillReturnRows(selectRows)

	sims, total, err := repo.List(ctx, "user-123", "", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, sims)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_List_LimitCapped(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"})
	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 100, 0).
		WillReturnRows(selectRows)

	opts := &ListOptions{Limit: 500, Offset: 0}
	_, _, err := repo.List(ctx, "user-123", "", opts)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_List_CountError(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnError(errors.New("count error"))

	opts := &ListOptions{Limit: 20, Offset: 0}
	sims, total, err := repo.List(ctx, "user-123", "", opts)

	assert.Error(t, err)
	assert.Nil(t, sims)
	assert.Equal(t, int64(0), total)
	assert.Contains(t, err.Error(), "failed to count simulations")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_List_SelectError(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(5))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 20, 0).
		WillReturnError(errors.New("select error"))

	opts := &ListOptions{Limit: 20, Offset: 0}
	sims, total, err := repo.List(ctx, "user-123", "", opts)

	assert.Error(t, err)
	assert.Nil(t, sims)
	assert.Equal(t, int64(0), total)
	assert.Contains(t, err.Error(), "failed to list simulations")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// LIST BY USER TESTS
// ============================================================

func TestPostgresSimulationRepository_ListByUser(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	tags := createTagsArray([]string{})
	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"}).
		AddRow("sim-1", "Sim 1", "WHAT_IF", tags, now)
	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 20, 0).
		WillReturnRows(selectRows)

	opts := &ListOptions{Limit: 20, Offset: 0}
	sims, total, err := repo.ListByUser(ctx, "user-123", opts)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, sims, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// GET BY USER AND ID TESTS
// ============================================================

func TestPostgresSimulationRepository_GetByUserAndID_Success(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	description := pgtype.Text{String: "Desc", Valid: true}
	baselineFlow := pgtype.Float8{Float64: 100.0, Valid: true}
	resultFlow := pgtype.Float8{Float64: 120.0, Valid: true}
	flowChangePercent := pgtype.Float8{Float64: 20.0, Valid: true}
	tags := createTagsArray([]string{})

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "name", "description", "simulation_type",
		"node_count", "edge_count", "computation_time_ms",
		"baseline_flow", "result_flow", "flow_change_percent",
		"graph_data", "request_data", "response_data", "tags",
		"created_at", "updated_at",
	}).AddRow(
		"sim-123", "user-123", "Test Sim", description, "WHAT_IF",
		10, 15, 100.5,
		baselineFlow, resultFlow, flowChangePercent,
		[]byte(`{}`), []byte(`{}`), []byte(`{}`), tags,
		now, now,
	)

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnRows(rows)

	sim, err := repo.GetByUserAndID(ctx, "user-123", "sim-123")

	require.NoError(t, err)
	assert.Equal(t, "sim-123", sim.ID)
	assert.Equal(t, "user-123", sim.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_GetByUserAndID_NotFound(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	sim, err := repo.GetByUserAndID(ctx, "user-123", "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, sim)
	assert.Equal(t, ErrSimulationNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_GetByUserAndID_AccessDenied(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()
	now := time.Now()

	description := pgtype.Text{String: "Desc", Valid: true}
	baselineFlow := pgtype.Float8{Valid: false}
	resultFlow := pgtype.Float8{Valid: false}
	flowChangePercent := pgtype.Float8{Valid: false}
	tags := createTagsArray([]string{})

	// Simulation exists but belongs to different user
	rows := pgxmock.NewRows([]string{
		"id", "user_id", "name", "description", "simulation_type",
		"node_count", "edge_count", "computation_time_ms",
		"baseline_flow", "result_flow", "flow_change_percent",
		"graph_data", "request_data", "response_data", "tags",
		"created_at", "updated_at",
	}).AddRow(
		"sim-123", "other-user", "Test Sim", description, "WHAT_IF",
		10, 15, 100.5,
		baselineFlow, resultFlow, flowChangePercent,
		nil, []byte(`{}`), []byte(`{}`), tags,
		now, now,
	)

	mock.ExpectQuery(`SELECT .* FROM simulations WHERE id = \$1`).
		WithArgs("sim-123").
		WillReturnRows(rows)

	sim, err := repo.GetByUserAndID(ctx, "user-123", "sim-123")

	assert.Error(t, err)
	assert.Nil(t, sim)
	assert.Equal(t, ErrAccessDenied, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// CONSTRUCTOR TEST
// ============================================================

func TestNewPostgresSimulationRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	adapter := &pgxMockAdapter{mock: mock}
	repo := NewPostgresSimulationRepository(adapter)

	assert.NotNil(t, repo)
}

// ============================================================
// PAGINATION EDGE CASES
// ============================================================

func TestPostgresSimulationRepository_List_NegativeLimit(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"})
	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 20, 0).
		WillReturnRows(selectRows)

	opts := &ListOptions{Limit: -5, Offset: 0}
	_, _, err := repo.List(ctx, "user-123", "", opts)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSimulationRepository_List_LargeOffset(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx := context.Background()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(100))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM simulations WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(countRows)

	selectRows := pgxmock.NewRows([]string{"id", "name", "simulation_type", "tags", "created_at"})
	mock.ExpectQuery(`SELECT id, name, simulation_type, tags, created_at FROM simulations`).
		WithArgs("user-123", 20, 1000).
		WillReturnRows(selectRows)

	opts := &ListOptions{Limit: 20, Offset: 1000}
	sims, total, err := repo.List(ctx, "user-123", "", opts)

	require.NoError(t, err)
	assert.Equal(t, int64(100), total)
	assert.Empty(t, sims)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================
// CONTEXT CANCELLATION TEST
// ============================================================

func TestPostgresSimulationRepository_Create_ContextCancelled(t *testing.T) {
	mock, repo := setupMockDB(t)
	defer mock.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sim := &Simulation{
		UserID:         "user-123",
		Name:           "Test",
		SimulationType: "WHAT_IF",
	}

	mock.ExpectQuery(`INSERT INTO simulations`).
		WithArgs(
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
		).
		WillReturnError(context.Canceled)

	err := repo.Create(ctx, sim)

	assert.Error(t, err)
}
