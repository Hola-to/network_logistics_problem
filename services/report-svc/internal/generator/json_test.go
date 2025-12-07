// services/report-svc/internal/generator/json_test.go

package generator

import (
	"context"
	"encoding/json"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestNewJSONGenerator(t *testing.T) {
	g := NewJSONGenerator()
	if g == nil {
		t.Fatal("NewJSONGenerator should not return nil")
	}
}

func TestJSONGenerator_Format(t *testing.T) {
	g := NewJSONGenerator()
	if g.Format() != reportv1.ReportFormat_REPORT_FORMAT_JSON {
		t.Errorf("Format() = %v, want JSON", g.Format())
	}
}

func TestJSONGenerator_Generate_Flow(t *testing.T) {
	g := NewJSONGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
		Options: &reportv1.ReportOptions{
			Title:  "Test Flow Report",
			Author: "Test Author",
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
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Проверяем валидность JSON
	var report JSONReport
	if err := json.Unmarshal(result, &report); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Проверяем метаданные
	if report.Metadata.Title != "Test Flow Report" {
		t.Errorf("Title = %v, want 'Test Flow Report'", report.Metadata.Title)
	}
	if report.Metadata.Author != "Test Author" {
		t.Errorf("Author = %v, want 'Test Author'", report.Metadata.Author)
	}
	if report.Metadata.ReportType != "REPORT_TYPE_FLOW" {
		t.Errorf("ReportType = %v, want 'REPORT_TYPE_FLOW'", report.Metadata.ReportType)
	}

	// Проверяем граф
	if report.Graph == nil {
		t.Fatal("Graph should not be nil")
	}
	if report.Graph.NodeCount != 2 {
		t.Errorf("NodeCount = %d, want 2", report.Graph.NodeCount)
	}

	// Проверяем результаты
	if report.FlowResult == nil {
		t.Fatal("FlowResult should not be nil")
	}
	if report.FlowResult.MaxFlow != 100.0 {
		t.Errorf("MaxFlow = %v, want 100.0", report.FlowResult.MaxFlow)
	}
}

func TestJSONGenerator_Generate_Analytics(t *testing.T) {
	g := NewJSONGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		AnalyticsData: &AnalyticsReportData{
			TotalCost: 1500.0,
			Currency:  "USD",
			CostBreakdown: &CostBreakdownData{
				TransportCost: 1000.0,
				FixedCost:     300.0,
				HandlingCost:  200.0,
			},
			Bottlenecks: []*BottleneckData{
				{From: 1, To: 2, Utilization: 0.95, ImpactScore: 0.8, Severity: "HIGH"},
			},
			Recommendations: []*RecommendationData{
				{Type: "increase_capacity", Description: "Increase capacity", EstimatedImprovement: 0.15},
			},
			Efficiency: &EfficiencyData{
				OverallEfficiency:   0.85,
				CapacityUtilization: 0.75,
				UnusedEdges:         5,
				SaturatedEdges:      3,
				Grade:               "B",
			},
		},
		Options: &reportv1.ReportOptions{IncludeRecommendations: true},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var report JSONReport
	if err := json.Unmarshal(result, &report); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if report.Analytics == nil {
		t.Fatal("Analytics should not be nil")
	}
	if report.Analytics.TotalCost != 1500.0 {
		t.Errorf("TotalCost = %v, want 1500.0", report.Analytics.TotalCost)
	}
	if report.Analytics.Currency != "USD" {
		t.Errorf("Currency = %v, want USD", report.Analytics.Currency)
	}
	if len(report.Analytics.Bottlenecks) != 1 {
		t.Errorf("Bottlenecks length = %d, want 1", len(report.Analytics.Bottlenecks))
	}
	if report.Analytics.Efficiency.Grade != "B" {
		t.Errorf("Grade = %v, want B", report.Analytics.Efficiency.Grade)
	}
}

func TestJSONGenerator_Generate_Simulation(t *testing.T) {
	g := NewJSONGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "monte-carlo",
			BaselineFlow:   100.0,
			BaselineCost:   500.0,
			MonteCarlo: &MonteCarloData{
				Iterations: 1000,
				MeanFlow:   98.5,
				StdDev:     5.2,
			},
			Sensitivity: []*SensitivityData{
				{ParameterId: "edge_1_2", Elasticity: 0.5, SensitivityIndex: 0.8, Level: "HIGH"},
			},
			Resilience: &ResilienceData{
				OverallScore:          0.85,
				SinglePointsOfFailure: 2,
				NMinusOneFeasible:     true,
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var report JSONReport
	if err := json.Unmarshal(result, &report); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if report.Simulation == nil {
		t.Fatal("Simulation should not be nil")
	}
	if report.Simulation.Type != "monte-carlo" {
		t.Errorf("Type = %v, want 'monte-carlo'", report.Simulation.Type)
	}
	if report.Simulation.MonteCarlo == nil {
		t.Fatal("MonteCarlo should not be nil")
	}
	if report.Simulation.MonteCarlo.Iterations != 1000 {
		t.Errorf("Iterations = %d, want 1000", report.Simulation.MonteCarlo.Iterations)
	}
}

func TestJSONGenerator_Generate_Comparison(t *testing.T) {
	g := NewJSONGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_COMPARISON,
		ComparisonData: []*ComparisonItemData{
			{Name: "Baseline", MaxFlow: 100.0, TotalCost: 500.0, Efficiency: 0.8, Metrics: map[string]float64{"metric1": 10.0}},
			{Name: "Scenario A", MaxFlow: 120.0, TotalCost: 550.0, Efficiency: 0.85, Metrics: map[string]float64{"metric1": 12.0}},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var report JSONReport
	if err := json.Unmarshal(result, &report); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(report.Comparison) != 2 {
		t.Errorf("Comparison length = %d, want 2", len(report.Comparison))
	}
	if report.Comparison[0].Name != "Baseline" {
		t.Errorf("First comparison name = %v, want 'Baseline'", report.Comparison[0].Name)
	}
}
