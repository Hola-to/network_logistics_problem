package v1_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/tests/integration/testutil"
)

func TestSimulationService_RunWhatIf(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.RunWhatIf(ctx, &simulationv1.RunWhatIfRequest{
		BaselineGraph: CreateSimpleGraph(),
		Modifications: []*simulationv1.Modification{
			{
				Type:   simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				EdgeKey: &commonv1.EdgeKey{
					From: 0,
					To:   1,
				},
				Change: &simulationv1.Modification_AbsoluteValue{
					AbsoluteValue: 20,
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &simulationv1.WhatIfOptions{
			CompareWithBaseline: true,
			CalculateCostImpact: true,
			FindNewBottlenecks:  true,
			ReturnModifiedGraph: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Baseline)
	assert.NotNil(t, resp.Modified)
	assert.NotNil(t, resp.Comparison)
	assert.NotNil(t, resp.Metadata)
}

func TestSimulationService_CompareScenarios(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	baseline := CreateSimpleGraph()

	scenarios := []*simulationv1.Scenario{
		{
			Name: "Increased Capacity",
			Modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
					EdgeKey: &commonv1.EdgeKey{From: 0, To: 1},
					Change:  &simulationv1.Modification_RelativeChange{RelativeChange: 0.5},
				},
			},
		},
		{
			Name: "Reduced Cost",
			Modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_COST,
					EdgeKey: &commonv1.EdgeKey{From: 0, To: 1},
					Change:  &simulationv1.Modification_RelativeChange{RelativeChange: -0.3},
				},
			},
		},
	}

	resp, err := client.CompareScenarios(ctx, &simulationv1.CompareScenariosRequest{
		BaselineGraph: baseline,
		Scenarios:     scenarios,
		Algorithm:     commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Baseline)
	assert.Len(t, resp.RankedScenarios, 2)
	assert.NotEmpty(t, resp.Recommendation)
}

func TestSimulationService_RunMonteCarlo(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.ContextWithDuration(t, 60*1e9) // 60 seconds
	defer cancel()

	resp, err := client.RunMonteCarlo(ctx, &simulationv1.RunMonteCarloRequest{
		Graph: CreateSimpleGraph(),
		Config: &simulationv1.MonteCarloConfig{
			NumIterations:   100,
			ConfidenceLevel: 0.95,
			Parallel:        true,
		},
		Uncertainties: []*simulationv1.UncertaintySpec{
			{
				Edge:   &commonv1.EdgeKey{From: 0, To: 1},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_NORMAL,
					Param1: 10, // mean
					Param2: 2,  // std dev
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.FlowStats)
	assert.Greater(t, resp.FlowStats.Mean, 0.0)
	assert.NotNil(t, resp.Metadata)
}

func TestSimulationService_RunMonteCarloStream(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.ContextWithDuration(t, 60*1e9)
	defer cancel()

	stream, err := client.RunMonteCarloStream(ctx, &simulationv1.RunMonteCarloRequest{
		Graph: CreateSimpleGraph(),
		Config: &simulationv1.MonteCarloConfig{
			NumIterations:   50,
			ConfidenceLevel: 0.95,
		},
		Uncertainties: []*simulationv1.UncertaintySpec{
			{
				Edge:   &commonv1.EdgeKey{From: 0, To: 1},
				Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				Distribution: &simulationv1.Distribution{
					Type:   simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM,
					Param1: 5,
					Param2: 15,
				},
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})
	require.NoError(t, err)

	progressCount := 0
	var lastProgress *simulationv1.MonteCarloProgress

	for {
		progress, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		progressCount++
		lastProgress = progress

		assert.GreaterOrEqual(t, progress.ProgressPercent, 0.0)
		assert.LessOrEqual(t, progress.ProgressPercent, 100.0)
	}

	assert.Greater(t, progressCount, 0)
	require.NotNil(t, lastProgress)
}

func TestSimulationService_AnalyzeSensitivity(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.AnalyzeSensitivity(ctx, &simulationv1.AnalyzeSensitivityRequest{
		Graph: CreateSimpleGraph(),
		Parameters: []*simulationv1.SensitivityParameter{
			{
				Edge:          &commonv1.EdgeKey{From: 0, To: 1},
				Target:        simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
				MinMultiplier: 0.5,
				MaxMultiplier: 2.0,
				NumSteps:      5,
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.ParameterResults)
	assert.NotNil(t, resp.Metadata)
}

func TestSimulationService_FindCriticalElements(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.FindCriticalElements(ctx, &simulationv1.FindCriticalElementsRequest{
		Graph: CreateSimpleGraph(),
		Config: &simulationv1.CriticalElementsConfig{
			AnalyzeEdges:     true,
			AnalyzeNodes:     true,
			TopN:             5,
			FailureThreshold: 0.1,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.GreaterOrEqual(t, resp.ResilienceScore, 0.0)
	assert.LessOrEqual(t, resp.ResilienceScore, 1.0)
}

func TestSimulationService_SimulateFailures(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.SimulateFailures(ctx, &simulationv1.SimulateFailuresRequest{
		Graph: CreateSimpleGraph(),
		FailureScenarios: []*simulationv1.FailureScenario{
			{
				Name:        "Edge 0->1 Failure",
				FailedEdges: []*commonv1.EdgeKey{{From: 0, To: 1}},
				Probability: 0.1,
			},
			{
				Name:        "Multiple Edge Failure",
				FailedEdges: []*commonv1.EdgeKey{{From: 0, To: 1}, {From: 0, To: 2}},
				Probability: 0.05,
			},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Baseline)
	assert.Len(t, resp.ScenarioResults, 2)
	assert.NotNil(t, resp.Stats)
}

func TestSimulationService_AnalyzeResilience(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.AnalyzeResilience(ctx, &simulationv1.AnalyzeResilienceRequest{
		Graph: CreateSimpleGraph(),
		Config: &simulationv1.ResilienceConfig{
			MaxFailuresToTest:     2,
			TestCascadingFailures: true,
			LoadFactor:            1.0,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Metrics)
	assert.GreaterOrEqual(t, resp.Metrics.OverallScore, 0.0)
	assert.LessOrEqual(t, resp.Metrics.OverallScore, 1.0)
}

func TestSimulationService_Health(t *testing.T) {
	client := SetupSimulationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Health(ctx, &simulationv1.HealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SERVING", resp.Status)
	assert.NotEmpty(t, resp.Version)
}
