package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// MOCK REPOSITORY FOR INTERFACE TESTING
// ============================================================

type MockSimulationRepository struct {
	mock.Mock
}

func (m *MockSimulationRepository) Create(ctx context.Context, sim *Simulation) error {
	args := m.Called(ctx, sim)
	return args.Error(0)
}

func (m *MockSimulationRepository) GetByID(ctx context.Context, id string) (*Simulation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Simulation), args.Error(1)
}

func (m *MockSimulationRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSimulationRepository) List(ctx context.Context, userID string, simType string, opts *ListOptions) ([]*SimulationSummary, int64, error) {
	args := m.Called(ctx, userID, simType, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*SimulationSummary), args.Get(1).(int64), args.Error(2)
}

func (m *MockSimulationRepository) ListByUser(ctx context.Context, userID string, opts *ListOptions) ([]*SimulationSummary, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*SimulationSummary), args.Get(1).(int64), args.Error(2)
}

func (m *MockSimulationRepository) GetByUserAndID(ctx context.Context, userID, id string) (*Simulation, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Simulation), args.Error(1)
}

// ============================================================
// ERROR TESTS
// ============================================================

func TestErrors(t *testing.T) {
	assert.Equal(t, "simulation not found", ErrSimulationNotFound.Error())
	assert.Equal(t, "access denied", ErrAccessDenied.Error())
}

// ============================================================
// MODEL TESTS
// ============================================================

