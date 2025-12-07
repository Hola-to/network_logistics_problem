// services/report-svc/internal/generator/excel_test.go

package generator

import (
	"context"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestNewExcelGenerator(t *testing.T) {
	g := NewExcelGenerator()
	if g == nil {
		t.Fatal("NewExcelGenerator should not return nil")
	}
}

func TestExcelGenerator_Format(t *testing.T) {
	g := NewExcelGenerator()
	if g.Format() != reportv1.ReportFormat_REPORT_FORMAT_EXCEL {
		t.Errorf("Format() = %v, want EXCEL", g.Format())
	}
}

func TestExcelGenerator_Generate_Flow(t *testing.T) {
	g := NewExcelGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
		Options: &reportv1.ReportOptions{
			Title:          "Excel Flow Report",
			IncludeRawData: true,
		},
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE}, {Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 100, Cost: 5}},
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
			{From: 1, To: 2, Flow: 100.0, Capacity: 100.0, Cost: 5.0, Utilization: 1.0},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Проверяем что результат не пустой и начинается с XLSX signature
	if len(result) < 4 {
		t.Error("Excel file too small")
	}

	// XLSX files start with PK (zip signature)
	if result[0] != 'P' || result[1] != 'K' {
		t.Error("Result doesn't look like a valid XLSX file")
	}
}

func TestExcelGenerator_Generate_Analytics(t *testing.T) {
	g := NewExcelGenerator()
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

	if len(result) < 100 {
		t.Error("Excel file seems too small for analytics report")
	}
}

func TestExcelGenerator_Generate_Simulation(t *testing.T) {
	g := NewExcelGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "monte-carlo",
			BaselineFlow:   100.0,
			BaselineCost:   500.0,
			Scenarios: []*ScenarioData{
				{Name: "Scenario A", MaxFlow: 120.0, TotalCost: 600.0},
			},
			MonteCarlo: &MonteCarloData{
				Iterations: 1000,
				MeanFlow:   98.5,
				StdDev:     5.2,
			},
			Sensitivity: []*SensitivityData{
				{ParameterId: "edge_1_2", Elasticity: 0.5, SensitivityIndex: 0.8, Level: "HIGH"},
			},
			Resilience: &ResilienceData{
				OverallScore:      0.85,
				NMinusOneFeasible: true,
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(result) < 100 {
		t.Error("Excel file seems too small")
	}
}

func TestExcelGenerator_Generate_Comparison(t *testing.T) {
	g := NewExcelGenerator()
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

	if len(result) < 100 {
		t.Error("Excel file seems too small")
	}
}

func TestCellAddr(t *testing.T) {
	tests := []struct {
		col      string
		row      int
		expected string
	}{
		{"A", 1, "A1"},
		{"B", 10, "B10"},
		{"AA", 100, "AA100"},
		{"Z", 999, "Z999"},
	}

	for _, tt := range tests {
		result := cellAddr(tt.col, tt.row)
		if result != tt.expected {
			t.Errorf("cellAddr(%q, %d) = %v, want %v", tt.col, tt.row, result, tt.expected)
		}
	}
}
