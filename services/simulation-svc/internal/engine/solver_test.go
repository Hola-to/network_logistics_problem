package engine

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSolverEngine(t *testing.T) {
	engine := NewSolverEngine(nil)
	require.NotNil(t, engine)
}

func TestToScenarioResult(t *testing.T) {
	tests := []struct {
		name   string
		result *client.SolveResult
		scName string
	}{
		{
			name:   "nil result",
			result: nil,
			scName: "test",
		},
		{
			name: "valid result",
			result: &client.SolveResult{
				MaxFlow:            100,
				TotalCost:          50,
				AverageUtilization: 0.75,
				SaturatedEdges:     5,
				ActivePaths:        3,
				Status:             commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
			},
			scName: "baseline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToScenarioResult(tt.result, tt.scName)
			require.NotNil(t, result)
			assert.Equal(t, tt.scName, result.Name)

			if tt.result != nil {
				assert.Equal(t, tt.result.MaxFlow, result.MaxFlow)
				assert.Equal(t, tt.result.TotalCost, result.TotalCost)
				assert.Equal(t, tt.result.AverageUtilization, result.AverageUtilization)
				assert.Equal(t, tt.result.SaturatedEdges, result.SaturatedEdges)
				assert.Equal(t, tt.result.ActivePaths, result.ActivePaths)
				assert.Equal(t, tt.result.Status, result.Status)
			}
		})
	}
}

func TestCompareResults(t *testing.T) {
	tests := []struct {
		name     string
		baseline *client.SolveResult
		modified *client.SolveResult
		validate func(t *testing.T, comparison *simulationv1.ScenarioComparison)
	}{
		{
			name:     "nil baseline",
			baseline: nil,
			modified: &client.SolveResult{MaxFlow: 100},
			validate: func(t *testing.T, comparison *simulationv1.ScenarioComparison) {
				assert.Equal(t, 0.0, comparison.FlowChange)
			},
		},
		{
			name:     "nil modified",
			baseline: &client.SolveResult{MaxFlow: 100},
			modified: nil,
			validate: func(t *testing.T, comparison *simulationv1.ScenarioComparison) {
				assert.Equal(t, 0.0, comparison.FlowChange)
			},
		},
		{
			name: "flow increase",
			baseline: &client.SolveResult{
				MaxFlow:            100,
				TotalCost:          50,
				AverageUtilization: 0.7,
			},
			modified: &client.SolveResult{
				MaxFlow:            150,
				TotalCost:          60,
				AverageUtilization: 0.8,
			},
			validate: func(t *testing.T, comparison *simulationv1.ScenarioComparison) {
				assert.Equal(t, 50.0, comparison.FlowChange)
				assert.Equal(t, 50.0, comparison.FlowChangePercent)
				assert.Equal(t, 10.0, comparison.CostChange)
				assert.Equal(t, 20.0, comparison.CostChangePercent)
				assert.InDelta(t, 0.1, comparison.UtilizationChange, 0.001)
			},
		},
		{
			name: "flow decrease",
			baseline: &client.SolveResult{
				MaxFlow:   100,
				TotalCost: 50,
			},
			modified: &client.SolveResult{
				MaxFlow:   70,
				TotalCost: 40,
			},
			validate: func(t *testing.T, comparison *simulationv1.ScenarioComparison) {
				assert.Equal(t, -30.0, comparison.FlowChange)
				assert.Equal(t, -30.0, comparison.FlowChangePercent)
				assert.Equal(t, -10.0, comparison.CostChange)
			},
		},
		{
			name: "zero baseline flow",
			baseline: &client.SolveResult{
				MaxFlow:   0,
				TotalCost: 0,
			},
			modified: &client.SolveResult{
				MaxFlow:   100,
				TotalCost: 50,
			},
			validate: func(t *testing.T, comparison *simulationv1.ScenarioComparison) {
				assert.Equal(t, 100.0, comparison.FlowChange)
				assert.Equal(t, 0.0, comparison.FlowChangePercent) // div by zero protection
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparison := CompareResults(tt.baseline, tt.modified)
			require.NotNil(t, comparison)
			tt.validate(t, comparison)
		})
	}
}

func TestDetermineImpactLevel(t *testing.T) {
	tests := []struct {
		changePercent float64
		expected      simulationv1.ImpactLevel
	}{
		{0.5, simulationv1.ImpactLevel_IMPACT_LEVEL_NONE},
		{-0.5, simulationv1.ImpactLevel_IMPACT_LEVEL_NONE},
		{3, simulationv1.ImpactLevel_IMPACT_LEVEL_LOW},
		{-3, simulationv1.ImpactLevel_IMPACT_LEVEL_LOW},
		{10, simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM},
		{-10, simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM},
		{20, simulationv1.ImpactLevel_IMPACT_LEVEL_HIGH},
		{-20, simulationv1.ImpactLevel_IMPACT_LEVEL_HIGH},
		{40, simulationv1.ImpactLevel_IMPACT_LEVEL_CRITICAL},
		{-40, simulationv1.ImpactLevel_IMPACT_LEVEL_CRITICAL},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := determineImpactLevel(tt.changePercent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateImpactSummary(t *testing.T) {
	tests := []struct {
		flowChange float64
		costChange float64
		level      simulationv1.ImpactLevel
		contains   []string
	}{
		{
			flowChange: 0,
			costChange: 0,
			level:      simulationv1.ImpactLevel_IMPACT_LEVEL_NONE,
			contains:   []string{"практически не влияют"},
		},
		{
			flowChange: 3,
			costChange: 2,
			level:      simulationv1.ImpactLevel_IMPACT_LEVEL_LOW,
			contains:   []string{"Незначительное", "увеличился"},
		},
		{
			flowChange: -10,
			costChange: 5,
			level:      simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM,
			contains:   []string{"Умеренное", "уменьшился"},
		},
		{
			flowChange: -25,
			costChange: 15,
			level:      simulationv1.ImpactLevel_IMPACT_LEVEL_HIGH,
			contains:   []string{"Значительное"},
		},
		{
			flowChange: -50,
			costChange: 30,
			level:      simulationv1.ImpactLevel_IMPACT_LEVEL_CRITICAL,
			contains:   []string{"КРИТИЧЕСКОЕ"},
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := generateImpactSummary(tt.flowChange, tt.costChange, tt.level)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-0.1, 0.1},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
