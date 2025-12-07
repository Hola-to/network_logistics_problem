// services/simulation-svc/internal/engine/sensitivity_test.go
package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
	"logistics/pkg/logger"
)

// ============================================================
// TEST SETUP
// ============================================================

func init() {
	// Инициализируем логгер для тестов
	logger.Init("error") // Минимальный уровень чтобы не засорять вывод
}

// ============================================================
// MOCK SOLVER CLIENT
// ============================================================

// MockSolverClient мок для SolverClientInterface
type MockSolverClient struct {
	results     []*client.SolveResult
	errors      []error
	callIndex   int
	defaultFlow float64
	flowPerStep float64
}

func NewMockSolverClient() *MockSolverClient {
	return &MockSolverClient{
		defaultFlow: 100,
		flowPerStep: 5,
	}
}

func (m *MockSolverClient) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	defer func() { m.callIndex++ }()

	// Если есть предопределённые ошибки
	if m.callIndex < len(m.errors) && m.errors[m.callIndex] != nil {
		return nil, m.errors[m.callIndex]
	}

	// Если есть предопределённые результаты
	if m.callIndex < len(m.results) && m.results[m.callIndex] != nil {
		return m.results[m.callIndex], nil
	}

	// Иначе генерируем результат на основе индекса вызова
	flow := m.defaultFlow + float64(m.callIndex)*m.flowPerStep
	return &client.SolveResult{
		MaxFlow:   flow,
		TotalCost: flow * 0.5,
		Status:    commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		Graph:     graph,
	}, nil
}

func (m *MockSolverClient) WithResults(results ...*client.SolveResult) *MockSolverClient {
	m.results = results
	return m
}

func (m *MockSolverClient) WithErrors(errs ...error) *MockSolverClient {
	m.errors = errs
	return m
}

func (m *MockSolverClient) Reset() {
	m.callIndex = 0
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func createSensitivityTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Name:     "sensitivity-test-graph",
		Nodes: []*commonv1.Node{
			{Id: 1, Name: "source", Type: commonv1.NodeType_NODE_TYPE_SOURCE, Supply: 100},
			{Id: 2, Name: "node2", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION},
			{Id: 3, Name: "node3", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION},
			{Id: 4, Name: "sink", Type: commonv1.NodeType_NODE_TYPE_SINK, Demand: 100},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 50, Cost: 1},
			{From: 1, To: 3, Capacity: 50, Cost: 2},
			{From: 2, To: 4, Capacity: 50, Cost: 1},
			{From: 3, To: 4, Capacity: 50, Cost: 1},
		},
	}
}

// ============================================================
// CONSTRUCTOR TESTS
// ============================================================

func TestNewSensitivityEngine(t *testing.T) {
	t.Run("with_mock_client", func(t *testing.T) {
		mockClient := NewMockSolverClient()
		engine := NewSensitivityEngine(mockClient)

		assert.NotNil(t, engine)
		assert.NotNil(t, engine.solverClient)
	})

	t.Run("with_nil_client", func(t *testing.T) {
		engine := NewSensitivityEngine(nil)

		assert.NotNil(t, engine)
		assert.Nil(t, engine.solverClient)
	})
}

// ============================================================
// ANALYZE SENSITIVITY TESTS
// ============================================================

func TestSensitivityEngine_AnalyzeSensitivity_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
	}

	config := &simulationv1.SensitivityConfig{
		Method:              simulationv1.SensitivityMethod_SENSITIVITY_METHOD_ONE_AT_A_TIME,
		CalculateElasticity: true,
		FindThresholds:      true,
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, config, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Len(t, result.ParameterResults, 1)
	assert.Len(t, result.Rankings, 1)
	// Metadata устанавливается в service, не в engine
}

