// services/report-svc/internal/generator/excel.go
package generator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// ExcelGenerator генератор Excel отчётов
type ExcelGenerator struct {
	BaseGenerator
}

// NewExcelGenerator создаёт новый генератор
func NewExcelGenerator() *ExcelGenerator {
	return &ExcelGenerator{}
}

// Format возвращает формат генератора
func (g *ExcelGenerator) Format() reportv1.ReportFormat {
	return reportv1.ReportFormat_REPORT_FORMAT_EXCEL
}

// Generate генерирует Excel отчёт
func (g *ExcelGenerator) Generate(ctx context.Context, data *ReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Удаляем дефолтный лист
	f.DeleteSheet("Sheet1")

	switch data.Type {
	case reportv1.ReportType_REPORT_TYPE_FLOW:
		g.writeFlowExcel(f, data)
	case reportv1.ReportType_REPORT_TYPE_ANALYTICS:
		g.writeAnalyticsExcel(f, data)
	case reportv1.ReportType_REPORT_TYPE_SIMULATION:
		g.writeSimulationExcel(f, data)
	case reportv1.ReportType_REPORT_TYPE_SUMMARY:
		g.writeSummaryExcel(f, data)
	case reportv1.ReportType_REPORT_TYPE_COMPARISON:
		g.writeComparisonExcel(f, data)
	default:
		g.writeFlowExcel(f, data)
	}

	// Записываем в буфер
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *ExcelGenerator) writeFlowExcel(f *excelize.File, data *ReportData) {
	// Лист с результатами
	sheetName := "Flow Results"
	f.NewSheet(sheetName)

	// Стили
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	row := 1

	// Заголовок
	f.SetCellValue(sheetName, cellAddr("A", row), "Flow Optimization Report")
	f.MergeCell(sheetName, cellAddr("A", row), cellAddr("D", row))
	row += 2

	// Метаданные
	if data.Graph != nil {
		f.SetCellValue(sheetName, cellAddr("A", row), "Graph Information")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("B", row), headerStyle)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Nodes")
		f.SetCellValue(sheetName, cellAddr("B", row), len(data.Graph.Nodes))
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Edges")
		f.SetCellValue(sheetName, cellAddr("B", row), len(data.Graph.Edges))
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Source")
		f.SetCellValue(sheetName, cellAddr("B", row), data.Graph.SourceId)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Sink")
		f.SetCellValue(sheetName, cellAddr("B", row), data.Graph.SinkId)
		row += 2
	}

	// Результаты
	if data.FlowResult != nil {
		f.SetCellValue(sheetName, cellAddr("A", row), "Optimization Results")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("B", row), headerStyle)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Max Flow")
		f.SetCellValue(sheetName, cellAddr("B", row), data.FlowResult.MaxFlow)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Total Cost")
		f.SetCellValue(sheetName, cellAddr("B", row), data.FlowResult.TotalCost)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Status")
		f.SetCellValue(sheetName, cellAddr("B", row), data.FlowResult.Status.String())
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Iterations")
		f.SetCellValue(sheetName, cellAddr("B", row), data.FlowResult.Iterations)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Computation Time (ms)")
		f.SetCellValue(sheetName, cellAddr("B", row), data.FlowResult.ComputationTimeMs)
		row += 2

		// Таблица потоков
		edges := data.FlowEdges
		if edges == nil && len(data.FlowResult.Edges) > 0 {
			edges = ConvertFlowEdges(data.FlowResult.Edges)
		}

		if len(edges) > 0 && g.ShouldIncludeRawData(data) {
			f.SetCellValue(sheetName, cellAddr("A", row), "Edge Flows")
			f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("F", row), headerStyle)
			row++

			headers := []string{"From", "To", "Flow", "Capacity", "Cost", "Utilization"}
			for i, h := range headers {
				f.SetCellValue(sheetName, cellAddr(string(rune('A'+i)), row), h)
			}
			f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("F", row), headerStyle)
			row++

			for _, edge := range edges {
				if edge.Flow > 0.001 {
					f.SetCellValue(sheetName, cellAddr("A", row), edge.From)
					f.SetCellValue(sheetName, cellAddr("B", row), edge.To)
					f.SetCellValue(sheetName, cellAddr("C", row), edge.Flow)
					f.SetCellValue(sheetName, cellAddr("D", row), edge.Capacity)
					f.SetCellValue(sheetName, cellAddr("E", row), edge.Cost)
					f.SetCellValue(sheetName, cellAddr("F", row), edge.Utilization)
					row++
				}
			}
		}
	}

	// Лист с узлами
	if data.Graph != nil && len(data.Graph.Nodes) > 0 && g.ShouldIncludeRawData(data) {
		nodesSheet := "Nodes"
		f.NewSheet(nodesSheet)

		headers := []string{"ID", "X", "Y", "Type", "Name", "Supply", "Demand"}
		for i, h := range headers {
			f.SetCellValue(nodesSheet, cellAddr(string(rune('A'+i)), 1), h)
		}
		f.SetCellStyle(nodesSheet, "A1", "G1", headerStyle)

		for i, node := range data.Graph.Nodes {
			row := i + 2
			f.SetCellValue(nodesSheet, cellAddr("A", row), node.Id)
			f.SetCellValue(nodesSheet, cellAddr("B", row), node.X)
			f.SetCellValue(nodesSheet, cellAddr("C", row), node.Y)
			f.SetCellValue(nodesSheet, cellAddr("D", row), node.Type.String())
			f.SetCellValue(nodesSheet, cellAddr("E", row), node.Name)
			f.SetCellValue(nodesSheet, cellAddr("F", row), node.Supply)
			f.SetCellValue(nodesSheet, cellAddr("G", row), node.Demand)
		}
	}

	// Лист с рёбрами
	if data.Graph != nil && len(data.Graph.Edges) > 0 && g.ShouldIncludeRawData(data) {
		edgesSheet := "Edges"
		f.NewSheet(edgesSheet)

		headers := []string{"From", "To", "Capacity", "Cost", "Length", "Road Type", "Current Flow"}
		for i, h := range headers {
			f.SetCellValue(edgesSheet, cellAddr(string(rune('A'+i)), 1), h)
		}
		f.SetCellStyle(edgesSheet, "A1", "G1", headerStyle)

		for i, edge := range data.Graph.Edges {
			row := i + 2
			f.SetCellValue(edgesSheet, cellAddr("A", row), edge.From)
			f.SetCellValue(edgesSheet, cellAddr("B", row), edge.To)
			f.SetCellValue(edgesSheet, cellAddr("C", row), edge.Capacity)
			f.SetCellValue(edgesSheet, cellAddr("D", row), edge.Cost)
			f.SetCellValue(edgesSheet, cellAddr("E", row), edge.Length)
			f.SetCellValue(edgesSheet, cellAddr("F", row), edge.RoadType.String())
			f.SetCellValue(edgesSheet, cellAddr("G", row), edge.CurrentFlow)
		}
	}

	// Авто-ширина колонок
	f.SetColWidth(sheetName, "A", "F", 15)
}

