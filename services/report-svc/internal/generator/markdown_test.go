// services/report-svc/internal/generator/markdown_test.go

package generator

import (
	"context"
	"strings"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestNewMarkdownGenerator(t *testing.T) {
	g := NewMarkdownGenerator()
	if g == nil {
		t.Fatal("NewMarkdownGenerator should not return nil")
	}
}

func TestMarkdownGenerator_Format(t *testing.T) {
	g := NewMarkdownGenerator()
	if g.Format() != reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN {
		t.Errorf("Format() = %v, want MARKDOWN", g.Format())
	}
}

func TestMarkdownGenerator_Generate_Flow(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
		Options: &reportv1.ReportOptions{
			Title:          "Flow Report",
			Author:         "Test",
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

	md := string(result)

	// Проверяем структуру Markdown
	if !strings.Contains(md, "# Flow Report") {
		t.Error("Should contain title")
	}
	if !strings.Contains(md, "## Network Information") {
		t.Error("Should contain network section")
	}
	if !strings.Contains(md, "## Optimization Results") {
		t.Error("Should contain optimization section")
	}
	if !strings.Contains(md, "**Maximum Flow:**") {
		t.Error("Should contain max flow")
	}
	if !strings.Contains(md, "| From | To | Flow |") {
		t.Error("Should contain edge table header")
	}
}

func TestMarkdownGenerator_Generate_Analytics(t *testing.T) {
	g := NewMarkdownGenerator()
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
				TransportCost:  1000.0,
				FixedCost:      300.0,
				HandlingCost:   200.0,
				CostByRoadType: map[string]float64{"highway": 800.0},
			},
			Bottlenecks: []*BottleneckData{
				{From: 1, To: 2, Utilization: 0.95, ImpactScore: 0.8, Severity: "HIGH"},
			},
			Recommendations: []*RecommendationData{
				{Type: "increase_capacity", Description: "Increase capacity of edge 1->2", EstimatedImprovement: 0.15},
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

	md := string(result)

	if !strings.Contains(md, "## Cost Analysis") {
		t.Error("Should contain cost section")
	}
	if !strings.Contains(md, "## Bottlenecks") {
		t.Error("Should contain bottlenecks section")
	}
	if !strings.Contains(md, "## Recommendations") {
		t.Error("Should contain recommendations section")
	}
	if !strings.Contains(md, "## Efficiency Metrics") {
		t.Error("Should contain efficiency section")
	}
	// ИСПРАВЛЕНО: проверяем правильный формат с Markdown форматированием
	if !strings.Contains(md, "**Grade:** B") {
		t.Error("Should contain grade")
	}
}

func TestMarkdownGenerator_Generate_Comparison(t *testing.T) {
	g := NewMarkdownGenerator()
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

	md := string(result)

	if !strings.Contains(md, "## Scenario Comparison") {
		t.Error("Should contain comparison section")
	}
	if !strings.Contains(md, "| Baseline |") {
		t.Error("Should contain baseline in table")
	}
	if !strings.Contains(md, "### Conclusions") {
		t.Error("Should contain conclusions")
	}
	if !strings.Contains(md, "Best scenario by flow") {
		t.Error("Should identify best scenario")
	}
}

func TestMarkdownGenerator_Generate_Simulation(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "Monte Carlo",
			BaselineFlow:   100.0,
			BaselineCost:   500.0,
			MonteCarlo: &MonteCarloData{
				Iterations:      1000,
				MeanFlow:        105.0,
				StdDev:          10.0,
				MinFlow:         80.0,
				MaxFlow:         130.0,
				P5:              85.0,
				P50:             105.0,
				P95:             125.0,
				ConfidenceLevel: 0.95,
				CiLow:           100.0,
				CiHigh:          110.0,
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := string(result)

	if !strings.Contains(md, "## Simulation Type: Monte Carlo") {
		t.Error("Should contain simulation type")
	}
	if !strings.Contains(md, "### Monte Carlo Results") {
		t.Error("Should contain Monte Carlo section")
	}
	if !strings.Contains(md, "**Mean Flow:**") {
		t.Error("Should contain mean flow")
	}
}

func TestMarkdownGenerator_Generate_Summary(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SUMMARY,
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 100}},
			SourceId: 1,
			SinkId:   2,
		},
		FlowResult: &commonv1.FlowResult{
			MaxFlow:   100.0,
			TotalCost: 500.0,
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := string(result)

	if !strings.Contains(md, "## Summary Report") {
		t.Error("Should contain summary section")
	}
}

func TestMarkdownGenerator_Generate_NoData(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		// AnalyticsData is nil
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := string(result)

	if !strings.Contains(md, "*No analytics data available*") {
		t.Error("Should contain no data message")
	}
}

func TestMarkdownGenerator_FindBest(t *testing.T) {
	g := NewMarkdownGenerator()

	items := []*ComparisonItemData{
		{Name: "A", MaxFlow: 100.0},
		{Name: "B", MaxFlow: 150.0},
		{Name: "C", MaxFlow: 80.0},
	}

	best := g.findBest(items)
	if best == nil {
		t.Fatal("findBest should not return nil")
	}
	if best.Name != "B" {
		t.Errorf("Best scenario = %v, want B", best.Name)
	}
}

func TestMarkdownGenerator_FindBest_Empty(t *testing.T) {
	g := NewMarkdownGenerator()

	best := g.findBest([]*ComparisonItemData{})
	if best != nil {
		t.Error("findBest([]) should return nil")
	}
}

func TestMarkdownGenerator_Generate_WithSensitivity(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "Sensitivity Analysis",
			BaselineFlow:   100.0,
			Sensitivity: []*SensitivityData{
				{ParameterId: "edge_1_2", Elasticity: 1.5, SensitivityIndex: 0.8, Level: "HIGH"},
				{ParameterId: "edge_2_3", Elasticity: 0.5, SensitivityIndex: 0.3, Level: "LOW"},
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := string(result)

	if !strings.Contains(md, "### Sensitivity Analysis") {
		t.Error("Should contain sensitivity section")
	}
	if !strings.Contains(md, "edge_1_2") {
		t.Error("Should contain parameter id")
	}
}

func TestMarkdownGenerator_Generate_WithResilience(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		SimulationData: &SimulationReportData{
			SimulationType: "Resilience Analysis",
			BaselineFlow:   100.0,
			Resilience: &ResilienceData{
				OverallScore:           0.85,
				SinglePointsOfFailure:  2,
				WorstCaseFlowReduction: 0.25,
				NMinusOneFeasible:      true,
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := string(result)

	if !strings.Contains(md, "### Resilience Analysis") {
		t.Error("Should contain resilience section")
	}
	if !strings.Contains(md, "**Overall Score:**") {
		t.Error("Should contain overall score")
	}
}

func TestMarkdownGenerator_Generate_EmptyComparison(t *testing.T) {
	g := NewMarkdownGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type:           reportv1.ReportType_REPORT_TYPE_COMPARISON,
		ComparisonData: []*ComparisonItemData{},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := string(result)

	if !strings.Contains(md, "*No comparison data available*") {
		t.Error("Should contain no data message for empty comparison")
	}
}