func TestSensitivityEngine_AnalyzeSensitivity_MultipleParameters(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
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
		{
			NodeId:        1,
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY,
			MinMultiplier: 0.9,
			MaxMultiplier: 1.1,
			NumSteps:      3,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Len(t, result.ParameterResults, 3)
	assert.Len(t, result.Rankings, 3)

	// Проверяем что ранги отсортированы
	for i := 0; i < len(result.Rankings)-1; i++ {
		assert.GreaterOrEqual(t, result.Rankings[i].SensitivityIndex, result.Rankings[i+1].SensitivityIndex)
	}
}

func TestSensitivityEngine_AnalyzeSensitivity_DefaultValues(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0, // Default to 0.5
			MaxMultiplier: 0, // Default to 1.5
			NumSteps:      0, // Default to 10
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	// С дефолтными 10 шагами должно быть 10 точек на кривой
	assert.Len(t, result.ParameterResults[0].Curve, 10)
}

func TestSensitivityEngine_AnalyzeSensitivity_EmptyParameters(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.ParameterResults)
	assert.Empty(t, result.Rankings)
}

func TestSensitivityEngine_AnalyzeSensitivity_SolverError(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient().WithErrors(errors.New("solver error"))
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      3,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	// Ошибка от первого solve (base result)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSensitivityEngine_AnalyzeSensitivity_PartialSolverErrors(t *testing.T) {
	ctx := context.Background()

	// Создаём mock с результатами для всех вызовов
	// Base result успешен, некоторые шаги с ошибками
	mockClient := &MockSolverClient{
		defaultFlow: 100,
		flowPerStep: 5,
		results: []*client.SolveResult{
			{MaxFlow: 100, TotalCost: 50, Status: commonv1.FlowStatus_FLOW_STATUS_OPTIMAL}, // base
			nil, // step 1 - будет ошибка
			{MaxFlow: 110, TotalCost: 55, Status: commonv1.FlowStatus_FLOW_STATUS_OPTIMAL}, // step 2
			{MaxFlow: 115, TotalCost: 57, Status: commonv1.FlowStatus_FLOW_STATUS_OPTIMAL}, // step 3
		},
		errors: []error{
			nil,                        // base - OK
			errors.New("step 1 error"), // step 1 - error
			nil,                        // step 2 - OK
			nil,                        // step 3 - OK
		},
	}

	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      3,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	// Частичные ошибки логируются, но не прерывают выполнение
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	// Все точки должны быть в кривой, с нулевыми значениями для ошибочных
	assert.Len(t, result.ParameterResults[0].Curve, 3)
}

func TestSensitivityEngine_AnalyzeSensitivity_FindThresholds(t *testing.T) {
	ctx := context.Background()

	// Создаём результаты с резким падением потока
	mockClient := NewMockSolverClient().WithResults(
		&client.SolveResult{MaxFlow: 100, TotalCost: 50}, // base
		&client.SolveResult{MaxFlow: 100, TotalCost: 50}, // step 1
		&client.SolveResult{MaxFlow: 95, TotalCost: 48},  // step 2
		&client.SolveResult{MaxFlow: 80, TotalCost: 40},  // step 3 - 15% drop
		&client.SolveResult{MaxFlow: 50, TotalCost: 25},  // step 4 - 37.5% drop (threshold!)
		&client.SolveResult{MaxFlow: 40, TotalCost: 20},  // step 5
	)

	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
	}

	config := &simulationv1.SensitivityConfig{
		FindThresholds: true,
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, config, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	// Должны найти пороговые точки где поток падает > 10%
	assert.NotEmpty(t, result.Thresholds)
}

func TestSensitivityEngine_AnalyzeSensitivity_NodeParameter(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			NodeId:        1,
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ParameterResults[0].ParameterId, "node_1")
}

func TestSensitivityEngine_AnalyzeSensitivity_DemandParameter(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			NodeId:        4,
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_DEMAND,
			MinMultiplier: 0.8,
			MaxMultiplier: 1.2,
			NumSteps:      3,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ParameterResults[0].ParameterId, "node_4")
	assert.Contains(t, result.ParameterResults[0].ParameterId, "DEMAND")
}

// ============================================================
// BUILD PARAM ID TESTS
// ============================================================

