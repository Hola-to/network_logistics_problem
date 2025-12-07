// services/report-svc/internal/generator/types_test.go
package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
)

func TestConvertBottlenecks(t *testing.T) {
	tests := []struct {
		name     string
		input    []*analyticsv1.Bottleneck
		expected []*BottleneckData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []*BottleneckData{},
		},
		{
			name:     "empty input",
			input:    []*analyticsv1.Bottleneck{},
			expected: []*BottleneckData{},
		},
		{
			name: "skip nil bottleneck",
			input: []*analyticsv1.Bottleneck{
				nil,
				{
					Edge:        &commonv1.Edge{From: 1, To: 2},
					Utilization: 0.95,
					ImpactScore: 0.8,
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
				},
			},
			expected: []*BottleneckData{
				{
					From:        1,
					To:          2,
					Utilization: 0.95,
					ImpactScore: 0.8,
					Severity:    "BOTTLENECK_SEVERITY_HIGH",
				},
			},
		},
		{
			name: "skip bottleneck with nil edge",
			input: []*analyticsv1.Bottleneck{
				{
					Edge:        nil,
					Utilization: 0.9,
				},
			},
			expected: []*BottleneckData{},
		},
		{
			name: "multiple bottlenecks",
			input: []*analyticsv1.Bottleneck{
				{
					Edge:        &commonv1.Edge{From: 1, To: 2},
					Utilization: 0.95,
					ImpactScore: 0.8,
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
				},
				{
					Edge:        &commonv1.Edge{From: 3, To: 4},
					Utilization: 1.0,
					ImpactScore: 1.0,
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
				},
			},
			expected: []*BottleneckData{
				{
					From:        1,
					To:          2,
					Utilization: 0.95,
					ImpactScore: 0.8,
					Severity:    "BOTTLENECK_SEVERITY_HIGH",
				},
				{
					From:        3,
					To:          4,
					Utilization: 1.0,
					ImpactScore: 1.0,
					Severity:    "BOTTLENECK_SEVERITY_CRITICAL",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertBottlenecks(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertRecommendations(t *testing.T) {
	tests := []struct {
		name     string
		input    []*analyticsv1.Recommendation
		expected []*RecommendationData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []*RecommendationData{},
		},
		{
			name:     "empty input",
			input:    []*analyticsv1.Recommendation{},
			expected: []*RecommendationData{},
		},
		{
			name: "skip nil recommendation",
			input: []*analyticsv1.Recommendation{
				nil,
				{
					Type:                 "increase_capacity",
					Description:          "Increase capacity of edge 1->2",
					EstimatedImprovement: 15.5,
					EstimatedCost:        1000.0,
				},
			},
			expected: []*RecommendationData{
				{
					Type:                 "increase_capacity",
					Description:          "Increase capacity of edge 1->2",
					EstimatedImprovement: 15.5,
					EstimatedCost:        1000.0,
				},
			},
		},
		{
			name: "with affected edge",
			input: []*analyticsv1.Recommendation{
				{
					Type:        "add_edge",
					Description: "Add parallel route",
					AffectedEdge: &commonv1.EdgeKey{
						From: 5,
						To:   6,
					},
					EstimatedImprovement: 20.0,
					EstimatedCost:        5000.0,
				},
			},
			expected: []*RecommendationData{
				{
					Type:                 "add_edge",
					Description:          "Add parallel route",
					AffectedEdgeFrom:     5,
					AffectedEdgeTo:       6,
					EstimatedImprovement: 20.0,
					EstimatedCost:        5000.0,
				},
			},
		},
		{
			name: "without affected edge",
			input: []*analyticsv1.Recommendation{
				{
					Type:                 "optimize_routes",
					Description:          "General optimization",
					AffectedEdge:         nil,
					EstimatedImprovement: 10.0,
				},
			},
			expected: []*RecommendationData{
				{
					Type:                 "optimize_routes",
					Description:          "General optimization",
					AffectedEdgeFrom:     0,
					AffectedEdgeTo:       0,
					EstimatedImprovement: 10.0,
					EstimatedCost:        0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertRecommendations(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertEfficiency(t *testing.T) {
	tests := []struct {
		name     string
		input    *analyticsv1.EfficiencyReport
		expected *EfficiencyData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "full efficiency report",
			input: &analyticsv1.EfficiencyReport{
				OverallEfficiency:   0.85,
				CapacityUtilization: 0.75,
				UnusedEdgesCount:    5,
				SaturatedEdgesCount: 3,
				Grade:               "B",
			},
			expected: &EfficiencyData{
				OverallEfficiency:   0.85,
				CapacityUtilization: 0.75,
				UnusedEdges:         5,
				SaturatedEdges:      3,
				Grade:               "B",
			},
		},
		{
			name: "zero values",
			input: &analyticsv1.EfficiencyReport{
				OverallEfficiency:   0,
				CapacityUtilization: 0,
				UnusedEdgesCount:    0,
				SaturatedEdgesCount: 0,
				Grade:               "",
			},
			expected: &EfficiencyData{
				OverallEfficiency:   0,
				CapacityUtilization: 0,
				UnusedEdges:         0,
				SaturatedEdges:      0,
				Grade:               "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertEfficiency(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertCostBreakdown(t *testing.T) {
	tests := []struct {
		name     string
		input    *analyticsv1.CostBreakdown
		expected *CostBreakdownData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "full breakdown",
			input: &analyticsv1.CostBreakdown{
				TransportCost:  1000.0,
				FixedCost:      500.0,
				HandlingCost:   200.0,
				RoadBaseCost:   300.0,
				DiscountAmount: 50.0,
				MarkupAmount:   100.0,
				CostByRoadType: map[string]float64{
					"highway":   600.0,
					"secondary": 400.0,
				},
				CostByNodeType: map[string]float64{
					"warehouse": 300.0,
					"delivery":  200.0,
				},
				ActiveEdges: 15,
				TotalFlow:   1000.0,
			},
			expected: &CostBreakdownData{
				TransportCost:  1000.0,
				FixedCost:      500.0,
				HandlingCost:   200.0,
				RoadBaseCost:   300.0,
				DiscountAmount: 50.0,
				MarkupAmount:   100.0,
				CostByRoadType: map[string]float64{
					"highway":   600.0,
					"secondary": 400.0,
				},
				CostByNodeType: map[string]float64{
					"warehouse": 300.0,
					"delivery":  200.0,
				},
				ActiveEdges: 15,
				TotalFlow:   1000.0,
			},
		},
		{
			name: "nil maps",
			input: &analyticsv1.CostBreakdown{
				TransportCost:  100.0,
				CostByRoadType: nil,
				CostByNodeType: nil,
			},
			expected: &CostBreakdownData{
				TransportCost:  100.0,
				CostByRoadType: nil,
				CostByNodeType: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertCostBreakdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertScenarioResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []*simulationv1.ScenarioResultWithRank
		expected []*ScenarioData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []*ScenarioData{},
		},
		{
			name:     "empty input",
			input:    []*simulationv1.ScenarioResultWithRank{},
			expected: []*ScenarioData{},
		},
		{
			name: "skip nil entries",
			input: []*simulationv1.ScenarioResultWithRank{
				nil,
				{
					Result: nil,
				},
				{
					Result: &simulationv1.ScenarioResult{
						Name:      "Scenario A",
						MaxFlow:   100.0,
						TotalCost: 500.0,
					},
					VsBaseline: &simulationv1.ScenarioComparison{
						FlowChangePercent: 10.0,
						ImpactLevel:       simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM,
					},
				},
			},
			expected: []*ScenarioData{
				{
					Name:              "Scenario A",
					MaxFlow:           100.0,
					TotalCost:         500.0,
					FlowChangePercent: 10.0,
					ImpactLevel:       "IMPACT_LEVEL_MEDIUM",
				},
			},
		},
		{
			name: "without vs_baseline",
			input: []*simulationv1.ScenarioResultWithRank{
				{
					Result: &simulationv1.ScenarioResult{
						Name:      "Scenario B",
						MaxFlow:   200.0,
						TotalCost: 1000.0,
					},
					VsBaseline: nil,
				},
			},
			expected: []*ScenarioData{
				{
					Name:              "Scenario B",
					MaxFlow:           200.0,
					TotalCost:         1000.0,
					FlowChangePercent: 0,
					ImpactLevel:       "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertScenarioResults(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertMonteCarloStats(t *testing.T) {
	tests := []struct {
		name     string
		input    *simulationv1.RunMonteCarloResponse
		expected *MonteCarloData
	}{
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
		{
			name: "nil flow stats",
			input: &simulationv1.RunMonteCarloResponse{
				FlowStats: nil,
			},
			expected: nil,
		},
		{
			name: "full response",
			input: &simulationv1.RunMonteCarloResponse{
				FlowStats: &simulationv1.MonteCarloStats{
					Mean:                   150.0,
					StdDev:                 10.0,
					Min:                    120.0,
					Max:                    180.0,
					ConfidenceIntervalLow:  140.0,
					ConfidenceIntervalHigh: 160.0,
				},
				FlowPercentiles: map[string]float64{
					"p5":  125.0,
					"p50": 150.0,
					"p95": 175.0,
				},
			},
			expected: &MonteCarloData{
				MeanFlow:        150.0,
				StdDev:          10.0,
				MinFlow:         120.0,
				MaxFlow:         180.0,
				ConfidenceLevel: 140.0,
				CiLow:           140.0,
				CiHigh:          160.0,
				P5:              125.0,
				P50:             150.0,
				P95:             175.0,
			},
		},
		{
			name: "empty percentiles",
			input: &simulationv1.RunMonteCarloResponse{
				FlowStats: &simulationv1.MonteCarloStats{
					Mean:   100.0,
					StdDev: 5.0,
				},
				FlowPercentiles: map[string]float64{},
			},
			expected: &MonteCarloData{
				MeanFlow: 100.0,
				StdDev:   5.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertMonteCarloStats(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSensitivityResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []*simulationv1.SensitivityResult
		expected []*SensitivityData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []*SensitivityData{},
		},
		{
			name:     "empty input",
			input:    []*simulationv1.SensitivityResult{},
			expected: []*SensitivityData{},
		},
		{
			name: "skip nil entries",
			input: []*simulationv1.SensitivityResult{
				nil,
				{
					ParameterId:      "edge_1_2_capacity",
					Elasticity:       1.5,
					SensitivityIndex: 0.8,
					Level:            simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
				},
			},
			expected: []*SensitivityData{
				{
					ParameterId:      "edge_1_2_capacity",
					Elasticity:       1.5,
					SensitivityIndex: 0.8,
					Level:            "SENSITIVITY_LEVEL_HIGH",
				},
			},
		},
		{
			name: "multiple results",
			input: []*simulationv1.SensitivityResult{
				{
					ParameterId:      "param1",
					Elasticity:       0.5,
					SensitivityIndex: 0.3,
					Level:            simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
				},
				{
					ParameterId:      "param2",
					Elasticity:       2.0,
					SensitivityIndex: 0.9,
					Level:            simulationv1.SensitivityLevel_SENSITIVITY_LEVEL_CRITICAL,
				},
			},
			expected: []*SensitivityData{
				{
					ParameterId:      "param1",
					Elasticity:       0.5,
					SensitivityIndex: 0.3,
					Level:            "SENSITIVITY_LEVEL_LOW",
				},
				{
					ParameterId:      "param2",
					Elasticity:       2.0,
					SensitivityIndex: 0.9,
					Level:            "SENSITIVITY_LEVEL_CRITICAL",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSensitivityResults(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertResilienceMetrics(t *testing.T) {
	tests := []struct {
		name     string
		input    *simulationv1.AnalyzeResilienceResponse
		expected *ResilienceData
	}{
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
		{
			name: "nil metrics",
			input: &simulationv1.AnalyzeResilienceResponse{
				Metrics: nil,
			},
			expected: nil,
		},
		{
			name: "full response",
			input: &simulationv1.AnalyzeResilienceResponse{
				Metrics: &simulationv1.ResilienceMetrics{
					OverallScore: 0.85,
				},
				NMinusOne: &simulationv1.NMinusOneAnalysis{
					AllScenariosFeasible:   true,
					WorstCaseFlowReduction: 0.15,
				},
				Weaknesses: []*simulationv1.ResilienceWeakness{
					{Type: simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE},
					{Type: simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE},
					{Type: simulationv1.WeaknessType_WEAKNESS_TYPE_CAPACITY_BOTTLENECK},
				},
			},
			expected: &ResilienceData{
				OverallScore:           0.85,
				SinglePointsOfFailure:  2,
				WorstCaseFlowReduction: 0.15,
				NMinusOneFeasible:      true,
			},
		},
		{
			name: "without n_minus_one",
			input: &simulationv1.AnalyzeResilienceResponse{
				Metrics: &simulationv1.ResilienceMetrics{
					OverallScore: 0.7,
				},
				NMinusOne:  nil,
				Weaknesses: nil,
			},
			expected: &ResilienceData{
				OverallScore: 0.7,
			},
		},
		{
			name: "nil weakness in list",
			input: &simulationv1.AnalyzeResilienceResponse{
				Metrics: &simulationv1.ResilienceMetrics{
					OverallScore: 0.9,
				},
				Weaknesses: []*simulationv1.ResilienceWeakness{
					nil,
					{Type: simulationv1.WeaknessType_WEAKNESS_TYPE_SINGLE_POINT_OF_FAILURE},
				},
			},
			expected: &ResilienceData{
				OverallScore:          0.9,
				SinglePointsOfFailure: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertResilienceMetrics(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertFlowEdges(t *testing.T) {
	tests := []struct {
		name     string
		input    []*commonv1.FlowEdge
		expected []*EdgeFlowData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []*EdgeFlowData{},
		},
		{
			name:     "empty input",
			input:    []*commonv1.FlowEdge{},
			expected: []*EdgeFlowData{},
		},
		{
			name: "skip nil entries",
			input: []*commonv1.FlowEdge{
				nil,
				{
					From:        1,
					To:          2,
					Flow:        50.0,
					Capacity:    100.0,
					Cost:        10.0,
					Utilization: 0.5,
				},
			},
			expected: []*EdgeFlowData{
				{
					From:        1,
					To:          2,
					Flow:        50.0,
					Capacity:    100.0,
					Cost:        10.0,
					Utilization: 0.5,
				},
			},
		},
		{
			name: "multiple edges",
			input: []*commonv1.FlowEdge{
				{
					From:        1,
					To:          2,
					Flow:        100.0,
					Capacity:    100.0,
					Cost:        5.0,
					Utilization: 1.0,
				},
				{
					From:        2,
					To:          3,
					Flow:        75.0,
					Capacity:    150.0,
					Cost:        8.0,
					Utilization: 0.5,
				},
			},
			expected: []*EdgeFlowData{
				{
					From:        1,
					To:          2,
					Flow:        100.0,
					Capacity:    100.0,
					Cost:        5.0,
					Utilization: 1.0,
				},
				{
					From:        2,
					To:          3,
					Flow:        75.0,
					Capacity:    150.0,
					Cost:        8.0,
					Utilization: 0.5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertFlowEdges(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Тесты для структур данных
func TestFlowReportData(t *testing.T) {
	data := &FlowReportData{
		AlgorithmUsed: "Dinic",
		GraphStats: &commonv1.GraphStatistics{
			NodeCount: 10,
			EdgeCount: 15,
		},
		FlowStats: &commonv1.FlowStatistics{
			TotalFlow: 100.0,
			TotalCost: 500.0,
		},
	}

	require.NotNil(t, data)
	assert.Equal(t, "Dinic", data.AlgorithmUsed)
	assert.Equal(t, int64(10), data.GraphStats.NodeCount)
	assert.Equal(t, 100.0, data.FlowStats.TotalFlow)
}

func TestAnalyticsReportData(t *testing.T) {
	data := &AnalyticsReportData{
		TotalCost: 1500.0,
		Currency:  "USD",
		CostBreakdown: &CostBreakdownData{
			TransportCost: 1000.0,
			FixedCost:     500.0,
		},
		Bottlenecks: []*BottleneckData{
			{From: 1, To: 2, Utilization: 0.95},
		},
		Recommendations: []*RecommendationData{
			{Type: "increase_capacity", Description: "Test"},
		},
		Efficiency: &EfficiencyData{
			OverallEfficiency: 0.85,
			Grade:             "B",
		},
	}

	require.NotNil(t, data)
	assert.Equal(t, 1500.0, data.TotalCost)
	assert.Equal(t, "USD", data.Currency)
	assert.Len(t, data.Bottlenecks, 1)
	assert.Len(t, data.Recommendations, 1)
	assert.Equal(t, "B", data.Efficiency.Grade)
}

func TestSimulationReportData(t *testing.T) {
	data := &SimulationReportData{
		SimulationType: "Monte Carlo",
		BaselineFlow:   100.0,
		BaselineCost:   500.0,
		MonteCarlo: &MonteCarloData{
			Iterations: 1000,
			MeanFlow:   105.0,
			StdDev:     10.0,
		},
	}

	require.NotNil(t, data)
	assert.Equal(t, "Monte Carlo", data.SimulationType)
	assert.NotNil(t, data.MonteCarlo)
	assert.Equal(t, int32(1000), data.MonteCarlo.Iterations)
}