func (g *ExcelGenerator) writeAnalyticsExcel(f *excelize.File, data *ReportData) {
	sheetName := "Analytics"
	f.NewSheet(sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	row := 1

	f.SetCellValue(sheetName, cellAddr("A", row), "Analytics Report")
	row += 2

	if data.AnalyticsData == nil {
		f.SetCellValue(sheetName, cellAddr("A", row), "No analytics data")
		return
	}

	ad := data.AnalyticsData

	// Стоимость
	f.SetCellValue(sheetName, cellAddr("A", row), "Cost Summary")
	f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("B", row), headerStyle)
	row++

	f.SetCellValue(sheetName, cellAddr("A", row), "Total Cost")
	f.SetCellValue(sheetName, cellAddr("B", row), ad.TotalCost)
	row++

	f.SetCellValue(sheetName, cellAddr("A", row), "Currency")
	f.SetCellValue(sheetName, cellAddr("B", row), ad.Currency)
	row += 2

	// Разбивка стоимости
	if ad.CostBreakdown != nil {
		f.SetCellValue(sheetName, cellAddr("A", row), "Cost Breakdown")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("B", row), headerStyle)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Transport Cost")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.CostBreakdown.TransportCost)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Fixed Cost")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.CostBreakdown.FixedCost)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Handling Cost")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.CostBreakdown.HandlingCost)
		row += 2
	}

	// Bottlenecks
	if len(ad.Bottlenecks) > 0 {
		f.SetCellValue(sheetName, cellAddr("A", row), "Bottlenecks")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("E", row), headerStyle)
		row++

		headers := []string{"From", "To", "Utilization", "Impact Score", "Severity"}
		for i, h := range headers {
			f.SetCellValue(sheetName, cellAddr(string(rune('A'+i)), row), h)
		}
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("E", row), headerStyle)
		row++

		for _, bn := range ad.Bottlenecks {
			f.SetCellValue(sheetName, cellAddr("A", row), bn.From)
			f.SetCellValue(sheetName, cellAddr("B", row), bn.To)
			f.SetCellValue(sheetName, cellAddr("C", row), bn.Utilization)
			f.SetCellValue(sheetName, cellAddr("D", row), bn.ImpactScore)
			f.SetCellValue(sheetName, cellAddr("E", row), bn.Severity)
			row++
		}
		row++
	}

	// Эффективность
	if ad.Efficiency != nil {
		f.SetCellValue(sheetName, cellAddr("A", row), "Efficiency Metrics")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("B", row), headerStyle)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Overall Efficiency")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.Efficiency.OverallEfficiency)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Capacity Utilization")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.Efficiency.CapacityUtilization)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Unused Edges")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.Efficiency.UnusedEdges)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Saturated Edges")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.Efficiency.SaturatedEdges)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Grade")
		f.SetCellValue(sheetName, cellAddr("B", row), ad.Efficiency.Grade)
	}

	f.SetColWidth(sheetName, "A", "E", 18)
}

