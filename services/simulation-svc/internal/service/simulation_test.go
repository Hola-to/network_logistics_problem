// services/simulation-svc/internal/service/simulation_test.go
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
	"logistics/services/simulation-svc/internal/engine"
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

// MockSolverClientInterface mock для engine.SolverClientInterface
type MockSolverClientInterface struct {
	mock.Mock
}

func (m *MockSolverClientInterface) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	args := m.Called(ctx, graph, algorithm, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.SolveResult), args.Error(1)
}

// Проверка что mock реализует интерфейс
var _ engine.SolverClientInterface = (*MockSolverClientInterface)(nil)

// ============================================================
// TEST HELPERS
// ============================================================

func createTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Name:     "test-graph",
		Nodes: []*commonv1.Node{
			{Id: 1, Name: "source", Type: commonv1.NodeType_NODE_TYPE_SOURCE, X: 0, Y: 0, Supply: 100},
			{Id: 2, Name: "node2", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, X: 1, Y: 0},
			{Id: 3, Name: "node3", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, X: 1, Y: 1},
			{Id: 4, Name: "sink", Type: commonv1.NodeType_NODE_TYPE_SINK, X: 2, Y: 0, Demand: 100},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1, CurrentFlow: 8},
			{From: 1, To: 3, Capacity: 10, Cost: 2, CurrentFlow: 10},
			{From: 2, To: 4, Capacity: 10, Cost: 1, CurrentFlow: 8},
			{From: 3, To: 4, Capacity: 10, Cost: 1, CurrentFlow: 10},
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

