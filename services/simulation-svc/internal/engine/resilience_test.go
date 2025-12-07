// services/simulation-svc/internal/engine/resilience_test.go
package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
)

// ============================================================
// MOCK SOLVER FOR RESILIENCE
// ============================================================

type ResilienceMockSolver struct {
	callCount      int
	baseFlow       float64
	failOnEdge     map[string]bool    // edge key -> should fail
	reduceFlowEdge map[string]float64 // edge key -> flow reduction
	returnError    bool
}

func NewResilienceMockSolver() *ResilienceMockSolver {
	return &ResilienceMockSolver{
		baseFlow:       100,
		failOnEdge:     make(map[string]bool),
		reduceFlowEdge: make(map[string]float64),
	}
}

func (m *ResilienceMockSolver) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	m.callCount++

	if m.returnError {
		return nil, errors.New("solver error")
	}

	// Проверяем какие рёбра отсутствуют
	flow := m.baseFlow
	edgeMap := make(map[string]bool)
	for _, e := range graph.Edges {
		key := edgeKey(e.From, e.To)
		edgeMap[key] = true
	}

	// Проверяем отсутствующие рёбра
	for key := range m.failOnEdge {
		if !edgeMap[key] {
			return &client.SolveResult{
				MaxFlow:   0,
				TotalCost: 0,
				Status:    commonv1.FlowStatus_FLOW_STATUS_INFEASIBLE,
				Graph:     graph,
			}, nil
		}
	}

	// Проверяем рёбра с уменьшением потока
	for key, reduction := range m.reduceFlowEdge {
		if !edgeMap[key] {
			flow -= reduction
		}
	}

	return &client.SolveResult{
		MaxFlow:   flow,
		TotalCost: flow * 0.5,
		Status:    commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		Graph:     graph,
	}, nil
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func createResilienceTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Nodes: []*commonv1.Node{
			{Id: 1, Name: "source", Type: commonv1.NodeType_NODE_TYPE_SOURCE, X: 0, Y: 0},
			{Id: 2, Name: "node2", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, X: 10, Y: 0},
			{Id: 3, Name: "node3", Type: commonv1.NodeType_NODE_TYPE_INTERSECTION, X: 10, Y: 10},
			{Id: 4, Name: "sink", Type: commonv1.NodeType_NODE_TYPE_SINK, X: 20, Y: 5},
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

func TestNewResilienceEngine(t *testing.T) {
	t.Run("with_client", func(t *testing.T) {
		engine := NewResilienceEngine(&client.SolverClient{})
		assert.NotNil(t, engine)
	})

	t.Run("with_nil_client", func(t *testing.T) {
		engine := NewResilienceEngine(nil)
		assert.NotNil(t, engine)
		assert.Nil(t, engine.solverClient)
	})
}

// ============================================================
// ANALYZE RESILIENCE TESTS
// ============================================================

func TestResilienceEngine_AnalyzeResilience_Success(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := createResilienceTestGraph()
	config := &simulationv1.ResilienceConfig{
		MaxFailuresToTest: 1,
	}

	result, err := engine.AnalyzeResilience(ctx, graph, config, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Metrics)
	assert.NotNil(t, result.NMinusOne)
}

func TestResilienceEngine_AnalyzeResilience_WithSPOF(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()
	// Помечаем ребро как критическое
	mockSolver.failOnEdge["1->2"] = true

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := createResilienceTestGraph()

	result, err := engine.AnalyzeResilience(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Должны быть обнаружены слабости
	assert.NotEmpty(t, result.Weaknesses)
}

func TestResilienceEngine_AnalyzeResilience_SolverError(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()
	mockSolver.returnError = true

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := createResilienceTestGraph()

	result, err := engine.AnalyzeResilience(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ============================================================
// N-1 ANALYSIS TESTS
// ============================================================

func TestResilienceEngine_PerformN1Analysis(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := createResilienceTestGraph()
	baseResult := &client.SolveResult{
		MaxFlow:   100,
		TotalCost: 50,
		Status:    commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}

	result := engine.performN1Analysis(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, baseResult)

	assert.NotNil(t, result)
	assert.NotNil(t, result.analysis)
	assert.GreaterOrEqual(t, result.analysis.ScenariosTested, int32(0))
}

func TestResilienceEngine_PerformN1Analysis_WithCriticalEdge(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()
	mockSolver.reduceFlowEdge["1->2"] = 50 // Removing this edge reduces flow significantly

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := createResilienceTestGraph()
	baseResult := &client.SolveResult{
		MaxFlow:   100,
		TotalCost: 50,
		Status:    commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}

	result := engine.performN1Analysis(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, baseResult)

	assert.NotNil(t, result)
	assert.Greater(t, result.analysis.WorstCaseFlowReduction, 0.0)
}

// ============================================================
// REMOVE EDGE TESTS
// ============================================================

func TestResilienceEngine_RemoveEdge(t *testing.T) {
	engine := NewResilienceEngine(nil)
	graph := createResilienceTestGraph()

	result := engine.removeEdge(graph, 1, 2)

	// Проверяем что ребро удалено
	assert.Equal(t, 3, len(result.Edges))
	for _, e := range result.Edges {
		assert.False(t, e.From == 1 && e.To == 2)
	}

	// Оригинальный граф не изменился
	assert.Equal(t, 4, len(graph.Edges))
}

func TestResilienceEngine_RemoveEdge_NonExistent(t *testing.T) {
	engine := NewResilienceEngine(nil)
	graph := createResilienceTestGraph()

	result := engine.removeEdge(graph, 99, 100)

	// Ничего не удалено
	assert.Equal(t, 4, len(result.Edges))
}

// ============================================================
// ESTIMATE MIN CUT TESTS
// ============================================================

func TestResilienceEngine_EstimateMinCut(t *testing.T) {
	engine := NewResilienceEngine(nil)

	t.Run("with_failed_scenarios", func(t *testing.T) {
		graph := createResilienceTestGraph()
		minCut := engine.estimateMinCut(graph, 1)
		assert.Equal(t, int32(1), minCut)
	})

	t.Run("no_failed_scenarios", func(t *testing.T) {
		graph := createResilienceTestGraph()
		minCut := engine.estimateMinCut(graph, 0)
		assert.Greater(t, minCut, int32(0))
	})
}

// ============================================================
// IDENTIFY WEAKNESSES TESTS
// ============================================================

func TestResilienceEngine_IdentifyWeaknesses(t *testing.T) {
	engine := NewResilienceEngine(nil)

	t.Run("with_spof", func(t *testing.T) {
		n1 := &n1Result{
			spofEdges: []*commonv1.EdgeKey{
				{From: 1, To: 2},
			},
			flowRobustness:  0.8,
			redundancyLevel: 2.0,
		}
		graph := createResilienceTestGraph()

		weaknesses := engine.identifyWeaknesses(n1, graph)

		assert.NotEmpty(t, weaknesses)
		hasSPOF := false
		for _, w := range weaknesses {
			if w.Type == simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE {
				hasSPOF = true
			}
		}
		assert.True(t, hasSPOF)
	})

	t.Run("low_flow_robustness", func(t *testing.T) {
		n1 := &n1Result{
			spofEdges:       []*commonv1.EdgeKey{},
			flowRobustness:  0.5, // < 0.7
			redundancyLevel: 2.0,
		}
		graph := createResilienceTestGraph()

		weaknesses := engine.identifyWeaknesses(n1, graph)

		hasBottleneck := false
		for _, w := range weaknesses {
			if w.Type == simulationv1.WeaknessType_WEAKNESS_TYPE_CAPACITY_BOTTLENECK {
				hasBottleneck = true
			}
		}
		assert.True(t, hasBottleneck)
	})

	t.Run("low_redundancy", func(t *testing.T) {
		n1 := &n1Result{
			spofEdges:       []*commonv1.EdgeKey{},
			flowRobustness:  0.9,
			redundancyLevel: 1.0, // < 1.5
		}
		graph := createResilienceTestGraph()

		weaknesses := engine.identifyWeaknesses(n1, graph)

		hasNoRedundancy := false
		for _, w := range weaknesses {
			if w.Type == simulationv1.WeaknessType_WEAKNESS_TYPE_NO_REDUNDANCY {
				hasNoRedundancy = true
			}
		}
		assert.True(t, hasNoRedundancy)
	})
}

// ============================================================
// GEOGRAPHIC CONCENTRATION TESTS
// ============================================================

func TestResilienceEngine_HasGeographicConcentration(t *testing.T) {
	engine := NewResilienceEngine(nil)

	tests := []struct {
		name     string
		graph    *commonv1.Graph
		expected bool
	}{
		{
			name: "concentrated_nodes",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, X: 100, Y: 100},
					{Id: 2, X: 101, Y: 100},
					{Id: 3, X: 100, Y: 101},
					{Id: 4, X: 101, Y: 101},
					{Id: 5, X: 100.5, Y: 100.5},
				},
			},
			expected: true,
		},
		{
			name: "distributed_nodes",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, X: 0, Y: 0},
					{Id: 2, X: 100, Y: 0},
					{Id: 3, X: 0, Y: 100},
					{Id: 4, X: 100, Y: 100},
				},
			},
			expected: false,
		},
		{
			name: "too_few_nodes",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, X: 1, Y: 1},
					{Id: 2, X: 2, Y: 2},
				},
			},
			expected: false,
		},
		{
			name: "nodes_without_coordinates",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, X: 0, Y: 0},
					{Id: 2, X: 0, Y: 0},
					{Id: 3, X: 0, Y: 0},
					{Id: 4, X: 1, Y: 1},
				},
			},
			expected: false,
		},
		{
			name:     "nil_graph",
			graph:    nil,
			expected: false,
		},
		{
			name: "empty_nodes",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{},
			},
			expected: false,
		},
		{
			name: "moderately_concentrated",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, X: 50, Y: 50},
					{Id: 2, X: 52, Y: 50},
					{Id: 3, X: 50, Y: 52},
					{Id: 4, X: 55, Y: 55},
				},
			},
			expected: true, // avgDist < 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.hasGeographicConcentration(tt.graph)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================