func (g *ExcelGenerator) writeSimulationExcel(f *excelize.File, data *ReportData) {
	sheetName := "Simulation"
	f.NewSheet(sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	row := 1

	if data.SimulationData == nil {
		f.SetCellValue(sheetName, cellAddr("A", row), "No simulation data")
		return
	}

	sd := data.SimulationData

	f.SetCellValue(sheetName, cellAddr("A", row), "Simulation Report")
	row += 2

	f.SetCellValue(sheetName, cellAddr("A", row), "Type")
	f.SetCellValue(sheetName, cellAddr("B", row), sd.SimulationType)
	row++

	f.SetCellValue(sheetName, cellAddr("A", row), "Baseline Flow")
	f.SetCellValue(sheetName, cellAddr("B", row), sd.BaselineFlow)
	row++

	f.SetCellValue(sheetName, cellAddr("A", row), "Baseline Cost")
	f.SetCellValue(sheetName, cellAddr("B", row), sd.BaselineCost)
	row += 2

	// Сценарии
	if len(sd.Scenarios) > 0 {
		f.SetCellValue(sheetName, cellAddr("A", row), "Scenarios")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("E", row), headerStyle)
		row++

		headers := []string{"Name", "Max Flow", "Cost", "Change %", "Impact"}
		for i, h := range headers {
			f.SetCellValue(sheetName, cellAddr(string(rune('A'+i)), row), h)
		}
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("E", row), headerStyle)
		row++

		for _, sc := range sd.Scenarios {
			f.SetCellValue(sheetName, cellAddr("A", row), sc.Name)
			f.SetCellValue(sheetName, cellAddr("B", row), sc.MaxFlow)
			f.SetCellValue(sheetName, cellAddr("C", row), sc.TotalCost)
			f.SetCellValue(sheetName, cellAddr("D", row), sc.FlowChangePercent)
			f.SetCellValue(sheetName, cellAddr("E", row), sc.ImpactLevel)
			row++
		}
		row++
	}

	// Monte Carlo на отдельном листе
	if sd.MonteCarlo != nil {
		mcSheet := "Monte Carlo"
		f.NewSheet(mcSheet)

		mc := sd.MonteCarlo
		mcRow := 1

		f.SetCellValue(mcSheet, cellAddr("A", mcRow), "Monte Carlo Results")
		mcRow += 2

		metrics := []struct {
			name  string
			value any
		}{
			{"Iterations", mc.Iterations},
			{"Mean Flow", mc.MeanFlow},
			{"Std Dev", mc.StdDev},
			{"Min Flow", mc.MinFlow},
			{"Max Flow", mc.MaxFlow},
			{"P5", mc.P5},
			{"P50 (Median)", mc.P50},
			{"P95", mc.P95},
			{"Confidence Level", mc.ConfidenceLevel},
			{"CI Low", mc.CiLow},
			{"CI High", mc.CiHigh},
		}

		for _, m := range metrics {
			f.SetCellValue(mcSheet, cellAddr("A", mcRow), m.name)
			f.SetCellValue(mcSheet, cellAddr("B", mcRow), m.value)
			mcRow++
		}

		f.SetColWidth(mcSheet, "A", "B", 20)
	}

	// Sensitivity Analysis
	if len(sd.Sensitivity) > 0 {
		sensSheet := "Sensitivity"
		f.NewSheet(sensSheet)

		headers := []string{"Parameter", "Elasticity", "Sensitivity Index", "Level"}
		for i, h := range headers {
			f.SetCellValue(sensSheet, cellAddr(string(rune('A'+i)), 1), h)
		}
		f.SetCellStyle(sensSheet, "A1", "D1", headerStyle)

		for i, sp := range sd.Sensitivity {
			row := i + 2
			f.SetCellValue(sensSheet, cellAddr("A", row), sp.ParameterId)
			f.SetCellValue(sensSheet, cellAddr("B", row), sp.Elasticity)
			f.SetCellValue(sensSheet, cellAddr("C", row), sp.SensitivityIndex)
			f.SetCellValue(sensSheet, cellAddr("D", row), sp.Level)
		}

		f.SetColWidth(sensSheet, "A", "D", 18)
	}

	// Resilience
	if sd.Resilience != nil {
		r := sd.Resilience
		f.SetCellValue(sheetName, cellAddr("A", row), "Resilience Analysis")
		f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("B", row), headerStyle)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Overall Score")
		f.SetCellValue(sheetName, cellAddr("B", row), r.OverallScore)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Single Points of Failure")
		f.SetCellValue(sheetName, cellAddr("B", row), r.SinglePointsOfFailure)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "Worst Case Flow Reduction")
		f.SetCellValue(sheetName, cellAddr("B", row), r.WorstCaseFlowReduction)
		row++

		f.SetCellValue(sheetName, cellAddr("A", row), "N-1 Feasible")
		f.SetCellValue(sheetName, cellAddr("B", row), r.NMinusOneFeasible)
	}

	f.SetColWidth(sheetName, "A", "E", 18)
}

