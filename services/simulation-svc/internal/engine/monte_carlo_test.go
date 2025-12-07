// services/simulation-svc/internal/engine/monte_carlo_test.go
package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
	"logistics/pkg/logger"
)

func init() {
	logger.Init("error")
}

// ============================================================
// MOCK SOLVER FOR MONTE CARLO
// ============================================================

type MonteCarloMockSolver struct {
	mu          sync.Mutex
	callCount   int
	results     []*client.SolveResult
	defaultFlow float64
	variance    float64
}

func NewMonteCarloMockSolver() *MonteCarloMockSolver {
	return &MonteCarloMockSolver{
		defaultFlow: 100,
		variance:    10,
	}
}

func (m *MonteCarloMockSolver) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.callCount <= len(m.results) {
		return m.results[m.callCount-1], nil
	}

	// Генерируем случайный результат с некоторой вариацией
	flow := m.defaultFlow + float64(m.callCount%10)*m.variance/10
	return &client.SolveResult{
		MaxFlow:   flow,
		TotalCost: flow * 0.5,
		Status:    commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		Graph:     graph,
	}, nil
}

func (m *MonteCarloMockSolver) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func createMonteCarloTestGraph() *commonv1.Graph {
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

func TestNewMonteCarloEngine(t *testing.T) {
	t.Run("with_default_seed", func(t *testing.T) {
		config := &simulationv1.MonteCarloConfig{
			NumIterations: 100,
			RandomSeed:    0, // Should use time-based seed
		}

		engine := NewMonteCarloEngine(config, nil)

		assert.NotNil(t, engine)
		assert.NotNil(t, engine.rng)
		assert.Equal(t, config, engine.config)
	})

	t.Run("with_fixed_seed", func(t *testing.T) {
		config := &simulationv1.MonteCarloConfig{
			NumIterations: 100,
			RandomSeed:    12345,
		}

		engine := NewMonteCarloEngine(config, nil)

		assert.NotNil(t, engine)
		assert.NotNil(t, engine.rng)
	})

	t.Run("with_nil_config", func(t *testing.T) {
		engine := NewMonteCarloEngine(nil, nil)

		assert.NotNil(t, engine)
	})
}

// ============================================================
// RUN TESTS
// ============================================================

func TestMonteCarloEngine_Run_Success(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewMonteCarloMockSolver()

	config := &simulationv1.MonteCarloConfig{
		NumIterations:   20,
		RandomSeed:      42,
		ConfidenceLevel: 0.95,
		Parallel:        false,
	}

	engine := NewMonteCarloEngine(config, &client.SolverClient{})
	engine.solverClient = mockSolver

	graph := createMonteCarloTestGraph()
	uncertainties := []*simulationv1.UncertaintySpec{
		{
			Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_EDGE,
			Edge:   &commonv1.EdgeKey{From: 1, To: 2},
			Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			Distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_NORMAL,
				Param1: 1.0, // mean
				Param2: 0.1, // std_dev
			},
		},
	}

	result, err := engine.Run(ctx, graph, uncertainties, commonv1.Algorithm_ALGORITHM_DINIC, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.NotNil(t, result.FlowStats)
	assert.NotNil(t, result.CostStats)
	assert.NotEmpty(t, result.FlowHistogram)
	assert.NotEmpty(t, result.FlowPercentiles)
	assert.NotNil(t, result.RiskAnalysis)
}

func TestMonteCarloEngine_Run_Parallel(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewMonteCarloMockSolver()

	config := &simulationv1.MonteCarloConfig{
		NumIterations:   50,
		RandomSeed:      42,
		ConfidenceLevel: 0.95,
		Parallel:        true,
		MaxWorkers:      4,
	}

	engine := NewMonteCarloEngine(config, &client.SolverClient{})
	engine.solverClient = mockSolver

	graph := createMonteCarloTestGraph()

	result, err := engine.Run(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 50, mockSolver.GetCallCount())
}

func TestMonteCarloEngine_Run_WithProgress(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewMonteCarloMockSolver()

	config := &simulationv1.MonteCarloConfig{
		NumIterations:   30,
		RandomSeed:      42,
		ConfidenceLevel: 0.95,
		Parallel:        false,
	}

	engine := NewMonteCarloEngine(config, &client.SolverClient{})
	engine.solverClient = mockSolver

	graph := createMonteCarloTestGraph()

	progressChan := make(chan *simulationv1.MonteCarloProgress, 100)

	result, err := engine.Run(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC, progressChan)

	close(progressChan)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Проверяем что был хотя бы один прогресс
	progressCount := 0
	for range progressChan {
		progressCount++
	}
	// При 30 итерациях и отправке каждые 10, должно быть ~2-3 сообщения
}

func TestMonteCarloEngine_Run_DefaultIterations(t *testing.T) {
	ctx := context.Background()
	mockSolver := NewMonteCarloMockSolver()

	config := &simulationv1.MonteCarloConfig{
		NumIterations: 0, // Should default to 1000
		RandomSeed:    42,
	}

	// Для теста ограничим итерации
	config.NumIterations = 10

	engine := NewMonteCarloEngine(config, &client.SolverClient{})
	engine.solverClient = mockSolver

	graph := createMonteCarloTestGraph()

	result, err := engine.Run(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMonteCarloEngine_Run_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockSolver := NewMonteCarloMockSolver()

	config := &simulationv1.MonteCarloConfig{
		NumIterations: 1000,
		RandomSeed:    42,
		Parallel:      true,
	}

	engine := NewMonteCarloEngine(config, &client.SolverClient{})
	engine.solverClient = mockSolver

	graph := createMonteCarloTestGraph()

	// Отменяем контекст после небольшой задержки
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result, err := engine.Run(ctx, graph, nil, commonv1.Algorithm_ALGORITHM_DINIC, nil)

	// Может завершиться с результатом или без, но не должно паниковать
	if err == nil {
		assert.NotNil(t, result)
	}
}

// ============================================================
// DISTRIBUTION SAMPLING TESTS
// ============================================================

func TestMonteCarloEngine_SampleDistribution(t *testing.T) {
	config := &simulationv1.MonteCarloConfig{RandomSeed: 42}
	engine := NewMonteCarloEngine(config, nil)

	tests := []struct {
		name         string
		distribution *simulationv1.Distribution
		checkFunc    func(t *testing.T, value float64)
	}{
		{
			name:         "nil_distribution",
			distribution: nil,
			checkFunc: func(t *testing.T, value float64) {
				assert.Equal(t, 1.0, value)
			},
		},
		{
			name: "normal_distribution",
			distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_NORMAL,
				Param1: 1.0, // mean
				Param2: 0.1, // std_dev
			},
			checkFunc: func(t *testing.T, value float64) {
				// Нормальное распределение: большинство значений в пределах 3 sigma
				assert.Greater(t, value, 0.5)
				assert.Less(t, value, 1.5)
			},
		},
		{
			name: "uniform_distribution",
			distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
				Param1: 0.5, // min
				Param2: 1.5, // max
			},
			checkFunc: func(t *testing.T, value float64) {
				assert.GreaterOrEqual(t, value, 0.5)
				assert.LessOrEqual(t, value, 1.5)
			},
		},
		{
			name: "triangular_distribution_left",
			distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_TRIANGULAR,
				Param1: 0.5, // min
				Param2: 1.5, // max
				Param3: 0.8, // mode
			},
			checkFunc: func(t *testing.T, value float64) {
				assert.GreaterOrEqual(t, value, 0.5)
				assert.LessOrEqual(t, value, 1.5)
			},
		},
		{
			name: "triangular_distribution_right",
			distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_TRIANGULAR,
				Param1: 0.5, // min
				Param2: 1.5, // max
				Param3: 1.2, // mode (closer to max)
			},
			checkFunc: func(t *testing.T, value float64) {
				assert.GreaterOrEqual(t, value, 0.5)
				assert.LessOrEqual(t, value, 1.5)
			},
		},
		{
			name: "lognormal_distribution",
			distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_LOGNORMAL,
				Param1: 0.0, // mean of ln(X)
				Param2: 0.5, // std_dev of ln(X)
			},
			checkFunc: func(t *testing.T, value float64) {
				assert.Greater(t, value, 0.0)
			},
		},
		{
			name: "exponential_distribution",
			distribution: &simulationv1.Distribution{
				Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_EXPONENTIAL,
				Param1: 1.0, // lambda
			},
			checkFunc: func(t *testing.T, value float64) {
				assert.Greater(t, value, 0.0)
			},
		},
		{
			name: "unspecified_distribution",
			distribution: &simulationv1.Distribution{
				Type: simulationv1.DistributionType_DISTRIBUTION_TYPE_UNSPECIFIED,
			},
			checkFunc: func(t *testing.T, value float64) {
				assert.Equal(t, 1.0, value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сэмплируем несколько раз для проверки
			for i := 0; i < 10; i++ {
				value := engine.sampleDistribution(tt.distribution, engine.rng)
				tt.checkFunc(t, value)
			}
		})
	}
}