// RESILIENCE METRICS TESTS
// ============================================================

func TestResilienceMetrics(t *testing.T) {
	metrics := &simulationv1.ResilienceMetrics{
		OverallScore:           0.85,
		ConnectivityRobustness: 0.9,
		FlowRobustness:         0.8,
		RedundancyLevel:        2.0,
		MinCutSize:             2,
	}

	assert.Equal(t, 0.85, metrics.OverallScore)
	assert.Equal(t, 0.9, metrics.ConnectivityRobustness)
	assert.Equal(t, 0.8, metrics.FlowRobustness)
	assert.Equal(t, 2.0, metrics.RedundancyLevel)
	assert.Equal(t, int32(2), metrics.MinCutSize)
}

func TestNMinusOneAnalysis(t *testing.T) {
	analysis := &simulationv1.NMinusOneAnalysis{
		AllScenariosFeasible:   true,
		WorstCaseFlowReduction: 10.0,
		MostCriticalEdge:       &commonv1.EdgeKey{From: 1, To: 2},
		ScenariosTested:        4,
		ScenariosFailed:        0,
	}

	assert.True(t, analysis.AllScenariosFeasible)
	assert.Equal(t, 10.0, analysis.WorstCaseFlowReduction)
	assert.NotNil(t, analysis.MostCriticalEdge)
	assert.Equal(t, int32(4), analysis.ScenariosTested)
	assert.Equal(t, int32(0), analysis.ScenariosFailed)
}

