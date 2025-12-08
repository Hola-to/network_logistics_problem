package v1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/tests/integration/testutil"
)

func TestAnalyticsService_CalculateCost(t *testing.T) {
	client := SetupAnalyticsClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	tests := []struct {
		name     string
		graph    *commonv1.Graph
		options  *analyticsv1.CostOptions
		wantCost bool
	}{
		{
			name:     "simple cost calculation",
			graph:    CreateSolvedGraph(),
			options:  nil,
			wantCost: true,
		},
		{
			name:  "cost with options",
			graph: CreateSolvedGraph(),
			options: &analyticsv1.CostOptions{
				Currency:          "USD",
				IncludeFixedCosts: true,
			},
			wantCost: true,
		},
		{
			name:  "cost with multipliers",
			graph: CreateSolvedGraph(),
			options: &analyticsv1.CostOptions{
				Currency: "EUR",
				CostMultipliers: map[string]float64{
					"ROAD_TYPE_HIGHWAY":  1.5,
					"ROAD_TYPE_REGIONAL": 1.2,
				},
			},
			wantCost: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.CalculateCost(ctx, &analyticsv1.CalculateCostRequest{
				Graph:   tt.graph,
				Options: tt.options,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.wantCost {
				assert.Greater(t, resp.TotalCost, 0.0)
			}

			assert.NotEmpty(t, resp.Currency)
		})
	}
}

func TestAnalyticsService_FindBottlenecks(t *testing.T) {
	client := SetupAnalyticsClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.FindBottlenecks(ctx, &analyticsv1.FindBottlenecksRequest{
		Graph:                CreateSolvedGraph(),
		UtilizationThreshold: 0.8,
		TopN:                 5,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Bottlenecks are edges with high utilization
	for _, b := range resp.Bottlenecks {
		assert.NotNil(t, b.Edge)
		assert.GreaterOrEqual(t, b.Utilization, 0.8)
	}
}

func TestAnalyticsService_AnalyzeFlow(t *testing.T) {
	client := SetupAnalyticsClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.AnalyzeFlow(ctx, &analyticsv1.AnalyzeFlowRequest{
		Graph: CreateSolvedGraph(),
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
			SuggestImprovements: true,
			BottleneckThreshold: 0.9,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Check flow statistics
	if resp.FlowStats != nil {
		assert.GreaterOrEqual(t, resp.FlowStats.TotalFlow, 0.0)
	}

	// Check graph statistics
	if resp.GraphStats != nil {
		assert.Greater(t, resp.GraphStats.NodeCount, int64(0))
		assert.Greater(t, resp.GraphStats.EdgeCount, int64(0))
	}

	// Check cost analysis
	if resp.Cost != nil {
		assert.NotEmpty(t, resp.Cost.Currency)
	}

	// Check efficiency report
	if resp.Efficiency != nil {
		assert.NotEmpty(t, resp.Efficiency.Grade)
	}
}

func TestAnalyticsService_CompareScenarios(t *testing.T) {
	client := SetupAnalyticsClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	baseline := CreateSolvedGraph()

	// Create modified scenario
	scenario1 := CreateSolvedGraph()
	scenario1.Edges[0].Capacity = 20 // Increase capacity

	scenario2 := CreateSolvedGraph()
	scenario2.Edges[0].Capacity = 5 // Decrease capacity

	resp, err := client.CompareScenarios(ctx, &analyticsv1.CompareScenariosRequest{
		Baseline:      baseline,
		Scenarios:     []*commonv1.Graph{scenario1, scenario2},
		ScenarioNames: []string{"Increased Capacity", "Decreased Capacity"},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 2)
	assert.NotEmpty(t, resp.ComparisonSummary)
}
