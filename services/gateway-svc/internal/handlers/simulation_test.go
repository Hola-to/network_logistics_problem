// services/gateway-svc/internal/handlers/simulation_test.go

package handlers

import (
	"testing"

	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
)

func TestSimulationHandler_ConvertMonteCarloConfig(t *testing.T) {
	h := &SimulationHandler{}

	// Test nil input
	result := h.convertMonteCarloConfig(nil)
	if result != nil {
		t.Error("convertMonteCarloConfig(nil) should return nil")
	}

	// Test valid input
	config := &gatewayv1.MonteCarloConfig{
		NumIterations:   1000,
		RandomSeed:      42,
		ConfidenceLevel: 0.95,
		Parallel:        true,
	}

	result = h.convertMonteCarloConfig(config)
	if result == nil {
		t.Fatal("convertMonteCarloConfig should not return nil for valid input")
	}

	if result.NumIterations != 1000 {
		t.Errorf("NumIterations = %d, want 1000", result.NumIterations)
	}
	if result.RandomSeed != 42 {
		t.Errorf("RandomSeed = %d, want 42", result.RandomSeed)
	}
	if result.ConfidenceLevel != 0.95 {
		t.Errorf("ConfidenceLevel = %v, want 0.95", result.ConfidenceLevel)
	}
	if !result.Parallel {
		t.Error("Parallel should be true")
	}
}

func TestSimulationHandler_ConvertUncertainties(t *testing.T) {
	h := &SimulationHandler{}

	// Test empty input
	result := h.convertUncertainties(nil)
	if len(result) != 0 {
		t.Errorf("convertUncertainties(nil) should return empty slice, got %d items", len(result))
	}

	// Test valid input
	specs := []*gatewayv1.UncertaintySpec{
		{
			NodeId: 1,
			Target: gatewayv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			Distribution: &gatewayv1.Distribution{
				Type:   gatewayv1.DistributionType_DISTRIBUTION_TYPE_NORMAL,
				Param1: 100.0,
				Param2: 10.0,
			},
		},
	}

	result = h.convertUncertainties(specs)
	if len(result) != 1 {
		t.Fatalf("convertUncertainties should return 1 item, got %d", len(result))
	}

	if result[0].NodeId != 1 {
		t.Errorf("NodeId = %d, want 1", result[0].NodeId)
	}
}

func TestSimulationHandler_ConvertMonteCarloStats_Valid(t *testing.T) {
	h := &SimulationHandler{}

	stats := &simulationv1.MonteCarloStats{
		Mean:                   100.0,
		StdDev:                 10.0,
		Min:                    80.0,
		Max:                    120.0,
		Median:                 99.0,
		ConfidenceIntervalLow:  90.0,
		ConfidenceIntervalHigh: 110.0,
	}

	result := h.convertMonteCarloStats(stats)
	if result == nil {
		t.Fatal("convertMonteCarloStats should not return nil for valid input")
	}

	if result.Mean != 100.0 {
		t.Errorf("Mean = %v, want 100.0", result.Mean)
	}
	if result.StdDev != 10.0 {
		t.Errorf("StdDev = %v, want 10.0", result.StdDev)
	}
	if result.Min != 80.0 {
		t.Errorf("Min = %v, want 80.0", result.Min)
	}
	if result.Max != 120.0 {
		t.Errorf("Max = %v, want 120.0", result.Max)
	}
}

func TestSimulationHandler_ConvertRiskAnalysis_Valid(t *testing.T) {
	h := &SimulationHandler{}

	risk := &simulationv1.RiskAnalysis{
		ProbabilityBelowThreshold: 0.05,
		ValueAtRisk:               85.0,
		WorstCaseFlow:             70.0,
		BestCaseFlow:              130.0,
	}

	result := h.convertRiskAnalysis(risk)
	if result == nil {
		t.Fatal("convertRiskAnalysis should not return nil for valid input")
	}

	if result.ProbabilityBelowThreshold != 0.05 {
		t.Errorf("ProbabilityBelowThreshold = %v, want 0.05", result.ProbabilityBelowThreshold)
	}
	if result.ValueAtRisk != 85.0 {
		t.Errorf("ValueAtRisk = %v, want 85.0", result.ValueAtRisk)
	}
}

func TestSimulationHandler_ConvertScenarioResult_Valid(t *testing.T) {
	h := &SimulationHandler{}

	scenario := &simulationv1.ScenarioResult{
		Name:               "Test Scenario",
		MaxFlow:            100.0,
		TotalCost:          500.0,
		AverageUtilization: 0.85,
	}

	result := h.convertScenarioResult(scenario)
	if result == nil {
		t.Fatal("convertScenarioResult should not return nil for valid input")
	}

	if result.Name != "Test Scenario" {
		t.Errorf("Name = %v, want 'Test Scenario'", result.Name)
	}
	if result.MaxFlow != 100.0 {
		t.Errorf("MaxFlow = %v, want 100.0", result.MaxFlow)
	}
	if result.TotalCost != 500.0 {
		t.Errorf("TotalCost = %v, want 500.0", result.TotalCost)
	}
}

func TestSimulationHandler_ConvertComparison_Valid(t *testing.T) {
	h := &SimulationHandler{}

	comparison := &simulationv1.ScenarioComparison{
		FlowChange:        10.0,
		FlowChangePercent: 10.0,
		CostChange:        -50.0,
		CostChangePercent: -10.0,
		ImpactSummary:     "Positive impact",
		ImpactLevel:       simulationv1.ImpactLevel_IMPACT_LEVEL_MEDIUM,
	}

	result := h.convertComparison(comparison)
	if result == nil {
		t.Fatal("convertComparison should not return nil for valid input")
	}

	if result.FlowChange != 10.0 {
		t.Errorf("FlowChange = %v, want 10.0", result.FlowChange)
	}
	if result.CostChange != -50.0 {
		t.Errorf("CostChange = %v, want -50.0", result.CostChange)
	}
}

func TestSimulationHandler_ConvertMetadata_Valid(t *testing.T) {
	h := &SimulationHandler{}

	metadata := &simulationv1.SimulationMetadata{
		SimulationId:      "sim-123",
		ComputationTimeMs: 1500,
		Iterations:        1000,
		MemoryUsedBytes:   1024 * 1024 * 10,
		AlgorithmUsed:     "DINIC",
	}

	result := h.convertMetadata(metadata)
	if result == nil {
		t.Fatal("convertMetadata should not return nil for valid input")
	}

	if result.SimulationId != "sim-123" {
		t.Errorf("SimulationId = %v, want 'sim-123'", result.SimulationId)
	}
	if result.Iterations != 1000 {
		t.Errorf("Iterations = %d, want 1000", result.Iterations)
	}
}