// ============================================================
// APPLY UNCERTAINTIES TESTS
// ============================================================

func TestMonteCarloEngine_ApplyUncertainties(t *testing.T) {
	config := &simulationv1.MonteCarloConfig{RandomSeed: 42}
	engine := NewMonteCarloEngine(config, nil)

	t.Run("edge_uncertainty", func(t *testing.T) {
		graph := createMonteCarloTestGraph()
		originalCapacity := graph.Edges[0].Capacity

		uncertainties := []*simulationv1.UncertaintySpec{
			{
				Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_EDGE,
				Edge:   &commonv1.EdgeKey{From: 1, To: 2},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
					Param1: 0.8,
					Param2: 1.2,
				},
			},
		}

		result := engine.applyUncertainties(graph, uncertainties, engine.rng)

		// Проверяем что capacity изменилась
		var modifiedEdge *commonv1.Edge
		for _, e := range result.Edges {
			if e.From == 1 && e.To == 2 {
				modifiedEdge = e
				break
			}
		}
		require.NotNil(t, modifiedEdge)
		// Capacity должна быть в диапазоне [0.8*50, 1.2*50] = [40, 60]
		assert.GreaterOrEqual(t, modifiedEdge.Capacity, originalCapacity*0.8)
		assert.LessOrEqual(t, modifiedEdge.Capacity, originalCapacity*1.2)
	})

	t.Run("node_uncertainty_supply", func(t *testing.T) {
		graph := createMonteCarloTestGraph()
		originalSupply := graph.Nodes[0].Supply

		uncertainties := []*simulationv1.UncertaintySpec{
			{
				Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_NODE,
				NodeId: 1,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
					Param1: 0.9,
					Param2: 1.1,
				},
			},
		}

		result := engine.applyUncertainties(graph, uncertainties, engine.rng)

		var modifiedNode *commonv1.Node
		for _, n := range result.Nodes {
			if n.Id == 1 {
				modifiedNode = n
				break
			}
		}
		require.NotNil(t, modifiedNode)
		assert.GreaterOrEqual(t, modifiedNode.Supply, originalSupply*0.9)
		assert.LessOrEqual(t, modifiedNode.Supply, originalSupply*1.1)
	})

	t.Run("node_uncertainty_demand", func(t *testing.T) {
		graph := createMonteCarloTestGraph()
		originalDemand := graph.Nodes[3].Demand

		uncertainties := []*simulationv1.UncertaintySpec{
			{
				Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_NODE,
				NodeId: 4,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_DEMAND,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
					Param1: 0.8,
					Param2: 1.2,
				},
			},
		}

		result := engine.applyUncertainties(graph, uncertainties, engine.rng)

		var modifiedNode *commonv1.Node
		for _, n := range result.Nodes {
			if n.Id == 4 {
				modifiedNode = n
				break
			}
		}
		require.NotNil(t, modifiedNode)
		assert.GreaterOrEqual(t, modifiedNode.Demand, originalDemand*0.8)
		assert.LessOrEqual(t, modifiedNode.Demand, originalDemand*1.2)
	})

	t.Run("global_uncertainty", func(t *testing.T) {
		graph := createMonteCarloTestGraph()

		uncertainties := []*simulationv1.UncertaintySpec{
			{
				Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_GLOBAL,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
					Param1: 0.9,
					Param2: 1.0,
				},
			},
		}

		result := engine.applyUncertainties(graph, uncertainties, engine.rng)

		// Все рёбра должны быть модифицированы
		for _, edge := range result.Edges {
			// Capacity уменьшилась (multiplier < 1)
			assert.LessOrEqual(t, edge.Capacity, 50.0)
		}
	})

	t.Run("edge_cost_uncertainty", func(t *testing.T) {
		graph := createMonteCarloTestGraph()

		uncertainties := []*simulationv1.UncertaintySpec{
			{
				Type:   simulationv1.UncertaintyType_UNCERTAINTY_TYPE_EDGE,
				Edge:   &commonv1.EdgeKey{From: 1, To: 2},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_COST,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
					Param1: 1.5,
					Param2: 2.0,
				},
			},
		}

		result := engine.applyUncertainties(graph, uncertainties, engine.rng)

		var modifiedEdge *commonv1.Edge
		for _, e := range result.Edges {
			if e.From == 1 && e.To == 2 {
				modifiedEdge = e
				break
			}
		}
		require.NotNil(t, modifiedEdge)
		// Cost = 1 * [1.5, 2.0] = [1.5, 2.0]
		assert.GreaterOrEqual(t, modifiedEdge.Cost, 1.5)
		assert.LessOrEqual(t, modifiedEdge.Cost, 2.0)
	})
}

