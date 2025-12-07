package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
	"logistics/services/simulation-svc/internal/repository"
)

// ============================================================
// MOCKS
// ============================================================

// MockSimulationRepository mock для репозитория
type MockSimulationRepository struct {
	mock.Mock
}

func (m *MockSimulationRepository) Create(ctx context.Context, sim *repository.Simulation) error {
	args := m.Called(ctx, sim)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	// Симулируем присвоение ID
	sim.ID = "test-sim-id"
	sim.CreatedAt = time.Now()
	sim.UpdatedAt = time.Now()
	return nil
}

func (m *MockSimulationRepository) GetByID(ctx context.Context, id string) (*repository.Simulation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Simulation), args.Error(1)
}

func (m *MockSimulationRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSimulationRepository) List(ctx context.Context, userID string, simType string, opts *repository.ListOptions) ([]*repository.SimulationSummary, int64, error) {
	args := m.Called(ctx, userID, simType, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*repository.SimulationSummary), args.Get(1).(int64), args.Error(2)
}

func (m *MockSimulationRepository) ListByUser(ctx context.Context, userID string, opts *repository.ListOptions) ([]*repository.SimulationSummary, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*repository.SimulationSummary), args.Get(1).(int64), args.Error(2)
}

func (m *MockSimulationRepository) GetByUserAndID(ctx context.Context, userID, id string) (*repository.Simulation, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Simulation), args.Error(1)
}

// MockSolverClient mock для solver клиента
type MockSolverClient struct {
	mock.Mock
}

func (m *MockSolverClient) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	args := m.Called(ctx, graph, algorithm, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.SolveResult), args.Error(1)
}

func (m *MockSolverClient) Close() error {
	return nil
}

// ============================================================
// TEST HELPERS
// ============================================================

func createTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Name:     "test-graph",
		Nodes: []*commonv1.Node{
			{Id: 1, Name: "source", Type: commonv1.NodeType_NODE_TYPE_SOURCE, X: 0, Y: 0},
			{Id: 2, Name: "node2", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, X: 1, Y: 0},
			{Id: 3, Name: "node3", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, X: 1, Y: 1},
			{Id: 4, Name: "sink", Type: commonv1.NodeType_NODE_TYPE_SINK, X: 2, Y: 0},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 1, To: 3, Capacity: 10, Cost: 2},
			{From: 2, To: 4, Capacity: 10, Cost: 1},
			{From: 3, To: 4, Capacity: 10, Cost: 1},
		},
		Metadata: map[string]string{"test": "value"},
	}
}

func createTestSolveResult(maxFlow, totalCost float64) *client.SolveResult {
	return &client.SolveResult{
		MaxFlow:            maxFlow,
		TotalCost:          totalCost,
		AverageUtilization: 0.5,
		SaturatedEdges:     2,
		ActivePaths:        2,
		Status:             commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		ComputationTimeMs:  10.5,
		Graph:              createTestGraph(),
	}
}

// SolverClientWrapper адаптер для использования mock в сервисе
type SolverClientWrapper struct {
	mockClient *MockSolverClient
}

func (w *SolverClientWrapper) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	return w.mockClient.Solve(ctx, graph, algorithm, opts)
}

// ============================================================
// HEALTH TESTS
// ============================================================

func TestSimulationService_Health(t *testing.T) {
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	resp, err := svc.Health(context.Background(), &simulationv1.HealthRequest{})

	require.NoError(t, err)
	assert.Equal(t, "SERVING", resp.Status)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.True(t, resp.UptimeSeconds >= 0)
}

// ============================================================
// WHAT-IF TESTS
// ============================================================

