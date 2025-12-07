// services/report-svc/internal/generator/markdown.go
package generator

import (
	"bytes"
	"context"
	"fmt"
	"time"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// MarkdownGenerator генератор Markdown отчётов
type MarkdownGenerator struct {
	BaseGenerator
}

// NewMarkdownGenerator создаёт новый генератор
func NewMarkdownGenerator() *MarkdownGenerator {
	return &MarkdownGenerator{}
}

// Format возвращает формат генератора
func (g *MarkdownGenerator) Format() reportv1.ReportFormat {
	return reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN
}

// Generate генерирует Markdown отчёт
func (g *MarkdownGenerator) Generate(ctx context.Context, data *ReportData) ([]byte, error) {
	var buf bytes.Buffer

	// Заголовок
	g.writeHeader(&buf, data)

	// Содержимое в зависимости от типа
	switch data.Type {
	case reportv1.ReportType_REPORT_TYPE_FLOW:
		g.writeFlowReport(&buf, data)
	case reportv1.ReportType_REPORT_TYPE_ANALYTICS:
		g.writeAnalyticsReport(&buf, data)
	case reportv1.ReportType_REPORT_TYPE_SIMULATION:
		g.writeSimulationReport(&buf, data)
	case reportv1.ReportType_REPORT_TYPE_SUMMARY:
		g.writeSummaryReport(&buf, data)
	case reportv1.ReportType_REPORT_TYPE_COMPARISON:
		g.writeComparisonReport(&buf, data)
	default:
		g.writeFlowReport(&buf, data)
	}

	// Футер
	g.writeFooter(&buf)

	return buf.Bytes(), nil
}

func (g *MarkdownGenerator) writeHeader(buf *bytes.Buffer, data *ReportData) {
	title := g.GetTitle(data)
	buf.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Метаданные
	buf.WriteString("## Report Information\n\n")
	buf.WriteString(fmt.Sprintf("- **Generated:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString(fmt.Sprintf("- **Author:** %s\n", g.GetAuthor(data)))

	if desc := g.GetDescription(data); desc != "" {
		buf.WriteString(fmt.Sprintf("- **Description:** %s\n", desc))
	}

	buf.WriteString("\n---\n\n")
}

func (g *MarkdownGenerator) writeFlowReport(buf *bytes.Buffer, data *ReportData) {
	// Информация о графе
	if data.Graph != nil {
		buf.WriteString("## Network Information\n\n")
		buf.WriteString(fmt.Sprintf("- **Nodes:** %d\n", len(data.Graph.Nodes)))
		buf.WriteString(fmt.Sprintf("- **Edges:** %d\n", len(data.Graph.Edges)))
		buf.WriteString(fmt.Sprintf("- **Source:** %d\n", data.Graph.SourceId))
		buf.WriteString(fmt.Sprintf("- **Sink:** %d\n", data.Graph.SinkId))
		buf.WriteString("\n")
	}

	// Результаты оптимизации
	if data.FlowResult != nil {
		buf.WriteString("## Optimization Results\n\n")
		buf.WriteString(fmt.Sprintf("- **Maximum Flow:** %.4f\n", data.FlowResult.MaxFlow))
		buf.WriteString(fmt.Sprintf("- **Total Cost:** %.2f\n", data.FlowResult.TotalCost))
		buf.WriteString(fmt.Sprintf("- **Status:** %s\n", data.FlowResult.Status.String()))
		buf.WriteString(fmt.Sprintf("- **Iterations:** %d\n", data.FlowResult.Iterations))
		buf.WriteString(fmt.Sprintf("- **Computation Time:** %.2f ms\n", data.FlowResult.ComputationTimeMs))
		buf.WriteString("\n")
	}

	// Таблица рёбер с потоком
	edges := data.FlowEdges
	if edges == nil && data.FlowResult != nil {
		edges = ConvertFlowEdges(data.FlowResult.Edges)
	}

	if len(edges) > 0 && g.ShouldIncludeRawData(data) {
		buf.WriteString("### Edge Flows\n\n")
		buf.WriteString("| From | To | Flow | Capacity | Utilization |\n")
		buf.WriteString("|------|-----|------|----------|-------------|\n")
		for _, edge := range edges {
			if edge.Flow > 0.001 {
				buf.WriteString(fmt.Sprintf("| %d | %d | %.4f | %.4f | %.1f%% |\n",
					edge.From, edge.To, edge.Flow, edge.Capacity, edge.Utilization*100))
			}
		}
		buf.WriteString("\n")
	}
}

func (g *MarkdownGenerator) writeAnalyticsReport(buf *bytes.Buffer, data *ReportData) {
	if data.AnalyticsData == nil {
		buf.WriteString("*No analytics data available*\n\n")
		return
	}

	ad := data.AnalyticsData

	// Стоимость
	buf.WriteString("## Cost Analysis\n\n")
	buf.WriteString(fmt.Sprintf("- **Total Cost:** %.2f %s\n", ad.TotalCost, ad.Currency))

	if ad.CostBreakdown != nil {
		buf.WriteString("\n### Cost Breakdown\n\n")
		buf.WriteString("| Category | Amount |\n")
		buf.WriteString("|----------|--------|\n")
		buf.WriteString(fmt.Sprintf("| Transport | %.2f |\n", ad.CostBreakdown.TransportCost))
		buf.WriteString(fmt.Sprintf("| Fixed | %.2f |\n", ad.CostBreakdown.FixedCost))
		buf.WriteString(fmt.Sprintf("| Handling | %.2f |\n", ad.CostBreakdown.HandlingCost))
		buf.WriteString("\n")

		if len(ad.CostBreakdown.CostByRoadType) > 0 {
			buf.WriteString("#### Cost by Road Type\n\n")
			buf.WriteString("| Road Type | Cost |\n")
			buf.WriteString("|-----------|------|\n")
			for roadType, cost := range ad.CostBreakdown.CostByRoadType {
				buf.WriteString(fmt.Sprintf("| %s | %.2f |\n", roadType, cost))
			}
			buf.WriteString("\n")
		}
	}

	// Bottlenecks
	if len(ad.Bottlenecks) > 0 {
		buf.WriteString("## Bottlenecks\n\n")
		buf.WriteString("| From → To | Utilization | Impact | Severity |\n")
		buf.WriteString("|-----------|-------------|--------|----------|\n")
		for _, bn := range ad.Bottlenecks {
			buf.WriteString(fmt.Sprintf("| %d → %d | %.1f%% | %.2f | %s |\n",
				bn.From, bn.To, bn.Utilization*100, bn.ImpactScore, bn.Severity))
		}
		buf.WriteString("\n")
	}

	// Рекомендации
	if len(ad.Recommendations) > 0 && g.ShouldIncludeRecommendations(data) {
		buf.WriteString("## Recommendations\n\n")
		for i, rec := range ad.Recommendations {
			buf.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, rec.Type))
			buf.WriteString(fmt.Sprintf("%s\n\n", rec.Description))
			if rec.EstimatedImprovement > 0 {
				buf.WriteString(fmt.Sprintf("- Expected improvement: **%.1f%%**\n", rec.EstimatedImprovement*100))
			}
			if rec.EstimatedCost > 0 {
				buf.WriteString(fmt.Sprintf("- Estimated cost: **%.2f**\n", rec.EstimatedCost))
			}
			buf.WriteString("\n")
		}
	}

	// Эффективность
	if ad.Efficiency != nil {
		buf.WriteString("## Efficiency Metrics\n\n")
		buf.WriteString(fmt.Sprintf("- **Overall Efficiency:** %.1f%%\n", ad.Efficiency.OverallEfficiency*100))
		buf.WriteString(fmt.Sprintf("- **Capacity Utilization:** %.1f%%\n", ad.Efficiency.CapacityUtilization*100))
		buf.WriteString(fmt.Sprintf("- **Unused Edges:** %d\n", ad.Efficiency.UnusedEdges))
		buf.WriteString(fmt.Sprintf("- **Saturated Edges:** %d\n", ad.Efficiency.SaturatedEdges))
		buf.WriteString(fmt.Sprintf("- **Grade:** %s\n", ad.Efficiency.Grade))
		buf.WriteString("\n")
	}
}

func (g *MarkdownGenerator) writeSimulationReport(buf *bytes.Buffer, data *ReportData) {
	if data.SimulationData == nil {
		buf.WriteString("*No simulation data available*\n\n")
		return
	}

	sd := data.SimulationData

	buf.WriteString(fmt.Sprintf("## Simulation Type: %s\n\n", sd.SimulationType))
	buf.WriteString("### Baseline\n\n")
	buf.WriteString(fmt.Sprintf("- **Baseline Flow:** %.4f\n", sd.BaselineFlow))
	buf.WriteString(fmt.Sprintf("- **Baseline Cost:** %.2f\n", sd.BaselineCost))
	buf.WriteString("\n")

	// Сценарии
	if len(sd.Scenarios) > 0 {
		buf.WriteString("### Scenario Results\n\n")
		buf.WriteString("| Scenario | Max Flow | Cost | Change | Impact |\n")
		buf.WriteString("|----------|----------|------|--------|--------|\n")
		for _, sc := range sd.Scenarios {
			buf.WriteString(fmt.Sprintf("| %s | %.4f | %.2f | %.1f%% | %s |\n",
				sc.Name, sc.MaxFlow, sc.TotalCost, sc.FlowChangePercent, sc.ImpactLevel))
		}
		buf.WriteString("\n")
	}

	// Monte Carlo
	if sd.MonteCarlo != nil {
		mc := sd.MonteCarlo
		buf.WriteString("### Monte Carlo Results\n\n")
		buf.WriteString(fmt.Sprintf("- **Iterations:** %d\n", mc.Iterations))
		buf.WriteString(fmt.Sprintf("- **Mean Flow:** %.4f ± %.4f\n", mc.MeanFlow, mc.StdDev))
		buf.WriteString(fmt.Sprintf("- **Range:** %.4f - %.4f\n", mc.MinFlow, mc.MaxFlow))
		buf.WriteString(fmt.Sprintf("- **Median (P50):** %.4f\n", mc.P50))
		buf.WriteString(fmt.Sprintf("- **P5 - P95:** %.4f - %.4f\n", mc.P5, mc.P95))
		buf.WriteString(fmt.Sprintf("- **Confidence Interval (%.0f%%):** %.4f - %.4f\n",
			mc.ConfidenceLevel*100, mc.CiLow, mc.CiHigh))
		buf.WriteString("\n")
	}

	// Sensitivity
	if len(sd.Sensitivity) > 0 {
		buf.WriteString("### Sensitivity Analysis\n\n")
		buf.WriteString("| Parameter | Elasticity | Index | Level |\n")
		buf.WriteString("|-----------|------------|-------|-------|\n")
		for _, sp := range sd.Sensitivity {
			buf.WriteString(fmt.Sprintf("| %s | %.4f | %.4f | %s |\n",
				sp.ParameterId, sp.Elasticity, sp.SensitivityIndex, sp.Level))
		}
		buf.WriteString("\n")
	}

	// Resilience
	if sd.Resilience != nil {
		r := sd.Resilience
		buf.WriteString("### Resilience Analysis\n\n")
		buf.WriteString(fmt.Sprintf("- **Overall Score:** %.2f\n", r.OverallScore))
		buf.WriteString(fmt.Sprintf("- **Single Points of Failure:** %d\n", r.SinglePointsOfFailure))
		buf.WriteString(fmt.Sprintf("- **Worst Case Flow Reduction:** %.1f%%\n", r.WorstCaseFlowReduction*100))
		buf.WriteString(fmt.Sprintf("- **N-1 Feasible:** %v\n", r.NMinusOneFeasible))
		buf.WriteString("\n")
	}
}

func (g *MarkdownGenerator) writeSummaryReport(buf *bytes.Buffer, data *ReportData) {
	buf.WriteString("## Summary Report\n\n")

	// Flow
	if data.FlowResult != nil {
		g.writeFlowReport(buf, data)
	}

	// Analytics
	if data.AnalyticsData != nil {
		g.writeAnalyticsReport(buf, data)
	}

	// Simulation
	if data.SimulationData != nil {
		g.writeSimulationReport(buf, data)
	}
}

func (g *MarkdownGenerator) writeComparisonReport(buf *bytes.Buffer, data *ReportData) {
	if len(data.ComparisonData) == 0 {
		buf.WriteString("*No comparison data available*\n\n")
		return
	}

	buf.WriteString("## Scenario Comparison\n\n")

	// Основная таблица
	buf.WriteString("| Scenario | Max Flow | Total Cost | Efficiency |\n")
	buf.WriteString("|----------|----------|------------|------------|\n")
	for _, item := range data.ComparisonData {
		buf.WriteString(fmt.Sprintf("| %s | %.4f | %.2f | %.1f%% |\n",
			item.Name, item.MaxFlow, item.TotalCost, item.Efficiency*100))
	}
	buf.WriteString("\n")

	// Детальное сравнение метрик
	if len(data.ComparisonData) > 0 && len(data.ComparisonData[0].Metrics) > 0 {
		buf.WriteString("### Detailed Metrics\n\n")

		// Собираем все ключи метрик
		metricsKeys := make(map[string]bool)
		for _, item := range data.ComparisonData {
			for k := range item.Metrics {
				metricsKeys[k] = true
			}
		}

		// Заголовок таблицы
		header := "| Metric |"
		separator := "|--------|"
		for _, item := range data.ComparisonData {
			header += fmt.Sprintf(" %s |", item.Name)
			separator += "--------|"
		}
		buf.WriteString(header + "\n")
		buf.WriteString(separator + "\n")

		// Строки с метриками
		for metric := range metricsKeys {
			row := fmt.Sprintf("| %s |", metric)
			for _, item := range data.ComparisonData {
				val := item.Metrics[metric]
				row += fmt.Sprintf(" %.4f |", val)
			}
			buf.WriteString(row + "\n")
		}
		buf.WriteString("\n")
	}

	// Выводы
	buf.WriteString("### Conclusions\n\n")
	best := g.findBest(data.ComparisonData)
	if best != nil {
		buf.WriteString(fmt.Sprintf("Best scenario by flow: **%s** (%.4f)\n\n", best.Name, best.MaxFlow))
	}
}

func (g *MarkdownGenerator) findBest(items []*ComparisonItemData) *ComparisonItemData {
	if len(items) == 0 {
		return nil
	}
	best := items[0]
	for _, item := range items[1:] {
		if item.MaxFlow > best.MaxFlow {
			best = item
		}
	}
	return best
}

func (g *MarkdownGenerator) writeFooter(buf *bytes.Buffer) {
	buf.WriteString("\n---\n\n")
	buf.WriteString("*Report generated automatically by Logistics Platform*\n")
	buf.WriteString(fmt.Sprintf("*%s*\n", time.Now().Format("2006-01-02 15:04:05")))
}
