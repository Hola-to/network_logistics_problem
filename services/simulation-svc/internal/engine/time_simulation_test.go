// services/simulation-svc/internal/engine/time_simulation_test.go
package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
)

// ============================================================
// MOCK SOLVER FOR TIME SIMULATION
// ============================================================

type TimeSimMockSolver struct {
	callCount   int
	baseFlow    float64
	flowPerStep float64
}

func NewTimeSimMockSolver() *TimeSimMockSolver {
	return &TimeSimMockSolver{
		baseFlow:    100,
		flowPerStep: 0,
	}
}

func (m *TimeSimMockSolver) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	m.callCount++

	flow := m.baseFlow + m.flowPerStep*float64(m.callCount-1)

	// Добавляем current flow к рёбрам
	resultGraph := CloneGraph(graph)
	for _, edge := range resultGraph.Edges {
		edge.CurrentFlow = edge.Capacity * 0.8
	}

	return &client.SolveResult{
		MaxFlow:            flow,
		TotalCost:          flow * 0.5,
		AverageUtilization: 0.8,
		SaturatedEdges:     1,
		Status:             commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		Graph:              resultGraph,
	}, nil
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func createTimeSimTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
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

func TestNewTimeSimulationEngine(t *testing.T) {
	t.Run("with_client", func(t *testing.T) {
		engine := NewTimeSimulationEngine(&client.SolverClient{})
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.rng)
	})

	t.Run("with_nil_client", func(t *testing.T) {
		engine := NewTimeSimulationEngine(nil)
		assert.NotNil(t, engine)
	})
}

// ============================================================
// RUN TIME SIMULATION TESTS
// ============================================================