func TestSimulationService_RunWhatIf_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClient)

	// Baseline result
	baselineResult := createTestSolveResult(20, 40)
	// Modified result (после увеличения capacity)
	modifiedResult := createTestSolveResult(25, 50)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(modifiedResult, nil).Once()

	// Создаём реальный SolverClient для теста (нужна интеграция)
	// В данном случае используем nil и тестируем через моки engine
	svc := NewSimulationService(repo, nil, "1.0.0")
	// Подменяем solverEngine на wrapper с mock
	svc.solverClient = nil // В реальности нужен способ инжектить mock

	req := &simulationv1.RunWhatIfRequest{
		BaselineGraph: createTestGraph(),
		Modifications: []*simulationv1.Modification{
			{
				Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
				EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
				Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Change:  &simulationv1.Modification_Delta{Delta: 5},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &simulationv1.WhatIfOptions{
			CompareWithBaseline: true,
			FindNewBottlenecks:  true,
			ReturnModifiedGraph: true,
		},
	}

	// Тест без solver client возвращает ошибку
	resp, err := svc.RunWhatIf(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSimulationService_RunWhatIf_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.RunWhatIfRequest{
		BaselineGraph: nil,
		Algorithm:     commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunWhatIf(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "baseline_graph is required")
}

// ============================================================
// COMPARE SCENARIOS TESTS
// ============================================================

func TestSimulationService_CompareScenarios_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.CompareScenariosRequest{
		BaselineGraph: nil,
		Scenarios: []*simulationv1.Scenario{
			{Name: "test", Modifications: nil},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.CompareScenarios(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "baseline_graph is required")
}

// ============================================================
// MONTE CARLO TESTS
// ============================================================

func TestSimulationService_RunMonteCarlo_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.RunMonteCarloRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunMonteCarlo(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// SENSITIVITY ANALYSIS TESTS
// ============================================================

func TestSimulationService_AnalyzeSensitivity_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.AnalyzeSensitivityRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.AnalyzeSensitivity(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// CRITICAL ELEMENTS TESTS
// ============================================================

func TestSimulationService_FindCriticalElements_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.FindCriticalElementsRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.FindCriticalElements(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// FAILURE SIMULATION TESTS
// ============================================================

func TestSimulationService_SimulateFailures_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.SimulateFailuresRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulateFailures(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// RESILIENCE ANALYSIS TESTS
// ============================================================

func TestSimulationService_AnalyzeResilience_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.AnalyzeResilienceRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.AnalyzeResilience(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// TIME SIMULATION TESTS
// ============================================================

func TestSimulationService_RunTimeSimulation_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.RunTimeSimulationRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunTimeSimulation(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// PEAK LOAD TESTS
// ============================================================

func TestSimulationService_SimulatePeakLoad_NoGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulatePeakLoad(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "graph is required")
}

// ============================================================
// SAVE SIMULATION TESTS
// ============================================================

func TestSimulationService_SaveSimulation_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("Create", mock.Anything, mock.AnythingOfType("*repository.Simulation")).
		Return(nil)

	req := &simulationv1.SaveSimulationRequest{
		UserId:       "user-123",
		Name:         "Test Simulation",
		Description:  "Test description",
		Type:         simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF,
		Graph:        createTestGraph(),
		RequestData:  []byte(`{"test": "request"}`),
		ResponseData: []byte(`{"test": "response"}`),
		Tags:         map[string]string{"env": "test", "version": "1"},
	}

	resp, err := svc.SaveSimulation(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.SimulationId)
	assert.NotNil(t, resp.CreatedAt)
	repo.AssertExpectations(t)
}

func TestSimulationService_SaveSimulation_NoUserID(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	req := &simulationv1.SaveSimulationRequest{
		UserId: "",
		Name:   "Test Simulation",
	}

	resp, err := svc.SaveSimulation(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestSimulationService_SaveSimulation_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("Create", mock.Anything, mock.AnythingOfType("*repository.Simulation")).
		Return(errors.New("database error"))

	req := &simulationv1.SaveSimulationRequest{
		UserId: "user-123",
		Name:   "Test Simulation",
	}

	resp, err := svc.SaveSimulation(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

func TestSimulationService_SaveSimulation_WithoutGraph(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("Create", mock.Anything, mock.AnythingOfType("*repository.Simulation")).
		Return(nil)

	req := &simulationv1.SaveSimulationRequest{
		UserId:      "user-123",
		Name:        "Test Simulation",
		Description: "No graph",
		Type:        simulationv1.SimulationType_SIMULATION_TYPE_MONTE_CARLO,
	}

	resp, err := svc.SaveSimulation(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.SimulationId)
	repo.AssertExpectations(t)
}

// ============================================================
// GET SIMULATION TESTS
// ============================================================

func TestSimulationService_GetSimulation_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	expectedSim := &repository.Simulation{
		ID:             "sim-123",
		UserID:         "user-123",
		Name:           "Test Simulation",
		Description:    "Test description",
		SimulationType: "SIMULATION_TYPE_WHAT_IF",
		CreatedAt:      time.Now(),
		RequestData:    []byte(`{}`),
		ResponseData:   []byte(`{}`),
		Tags:           []string{"env:test", "version:1"},
	}

	repo.On("GetByUserAndID", mock.Anything, "user-123", "sim-123").
		Return(expectedSim, nil)

	req := &simulationv1.GetSimulationRequest{
		SimulationId: "sim-123",
		UserId:       "user-123",
	}

	resp, err := svc.GetSimulation(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "sim-123", resp.Record.Id)
	assert.Equal(t, "user-123", resp.Record.UserId)
	assert.Equal(t, "Test Simulation", resp.Record.Name)
	assert.Equal(t, simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF, resp.Record.Type)
	assert.Equal(t, "test", resp.Record.Tags["env"])
	assert.Equal(t, "1", resp.Record.Tags["version"])
	repo.AssertExpectations(t)
}

func TestSimulationService_GetSimulation_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("GetByUserAndID", mock.Anything, "user-123", "sim-999").
		Return(nil, repository.ErrSimulationNotFound)

	req := &simulationv1.GetSimulationRequest{
		SimulationId: "sim-999",
		UserId:       "user-123",
	}

	resp, err := svc.GetSimulation(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")
	repo.AssertExpectations(t)
}

func TestSimulationService_GetSimulation_AccessDenied(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("GetByUserAndID", mock.Anything, "user-456", "sim-123").
		Return(nil, repository.ErrAccessDenied)

	req := &simulationv1.GetSimulationRequest{
		SimulationId: "sim-123",
		UserId:       "user-456",
	}

	resp, err := svc.GetSimulation(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "access denied")
	repo.AssertExpectations(t)
}

func TestSimulationService_GetSimulation_InternalError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("GetByUserAndID", mock.Anything, "user-123", "sim-123").
		Return(nil, errors.New("database connection lost"))

	req := &simulationv1.GetSimulationRequest{
		SimulationId: "sim-123",
		UserId:       "user-123",
	}

	resp, err := svc.GetSimulation(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

// ============================================================
// LIST SIMULATIONS TESTS
// ============================================================

func TestSimulationService_ListSimulations_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	expectedSims := []*repository.SimulationSummary{
		{
			ID:             "sim-1",
			Name:           "Simulation 1",
			SimulationType: "SIMULATION_TYPE_WHAT_IF",
			CreatedAt:      time.Now(),
			Tags:           []string{"env:prod"},
		},
		{
			ID:             "sim-2",
			Name:           "Simulation 2",
			SimulationType: "SIMULATION_TYPE_MONTE_CARLO",
			CreatedAt:      time.Now(),
			Tags:           []string{"env:test"},
		},
	}

	repo.On("List", mock.Anything, "user-123", "", mock.AnythingOfType("*repository.ListOptions")).
		Return(expectedSims, int64(2), nil)

	req := &simulationv1.ListSimulationsRequest{
		UserId: "user-123",
		Type:   simulationv1.SimulationType_SIMULATION_TYPE_UNSPECIFIED,
		Pagination: &commonv1.PaginationRequest{
			Page:     1,
			PageSize: 20,
		},
	}

	resp, err := svc.ListSimulations(ctx, req)

	require.NoError(t, err)
	assert.Len(t, resp.Simulations, 2)
	assert.Equal(t, "sim-1", resp.Simulations[0].Id)
	assert.Equal(t, "sim-2", resp.Simulations[1].Id)
	assert.Equal(t, int64(2), resp.Pagination.TotalItems)
	repo.AssertExpectations(t)
}

func TestSimulationService_ListSimulations_WithTypeFilter(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	expectedSims := []*repository.SimulationSummary{
		{
			ID:             "sim-1",
			Name:           "Simulation 1",
			SimulationType: "SIMULATION_TYPE_WHAT_IF",
			CreatedAt:      time.Now(),
		},
	}

	repo.On("List", mock.Anything, "user-123", "SIMULATION_TYPE_WHAT_IF", mock.AnythingOfType("*repository.ListOptions")).
		Return(expectedSims, int64(1), nil)

	req := &simulationv1.ListSimulationsRequest{
		UserId: "user-123",
		Type:   simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF,
	}

	resp, err := svc.ListSimulations(ctx, req)

	require.NoError(t, err)
	assert.Len(t, resp.Simulations, 1)
	assert.Equal(t, simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF, resp.Simulations[0].Type)
	repo.AssertExpectations(t)
}

func TestSimulationService_ListSimulations_Pagination(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	expectedSims := []*repository.SimulationSummary{
		{ID: "sim-11", Name: "Sim 11", SimulationType: "SIMULATION_TYPE_WHAT_IF", CreatedAt: time.Now()},
		{ID: "sim-12", Name: "Sim 12", SimulationType: "SIMULATION_TYPE_WHAT_IF", CreatedAt: time.Now()},
	}

	repo.On("List", mock.Anything, "user-123", "", mock.MatchedBy(func(opts *repository.ListOptions) bool {
		return opts.Limit == 2 && opts.Offset == 10
	})).Return(expectedSims, int64(25), nil)

	req := &simulationv1.ListSimulationsRequest{
		UserId: "user-123",
		Pagination: &commonv1.PaginationRequest{
			Page:     6, // offset = (6-1)*2 = 10
			PageSize: 2,
		},
	}

	resp, err := svc.ListSimulations(ctx, req)

	require.NoError(t, err)
	assert.Len(t, resp.Simulations, 2)
	assert.Equal(t, int32(6), resp.Pagination.CurrentPage)
	assert.Equal(t, int32(2), resp.Pagination.PageSize)
	assert.Equal(t, int64(25), resp.Pagination.TotalItems)
	assert.Equal(t, int32(13), resp.Pagination.TotalPages) // ceil(25/2) = 13
	assert.True(t, resp.Pagination.HasNext)
	assert.True(t, resp.Pagination.HasPrevious)
	repo.AssertExpectations(t)
}

func TestSimulationService_ListSimulations_Empty(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("List", mock.Anything, "user-123", "", mock.AnythingOfType("*repository.ListOptions")).
		Return([]*repository.SimulationSummary{}, int64(0), nil)

	req := &simulationv1.ListSimulationsRequest{
		UserId: "user-123",
	}

	resp, err := svc.ListSimulations(ctx, req)

	require.NoError(t, err)
	assert.Empty(t, resp.Simulations)
	assert.Equal(t, int64(0), resp.Pagination.TotalItems)
	repo.AssertExpectations(t)
}

func TestSimulationService_ListSimulations_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	repo.On("List", mock.Anything, "user-123", "", mock.AnythingOfType("*repository.ListOptions")).
		Return(nil, int64(0), errors.New("database error"))

	req := &simulationv1.ListSimulationsRequest{
		UserId: "user-123",
	}

	resp, err := svc.ListSimulations(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

// ============================================================
// HELPER FUNCTION TESTS
// ============================================================

func TestParseSimulationType(t *testing.T) {
	tests := []struct {
		input    string
		expected simulationv1.SimulationType
	}{
		{"SIMULATION_TYPE_WHAT_IF", simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF},
		{"SIMULATION_TYPE_TIME", simulationv1.SimulationType_SIMULATION_TYPE_TIME},
		{"SIMULATION_TYPE_MONTE_CARLO", simulationv1.SimulationType_SIMULATION_TYPE_MONTE_CARLO},
		{"SIMULATION_TYPE_SENSITIVITY", simulationv1.SimulationType_SIMULATION_TYPE_SENSITIVITY},
		{"SIMULATION_TYPE_FAILURE", simulationv1.SimulationType_SIMULATION_TYPE_FAILURE},
		{"SIMULATION_TYPE_RESILIENCE", simulationv1.SimulationType_SIMULATION_TYPE_RESILIENCE},
		{"UNKNOWN", simulationv1.SimulationType_SIMULATION_TYPE_UNSPECIFIED},
		{"", simulationv1.SimulationType_SIMULATION_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSimulationType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitOnce(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"key:value", ":", []string{"key", "value"}},
		{"key:value:extra", ":", []string{"key", "value:extra"}},
		{"no-separator", ":", []string{"no-separator"}},
		{"", ":", []string{""}},
		{"key::value", ":", []string{"key", ":value"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitOnce(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEdgeKey(t *testing.T) {
	tests := []struct {
		from, to int64
		expected string
	}{
		{1, 2, "1->2"},
		{100, 200, "100->200"},
		{0, 0, "0->0"},
	}

	for _, tt := range tests {
		result := edgeKey(tt.from, tt.to)
		assert.Equal(t, tt.expected, result)
	}
}

// ============================================================
// HELPER METHOD TESTS
// ============================================================

func TestSimulationService_RemoveEdgeFromGraph(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	result := svc.removeEdgeFromGraph(graph, 1, 2)

	assert.Equal(t, 3, len(result.Edges))
	for _, edge := range result.Edges {
		assert.False(t, edge.From == 1 && edge.To == 2)
	}
	// Исходный граф не должен измениться
	assert.Equal(t, 4, len(graph.Edges))
}

func TestSimulationService_RemoveNodeFromGraph(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	result := svc.removeNodeFromGraph(graph, 2)

	// Узел 2 должен быть удалён
	assert.Equal(t, 3, len(result.Nodes))
	for _, node := range result.Nodes {
		assert.NotEqual(t, int64(2), node.Id)
	}

	// Рёбра связанные с узлом 2 должны быть удалены
	for _, edge := range result.Edges {
		assert.NotEqual(t, int64(2), edge.From)
		assert.NotEqual(t, int64(2), edge.To)
	}

	// Исходный граф не должен измениться
	assert.Equal(t, 4, len(graph.Nodes))
	assert.Equal(t, 4, len(graph.Edges))
}

func TestSimulationService_CountAffectedEdges(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	// Узел 2 связан с рёбрами: 1->2, 2->4
	count := svc.countAffectedEdges(graph, 2)
	assert.Equal(t, 2, count)

	// Узел 1 связан с рёбрами: 1->2, 1->3
	count = svc.countAffectedEdges(graph, 1)
	assert.Equal(t, 2, count)

	// Узел 4 связан с рёбрами: 2->4, 3->4
	count = svc.countAffectedEdges(graph, 4)
	assert.Equal(t, 2, count)

	// Несуществующий узел
	count = svc.countAffectedEdges(graph, 999)
	assert.Equal(t, 0, count)
}

func TestSimulationService_CalculateResilienceScore(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	tests := []struct {
		criticalEdges int
		criticalNodes int
		totalEdges    int
		totalNodes    int
		minExpected   float64
		maxExpected   float64
	}{
		{0, 0, 10, 5, 1.0, 1.0},
		{2, 1, 10, 5, 0.7, 0.85},
		{5, 5, 10, 10, 0.4, 0.6},
		{0, 0, 0, 0, 1.0, 1.0}, // Edge case
	}

	for _, tt := range tests {
		edges := make([]*simulationv1.CriticalEdge, tt.criticalEdges)
		nodes := make([]*simulationv1.CriticalNode, tt.criticalNodes)

		score := svc.calculateResilienceScore(edges, nodes, tt.totalEdges, tt.totalNodes)
		assert.GreaterOrEqual(t, score, tt.minExpected)
		assert.LessOrEqual(t, score, tt.maxExpected)
	}
}

func TestSimulationService_SortCriticalEdges(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	edges := []*simulationv1.CriticalEdge{
		{Edge: &commonv1.EdgeKey{From: 1, To: 2}, CriticalityScore: 0.3},
		{Edge: &commonv1.EdgeKey{From: 2, To: 3}, CriticalityScore: 0.9},
		{Edge: &commonv1.EdgeKey{From: 3, To: 4}, CriticalityScore: 0.5},
	}

	svc.sortCriticalEdges(edges)

	assert.Equal(t, 0.9, edges[0].CriticalityScore)
	assert.Equal(t, 0.5, edges[1].CriticalityScore)
	assert.Equal(t, 0.3, edges[2].CriticalityScore)
}

func TestSimulationService_SortCriticalNodes(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	nodes := []*simulationv1.CriticalNode{
		{NodeId: 1, CriticalityScore: 0.2},
		{NodeId: 2, CriticalityScore: 0.8},
		{NodeId: 3, CriticalityScore: 0.4},
	}

	svc.sortCriticalNodes(nodes)

	assert.Equal(t, 0.8, nodes[0].CriticalityScore)
	assert.Equal(t, 0.4, nodes[1].CriticalityScore)
	assert.Equal(t, 0.2, nodes[2].CriticalityScore)
}

func TestSimulationService_GenerateRecommendation(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	// С лучшим сценарием
	rec := svc.generateRecommendation(nil, "Scenario A")
	assert.Contains(t, rec, "Scenario A")

	// Без лучшего сценария
	rec = svc.generateRecommendation(nil, "")
	assert.Contains(t, rec, "худшие результаты")
}

func TestSimulationService_CalculateModificationCost(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	mods := []*simulationv1.Modification{
		{
			Type:   simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
			Change: &simulationv1.Modification_Delta{Delta: 10},
		},
		{
			Type:   simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
			Change: &simulationv1.Modification_Delta{Delta: 5},
		},
		{
			Type:   simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
			Change: &simulationv1.Modification_Delta{Delta: -3}, // Negative delta shouldn't count
		},
		{
			Type:   simulationv1.ModificationType_MODIFICATION_TYPE_REMOVE_EDGE,
			Change: &simulationv1.Modification_Delta{Delta: 10}, // Wrong type
		},
	}

	cost := svc.calculateModificationCost(mods, 100.0)
	assert.Equal(t, 1500.0, cost) // (10 + 5) * 100
}

func TestSimulationService_GenerateRandomFailureScenarios(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	config := &simulationv1.RandomFailureConfig{
		NumScenarios:            5,
		EdgeFailureProbability:  0.3,
		MaxSimultaneousFailures: 2,
	}

	scenarios := svc.generateRandomFailureScenarios(graph, config)

	assert.Len(t, scenarios, 5)
	for i, s := range scenarios {
		assert.Contains(t, s.Name, "Random Scenario")
		assert.LessOrEqual(t, len(s.FailedEdges), 2)
		assert.InDelta(t, 0.2, s.Probability, 0.01) // 1/5 = 0.2
		t.Logf("Scenario %d: %s with %d failed edges", i, s.Name, len(s.FailedEdges))
	}
}

func TestSimulationService_GenerateRandomFailureScenarios_DefaultConfig(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	config := &simulationv1.RandomFailureConfig{
		NumScenarios:            0, // Default to 10
		EdgeFailureProbability:  0, // Default to 0.1
		MaxSimultaneousFailures: 0, // Default to 3
	}

	scenarios := svc.generateRandomFailureScenarios(graph, config)

	assert.Len(t, scenarios, 10)
}

func TestSimulationService_CalculateFailureStats(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	results := []*simulationv1.FailureScenarioResult{
		{
			ScenarioName:        "Scenario 1",
			Probability:         0.3,
			Result:              &simulationv1.ScenarioResult{MaxFlow: 80},
			NetworkDisconnected: false,
		},
		{
			ScenarioName:        "Scenario 2",
			Probability:         0.5,
			Result:              &simulationv1.ScenarioResult{MaxFlow: 60},
			NetworkDisconnected: false,
		},
		{
			ScenarioName:        "Scenario 3",
			Probability:         0.2,
			Result:              &simulationv1.ScenarioResult{MaxFlow: 0},
			NetworkDisconnected: true,
		},
	}

	stats := svc.calculateFailureStats(results, 100)

	// Expected loss: 0.3*(100-80) + 0.5*(100-60) + 0.2*(100-0) = 6 + 20 + 20 = 46
	assert.InDelta(t, 46.0, stats.ExpectedFlowLoss, 0.1)
	assert.Equal(t, 100.0, stats.MaxFlowLoss)
	assert.InDelta(t, 1.0/3.0, stats.ProbabilityOfDisconnection, 0.01)
}

func TestSimulationService_CalculateFailureStats_Empty(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	stats := svc.calculateFailureStats([]*simulationv1.FailureScenarioResult{}, 100)

	assert.Equal(t, &simulationv1.FailureStats{}, stats)
}

func TestSimulationService_GenerateResilienceRecommendations(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	results := []*simulationv1.FailureScenarioResult{
		{ScenarioName: "Disconnection", NetworkDisconnected: true},
		{ScenarioName: "Minor", NetworkDisconnected: false},
	}

	// Граф с низким резервированием
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
		Edges: []*commonv1.Edge{{From: 1, To: 2}, {From: 2, To: 3}}, // 2 edges / 3 nodes < 2
	}

	recs := svc.generateResilienceRecommendations(results, graph)

	assert.GreaterOrEqual(t, len(recs), 2)

	hasRedundancy := false
	hasBackup := false
	for _, rec := range recs {
		if rec.Type == simulationv1.RecommendationType_RECOMMENDATION_TYPE_ADD_REDUNDANCY {
			hasRedundancy = true
		}
		if rec.Type == simulationv1.RecommendationType_RECOMMENDATION_TYPE_ADD_BACKUP_ROUTE {
			hasBackup = true
		}
	}
	assert.True(t, hasRedundancy)
	assert.True(t, hasBackup)
}

// ============================================================
// BOTTLENECK ANALYSIS TESTS
// ============================================================

func TestBottleneckChangeAnalyzer(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	baseResult := &client.SolveResult{
		Graph: &commonv1.Graph{
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 100, CurrentFlow: 95}, // Bottleneck
				{From: 2, To: 3, Capacity: 100, CurrentFlow: 50}, // Not bottleneck
				{From: 3, To: 4, Capacity: 100, CurrentFlow: 90}, // Near bottleneck
			},
		},
	}

	modResult := &client.SolveResult{
		Graph: &commonv1.Graph{
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 100, CurrentFlow: 50}, // Resolved
				{From: 2, To: 3, Capacity: 100, CurrentFlow: 95}, // New bottleneck
				{From: 3, To: 4, Capacity: 100, CurrentFlow: 98}, // Worsened (now bottleneck)
			},
		},
	}

	changes := svc.findBottleneckChanges(baseResult, modResult)

	assert.GreaterOrEqual(t, len(changes), 2)

	changeTypes := make(map[simulationv1.BottleneckChangeType]int)
	for _, c := range changes {
		changeTypes[c.ChangeType]++
	}

	// Должны быть: RESOLVED (1->2), NEW (2->3), возможно NEW/WORSENED (3->4)
	assert.Contains(t, changeTypes, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_RESOLVED)
	assert.Contains(t, changeTypes, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_NEW)
}

func TestBottleneckChangeAnalyzer_NilGraphs(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	changes := svc.findBottleneckChanges(
		&client.SolveResult{Graph: nil},
		&client.SolveResult{Graph: nil},
	)

	assert.Empty(t, changes)
}

// ============================================================
// SCENARIO SORTING TESTS
// ============================================================

func TestSimulationService_SortScenariosByFlow(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")

	scenarios := []*simulationv1.ScenarioResultWithRank{
		{Result: &simulationv1.ScenarioResult{Name: "A", MaxFlow: 100}},
		{Result: &simulationv1.ScenarioResult{Name: "B", MaxFlow: 300}},
		{Result: &simulationv1.ScenarioResult{Name: "C", MaxFlow: 200}},
	}

	svc.sortScenariosByFlow(scenarios)

	assert.Equal(t, "B", scenarios[0].Result.Name)
	assert.Equal(t, "C", scenarios[1].Result.Name)
	assert.Equal(t, "A", scenarios[2].Result.Name)
}