func TestResilienceWeakness(t *testing.T) {
	weakness := &simulationv1.ResilienceWeakness{
		Type:                 simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE,
		Description:          "Critical edge found",
		Severity:             1.0,
		AffectedEdges:        []*commonv1.EdgeKey{{From: 1, To: 2}},
		MitigationSuggestion: "Add redundant path",
	}

	assert.Equal(t, simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE, weakness.Type)
	assert.NotEmpty(t, weakness.Description)
	assert.Equal(t, 1.0, weakness.Severity)
	assert.Len(t, weakness.AffectedEdges, 1)
	assert.NotEmpty(t, weakness.MitigationSuggestion)
}

// ============================================================
// EDGE CASES
// ============================================================

func TestResilienceEngine_AnalyzeResilience_EmptyGraph(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   2,
		Nodes:    []*commonv1.Node{},
		Edges:    []*commonv1.Edge{},
	}

	result, err := engine.AnalyzeResilience(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResilienceEngine_AnalyzeResilience_SingleEdge(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewResilienceMockSolver()
	mockSolver.failOnEdge["1->2"] = true

	engine := &ResilienceEngine{solverClient: mockSolver}

	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   2,
		Nodes: []*commonv1.Node{
			{Id: 1},
			{Id: 2},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 100},
		},
	}

	result, err := engine.AnalyzeResilience(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Единственное ребро - критическое
}