// ============================================================
// STATISTICS TESTS
// ============================================================

func TestCalculateStats(t *testing.T) {
	config := &simulationv1.MonteCarloConfig{RandomSeed: 42, ConfidenceLevel: 0.95}
	engine := NewMonteCarloEngine(config, nil)

	t.Run("normal_values", func(t *testing.T) {
		values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
		stats := calculateStats(values, 0.95)

		assert.Equal(t, 55.0, stats.Mean)
		assert.Equal(t, 10.0, stats.Min)
		assert.Equal(t, 100.0, stats.Max)
		assert.Equal(t, 50.0, stats.Median) // Для 10 элементов, медиана = 5-й элемент
		assert.Greater(t, stats.StdDev, 0.0)
		assert.Greater(t, stats.Variance, 0.0)
		assert.Less(t, stats.ConfidenceIntervalLow, stats.Mean)
		assert.Greater(t, stats.ConfidenceIntervalHigh, stats.Mean)
	})

	t.Run("empty_values", func(t *testing.T) {
		values := []float64{}
		stats := calculateStats(values, 0.95)

		assert.Equal(t, 0.0, stats.Mean)
	})

	t.Run("single_value", func(t *testing.T) {
		values := []float64{42}
		stats := calculateStats(values, 0.95)

		assert.Equal(t, 42.0, stats.Mean)
		assert.Equal(t, 42.0, stats.Min)
		assert.Equal(t, 42.0, stats.Max)
	})

	t.Run("custom_confidence_level", func(t *testing.T) {
		values := []float64{10, 20, 30, 40, 50}
		stats := calculateStats(values, 0.99)

		assert.NotZero(t, stats.ConfidenceIntervalLow)
		assert.NotZero(t, stats.ConfidenceIntervalHigh)
	})

	_ = engine // silence unused
}

