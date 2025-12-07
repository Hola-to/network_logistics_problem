// services/report-svc/internal/generator/csv.go
package generator

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// CSVGenerator генератор CSV отчётов
type CSVGenerator struct {
	BaseGenerator
}

// NewCSVGenerator создаёт новый генератор
func NewCSVGenerator() *CSVGenerator {
	return &CSVGenerator{}
}

// Format возвращает формат генератора
func (g *CSVGenerator) Format() reportv1.ReportFormat {
	return reportv1.ReportFormat_REPORT_FORMAT_CSV
}

// csvWriter обёртка для отслеживания ошибок
type csvWriter struct {
	w   *csv.Writer
	err error
}

func (cw *csvWriter) Write(record []string) {
	if cw.err != nil {
		return
	}
	cw.err = cw.w.Write(record)
}

func (cw *csvWriter) Flush() {
	if cw.err != nil {
		return
	}
	cw.w.Flush()
	cw.err = cw.w.Error()
}

func (cw *csvWriter) Error() error {
	return cw.err
}

// Generate генерирует CSV отчёт
func (g *CSVGenerator) Generate(ctx context.Context, data *ReportData) ([]byte, error) {
	var buf bytes.Buffer
	cw := &csvWriter{w: csv.NewWriter(&buf)}

	switch data.Type {
	case reportv1.ReportType_REPORT_TYPE_FLOW:
		g.writeFlowCSV(cw, data)
	case reportv1.ReportType_REPORT_TYPE_ANALYTICS:
		g.writeAnalyticsCSV(cw, data)
	case reportv1.ReportType_REPORT_TYPE_SIMULATION:
		g.writeSimulationCSV(cw, data)
	case reportv1.ReportType_REPORT_TYPE_COMPARISON:
		g.writeComparisonCSV(cw, data)
	case reportv1.ReportType_REPORT_TYPE_SUMMARY:
		g.writeSummaryCSV(cw, data)
	default:
		g.writeFlowCSV(cw, data)
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return nil, fmt.Errorf("csv write error: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *CSVGenerator) writeFlowCSV(w *csvWriter, data *ReportData) {
	w.Write([]string{"# Flow Report"})
	w.Write([]string{""})

	if data.Graph != nil {
		w.Write([]string{"Graph Info"})
		w.Write([]string{"Nodes", fmt.Sprintf("%d", len(data.Graph.Nodes))})
		w.Write([]string{"Edges", fmt.Sprintf("%d", len(data.Graph.Edges))})
		w.Write([]string{"Source", fmt.Sprintf("%d", data.Graph.SourceId)})
		w.Write([]string{"Sink", fmt.Sprintf("%d", data.Graph.SinkId)})
		w.Write([]string{""})
	}

	if data.FlowResult != nil {
		w.Write([]string{"Flow Results"})
		w.Write([]string{"Max Flow", g.FormatFloat(data.FlowResult.MaxFlow, 4)})
		w.Write([]string{"Total Cost", g.FormatFloat(data.FlowResult.TotalCost, 4)})
		w.Write([]string{"Status", data.FlowResult.Status.String()})
		w.Write([]string{"Iterations", fmt.Sprintf("%d", data.FlowResult.Iterations)})
		w.Write([]string{"Computation Time (ms)", g.FormatFloat(data.FlowResult.ComputationTimeMs, 2)})
		w.Write([]string{""})

		// Используем FlowEdges если есть, иначе берём из FlowResult
		edges := data.FlowEdges
		if edges == nil && len(data.FlowResult.Edges) > 0 {
			edges = ConvertFlowEdges(data.FlowResult.Edges)
		}

		if len(edges) > 0 && g.ShouldIncludeRawData(data) {
			w.Write([]string{"Edge Flows"})
			w.Write([]string{"From", "To", "Flow", "Capacity", "Cost", "Utilization"})
			for _, edge := range edges {
				if edge.Flow > 0.001 {
					w.Write([]string{
						fmt.Sprintf("%d", edge.From),
						fmt.Sprintf("%d", edge.To),
						g.FormatFloat(edge.Flow, 4),
						g.FormatFloat(edge.Capacity, 4),
						g.FormatFloat(edge.Cost, 4),
						g.FormatFloat(edge.Utilization, 4),
					})
				}
			}
		}
	}
}

func (g *CSVGenerator) writeAnalyticsCSV(w *csvWriter, data *ReportData) {
	w.Write([]string{"# Analytics Report"})
	w.Write([]string{""})

	if data.AnalyticsData == nil {
		w.Write([]string{"No analytics data"})
		return
	}

	ad := data.AnalyticsData

	w.Write([]string{"Cost Summary"})
	w.Write([]string{"Total Cost", g.FormatFloat(ad.TotalCost, 4)})
	w.Write([]string{"Currency", ad.Currency})
	w.Write([]string{""})

	if ad.CostBreakdown != nil {
		w.Write([]string{"Cost Breakdown"})
		w.Write([]string{"Category", "Amount"})
		w.Write([]string{"Transport Cost", g.FormatFloat(ad.CostBreakdown.TransportCost, 4)})
		w.Write([]string{"Fixed Cost", g.FormatFloat(ad.CostBreakdown.FixedCost, 4)})
		w.Write([]string{"Handling Cost", g.FormatFloat(ad.CostBreakdown.HandlingCost, 4)})
		w.Write([]string{""})

		if len(ad.CostBreakdown.CostByRoadType) > 0 {
			w.Write([]string{"Cost by Road Type"})
			w.Write([]string{"Road Type", "Cost"})
			for rt, cost := range ad.CostBreakdown.CostByRoadType {
				w.Write([]string{rt, g.FormatFloat(cost, 4)})
			}
			w.Write([]string{""})
		}
	}

	if len(ad.Bottlenecks) > 0 {
		w.Write([]string{"Bottlenecks"})
		w.Write([]string{"From", "To", "Utilization", "Impact Score", "Severity"})
		for _, bn := range ad.Bottlenecks {
			w.Write([]string{
				fmt.Sprintf("%d", bn.From),
				fmt.Sprintf("%d", bn.To),
				g.FormatFloat(bn.Utilization, 4),
				g.FormatFloat(bn.ImpactScore, 4),
				bn.Severity,
			})
		}
		w.Write([]string{""})
	}

	if len(ad.Recommendations) > 0 && g.ShouldIncludeRecommendations(data) {
		w.Write([]string{"Recommendations"})
		w.Write([]string{"Type", "Description", "Estimated Improvement", "Estimated Cost"})
		for _, rec := range ad.Recommendations {
			w.Write([]string{
				rec.Type,
				rec.Description,
				g.FormatFloat(rec.EstimatedImprovement, 4),
				g.FormatFloat(rec.EstimatedCost, 4),
			})
		}
		w.Write([]string{""})
	}

	if ad.Efficiency != nil {
		w.Write([]string{"Efficiency Metrics"})
		w.Write([]string{"Metric", "Value"})
		w.Write([]string{"Overall Efficiency", g.FormatFloat(ad.Efficiency.OverallEfficiency, 4)})
		w.Write([]string{"Capacity Utilization", g.FormatFloat(ad.Efficiency.CapacityUtilization, 4)})
		w.Write([]string{"Unused Edges", fmt.Sprintf("%d", ad.Efficiency.UnusedEdges)})
		w.Write([]string{"Saturated Edges", fmt.Sprintf("%d", ad.Efficiency.SaturatedEdges)})
		w.Write([]string{"Grade", ad.Efficiency.Grade})
	}
}

func (g *CSVGenerator) writeSimulationCSV(w *csvWriter, data *ReportData) {
	w.Write([]string{"# Simulation Report"})
	w.Write([]string{""})

	if data.SimulationData == nil {
		w.Write([]string{"No simulation data"})
		return
	}

	sd := data.SimulationData

	w.Write([]string{"Simulation Type", sd.SimulationType})
	w.Write([]string{"Baseline Flow", g.FormatFloat(sd.BaselineFlow, 4)})
	w.Write([]string{"Baseline Cost", g.FormatFloat(sd.BaselineCost, 4)})
	w.Write([]string{""})

	if len(sd.Scenarios) > 0 {
		w.Write([]string{"Scenarios"})
		w.Write([]string{"Name", "Max Flow", "Total Cost", "Flow Change %", "Impact Level"})
		for _, sc := range sd.Scenarios {
			w.Write([]string{
				sc.Name,
				g.FormatFloat(sc.MaxFlow, 4),
				g.FormatFloat(sc.TotalCost, 4),
				g.FormatFloat(sc.FlowChangePercent, 2),
				sc.ImpactLevel,
			})
		}
		w.Write([]string{""})
	}

	if sd.MonteCarlo != nil {
		mc := sd.MonteCarlo
		w.Write([]string{"Monte Carlo Results"})
		w.Write([]string{"Metric", "Value"})
		w.Write([]string{"Iterations", fmt.Sprintf("%d", mc.Iterations)})
		w.Write([]string{"Mean Flow", g.FormatFloat(mc.MeanFlow, 4)})
		w.Write([]string{"Std Dev", g.FormatFloat(mc.StdDev, 4)})
		w.Write([]string{"Min Flow", g.FormatFloat(mc.MinFlow, 4)})
		w.Write([]string{"Max Flow", g.FormatFloat(mc.MaxFlow, 4)})
		w.Write([]string{"P5", g.FormatFloat(mc.P5, 4)})
		w.Write([]string{"P50 (Median)", g.FormatFloat(mc.P50, 4)})
		w.Write([]string{"P95", g.FormatFloat(mc.P95, 4)})
		w.Write([]string{"Confidence Level", g.FormatFloat(mc.ConfidenceLevel, 4)})
		w.Write([]string{"CI Low", g.FormatFloat(mc.CiLow, 4)})
		w.Write([]string{"CI High", g.FormatFloat(mc.CiHigh, 4)})
		w.Write([]string{""})
	}

	if len(sd.Sensitivity) > 0 {
		w.Write([]string{"Sensitivity Analysis"})
		w.Write([]string{"Parameter", "Elasticity", "Sensitivity Index", "Level"})
		for _, sp := range sd.Sensitivity {
			w.Write([]string{
				sp.ParameterId,
				g.FormatFloat(sp.Elasticity, 4),
				g.FormatFloat(sp.SensitivityIndex, 4),
				sp.Level,
			})
		}
		w.Write([]string{""})
	}

	if sd.Resilience != nil {
		r := sd.Resilience
		w.Write([]string{"Resilience Analysis"})
		w.Write([]string{"Metric", "Value"})
		w.Write([]string{"Overall Score", g.FormatFloat(r.OverallScore, 4)})
		w.Write([]string{"Single Points of Failure", fmt.Sprintf("%d", r.SinglePointsOfFailure)})
		w.Write([]string{"Worst Case Flow Reduction", g.FormatFloat(r.WorstCaseFlowReduction, 4)})
		w.Write([]string{"N-1 Feasible", fmt.Sprintf("%v", r.NMinusOneFeasible)})
	}
}

func (g *CSVGenerator) writeComparisonCSV(w *csvWriter, data *ReportData) {
	w.Write([]string{"# Comparison Report"})
	w.Write([]string{""})

	if len(data.ComparisonData) == 0 {
		w.Write([]string{"No comparison data"})
		return
	}

	w.Write([]string{"Scenario Comparison"})
	w.Write([]string{"Name", "Max Flow", "Total Cost", "Efficiency"})
	for _, item := range data.ComparisonData {
		w.Write([]string{
			item.Name,
			g.FormatFloat(item.MaxFlow, 4),
			g.FormatFloat(item.TotalCost, 4),
			g.FormatFloat(item.Efficiency, 4),
		})
	}
	w.Write([]string{""})

	// Детальные метрики
	if len(data.ComparisonData) > 0 && len(data.ComparisonData[0].Metrics) > 0 {
		var keys []string
		for k := range data.ComparisonData[0].Metrics {
			keys = append(keys, k)
		}

		header := []string{"Metric"}
		for _, item := range data.ComparisonData {
			header = append(header, item.Name)
		}
		w.Write(header)

		for _, key := range keys {
			row := []string{key}
			for _, item := range data.ComparisonData {
				row = append(row, g.FormatFloat(item.Metrics[key], 4))
			}
			w.Write(row)
		}
	}
}

func (g *CSVGenerator) writeSummaryCSV(w *csvWriter, data *ReportData) {
	w.Write([]string{"# Summary Report"})
	w.Write([]string{""})

	g.writeFlowCSV(w, data)

	if data.AnalyticsData != nil {
		w.Write([]string{""})
		w.Write([]string{"=== ANALYTICS ==="})
		g.writeAnalyticsCSV(w, data)
	}

	if data.SimulationData != nil {
		w.Write([]string{""})
		w.Write([]string{"=== SIMULATION ==="})
		g.writeSimulationCSV(w, data)
	}
}