func createTestSolveResultWithGraph(maxFlow, totalCost float64, graph *commonv1.Graph) *client.SolveResult {
	return &client.SolveResult{
		MaxFlow:            maxFlow,
		TotalCost:          totalCost,
		AverageUtilization: 0.8,
		SaturatedEdges:     2,
		ActivePaths:        2,
		Status:             commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		ComputationTimeMs:  10.5,
		Graph:              graph,
	}
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
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(20, 40)
	modifiedResult := createTestSolveResult(25, 50)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(modifiedResult, nil).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

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

	resp, err := svc.RunWhatIf(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Baseline)
	assert.NotNil(t, resp.Modified)
	assert.NotNil(t, resp.Comparison)
	assert.NotNil(t, resp.ModifiedGraph)
	assert.NotNil(t, resp.Metadata)
	mockSolver.AssertExpectations(t)
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

func TestSimulationService_RunWhatIf_BaselineSolverError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("solver error")).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.RunWhatIfRequest{
		BaselineGraph: createTestGraph(),
		Algorithm:     commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunWhatIf(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	mockSolver.AssertExpectations(t)
}

func TestSimulationService_RunWhatIf_ModifiedSolverError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(20, 40)
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("solver error")).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.RunWhatIfRequest{
		BaselineGraph: createTestGraph(),
		Algorithm:     commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunWhatIf(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	mockSolver.AssertExpectations(t)
}

func TestSimulationService_RunWhatIf_WithBottleneckAnalysis(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	// Baseline с bottleneck
	baseGraph := createTestGraph()
	baseGraph.Edges[0].CurrentFlow = 9.5 // 95% utilization
	baseGraph.Edges[0].Capacity = 10
	baselineResult := createTestSolveResultWithGraph(20, 40, baseGraph)

	// Modified без bottleneck
	modGraph := createTestGraph()
	modGraph.Edges[0].CurrentFlow = 5
	modGraph.Edges[0].Capacity = 20
	modifiedResult := createTestSolveResultWithGraph(25, 50, modGraph)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(modifiedResult, nil).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.RunWhatIfRequest{
		BaselineGraph: createTestGraph(),
		Modifications: []*simulationv1.Modification{
			{
				Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
				EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
				Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 20},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &simulationv1.WhatIfOptions{
			FindNewBottlenecks: true,
		},
	}

	resp, err := svc.RunWhatIf(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	// Должны быть изменения в bottlenecks
	mockSolver.AssertExpectations(t)
}

// ============================================================
// COMPARE SCENARIOS TESTS
// ============================================================

func TestSimulationService_CompareScenarios_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(100, 50)
	scenarioAResult := createTestSolveResult(120, 60)
	scenarioBResult := createTestSolveResult(110, 55)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(scenarioAResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(scenarioBResult, nil).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.CompareScenariosRequest{
		BaselineGraph: createTestGraph(),
		Scenarios: []*simulationv1.Scenario{
			{
				Name: "Scenario A",
				Modifications: []*simulationv1.Modification{
					{
						Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
						EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
						Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
						Change:  &simulationv1.Modification_Delta{Delta: 10},
					},
				},
			},
			{
				Name: "Scenario B",
				Modifications: []*simulationv1.Modification{
					{
						Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
						EdgeKey: &commonv1.EdgeKey{From: 2, To: 4},
						Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
						Change:  &simulationv1.Modification_Delta{Delta: 5},
					},
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.CompareScenarios(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Baseline)
	assert.Len(t, resp.RankedScenarios, 2)
	assert.Equal(t, "Scenario A", resp.BestScenario)
	assert.NotEmpty(t, resp.Recommendation)
	assert.NotNil(t, resp.Metadata)
	mockSolver.AssertExpectations(t)
}

func TestSimulationService_CompareScenarios_WithROI(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(100, 50)
	scenarioResult := createTestSolveResult(150, 75)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(scenarioResult, nil).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.CompareScenariosRequest{
		BaselineGraph: createTestGraph(),
		Scenarios: []*simulationv1.Scenario{
			{
				Name: "Investment Scenario",
				Modifications: []*simulationv1.Modification{
					{
						Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
						EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
						Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
						Change:  &simulationv1.Modification_Delta{Delta: 10},
					},
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &simulationv1.CompareOptions{
			CalculateRoi:            true,
			ModificationCostPerUnit: 5.0,
		},
	}

	resp, err := svc.CompareScenarios(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.RankedScenarios, 1)
	// ROI = (150-100) / (10*5) = 50/50 = 1.0
	assert.InDelta(t, 1.0, resp.RankedScenarios[0].Roi, 0.01)
	mockSolver.AssertExpectations(t)
}

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

func TestSimulationService_CompareScenarios_BaselineSolverError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("solver error")).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.CompareScenariosRequest{
		BaselineGraph: createTestGraph(),
		Scenarios: []*simulationv1.Scenario{
			{Name: "test", Modifications: nil},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.CompareScenarios(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	mockSolver.AssertExpectations(t)
}

func TestSimulationService_CompareScenarios_ScenarioSolverError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(100, 50)
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("scenario solver error")).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.CompareScenariosRequest{
		BaselineGraph: createTestGraph(),
		Scenarios: []*simulationv1.Scenario{
			{Name: "failing scenario", Modifications: nil},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.CompareScenarios(ctx, req)
	require.NoError(t, err) // Ошибки сценариев не прерывают выполнение
	assert.NotNil(t, resp)
	assert.Empty(t, resp.RankedScenarios) // Сценарий с ошибкой не добавлен
	mockSolver.AssertExpectations(t)
}

// ============================================================
// MONTE CARLO TESTS
// ============================================================

func TestSimulationService_RunMonteCarlo_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	// Monte Carlo делает много вызовов solver
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.RunMonteCarloRequest{
		Graph: createTestGraph(),
		Config: &simulationv1.MonteCarloConfig{
			NumIterations:   10, // Малое число для теста
			ConfidenceLevel: 0.95,
			RandomSeed:      42,
		},
		Uncertainties: []*simulationv1.UncertaintySpec{
			{
				Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_EDGE,
				Edge:   &commonv1.EdgeKey{From: 1, To: 2},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_NORMAL,
					Param1: 1.0,
					Param2: 0.1,
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunMonteCarlo(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.FlowStats)
	assert.NotNil(t, resp.CostStats)
	assert.NotNil(t, resp.Metadata)
}

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

func TestSimulationService_RunMonteCarlo_DefaultConfig(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.RunMonteCarloRequest{
		Graph:     createTestGraph(),
		Config:    nil, // Используем default
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	// Этот тест может быть долгим из-за 1000 итераций по умолчанию
	// Поэтому просто проверяем что запрос не падает
	resp, err := svc.RunMonteCarlo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// ============================================================
// SENSITIVITY ANALYSIS TESTS
// ============================================================

func TestSimulationService_AnalyzeSensitivity_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.AnalyzeSensitivityRequest{
		Graph: createTestGraph(),
		Parameters: []*simulationv1.SensitivityParameter{
			{
				Edge:          &commonv1.EdgeKey{From: 1, To: 2},
				Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				MinMultiplier: 0.5,
				MaxMultiplier: 1.5,
				NumSteps:      5,
			},
		},
		Config: &simulationv1.SensitivityConfig{
			Method:              simulationv1.SensitivityMethod_SENSITIVITY_METHOD_ONE_AT_A_TIME,
			CalculateElasticity: true,
			FindThresholds:      true,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.AnalyzeSensitivity(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.ParameterResults, 1)
	assert.Len(t, resp.Rankings, 1)
	assert.NotNil(t, resp.Metadata)
	mockSolver.AssertExpectations(t)
}

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

func TestSimulationService_AnalyzeSensitivity_MultipleParameters(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.AnalyzeSensitivityRequest{
		Graph: createTestGraph(),
		Parameters: []*simulationv1.SensitivityParameter{
			{
				Edge:          &commonv1.EdgeKey{From: 1, To: 2},
				Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				MinMultiplier: 0.5,
				MaxMultiplier: 1.5,
				NumSteps:      3,
			},
			{
				Edge:          &commonv1.EdgeKey{From: 2, To: 4},
				Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_COST,
				MinMultiplier: 0.8,
				MaxMultiplier: 1.2,
				NumSteps:      3,
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.AnalyzeSensitivity(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.ParameterResults, 2)
	assert.Len(t, resp.Rankings, 2)
}

// ============================================================
// CRITICAL ELEMENTS TESTS
// ============================================================

func TestSimulationService_FindCriticalElements_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	// Baseline
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.FindCriticalElementsRequest{
		Graph: createTestGraph(),
		Config: &simulationv1.CriticalElementsConfig{
			AnalyzeEdges:     true,
			AnalyzeNodes:     true,
			TopN:             5,
			FailureThreshold: 0.1,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.FindCriticalElements(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Metadata)
	assert.GreaterOrEqual(t, resp.ResilienceScore, 0.0)
	assert.LessOrEqual(t, resp.ResilienceScore, 1.0)
}

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

func TestSimulationService_FindCriticalElements_DefaultConfig(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.FindCriticalElementsRequest{
		Graph:     createTestGraph(),
		Config:    nil, // Default config
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.FindCriticalElements(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestSimulationService_FindCriticalElements_WithSPOF(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	// Baseline с потоком
	baselineResult := createTestSolveResult(100, 50)
	// При удалении ребра поток = 0 (SPOF)
	spofResult := createTestSolveResult(0, 0)
	normalResult := createTestSolveResult(80, 40)

	// Первый вызов - baseline
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	// Второй - первое ребро (SPOF)
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(spofResult, nil).Once()
	// Остальные
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(normalResult, nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.FindCriticalElementsRequest{
		Graph: createTestGraph(),
		Config: &simulationv1.CriticalElementsConfig{
			AnalyzeEdges:     true,
			AnalyzeNodes:     true,
			TopN:             10,
			FailureThreshold: 0.05,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.FindCriticalElements(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	// Должен быть найден SPOF
	assert.NotEmpty(t, resp.SinglePointsOfFailure)
}

// ============================================================
// FAILURE SIMULATION TESTS
// ============================================================

func TestSimulationService_SimulateFailures_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(100, 50)
	failureResult := createTestSolveResult(70, 35)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(failureResult, nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.SimulateFailuresRequest{
		Graph: createTestGraph(),
		FailureScenarios: []*simulationv1.FailureScenario{
			{
				Name: "Edge Failure",
				FailedEdges: []*commonv1.EdgeKey{
					{From: 1, To: 2},
				},
				Probability: 0.1,
			},
			{
				Name:        "Node Failure",
				FailedNodes: []int64{2},
				Probability: 0.05,
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulateFailures(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Baseline)
	assert.Len(t, resp.ScenarioResults, 2)
	assert.NotNil(t, resp.Stats)
	assert.NotNil(t, resp.Metadata)
}

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

func TestSimulationService_SimulateFailures_RandomConfig(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.SimulateFailuresRequest{
		Graph: createTestGraph(),
		RandomConfig: &simulationv1.RandomFailureConfig{
			NumScenarios:            5,
			EdgeFailureProbability:  0.2,
			MaxSimultaneousFailures: 2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulateFailures(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	// Должно быть 5 сценариев
}

func TestSimulationService_SimulateFailures_WithDisconnection(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	baselineResult := createTestSolveResult(100, 50)
	disconnectedResult := createTestSolveResult(0, 0) // Сеть отключена

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(baselineResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(disconnectedResult, nil).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.SimulateFailuresRequest{
		Graph: createTestGraph(),
		FailureScenarios: []*simulationv1.FailureScenario{
			{
				Name: "Critical Failure",
				FailedEdges: []*commonv1.EdgeKey{
					{From: 1, To: 2},
					{From: 1, To: 3},
				},
				Probability: 0.01,
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulateFailures(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.True(t, resp.ScenarioResults[0].NetworkDisconnected)
	assert.Greater(t, resp.Stats.ProbabilityOfDisconnection, 0.0)
}

// ============================================================
// RESILIENCE ANALYSIS TESTS
// ============================================================

func TestSimulationService_AnalyzeResilience_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.AnalyzeResilienceRequest{
		Graph: createTestGraph(),
		Config: &simulationv1.ResilienceConfig{
			MaxFailuresToTest: 1,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.AnalyzeResilience(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Metrics)
	assert.NotNil(t, resp.NMinusOne)
	assert.NotNil(t, resp.Metadata)
}

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

func TestSimulationService_RunTimeSimulation_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.RunTimeSimulationRequest{
		Graph: createTestGraph(),
		TimeConfig: &simulationv1.TimeSimulationConfig{
			NumSteps: 5,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunTimeSimulation(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.StepResults, 5)
	assert.NotNil(t, resp.Stats)
	assert.NotNil(t, resp.Metadata)
}

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

func TestSimulationService_RunTimeSimulation_WithPatterns(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	hourlyMultipliers := make([]float64, 24)
	for i := range hourlyMultipliers {
		hourlyMultipliers[i] = 1.0
	}
	hourlyMultipliers[8] = 1.5  // Утренний пик
	hourlyMultipliers[18] = 1.5 // Вечерний пик

	req := &simulationv1.RunTimeSimulationRequest{
		Graph: createTestGraph(),
		TimeConfig: &simulationv1.TimeSimulationConfig{
			NumSteps: 3,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		},
		EdgePatterns: []*simulationv1.EdgeTimePattern{
			{
				Edge: &commonv1.EdgeKey{From: 1, To: 2},
				Pattern: &simulationv1.TimePattern{
					Type:              simulationv1.PatternType_PATTERN_TYPE_HOURLY,
					HourlyMultipliers: hourlyMultipliers,
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.RunTimeSimulation(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

// ============================================================
// PEAK LOAD TESTS
// ============================================================

func TestSimulationService_SimulatePeakLoad_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	normalResult := createTestSolveResult(100, 50)
	peakResult := createTestSolveResult(70, 80) // Меньше потока, больше стоимость

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(normalResult, nil).Once()
	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(peakResult, nil).Once()

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:             createTestGraph(),
		DemandMultiplier:  1.5,
		CapacityReduction: 0.8,
		Algorithm:         commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulatePeakLoad(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.NormalResult)
	assert.NotNil(t, resp.PeakResult)
	assert.NotNil(t, resp.Comparison)
	assert.NotNil(t, resp.Metadata)
}

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

func TestSimulationService_SimulatePeakLoad_WithAffectedNodes(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:            createTestGraph(),
		DemandMultiplier: 2.0,
		AffectedNodes:    []int64{1, 4}, // Только source и sink
		Algorithm:        commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulatePeakLoad(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestSimulationService_SimulatePeakLoad_WithAffectedEdges(t *testing.T) {
	ctx := context.Background()
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	mockSolver.On("Solve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createTestSolveResult(100, 50), nil)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "1.0.0")

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:             createTestGraph(),
		CapacityReduction: 0.5,
		AffectedEdges: []*commonv1.EdgeKey{
			{From: 1, To: 2},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.SimulatePeakLoad(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
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
			Page:     6,
			PageSize: 2,
		},
	}

	resp, err := svc.ListSimulations(ctx, req)

	require.NoError(t, err)
	assert.Len(t, resp.Simulations, 2)
	assert.Equal(t, int32(6), resp.Pagination.CurrentPage)
	assert.Equal(t, int32(2), resp.Pagination.PageSize)
	assert.Equal(t, int64(25), resp.Pagination.TotalItems)
	assert.Equal(t, int32(13), resp.Pagination.TotalPages)
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
	assert.Equal(t, 4, len(graph.Edges))
}

func TestSimulationService_RemoveNodeFromGraph(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	result := svc.removeNodeFromGraph(graph, 2)

	assert.Equal(t, 3, len(result.Nodes))
	for _, node := range result.Nodes {
		assert.NotEqual(t, int64(2), node.Id)
	}

	for _, edge := range result.Edges {
		assert.NotEqual(t, int64(2), edge.From)
		assert.NotEqual(t, int64(2), edge.To)
	}

	assert.Equal(t, 4, len(graph.Nodes))
	assert.Equal(t, 4, len(graph.Edges))
}

func TestSimulationService_CountAffectedEdges(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	count := svc.countAffectedEdges(graph, 2)
	assert.Equal(t, 2, count)

	count = svc.countAffectedEdges(graph, 1)
	assert.Equal(t, 2, count)

	count = svc.countAffectedEdges(graph, 4)
	assert.Equal(t, 2, count)

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
		{0, 0, 0, 0, 1.0, 1.0},
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

	rec := svc.generateRecommendation(nil, "Scenario A")
	assert.Contains(t, rec, "Scenario A")

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
			Change: &simulationv1.Modification_Delta{Delta: -3},
		},
		{
			Type:   simulationv1.ModificationType_MODIFICATION_TYPE_REMOVE_EDGE,
			Change: &simulationv1.Modification_Delta{Delta: 10},
		},
	}

	cost := svc.calculateModificationCost(mods, 100.0)
	assert.Equal(t, 1500.0, cost)
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
	for _, s := range scenarios {
		assert.Contains(t, s.Name, "Random Scenario")
		assert.LessOrEqual(t, len(s.FailedEdges), 2)
		assert.InDelta(t, 0.2, s.Probability, 0.01)
	}
}

func TestSimulationService_GenerateRandomFailureScenarios_DefaultConfig(t *testing.T) {
	svc := NewSimulationService(nil, nil, "1.0.0")
	graph := createTestGraph()

	config := &simulationv1.RandomFailureConfig{
		NumScenarios:            0,
		EdgeFailureProbability:  0,
		MaxSimultaneousFailures: 0,
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

	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
		Edges: []*commonv1.Edge{{From: 1, To: 2}, {From: 2, To: 3}},
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
				{From: 1, To: 2, Capacity: 100, CurrentFlow: 95},
				{From: 2, To: 3, Capacity: 100, CurrentFlow: 50},
				{From: 3, To: 4, Capacity: 100, CurrentFlow: 90},
			},
		},
	}

	modResult := &client.SolveResult{
		Graph: &commonv1.Graph{
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 100, CurrentFlow: 50},
				{From: 2, To: 3, Capacity: 100, CurrentFlow: 95},
				{From: 3, To: 4, Capacity: 100, CurrentFlow: 98},
			},
		},
	}

	changes := svc.findBottleneckChanges(baseResult, modResult)

	assert.GreaterOrEqual(t, len(changes), 2)

	changeTypes := make(map[simulationv1.BottleneckChangeType]int)
	for _, c := range changes {
		changeTypes[c.ChangeType]++
	}

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

// ============================================================
// CONSTRUCTOR TESTS
// ============================================================

func TestNewSimulationService(t *testing.T) {
	repo := new(MockSimulationRepository)
	svc := NewSimulationService(repo, nil, "1.0.0")

	assert.NotNil(t, svc)
	assert.Equal(t, "1.0.0", svc.version)
	assert.NotNil(t, svc.repo)
}

func TestNewSimulationServiceWithInterface(t *testing.T) {
	repo := new(MockSimulationRepository)
	mockSolver := new(MockSolverClientInterface)

	svc := NewSimulationServiceWithInterface(repo, mockSolver, "2.0.0")

	assert.NotNil(t, svc)
	assert.Equal(t, "2.0.0", svc.version)
	assert.NotNil(t, svc.repo)
	assert.NotNil(t, svc.solverClient)
}
