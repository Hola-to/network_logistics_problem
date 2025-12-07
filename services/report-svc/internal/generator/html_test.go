// services/report-svc/internal/generator/html_test.go

package generator

import (
	"context"
	"strings"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestNewHTMLGenerator(t *testing.T) {
	g := NewHTMLGenerator()
	if g == nil {
		t.Fatal("NewHTMLGenerator should not return nil")
	}
}

func TestHTMLGenerator_Format(t *testing.T) {
	g := NewHTMLGenerator()
	if g.Format() != reportv1.ReportFormat_REPORT_FORMAT_HTML {
		t.Errorf("Format() = %v, want HTML", g.Format())
	}
}

func TestHTMLGenerator_Generate_Flow(t *testing.T) {
	g := NewHTMLGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
		Options: &reportv1.ReportOptions{
			Title:          "Test HTML Report",
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
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := string(result)

	// Проверяем структуру HTML
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Should contain DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
		t.Error("Should contain html tag")
	}
	if !strings.Contains(html, "<head>") {
		t.Error("Should contain head tag")
	}
	if !strings.Contains(html, "<body>") {
		t.Error("Should contain body tag")
	}
	if !strings.Contains(html, "Test HTML Report") {
		t.Error("Should contain title")
	}
	if !strings.Contains(html, "Maximum Flow") {
		t.Error("Should contain max flow label")
	}
	if !strings.Contains(html, "100.0000") {
		t.Error("Should contain max flow value")
	}
}

func TestHTMLGenerator_Generate_Analytics(t *testing.T) {
	g := NewHTMLGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		AnalyticsData: &AnalyticsReportData{
			TotalCost: 1500.0,
			Currency:  "RUB",
			Bottlenecks: []*BottleneckData{
				{From: 1, To: 2, Utilization: 0.95, ImpactScore: 0.8, Severity: "HIGH"},
			},
			Recommendations: []*RecommendationData{
				{Type: "increase_capacity", Description: "Increase capacity"},
			},
			Efficiency: &EfficiencyData{
				OverallEfficiency: 0.85,
				Grade:             "B",
			},
		},
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := string(result)

	if !strings.Contains(html, "Analytics") {
		t.Error("Should contain Analytics")
	}
	if !strings.Contains(html, "1500") {
		t.Error("Should contain cost value")
	}
	if !strings.Contains(html, "RUB") {
		t.Error("Should contain currency")
	}
	if !strings.Contains(html, "Bottlenecks") {
		t.Error("Should contain Bottlenecks section")
	}
}

func TestHTMLGenerator_Generate_ValidHTML(t *testing.T) {
	g := NewHTMLGenerator()
	ctx := context.Background()

	data := &ReportData{
		Type: reportv1.ReportType_REPORT_TYPE_FLOW,
	}

	result, err := g.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := string(result)

	// Проверяем закрытие тегов
	if strings.Count(html, "<html") != strings.Count(html, "</html>") {
		t.Error("HTML tags not balanced")
	}
	if strings.Count(html, "<body>") != strings.Count(html, "</body>") {
		t.Error("Body tags not balanced")
	}
	if strings.Count(html, "<div") != strings.Count(html, "</div>") {
		t.Error("Div tags not balanced")
	}
}
