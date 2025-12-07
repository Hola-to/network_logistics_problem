// services/report-svc/internal/generator/pdf_test.go

package generator

import (
	"context"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestNewPDFGenerator(t *testing.T) {
	g := NewPDFGenerator()
	if g == nil {
		t.Fatal("NewPDFGenerator should not return nil")
	}
}

func TestPDFGenerator_Format(t *testing.T) {
	g := NewPDFGenerator()
	if g.Format() != reportv1.ReportFormat_REPORT_FORMAT_PDF {
		t.Errorf("Format() = %v, want PDF", g.Format())
	}
}

func TestPDFGenerator_Generate_Flow(t *testing.T) {
	g := NewPDFGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
		Options: &reportv1.ReportOptions{
			Title:          "PDF Flow Report",
			Author:         "Test Author",
			IncludeRawData: true,
		},
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 100}},
			SourceId: 1,
			SinkId:   2,
		},
		FlowResult: &commonv1.FlowResult{
			MaxFlow:           100.0,
			TotalCost:         500.0,
			Status:            commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
			Iterations:        10,
			ComputationTimeMs: 50.0,
		},
		FlowEdges: []*EdgeFlowData{
			{From: 1, To: 2, Flow: 100.0, Capacity: 100.0, Utilization: 1.0},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// PDF signature: %PDF-
	if len(result) < 5 {
		t.Fatal("PDF file too small")
	}
	if string(result[:5]) != "%PDF-" {
		t.Error("Result doesn't look like a valid PDF file")
	}
}

func TestPDFGenerator_Generate_Analytics(t *testing.T) {
	g := NewPDFGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		Options: &reportv1.ReportOptions{
			IncludeRecommendations: true,
		},
		AnalyticsData: &AnalyticsReportData{
			TotalCost: 1500.0,
			Currency:  "RUB",
			CostBreakdown: &CostBreakdownData{
				TransportCost: 1000.0,
				FixedCost:     300.0,
				HandlingCost:  200.0,
			},
			Bottlenecks: []*BottleneckData{
				{From: 1, To: 2, Utilization: 0.95, ImpactScore: 0.8, Severity: "HIGH"},
			},
			Recommendations: []*RecommendationData{
				{Type: "increase_capacity", Description: "Increase capacity", EstimatedImprovement: 0.15, EstimatedCost: 1000.0},
			},
			Efficiency: &EfficiencyData{
				OverallEfficiency:   0.85,
				CapacityUtilization: 0.75,
				UnusedEdges:         5,
				SaturatedEdges:      3,
				Grade:               "B",
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if string(result[:5]) != "%PDF-" {
		t.Error("Result doesn't look like a valid PDF file")
	}
}

func TestPDFGenerator_Generate_Simulation(t *testing.T) {
	g := NewPDFGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "monte-carlo",
			BaselineFlow:   100.0,
			BaselineCost:   500.0,
			Scenarios: []*ScenarioData{
				{Name: "Scenario A", MaxFlow: 120.0, TotalCost: 600.0, FlowChangePercent: 20.0, ImpactLevel: "MEDIUM"},
			},
			MonteCarlo: &MonteCarloData{
				Iterations:      1000,
				MeanFlow:        98.5,
				StdDev:          5.2,
				MinFlow:         85.0,
				MaxFlow:         115.0,
				P5:              90.0,
				P50:             98.0,
				P95:             108.0,
				ConfidenceLevel: 0.95,
				CiLow:           88.0,
				CiHigh:          109.0,
			},
			Sensitivity: []*SensitivityData{
				{ParameterId: "edge_1_2", Elasticity: 0.5, SensitivityIndex: 0.8, Level: "HIGH"},
			},
			Resilience: &ResilienceData{
				OverallScore:           0.85,
				SinglePointsOfFailure:  2,
				WorstCaseFlowReduction: 0.3,
				NMinusOneFeasible:      true,
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if string(result[:5]) != "%PDF-" {
		t.Error("Result doesn't look like a valid PDF file")
	}
}

func TestPDFGenerator_Generate_Comparison(t *testing.T) {
	g := NewPDFGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_COMPARISON,
		ComparisonData: []*ComparisonItemData{
			{Name: "Baseline", MaxFlow: 100.0, TotalCost: 500.0, Efficiency: 0.8, Metrics: map[string]float64{"utilization": 0.8}},
			{Name: "Scenario A", MaxFlow: 120.0, TotalCost: 550.0, Efficiency: 0.85, Metrics: map[string]float64{"utilization": 0.85}},
			{Name: "Scenario B", MaxFlow: 90.0, TotalCost: 450.0, Efficiency: 0.75, Metrics: map[string]float64{"utilization": 0.75}},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if string(result[:5]) != "%PDF-" {
		t.Error("Result doesn't look like a valid PDF file")
	}
}

func TestPDFGenerator_Generate_Summary(t *testing.T) {
	g := NewPDFGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SUMMARY,
		FlowResult: &commonv1.FlowResult{
			MaxFlow:   100.0,
			TotalCost: 500.0,
		},
		AnalyticsData: &AnalyticsReportData{
			TotalCost: 500.0,
			Currency:  "RUB",
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if string(result[:5]) != "%PDF-" {
		t.Error("Result doesn't look like a valid PDF file")
	}
}

func TestPDFGenerator_Generate_History(t *testing.T) {
	g := NewPDFGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_HISTORY,
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if string(result[:5]) != "%PDF-" {
		t.Error("Result doesn't look like a valid PDF file")
	}
}

func TestPDFGenerator_FindBestScenario(t *testing.T) {
	g := NewPDFGenerator()

	tests := []struct {
		name     string
		items    []*ComparisonItemData
		expected string
	}{
		{
			name: "find best",
			items: []*ComparisonItemData{
				{Name: "A", MaxFlow: 100.0},
				{Name: "B", MaxFlow: 150.0},
				{Name: "C", MaxFlow: 80.0},
			},
			expected: "B",
		},
		{
			name:     "empty list",
			items:    []*ComparisonItemData{},
			expected: "",
		},
		{
			name: "single item",
			items: []*ComparisonItemData{
				{Name: "Only", MaxFlow: 100.0},
			},
			expected: "Only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.findBestScenario(tt.items)
			if tt.expected == "" {
				if result != nil {
					t.Error("Expected nil for empty list")
				}
			} else {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				if result.Name != tt.expected {
					t.Errorf("Best = %v, want %v", result.Name, tt.expected)
				}
			}
		})
	}
}
