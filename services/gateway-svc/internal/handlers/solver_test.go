// services/gateway-svc/internal/handlers/solver_test.go

package handlers

import (
	"testing"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
)

func TestSolverHandler_ConvertAnalytics(t *testing.T) {
	h := &SolverHandler{}

	// Test with all fields
	resp := &analyticsv1.AnalyzeFlowResponse{
		FlowStats: &commonv1.FlowStatistics{
			TotalFlow:          100.0,
			TotalCost:          500.0,
			AverageUtilization: 0.75,
		},
		Cost: &analyticsv1.CalculateCostResponse{
			TotalCost: 500.0,
			Currency:  "RUB",
			Breakdown: &analyticsv1.CostBreakdown{
				TransportCost: 400.0,
				FixedCost:     100.0,
			},
		},
		Bottlenecks: &analyticsv1.FindBottlenecksResponse{
			Bottlenecks: []*analyticsv1.Bottleneck{
				{
					Edge:        &commonv1.Edge{From: 1, To: 2},
					Utilization: 0.95,
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
				},
			},
			Recommendations: []*analyticsv1.Recommendation{
				{
					Type:        "increase_capacity",
					Description: "Increase capacity of edge 1->2",
				},
			},
		},
		Efficiency: &analyticsv1.EfficiencyReport{
			OverallEfficiency: 0.85,
			Grade:             "B",
		},
	}

	result := h.convertAnalytics(resp)

	if result == nil {
		t.Fatal("convertAnalytics should not return nil")
	}

	if result.FlowStats == nil {
		t.Error("FlowStats should not be nil")
	}

	if result.TotalCost != 500.0 {
		t.Errorf("TotalCost = %v, want 500.0", result.TotalCost)
	}

	if result.Currency != "RUB" {
		t.Errorf("Currency = %v, want RUB", result.Currency)
	}

	if result.BottlenecksCount != 1 {
		t.Errorf("BottlenecksCount = %d, want 1", result.BottlenecksCount)
	}

	if result.Efficiency == nil || result.Efficiency.Grade != "B" {
		t.Error("Efficiency should have grade B")
	}
}

func TestSolverHandler_ConvertAnalytics_NilFields(t *testing.T) {
	h := &SolverHandler{}

	// Test with nil fields
	resp := &analyticsv1.AnalyzeFlowResponse{
		FlowStats:   nil,
		Cost:        nil,
		Bottlenecks: nil,
		Efficiency:  nil,
	}

	result := h.convertAnalytics(resp)

	// Should not panic
	if result == nil {
		t.Fatal("convertAnalytics should not return nil")
	}

	if result.TotalCost != 0 {
		t.Errorf("TotalCost should be 0 when Cost is nil, got %v", result.TotalCost)
	}
}

func TestSolverHandler_ConvertMetrics_Valid(t *testing.T) {
	h := &SolverHandler{}

	metrics := &optimizationv1.SolveMetrics{
		ComputationTimeMs:    150.5,
		Iterations:           42,
		AugmentingPathsFound: 10,
		MemoryUsedBytes:      1024 * 1024,
	}

	result := h.convertMetrics(metrics)

	if result == nil {
		t.Fatal("convertMetrics should not return nil for valid input")
	}

	if result.ComputationTimeMs != 150.5 {
		t.Errorf("ComputationTimeMs = %v, want 150.5", result.ComputationTimeMs)
	}

	if result.Iterations != 42 {
		t.Errorf("Iterations = %d, want 42", result.Iterations)
	}

	if result.AugmentingPathsFound != 10 {
		t.Errorf("AugmentingPathsFound = %d, want 10", result.AugmentingPathsFound)
	}

	if result.MemoryUsedBytes != 1024*1024 {
		t.Errorf("MemoryUsedBytes = %d, want %d", result.MemoryUsedBytes, 1024*1024)
	}
}