func TestSimulationModel(t *testing.T) {
	now := time.Now()
	baselineFlow := 100.0
	resultFlow := 120.0
	flowChangePercent := 20.0

	sim := &Simulation{
		ID:                "test-id",
		UserID:            "user-123",
		Name:              "Test Simulation",
		Description:       "Test description",
		SimulationType:    "WHAT_IF",
		NodeCount:         10,
		EdgeCount:         15,
		ComputationTimeMs: 123.45,
		BaselineFlow:      &baselineFlow,
		ResultFlow:        &resultFlow,
		FlowChangePercent: &flowChangePercent,
		GraphData:         []byte(`{"nodes": []}`),
		RequestData:       []byte(`{"test": "request"}`),
		ResponseData:      []byte(`{"test": "response"}`),
		Tags:              []string{"env:test", "version:1"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	assert.Equal(t, "test-id", sim.ID)
	assert.Equal(t, "user-123", sim.UserID)
	assert.Equal(t, "Test Simulation", sim.Name)
	assert.Equal(t, "Test description", sim.Description)
	assert.Equal(t, "WHAT_IF", sim.SimulationType)
	assert.Equal(t, 10, sim.NodeCount)
	assert.Equal(t, 15, sim.EdgeCount)
	assert.Equal(t, 123.45, sim.ComputationTimeMs)
	assert.Equal(t, 100.0, *sim.BaselineFlow)
	assert.Equal(t, 120.0, *sim.ResultFlow)
	assert.Equal(t, 20.0, *sim.FlowChangePercent)
	assert.Equal(t, []byte(`{"nodes": []}`), sim.GraphData)
	assert.Len(t, sim.Tags, 2)
}

func TestSimulationSummaryModel(t *testing.T) {
	now := time.Now()

	summary := &SimulationSummary{
		ID:             "test-id",
		Name:           "Test Simulation",
		SimulationType: "MONTE_CARLO",
		CreatedAt:      now,
		Tags:           []string{"env:prod"},
	}

	assert.Equal(t, "test-id", summary.ID)
	assert.Equal(t, "Test Simulation", summary.Name)
	assert.Equal(t, "MONTE_CARLO", summary.SimulationType)
	assert.Equal(t, now, summary.CreatedAt)
	assert.Len(t, summary.Tags, 1)
}

func TestListOptions(t *testing.T) {
	opts := &ListOptions{
		Limit:  20,
		Offset: 40,
	}

	assert.Equal(t, 20, opts.Limit)
	assert.Equal(t, 40, opts.Offset)
}

// ============================================================
// INTERFACE IMPLEMENTATION TESTS
// ============================================================

func TestMockRepository_Create(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	sim := &Simulation{
		UserID:         "user-123",
		Name:           "Test",
		SimulationType: "WHAT_IF",
	}

	mockRepo.On("Create", ctx, sim).Return(nil)

	err := mockRepo.Create(ctx, sim)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	expectedSim := &Simulation{
		ID:     "sim-123",
		UserID: "user-123",
		Name:   "Test",
	}

	mockRepo.On("GetByID", ctx, "sim-123").Return(expectedSim, nil)

	sim, err := mockRepo.GetByID(ctx, "sim-123")
	assert.NoError(t, err)
	assert.Equal(t, expectedSim, sim)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, ErrSimulationNotFound)

	sim, err := mockRepo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, sim)
	assert.Equal(t, ErrSimulationNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_Delete(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	mockRepo.On("Delete", ctx, "sim-123").Return(nil)

	err := mockRepo.Delete(ctx, "sim-123")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	mockRepo.On("Delete", ctx, "nonexistent").Return(ErrSimulationNotFound)

	err := mockRepo.Delete(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrSimulationNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_List(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	expectedSummaries := []*SimulationSummary{
		{ID: "sim-1", Name: "Sim 1"},
		{ID: "sim-2", Name: "Sim 2"},
	}
	opts := &ListOptions{Limit: 10, Offset: 0}

	mockRepo.On("List", ctx, "user-123", "WHAT_IF", opts).
		Return(expectedSummaries, int64(2), nil)

	summaries, total, err := mockRepo.List(ctx, "user-123", "WHAT_IF", opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, summaries, 2)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_ListByUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	expectedSummaries := []*SimulationSummary{
		{ID: "sim-1", Name: "Sim 1"},
	}
	opts := &ListOptions{Limit: 5, Offset: 0}

	mockRepo.On("ListByUser", ctx, "user-123", opts).
		Return(expectedSummaries, int64(1), nil)

	summaries, total, err := mockRepo.ListByUser(ctx, "user-123", opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, summaries, 1)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetByUserAndID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	expectedSim := &Simulation{
		ID:     "sim-123",
		UserID: "user-123",
		Name:   "Test",
	}

	mockRepo.On("GetByUserAndID", ctx, "user-123", "sim-123").Return(expectedSim, nil)

	sim, err := mockRepo.GetByUserAndID(ctx, "user-123", "sim-123")
	assert.NoError(t, err)
	assert.Equal(t, expectedSim, sim)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetByUserAndID_AccessDenied(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSimulationRepository)

	mockRepo.On("GetByUserAndID", ctx, "wrong-user", "sim-123").
		Return(nil, ErrAccessDenied)

	sim, err := mockRepo.GetByUserAndID(ctx, "wrong-user", "sim-123")
	assert.Error(t, err)
	assert.Nil(t, sim)
	assert.Equal(t, ErrAccessDenied, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================
// NIL HANDLING TESTS
// ============================================================

// normalizeListOptions нормализует опции списка
func normalizeListOptions(opts *ListOptions) *ListOptions {
	if opts == nil {
		return &ListOptions{Limit: 20, Offset: 0}
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}
	return opts
}

func TestNormalizeListOptions_Nil(t *testing.T) {
	result := normalizeListOptions(nil)

	assert.NotNil(t, result)
	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)
}

func TestNormalizeListOptions_Valid(t *testing.T) {
	opts := &ListOptions{Limit: 50, Offset: 100}

	result := normalizeListOptions(opts)

	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 100, result.Offset)
}

func TestNormalizeListOptions_ZeroLimit(t *testing.T) {
	opts := &ListOptions{Limit: 0, Offset: 10}

	result := normalizeListOptions(opts)

	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 10, result.Offset)
}

func TestNormalizeListOptions_NegativeLimit(t *testing.T) {
	opts := &ListOptions{Limit: -5, Offset: 0}

	result := normalizeListOptions(opts)

	assert.Equal(t, 20, result.Limit)
}

func TestNormalizeListOptions_ExceedsMax(t *testing.T) {
	opts := &ListOptions{Limit: 500, Offset: 0}

	result := normalizeListOptions(opts)

	assert.Equal(t, 100, result.Limit)
}

func TestSimulation_NilOptionalFields(t *testing.T) {
	sim := &Simulation{
		ID:             "test-id",
		UserID:         "user-123",
		Name:           "Test",
		SimulationType: "WHAT_IF",
		// Optional fields are nil
		BaselineFlow:      nil,
		ResultFlow:        nil,
		FlowChangePercent: nil,
	}

	assert.Nil(t, sim.BaselineFlow)
	assert.Nil(t, sim.ResultFlow)
	assert.Nil(t, sim.FlowChangePercent)
}

// ============================================================
// EDGE CASES
// ============================================================

func TestSimulation_EmptyTags(t *testing.T) {
	sim := &Simulation{
		ID:   "test-id",
		Tags: []string{},
	}

	assert.Empty(t, sim.Tags)
	assert.Len(t, sim.Tags, 0)
}

func TestSimulation_EmptyGraphData(t *testing.T) {
	sim := &Simulation{
		ID:        "test-id",
		GraphData: nil,
	}

	assert.Nil(t, sim.GraphData)
}

func TestListOptions_ZeroValues(t *testing.T) {
	opts := &ListOptions{
		Limit:  0,
		Offset: 0,
	}

	assert.Equal(t, 0, opts.Limit)
	assert.Equal(t, 0, opts.Offset)
}

func TestListOptions_LargeValues(t *testing.T) {
	opts := &ListOptions{
		Limit:  1000000,
		Offset: 9999999,
	}

	assert.Equal(t, 1000000, opts.Limit)
	assert.Equal(t, 9999999, opts.Offset)
}