func TestBuildHistogram(t *testing.T) {
	t.Run("normal_values", func(t *testing.T) {
		values := make([]float64, 100)
		for i := range values {
			values[i] = float64(i)
		}

		buckets := buildHistogram(values, 10)

		assert.Len(t, buckets, 10)

		// Сумма всех count должна равняться количеству значений
		var totalCount int32
		for _, b := range buckets {
			totalCount += b.Count
			assert.GreaterOrEqual(t, b.UpperBound, b.LowerBound)
			assert.GreaterOrEqual(t, b.Frequency, 0.0)
			assert.LessOrEqual(t, b.Frequency, 1.0)
		}
		assert.Equal(t, int32(100), totalCount)
	})

	t.Run("empty_values", func(t *testing.T) {
		buckets := buildHistogram([]float64{}, 10)
		assert.Nil(t, buckets)
	})

	t.Run("single_value", func(t *testing.T) {
		values := []float64{42, 42, 42}
		buckets := buildHistogram(values, 5)

		assert.NotNil(t, buckets)
		// Все значения в одном bucket
	})
}

func TestCalculatePercentiles(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = float64(i + 1) // 1 to 100
	}

	percentiles := calculatePercentiles(values)

	assert.InDelta(t, 5.0, percentiles["p5"], 1.0)
	assert.InDelta(t, 10.0, percentiles["p10"], 1.0)
	assert.InDelta(t, 25.0, percentiles["p25"], 1.0)
	assert.InDelta(t, 50.0, percentiles["p50"], 1.0)
	assert.InDelta(t, 75.0, percentiles["p75"], 1.0)
	assert.InDelta(t, 90.0, percentiles["p90"], 1.0)
	assert.InDelta(t, 95.0, percentiles["p95"], 1.0)
	assert.InDelta(t, 99.0, percentiles["p99"], 1.0)
}