func TestSensitivityEngine_BuildParamID(t *testing.T) {
	engine := NewSensitivityEngine(nil)

	tests := []struct {
		name     string
		param    *simulationv1.SensitivityParameter
		expected string
	}{
		{
			name: "edge_parameter_capacity",
			param: &simulationv1.SensitivityParameter{
				Edge:   &commonv1.EdgeKey{From: 1, To: 2},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			},
			expected: "edge_1_2_MODIFICATION_TARGET_CAPACITY",
		},
		{
			name: "edge_parameter_cost",
			param: &simulationv1.SensitivityParameter{
				Edge:   &commonv1.EdgeKey{From: 3, To: 4},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_COST,
			},
			expected: "edge_3_4_MODIFICATION_TARGET_COST",
		},
		{
			name: "node_parameter_supply",
			param: &simulationv1.SensitivityParameter{
				NodeId: 5,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY,
			},
			expected: "node_5_MODIFICATION_TARGET_SUPPLY",
		},
		{
			name: "node_parameter_demand",
			param: &simulationv1.SensitivityParameter{
				NodeId: 10,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_DEMAND,
			},
			expected: "node_10_MODIFICATION_TARGET_DEMAND",
		},
		{
			name: "generic_parameter",
			param: &simulationv1.SensitivityParameter{
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_LENGTH,
			},
			expected: "param_MODIFICATION_TARGET_LENGTH",
		},
		{
			name: "edge_takes_precedence_over_node",
			param: &simulationv1.SensitivityParameter{
				Edge:   &commonv1.EdgeKey{From: 1, To: 2},
				NodeId: 5,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			},
			expected: "edge_1_2_MODIFICATION_TARGET_CAPACITY",
		},
		{
			name: "zero_node_id_uses_generic",
			param: &simulationv1.SensitivityParameter{
				NodeId: 0,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			},
			expected: "param_MODIFICATION_TARGET_CAPACITY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.buildParamID(tt.param)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================
// DETERMINE SENSITIVITY LEVEL TESTS
// ============================================================

func TestDetermineSensitivityLevel(t *testing.T) {
	tests := []struct {
		name     string
		index    float64
		expected simulationv1.SensitivityLevel
	}{
		{"negligible_zero", 0.0, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_NEGLIGIBLE},
		{"negligible_small", 0.005, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_NEGLIGIBLE},
		{"negligible_boundary", 0.009, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_NEGLIGIBLE},
		{"low_lower_bound", 0.01, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_LOW},
		{"low_mid", 0.03, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_LOW},
		{"low_upper_bound", 0.049, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_LOW},
		{"medium_lower_bound", 0.05, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM},
		{"medium_mid", 0.10, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM},
		{"medium_upper_bound", 0.149, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM},
		{"high_lower_bound", 0.15, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH},
		{"high_mid", 0.22, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH},
		{"high_upper_bound", 0.299, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH},
		{"critical_lower_bound", 0.30, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_CRITICAL},
		{"critical_high", 0.50, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_CRITICAL},
		{"critical_very_high", 1.0, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_CRITICAL},
		{"critical_above_one", 1.5, simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_CRITICAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineSensitivityLevel(tt.index)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================
// DESCRIBE RANK TESTS
// ============================================================

func TestSensitivityEngine_DescribeRank(t *testing.T) {
	engine := NewSensitivityEngine(nil)

	tests := []struct {
		rank             int32
		expectedContains string
	}{
		{1, "Наиболее критичный"},
		{2, "Высокоприоритетный"},
		{3, "Высокоприоритетный"},
		{4, "средней важности"},
		{5, "средней важности"},
		{6, "низкой важности"},
		{10, "низкой важности"},
		{100, "низкой важности"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rank_%d", tt.rank), func(t *testing.T) {
			ranking := &simulationv1.ParameterRanking{Rank: tt.rank}
			result := engine.describeRank(ranking)
			assert.Contains(t, result, tt.expectedContains)
		})
	}
}

// ============================================================
// SENSITIVITY RESULT ANALYSIS TESTS
// ============================================================

func TestSensitivityResult_Elasticity(t *testing.T) {
	ctx := context.Background()

	// Создаём линейную зависимость: flow увеличивается пропорционально параметру
	mockClient := NewMockSolverClient().WithResults(
		&client.SolveResult{MaxFlow: 100, TotalCost: 50}, // base (multiplier = 1.0)
		&client.SolveResult{MaxFlow: 50, TotalCost: 25},  // multiplier = 0.5
		&client.SolveResult{MaxFlow: 75, TotalCost: 37},  // multiplier = 0.75
		&client.SolveResult{MaxFlow: 100, TotalCost: 50}, // multiplier = 1.0
		&client.SolveResult{MaxFlow: 125, TotalCost: 62}, // multiplier = 1.25
		&client.SolveResult{MaxFlow: 150, TotalCost: 75}, // multiplier = 1.5
	)

	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
	}

	config := &simulationv1.SensitivityConfig{
		CalculateElasticity: true,
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, config, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	// Проверяем что эластичность рассчитана
	paramResult := result.ParameterResults[0]
	assert.NotZero(t, paramResult.SensitivityIndex)
	assert.NotZero(t, paramResult.ImpactRange)
}

func TestSensitivityResult_CurvePoints(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)

	curve := result.ParameterResults[0].Curve
	assert.Len(t, curve, 5)

	// Проверяем что точки кривой идут от min к max multiplier
	for i := 1; i < len(curve); i++ {
		assert.Greater(t, curve[i].ParameterValue, curve[i-1].ParameterValue)
	}

	// Первая точка должна быть около 0.5, последняя около 1.5
	assert.InDelta(t, 0.5, curve[0].ParameterValue, 0.01)
	assert.InDelta(t, 1.5, curve[len(curve)-1].ParameterValue, 0.01)
}

func TestSensitivityResult_Rankings(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
		{
			Edge:          &commonv1.EdgeKey{From: 2, To: 4},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      5,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Rankings, 2)

	// Проверяем что ранги от 1 до N
	for i, ranking := range result.Rankings {
		assert.Equal(t, int32(i+1), ranking.Rank)
		assert.NotEmpty(t, ranking.ParameterId)
		assert.NotEmpty(t, ranking.Description)
	}
}

// ============================================================
// EDGE CASES
// ============================================================

func TestSensitivityEngine_AnalyzeSensitivity_ZeroBaseFlow(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient().WithResults(
		&client.SolveResult{MaxFlow: 0, TotalCost: 0}, // base with zero flow
		&client.SolveResult{MaxFlow: 0, TotalCost: 0},
		&client.SolveResult{MaxFlow: 0, TotalCost: 0},
	)
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 0.5,
			MaxMultiplier: 1.5,
			NumSteps:      3,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	// С нулевым базовым потоком эластичность и индекс должны быть 0
	assert.Equal(t, 0.0, result.ParameterResults[0].Elasticity)
}

func TestSensitivityEngine_AnalyzeSensitivity_SingleStep(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			MinMultiplier: 1.0,
			MaxMultiplier: 1.0,
			NumSteps:      1,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Len(t, result.ParameterResults[0].Curve, 1)
}

func TestSensitivityEngine_AnalyzeSensitivity_LengthTarget(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	// Добавляем length к рёбрам
	for _, edge := range graph.Edges {
		edge.Length = 10
	}

	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_LENGTH,
			MinMultiplier: 0.5,
			MaxMultiplier: 2.0,
			NumSteps:      5,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ParameterResults[0].ParameterId, "LENGTH")
}

// ============================================================
// CONTEXT CANCELLATION TEST
// ============================================================

func TestSensitivityEngine_AnalyzeSensitivity_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	mockClient := NewMockSolverClient().WithErrors(context.Canceled)
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:     &commonv1.EdgeKey{From: 1, To: 2},
			Target:   simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			NumSteps: 3,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ============================================================
// NIL CLIENT TEST
// ============================================================

func TestSensitivityEngine_AnalyzeSensitivity_NilClient(t *testing.T) {
	ctx := context.Background()
	engine := NewSensitivityEngine(nil)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:     &commonv1.EdgeKey{From: 1, To: 2},
			Target:   simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			NumSteps: 3,
		},
	}

	// С nil клиентом должна быть паника или ошибка
	// Но лучше добавить проверку в engine
	assert.Panics(t, func() {
		_, _ = engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)
	})
}

// ============================================================
// COST TARGET TEST
// ============================================================

func TestSensitivityEngine_AnalyzeSensitivity_CostTarget(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSolverClient()
	engine := NewSensitivityEngine(mockClient)

	graph := createSensitivityTestGraph()
	params := []*simulationv1.SensitivityParameter{
		{
			Edge:          &commonv1.EdgeKey{From: 1, To: 2},
			Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_COST,
			MinMultiplier: 0.5,
			MaxMultiplier: 2.0,
			NumSteps:      5,
		},
	}

	result, err := engine.AnalyzeSensitivity(ctx, graph, params, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ParameterResults[0].ParameterId, "COST")
}