func (g *ExcelGenerator) writeSummaryExcel(f *excelize.File, data *ReportData) {
	g.writeFlowExcel(f, data)
	if data.AnalyticsData != nil {
		g.writeAnalyticsExcel(f, data)
	}
	if data.SimulationData != nil {
		g.writeSimulationExcel(f, data)
	}
}

func (g *ExcelGenerator) writeComparisonExcel(f *excelize.File, data *ReportData) {
	sheetName := "Comparison"
	f.NewSheet(sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	row := 1

	f.SetCellValue(sheetName, cellAddr("A", row), "Scenario Comparison")
	row += 2

	if len(data.ComparisonData) == 0 {
		f.SetCellValue(sheetName, cellAddr("A", row), "No comparison data")
		return
	}

	// Основная таблица
	headers := []string{"Name", "Max Flow", "Total Cost", "Efficiency"}
	for i, h := range headers {
		f.SetCellValue(sheetName, cellAddr(string(rune('A'+i)), row), h)
	}
	f.SetCellStyle(sheetName, cellAddr("A", row), cellAddr("D", row), headerStyle)
	row++

	for _, item := range data.ComparisonData {
		f.SetCellValue(sheetName, cellAddr("A", row), item.Name)
		f.SetCellValue(sheetName, cellAddr("B", row), item.MaxFlow)
		f.SetCellValue(sheetName, cellAddr("C", row), item.TotalCost)
		f.SetCellValue(sheetName, cellAddr("D", row), item.Efficiency)
		row++
	}

	// Детальные метрики на отдельном листе
	if len(data.ComparisonData) > 0 && len(data.ComparisonData[0].Metrics) > 0 {
		metricsSheet := "Detailed Metrics"
		f.NewSheet(metricsSheet)

		// Собираем ключи
		var keys []string
		for k := range data.ComparisonData[0].Metrics {
			keys = append(keys, k)
		}

		// Заголовок
		f.SetCellValue(metricsSheet, "A1", "Metric")
		for i, item := range data.ComparisonData {
			f.SetCellValue(metricsSheet, cellAddr(string(rune('B'+i)), 1), item.Name)
		}

		// Данные
		for i, key := range keys {
			row := i + 2
			f.SetCellValue(metricsSheet, cellAddr("A", row), key)
			for j, item := range data.ComparisonData {
				f.SetCellValue(metricsSheet, cellAddr(string(rune('B'+j)), row), item.Metrics[key])
			}
		}
	}

	f.SetColWidth(sheetName, "A", "D", 18)
}

// cellAddr формирует адрес ячейки
func cellAddr(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}