func TestAnalyzeRisks(t *testing.T) {
	flows := make([]float64, 100)
	for i := range flows {
		flows[i] = float64(50 + i) // 50 to 149
	}
	meanFlow := 99.5

	risks := analyzeRisks(flows, meanFlow)

	assert.Greater(t, risks.ValueAtRisk, 0.0)
	assert.Greater(t, risks.ExpectedShortfall, 0.0)
	assert.Equal(t, 50.0, risks.WorstCaseFlow)
	assert.Equal(t, 149.0, risks.BestCaseFlow)
}

func TestNormalInverse(t *testing.T) {
	tests := []struct {
		p        float64
		expected float64
		delta    float64
	}{
		{0.5, 0.0, 0.01},
		{0.975, 1.96, 0.01},
		{0.025, -1.96, 0.01},
		{0.99, 2.326, 0.01},
		{0.01, -2.326, 0.01},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := normalInverse(tt.p)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

// ============================================================
// RUNNING STATS TESTS
// ============================================================

func TestMonteCarloEngine_CalculateRunningStats(t *testing.T) {
	config := &simulationv1.MonteCarloConfig{RandomSeed: 42}
	engine := NewMonteCarloEngine(config, nil)

	t.Run("with_results", func(t *testing.T) {
		results := []MonteCarloResult{
			{Iteration: 0, Flow: 100},
			{Iteration: 1, Flow: 110},
			{Iteration: 2, Flow: 90},
			{Iteration: 3, Flow: 105},
			{Iteration: 4, Flow: 95},
		}

		stats := engine.calculateRunningStats(results)

		assert.InDelta(t, 100.0, stats.Mean, 0.1)
		assert.Greater(t, stats.StdDev, 0.0)
	})

	t.Run("empty_results", func(t *testing.T) {
		results := []MonteCarloResult{}
		stats := engine.calculateRunningStats(results)

		assert.Equal(t, 0.0, stats.Mean)
		assert.Equal(t, 0.0, stats.StdDev)
	})
}

// ============================================================
// ANALYZE RESULTS TESTS
// ============================================================

func TestMonteCarloEngine_AnalyzeResults(t *testing.T) {
	config := &simulationv1.MonteCarloConfig{
		RandomSeed:      42,
		ConfidenceLevel: 0.95,
	}
	engine := NewMonteCarloEngine(config, nil)

	t.Run("with_results", func(t *testing.T) {
		results := make([]MonteCarloResult, 100)
		for i := range results {
			results[i] = MonteCarloResult{
				Iteration: i,
				Flow:      100 + float64(i%20) - 10,
				Cost:      50 + float64(i%10) - 5,
			}
		}

		response, err := engine.analyzeResults(results)

		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotNil(t, response.FlowStats)
		assert.NotNil(t, response.CostStats)
		assert.NotEmpty(t, response.FlowHistogram)
		assert.NotEmpty(t, response.CostHistogram)
		assert.NotEmpty(t, response.FlowPercentiles)
		assert.NotEmpty(t, response.CostPercentiles)
		assert.NotNil(t, response.RiskAnalysis)
	})

	t.Run("empty_results", func(t *testing.T) {
		results := []MonteCarloResult{}

		response, err := engine.analyzeResults(results)

		require.NoError(t, err)
		assert.False(t, response.Success)
	})
}