func TestTimeSimulationEngine_RunTimeSimulation_Success(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := &TimeSimulationEngine{
		solverClient: mockSolver,
		rng:          nil,
	}
	engine.rng = engine.rng // Will be initialized in NewTimeSimulationEngine

	req := &simulationv1.RunTimeSimulationRequest{
		Graph: createTimeSimTestGraph(),
		TimeConfig: &simulationv1.TimeSimulationConfig{
			NumSteps: 10,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	result, err := engine.RunTimeSimulation(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Len(t, result.StepResults, 10)
	assert.NotNil(t, result.Stats)
}

func TestTimeSimulationEngine_RunTimeSimulation_DefaultConfig(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := &TimeSimulationEngine{
		solverClient: mockSolver,
	}

	req := &simulationv1.RunTimeSimulationRequest{
		Graph:      createTimeSimTestGraph(),
		TimeConfig: nil, // Should use defaults
		Algorithm:  commonv1.Algorithm_ALGORITHM_DINIC,
	}

	result, err := engine.RunTimeSimulation(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.StepResults, 24) // Default
}

func TestTimeSimulationEngine_RunTimeSimulation_WithEdgePatterns(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := NewTimeSimulationEngine(&client.SolverClient{})
	engine.solverClient = mockSolver

	req := &simulationv1.RunTimeSimulationRequest{
		Graph: createTimeSimTestGraph(),
		TimeConfig: &simulationv1.TimeSimulationConfig{
			NumSteps: 5,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		},
		EdgePatterns: []*simulationv1.EdgeTimePattern{
			{
				Edge: &commonv1.EdgeKey{From: 1, To: 2},
				Pattern: &simulationv1.TimePattern{
					Type:              simulationv1.PatternType_PATTERN_TYPE_HOURLY,
					HourlyMultipliers: make([]float64, 24),
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	// Fill hourly multipliers
	for i := range req.EdgePatterns[0].Pattern.HourlyMultipliers {
		req.EdgePatterns[0].Pattern.HourlyMultipliers[i] = 1.0
	}

	result, err := engine.RunTimeSimulation(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestTimeSimulationEngine_RunTimeSimulation_WithNodePatterns(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := NewTimeSimulationEngine(&client.SolverClient{})
	engine.solverClient = mockSolver

	req := &simulationv1.RunTimeSimulationRequest{
		Graph: createTimeSimTestGraph(),
		TimeConfig: &simulationv1.TimeSimulationConfig{
			NumSteps: 5,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		},
		NodePatterns: []*simulationv1.NodeTimePattern{
			{
				NodeId: 1,
				Target: simulationv1.PatternTarget_PATTERN_TARGET_SUPPLY,
				Pattern: &simulationv1.TimePattern{
					Type: simulationv1.PatternType_PATTERN_TYPE_CONSTANT,
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	result, err := engine.RunTimeSimulation(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
}

// ============================================================
// SIMULATE PEAK LOAD TESTS
// ============================================================

func TestTimeSimulationEngine_SimulatePeakLoad_Success(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := &TimeSimulationEngine{
		solverClient: mockSolver,
	}

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:             createTimeSimTestGraph(),
		DemandMultiplier:  1.5,
		CapacityReduction: 0.8,
		Algorithm:         commonv1.Algorithm_ALGORITHM_DINIC,
	}

	result, err := engine.SimulatePeakLoad(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.NotNil(t, result.NormalResult)
	assert.NotNil(t, result.PeakResult)
	assert.NotNil(t, result.Comparison)
}

func TestTimeSimulationEngine_SimulatePeakLoad_WithAffectedNodes(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := &TimeSimulationEngine{
		solverClient: mockSolver,
	}

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:            createTimeSimTestGraph(),
		DemandMultiplier: 2.0,
		AffectedNodes:    []int64{1, 4},
		Algorithm:        commonv1.Algorithm_ALGORITHM_DINIC,
	}

	result, err := engine.SimulatePeakLoad(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestTimeSimulationEngine_SimulatePeakLoad_WithAffectedEdges(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewTimeSimMockSolver()

	engine := &TimeSimulationEngine{
		solverClient: mockSolver,
	}

	req := &simulationv1.SimulatePeakLoadRequest{
		Graph:             createTimeSimTestGraph(),
		CapacityReduction: 0.5,
		AffectedEdges: []*commonv1.EdgeKey{
			{From: 1, To: 2},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	result, err := engine.SimulatePeakLoad(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
}

// ============================================================
// GET STEP DURATION TESTS
// ============================================================

func TestTimeSimulationEngine_GetStepDuration(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	tests := []struct {
		step     simulationv1.TimeStep
		expected time.Duration
	}{
		{simulationv1.TimeStep_TIME_STEP_MINUTE, time.Minute},
		{simulationv1.TimeStep_TIME_STEP_HOUR, time.Hour},
		{simulationv1.TimeStep_TIME_STEP_DAY, 24 * time.Hour},
		{simulationv1.TimeStep_TIME_STEP_WEEK, 7 * 24 * time.Hour},
		{simulationv1.TimeStep_TIME_STEP_UNSPECIFIED, time.Hour}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.step.String(), func(t *testing.T) {
			result := engine.getStepDuration(tt.step)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================
// NORMALIZE CONFIG TESTS
// ============================================================

func TestTimeSimulationEngine_NormalizeConfig(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	t.Run("nil_config", func(t *testing.T) {
		result := engine.normalizeConfig(nil)
		assert.NotNil(t, result)
		assert.Equal(t, int32(24), result.NumSteps)
		assert.Equal(t, simulationv1.TimeStep_TIME_STEP_HOUR, result.TimeStep)
	})

	t.Run("existing_config", func(t *testing.T) {
		config := &simulationv1.TimeSimulationConfig{
			NumSteps: 10,
			TimeStep: simulationv1.TimeStep_TIME_STEP_DAY,
		}
		result := engine.normalizeConfig(config)
		assert.Equal(t, config, result)
	})
}

// ============================================================
// GET STEPS TESTS
// ============================================================

func TestTimeSimulationEngine_GetSteps(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	t.Run("positive_steps", func(t *testing.T) {
		config := &simulationv1.TimeSimulationConfig{NumSteps: 10}
		result := engine.getSteps(config)
		assert.Equal(t, 10, result)
	})

	t.Run("zero_steps", func(t *testing.T) {
		config := &simulationv1.TimeSimulationConfig{NumSteps: 0}
		result := engine.getSteps(config)
		assert.Equal(t, 24, result)
	})

	t.Run("negative_steps", func(t *testing.T) {
		config := &simulationv1.TimeSimulationConfig{NumSteps: -5}
		result := engine.getSteps(config)
		assert.Equal(t, 24, result)
	})
}

// ============================================================
// GET START TIME TESTS
// ============================================================

func TestTimeSimulationEngine_GetStartTime(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	t.Run("with_start_time", func(t *testing.T) {
		expectedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		config := &simulationv1.TimeSimulationConfig{
			StartTime: timestamppb.New(expectedTime),
		}
		result := engine.getStartTime(config)
		assert.Equal(t, expectedTime, result)
	})

	t.Run("nil_start_time", func(t *testing.T) {
		config := &simulationv1.TimeSimulationConfig{}
		before := time.Now()
		result := engine.getStartTime(config)
		after := time.Now()
		assert.True(t, result.After(before) || result.Equal(before))
		assert.True(t, result.Before(after) || result.Equal(after))
	})
}

// ============================================================
// GET MULTIPLIER TESTS
// ============================================================

func TestTimeSimulationEngine_GetMultiplier(t *testing.T) {
	engine := NewTimeSimulationEngine(&client.SolverClient{})

	t.Run("nil_pattern", func(t *testing.T) {
		result := engine.getMultiplier(nil, 0, time.Now())
		assert.Equal(t, 1.0, result)
	})

	t.Run("constant_pattern", func(t *testing.T) {
		pattern := &simulationv1.TimePattern{
			Type: simulationv1.PatternType_PATTERN_TYPE_CONSTANT,
		}
		result := engine.getMultiplier(pattern, 0, time.Now())
		assert.Equal(t, 1.0, result)
	})

	t.Run("hourly_pattern", func(t *testing.T) {
		multipliers := make([]float64, 24)
		for i := range multipliers {
			multipliers[i] = float64(i) / 10.0
		}
		pattern := &simulationv1.TimePattern{
			Type:              simulationv1.PatternType_PATTERN_TYPE_HOURLY,
			HourlyMultipliers: multipliers,
		}
		testTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		result := engine.getMultiplier(pattern, 0, testTime)
		assert.Equal(t, 1.0, result) // Hour 10 -> multiplier 1.0
	})

	t.Run("daily_pattern", func(t *testing.T) {
		multipliers := []float64{0.8, 1.0, 1.0, 1.0, 1.0, 0.9, 0.8}
		pattern := &simulationv1.TimePattern{
			Type:             simulationv1.PatternType_PATTERN_TYPE_DAILY,
			DailyMultipliers: multipliers,
		}
		// Sunday
		testTime := time.Date(2024, 1, 7, 12, 0, 0, 0, time.UTC)
		result := engine.getMultiplier(pattern, 0, testTime)
		assert.Equal(t, 0.8, result)
	})

	t.Run("custom_pattern", func(t *testing.T) {
		pattern := &simulationv1.TimePattern{
			Type: simulationv1.PatternType_PATTERN_TYPE_CUSTOM,
			CustomPoints: []*simulationv1.TimePoint{
				{Step: 0, Multiplier: 0.5},
				{Step: 1, Multiplier: 0.8},
				{Step: 2, Multiplier: 1.2},
			},
		}
		result := engine.getMultiplier(pattern, 1, time.Now())
		assert.Equal(t, 0.8, result)
	})

	t.Run("custom_pattern_no_match", func(t *testing.T) {
		pattern := &simulationv1.TimePattern{
			Type: simulationv1.PatternType_PATTERN_TYPE_CUSTOM,
			CustomPoints: []*simulationv1.TimePoint{
				{Step: 0, Multiplier: 0.5},
			},
		}
		result := engine.getMultiplier(pattern, 5, time.Now())
		assert.Equal(t, 1.0, result)
	})

	t.Run("random_normal_pattern", func(t *testing.T) {
		pattern := &simulationv1.TimePattern{
			Type:     simulationv1.PatternType_PATTERN_TYPE_RANDOM_NORMAL,
			Mean:     1.0,
			StdDev:   0.1,
			MinValue: 0.5,
			MaxValue: 1.5,
		}
		result := engine.getMultiplier(pattern, 0, time.Now())
		assert.GreaterOrEqual(t, result, 0.5)
		assert.LessOrEqual(t, result, 1.5)
	})

	t.Run("random_uniform_pattern", func(t *testing.T) {
		pattern := &simulationv1.TimePattern{
			Type:     simulationv1.PatternType_PATTERN_TYPE_RANDOM_UNIFORM,
			MinValue: 0.8,
			MaxValue: 1.2,
		}
		result := engine.getMultiplier(pattern, 0, time.Now())
		assert.GreaterOrEqual(t, result, 0.8)
		assert.LessOrEqual(t, result, 1.2)
	})
}

// ============================================================
// APPLY TIME PATTERNS TESTS
// ============================================================

func TestTimeSimulationEngine_ApplyTimePatterns(t *testing.T) {
	engine := NewTimeSimulationEngine(&client.SolverClient{})

	t.Run("with_edge_patterns", func(t *testing.T) {
		graph := createTimeSimTestGraph()
		edgePatterns := []*simulationv1.EdgeTimePattern{
			{
				Edge: &commonv1.EdgeKey{From: 1, To: 2},
				Pattern: &simulationv1.TimePattern{
					Type: simulationv1.PatternType_PATTERN_TYPE_CONSTANT,
				},
			},
		}

		result := engine.applyTimePatterns(graph, 0, time.Now(), edgePatterns, nil)

		assert.NotNil(t, result)
		assert.Equal(t, len(graph.Edges), len(result.Edges))
	})

	t.Run("with_node_patterns", func(t *testing.T) {
		graph := createTimeSimTestGraph()
		nodePatterns := []*simulationv1.NodeTimePattern{
			{
				NodeId: 1,
				Target: simulationv1.PatternTarget_PATTERN_TARGET_DEMAND,
				Pattern: &simulationv1.TimePattern{
					Type: simulationv1.PatternType_PATTERN_TYPE_CONSTANT,
				},
			},
		}

		result := engine.applyTimePatterns(graph, 0, time.Now(), nil, nodePatterns)

		assert.NotNil(t, result)
	})
}

// ============================================================
// FIND BOTTLENECKS TESTS
// ============================================================

func TestTimeSimulationEngine_FindBottlenecks(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	t.Run("with_bottlenecks", func(t *testing.T) {
		graph := &commonv1.Graph{
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 100, CurrentFlow: 98},  // 98% utilized
				{From: 2, To: 3, Capacity: 100, CurrentFlow: 50},  // 50% utilized
				{From: 3, To: 4, Capacity: 100, CurrentFlow: 100}, // 100% utilized
			},
		}

		bottlenecks := engine.findBottlenecks(graph)

		assert.Len(t, bottlenecks, 2) // 98% and 100% are bottlenecks (>= 95%)
	})

	t.Run("no_bottlenecks", func(t *testing.T) {
		graph := &commonv1.Graph{
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 100, CurrentFlow: 50},
				{From: 2, To: 3, Capacity: 100, CurrentFlow: 60},
			},
		}

		bottlenecks := engine.findBottlenecks(graph)

		assert.Empty(t, bottlenecks)
	})

	t.Run("nil_graph", func(t *testing.T) {
		bottlenecks := engine.findBottlenecks(nil)
		assert.Nil(t, bottlenecks)
	})
}

// ============================================================
// AGGREGATE TIME RESULTS TESTS
// ============================================================

func TestTimeSimulationEngine_AggregateTimeResults(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	t.Run("with_results", func(t *testing.T) {
		results := []*simulationv1.TimeStepResult{
			{Step: 0, MaxFlow: 100, TotalCost: 50, Bottlenecks: nil},
			{Step: 1, MaxFlow: 110, TotalCost: 55, Bottlenecks: []*commonv1.EdgeKey{{From: 1, To: 2}}},
			{Step: 2, MaxFlow: 90, TotalCost: 45, Bottlenecks: nil},
		}

		stats := engine.aggregateTimeResults(results)

		assert.Equal(t, 90.0, stats.MinFlow)
		assert.Equal(t, 110.0, stats.MaxFlow)
		assert.Equal(t, 100.0, stats.AvgFlow)
		assert.Equal(t, 45.0, stats.MinCost)
		assert.Equal(t, 55.0, stats.MaxCost)
		assert.Equal(t, 50.0, stats.AvgCost)
		assert.Equal(t, int32(3), stats.TotalSteps)
		assert.Equal(t, int32(1), stats.StepsWithBottlenecks)
	})

	t.Run("empty_results", func(t *testing.T) {
		stats := engine.aggregateTimeResults([]*simulationv1.TimeStepResult{})
		assert.NotNil(t, stats)
	})
}

// ============================================================
// DESCRIBE IMPACT TESTS
// ============================================================

func TestTimeSimulationEngine_DescribeImpact(t *testing.T) {
	engine := NewTimeSimulationEngine(nil)

	tests := []struct {
		impact   float64
		contains string
	}{
		{35, "Критическое"},
		{20, "Значительное"},
		{8, "Умеренное"},
		{3, "Незначительное"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := engine.describeImpact(tt.impact)
			assert.Contains(t, result, tt.contains)
		})
	}
}

// ============================================================
// HELPER FUNCTIONS TESTS
// ============================================================

func TestContainsNode(t *testing.T) {
	nodes := []int64{1, 2, 3, 5, 8}

	assert.True(t, containsNode(nodes, 1))
	assert.True(t, containsNode(nodes, 5))
	assert.False(t, containsNode(nodes, 4))
	assert.False(t, containsNode(nodes, 0))
}

func TestContainsEdge(t *testing.T) {
	edges := []*commonv1.EdgeKey{
		{From: 1, To: 2},
		{From: 2, To: 3},
	}

	assert.True(t, containsEdge(edges, 1, 2))
	assert.True(t, containsEdge(edges, 2, 3))
	assert.False(t, containsEdge(edges, 1, 3))
	assert.False(t, containsEdge(edges, 3, 2))
}

func TestCalculateChangePercent(t *testing.T) {
	tests := []struct {
		baseline float64
		modified float64
		expected float64
	}{
		{100, 110, 10},
		{100, 90, -10},
		{100, 100, 0},
		{0, 50, 100},
		{0, 0, 0},
	}

	for _, tt := range tests {
		result := calculateChangePercent(tt.baseline, tt.modified)
		assert.Equal(t, tt.expected, result)
	}
}

// ============================================================
// CRITICAL PERIOD TRACKER TESTS
// ============================================================

func TestCriticalPeriodTracker(t *testing.T) {
	t.Run("track_critical_period", func(t *testing.T) {
		tracker := newCriticalPeriodTracker()
		stats := &timeSimulationStats{maxFlow: 100}
		stepDuration := time.Hour

		// Нормальный шаг
		normalStep := &simulationv1.TimeStepResult{
			MaxFlow:     95,
			Bottlenecks: nil,
		}
		tracker.track(0, time.Now(), normalStep, stats, stepDuration)
		assert.Nil(t, tracker.current)

		// Критический шаг (поток < 80%)
		criticalStep := &simulationv1.TimeStepResult{
			MaxFlow:     70,
			Bottlenecks: []*commonv1.EdgeKey{{From: 1, To: 2}},
		}
		tracker.track(1, time.Now(), criticalStep, stats, stepDuration)
		assert.NotNil(t, tracker.current)

		// Ещё один критический
		tracker.track(2, time.Now(), criticalStep, stats, stepDuration)

		// Возврат к нормальному
		tracker.track(3, time.Now(), normalStep, stats, stepDuration)
		assert.Nil(t, tracker.current)
		assert.Len(t, tracker.periods, 1)
	})

	t.Run("finalize_open_period", func(t *testing.T) {
		tracker := newCriticalPeriodTracker()
		stats := &timeSimulationStats{maxFlow: 100}
		stepDuration := time.Hour
		startTime := time.Now()

		criticalStep := &simulationv1.TimeStepResult{
			MaxFlow:     70,
			Bottlenecks: []*commonv1.EdgeKey{{From: 1, To: 2}, {From: 2, To: 3}, {From: 3, To: 4}},
		}
		tracker.track(0, startTime, criticalStep, stats, stepDuration)

		tracker.finalize(5, startTime, stepDuration)

		assert.Nil(t, tracker.current)
		assert.Len(t, tracker.periods, 1)
	})
}

// ============================================================
// TIME SIMULATION STATS TESTS
// ============================================================

func TestTimeSimulationStats(t *testing.T) {
	stats := newTimeSimulationStats()

	step1 := &simulationv1.TimeStepResult{
		MaxFlow:     100,
		TotalCost:   50,
		Bottlenecks: nil,
	}
	stats.update(step1)

	step2 := &simulationv1.TimeStepResult{
		MaxFlow:     110,
		TotalCost:   55,
		Bottlenecks: []*commonv1.EdgeKey{{From: 1, To: 2}},
	}
	stats.update(step2)

	step3 := &simulationv1.TimeStepResult{
		MaxFlow:     90,
		TotalCost:   45,
		Bottlenecks: nil,
	}
	stats.update(step3)

	result := stats.finalize(3)

	assert.Equal(t, 90.0, result.MinFlow)
	assert.Equal(t, 110.0, result.MaxFlow)
	assert.InDelta(t, 100.0, result.AvgFlow, 0.1)
	assert.Greater(t, result.StdDevFlow, 0.0)
	assert.Equal(t, 45.0, result.MinCost)
	assert.Equal(t, 55.0, result.MaxCost)
	assert.Equal(t, int32(1), result.StepsWithBottlenecks)
}
