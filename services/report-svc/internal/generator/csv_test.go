// services/report-svc/internal/generator/csv_test.go

package generator

import (
	"context"
	"strings"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestNewCSVGenerator(t *testing.T) {
	g := NewCSVGenerator()
	if g == nil {
		t.Fatal("NewCSVGenerator should not return nil")
	}
}

func TestCSVGenerator_Format(t *testing.T) {
	g := NewCSVGenerator()
	if g.Format() != reportv1.ReportFormat_REPORT_FORMAT_CSV {
		t.Errorf("Format() = %v, want CSV", g.Format())
	}
}

func TestCSVGenerator_Generate_Flow(t *testing.T) {
	g := NewCSVGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
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
		Options: &reportv1.ReportOptions{IncludeRawData: true},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	csv := string(result)

	// Проверяем наличие ключевых элементов
	if !strings.Contains(csv, "Flow Report") {
		t.Error("CSV should contain 'Flow Report'")
	}
	if !strings.Contains(csv, "100") { // MaxFlow
		t.Error("CSV should contain max flow value")
	}
	if !strings.Contains(csv, "500") { // TotalCost
		t.Error("CSV should contain total cost value")
	}
}

func TestCSVGenerator_Generate_Analytics(t *testing.T) {
	g := NewCSVGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
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
			Efficiency: &EfficiencyData{
				OverallEfficiency:   0.85,
				CapacityUtilization: 0.75,
				Grade:               "B",
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	csv := string(result)

	if !strings.Contains(csv, "Analytics Report") {
		t.Error("CSV should contain 'Analytics Report'")
	}
	if !strings.Contains(csv, "1500") {
		t.Error("CSV should contain total cost")
	}
	if !strings.Contains(csv, "RUB") {
		t.Error("CSV should contain currency")
	}
}

func TestCSVGenerator_Generate_Simulation(t *testing.T) {
	g := NewCSVGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "what-if",
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
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	csv := string(result)

	if !strings.Contains(csv, "Simulation Report") {
		t.Error("CSV should contain 'Simulation Report'")
	}
	if !strings.Contains(csv, "Monte Carlo") {
		t.Error("CSV should contain 'Monte Carlo'")
	}
}

func TestCSVGenerator_Generate_Comparison(t *testing.T) {
	g := NewCSVGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_COMPARISON,
		ComparisonData: []*ComparisonItemData{
			{Name: "Baseline", MaxFlow: 100.0, TotalCost: 500.0, Efficiency: 0.8},
			{Name: "Scenario A", MaxFlow: 120.0, TotalCost: 550.0, Efficiency: 0.85},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	csv := string(result)

	if !strings.Contains(csv, "Comparison Report") {
		t.Error("CSV should contain 'Comparison Report'")
	}
	if !strings.Contains(csv, "Baseline") {
		t.Error("CSV should contain 'Baseline'")
	}
	if !strings.Contains(csv, "Scenario A") {
		t.Error("CSV should contain 'Scenario A'")
	}
}

func TestCSVGenerator_Generate_NoData(t *testing.T) {
	g := NewCSVGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type:          reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		AnalyticsData: nil,
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	csv := string(result)
	if !strings.Contains(csv, "No analytics data") {
		t.Error("CSV should indicate no data available")
	}
}
