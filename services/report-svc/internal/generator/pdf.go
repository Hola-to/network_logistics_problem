// services/report-svc/internal/generator/pdf.go
package generator

import (
	"context"
	"fmt"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// PDFGenerator генератор PDF отчётов
type PDFGenerator struct {
	BaseGenerator
}

// NewPDFGenerator создаёт новый генератор
func NewPDFGenerator() *PDFGenerator {
	return &PDFGenerator{}
}

// Format возвращает формат генератора
func (g *PDFGenerator) Format() reportv1.ReportFormat {
	return reportv1.ReportFormat_REPORT_FORMAT_PDF
}

// Стили
var (
	// Цвета
	primaryColor   = &props.Color{Red: 52, Green: 152, Blue: 219}  // #3498db
	headerBgColor  = &props.Color{Red: 44, Green: 62, Blue: 80}    // #2c3e50
	successColor   = &props.Color{Red: 39, Green: 174, Blue: 96}   // #27ae60
	warningColor   = &props.Color{Red: 243, Green: 156, Blue: 18}  // #f39c12
	dangerColor    = &props.Color{Red: 231, Green: 76, Blue: 60}   // #e74c3c
	lightGrayColor = &props.Color{Red: 236, Green: 240, Blue: 241} // #ecf0f1
	darkGrayColor  = &props.Color{Red: 127, Green: 140, Blue: 141} // #7f8c8d

	// Стили текста
	titleStyle = props.Text{
		Size:  24,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: headerBgColor,
	}

	h2Style = props.Text{
		Size:  16,
		Style: fontstyle.Bold,
		Color: headerBgColor,
		Top:   5,
	}

	h3Style = props.Text{
		Size:  12,
		Style: fontstyle.Bold,
		Color: darkGrayColor,
		Top:   3,
	}

	normalStyle = props.Text{
		Size: 10,
	}

	boldStyle = props.Text{
		Size:  10,
		Style: fontstyle.Bold,
	}

	smallStyle = props.Text{
		Size:  8,
		Color: darkGrayColor,
	}

	metricValueStyle = props.Text{
		Size:  20,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: primaryColor,
	}

	metricLabelStyle = props.Text{
		Size:  9,
		Align: align.Center,
		Color: darkGrayColor,
	}

	tableHeaderStyle = &props.Cell{
		BackgroundColor: primaryColor,
	}

	tableHeaderTextStyle = props.Text{
		Size:  9,
		Style: fontstyle.Bold,
		Color: &props.Color{Red: 255, Green: 255, Blue: 255},
		Align: align.Center,
	}

	tableCellStyle = &props.Cell{
		BorderType:  border.Bottom,
		BorderColor: lightGrayColor,
	}

	tableCellTextStyle = props.Text{
		Size:  9,
		Align: align.Center,
	}
)

// Generate генерирует PDF отчёт
func (g *PDFGenerator) Generate(ctx context.Context, data *ReportData) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()

	m := maroto.New(cfg)

	// Заголовок документа
	g.addHeader(m, data)

	// Содержимое в зависимости от типа
	switch data.Type {
	case reportv1.ReportType_REPORT_TYPE_FLOW:
		g.addFlowContent(m, data)
	case reportv1.ReportType_REPORT_TYPE_ANALYTICS:
		g.addAnalyticsContent(m, data)
	case reportv1.ReportType_REPORT_TYPE_SIMULATION:
		g.addSimulationContent(m, data)
	case reportv1.ReportType_REPORT_TYPE_SUMMARY:
		g.addSummaryContent(m, data)
	case reportv1.ReportType_REPORT_TYPE_COMPARISON:
		g.addComparisonContent(m, data)
	case reportv1.ReportType_REPORT_TYPE_HISTORY:
		g.addHistoryContent(m, data)
	default:
		g.addFlowContent(m, data)
	}

	// Футер
	g.addFooter(m)

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

func (g *PDFGenerator) addHeader(m core.Maroto, data *ReportData) {
	m.AddRow(15,
		text.NewCol(12, g.GetTitle(data), titleStyle),
	)

	m.AddRow(5,
		line.NewCol(12),
	)

	// Метаданные
	m.AddRow(6,
		text.NewCol(6, fmt.Sprintf("Author: %s", g.GetAuthor(data)), smallStyle),
		text.NewCol(6, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")),
			props.Text{Size: 8, Color: darkGrayColor, Align: align.Right}),
	)

	if desc := g.GetDescription(data); desc != "" {
		m.AddRow(5,
			text.NewCol(12, desc, smallStyle),
		)
	}

	m.AddRow(8) // Отступ
}

func (g *PDFGenerator) addFlowContent(m core.Maroto, data *ReportData) {
	// Информация о сети
	if data.Graph != nil {
		g.addSection(m, "Network Information")
		g.addMetricCards(m, []metricCard{
			{Label: "Nodes", Value: fmt.Sprintf("%d", len(data.Graph.Nodes))},
			{Label: "Edges", Value: fmt.Sprintf("%d", len(data.Graph.Edges))},
			{Label: "Source", Value: fmt.Sprintf("%d", data.Graph.SourceId)},
			{Label: "Sink", Value: fmt.Sprintf("%d", data.Graph.SinkId)},
		})
	}

	// Результаты оптимизации
	if data.FlowResult != nil {
		g.addSection(m, "Optimization Results")

		// Главные метрики
		g.addMetricCards(m, []metricCard{
			{Label: "Maximum Flow", Value: g.FormatFloat(data.FlowResult.MaxFlow, 4), Highlight: true},
			{Label: "Total Cost", Value: g.FormatFloat(data.FlowResult.TotalCost, 2), Highlight: true},
		})

		// Дополнительные метрики
		m.AddRow(5)
		g.addMetricCards(m, []metricCard{
			{Label: "Status", Value: data.FlowResult.Status.String()},
			{Label: "Iterations", Value: fmt.Sprintf("%d", data.FlowResult.Iterations)},
			{Label: "Computation Time", Value: fmt.Sprintf("%.2f ms", data.FlowResult.ComputationTimeMs)},
		})

		// Таблица потоков по рёбрам
		if len(data.FlowEdges) > 0 && g.ShouldIncludeRawData(data) {
			g.addSection(m, "Edge Flows")
			g.addEdgeFlowsTable(m, data.FlowEdges)
		}
	}

	// Статистика графа из FlowData
	if data.FlowData != nil && data.FlowData.GraphStats != nil {
		g.addSection(m, "Graph Statistics")
		stats := data.FlowData.GraphStats
		g.addKeyValueTable(m, []keyValue{
			{"Total Capacity", g.FormatFloat(stats.TotalCapacity, 2)},
			{"Average Edge Length", g.FormatFloat(stats.AverageEdgeLength, 2)},
			{"Warehouses", fmt.Sprintf("%d", stats.WarehouseCount)},
			{"Delivery Points", fmt.Sprintf("%d", stats.DeliveryPointCount)},
			{"Connected", fmt.Sprintf("%v", stats.IsConnected)},
			{"Density", g.FormatFloat(stats.Density, 4)},
		})
	}
}

func (g *PDFGenerator) addAnalyticsContent(m core.Maroto, data *ReportData) {
	if data.AnalyticsData == nil {
		g.addSection(m, "No Analytics Data")
		return
	}

	ad := data.AnalyticsData

	// Стоимость
	g.addSection(m, "Cost Analysis")
	g.addMetricCards(m, []metricCard{
		{Label: "Total Cost", Value: fmt.Sprintf("%s %s", g.FormatFloat(ad.TotalCost, 2), ad.Currency), Highlight: true},
	})

	// Разбивка затрат
	if ad.CostBreakdown != nil {
		m.AddRow(5)
		g.addKeyValueTable(m, []keyValue{
			{"Transport Cost", g.FormatFloat(ad.CostBreakdown.TransportCost, 2)},
			{"Fixed Cost", g.FormatFloat(ad.CostBreakdown.FixedCost, 2)},
			{"Handling Cost", g.FormatFloat(ad.CostBreakdown.HandlingCost, 2)},
		})
	}

	// Узкие места
	if len(ad.Bottlenecks) > 0 {
		g.addSection(m, "Bottlenecks")
		g.addBottlenecksTable(m, ad.Bottlenecks)
	}

	// Рекомендации
	if len(ad.Recommendations) > 0 && g.ShouldIncludeRecommendations(data) {
		g.addSection(m, "Recommendations")
		for i, rec := range ad.Recommendations {
			g.addRecommendation(m, i+1, rec)
		}
	}

	// Эффективность
	if ad.Efficiency != nil {
		g.addSection(m, "Efficiency Metrics")
		g.addMetricCards(m, []metricCard{
			{Label: "Overall Efficiency", Value: g.FormatPercent(ad.Efficiency.OverallEfficiency)},
			{Label: "Capacity Utilization", Value: g.FormatPercent(ad.Efficiency.CapacityUtilization)},
			{Label: "Grade", Value: ad.Efficiency.Grade, Highlight: true},
		})

		m.AddRow(5)
		g.addKeyValueTable(m, []keyValue{
			{"Unused Edges", fmt.Sprintf("%d", ad.Efficiency.UnusedEdges)},
			{"Saturated Edges", fmt.Sprintf("%d", ad.Efficiency.SaturatedEdges)},
		})
	}
}

func (g *PDFGenerator) addSimulationContent(m core.Maroto, data *ReportData) {
	if data.SimulationData == nil {
		g.addSection(m, "No Simulation Data")
		return
	}

	sd := data.SimulationData

	g.addSection(m, fmt.Sprintf("Simulation: %s", sd.SimulationType))

	// Базовые показатели
	g.addMetricCards(m, []metricCard{
		{Label: "Baseline Flow", Value: g.FormatFloat(sd.BaselineFlow, 4)},
		{Label: "Baseline Cost", Value: g.FormatFloat(sd.BaselineCost, 2)},
	})

	// Сценарии
	if len(sd.Scenarios) > 0 {
		m.AddRow(8)
		g.addSubSection(m, "Scenarios")
		g.addScenariosTable(m, sd.Scenarios)
	}

	// Monte Carlo
	if sd.MonteCarlo != nil {
		m.AddRow(8)
		g.addSubSection(m, "Monte Carlo Results")
		mc := sd.MonteCarlo

		g.addMetricCards(m, []metricCard{
			{Label: "Mean Flow", Value: g.FormatFloat(mc.MeanFlow, 4)},
			{Label: "Std Dev", Value: g.FormatFloat(mc.StdDev, 4)},
			{Label: "Iterations", Value: fmt.Sprintf("%d", mc.Iterations)},
		})

		m.AddRow(5)
		g.addKeyValueTable(m, []keyValue{
			{"Min Flow", g.FormatFloat(mc.MinFlow, 4)},
			{"Max Flow", g.FormatFloat(mc.MaxFlow, 4)},
			{"P5", g.FormatFloat(mc.P5, 4)},
			{"P50 (Median)", g.FormatFloat(mc.P50, 4)},
			{"P95", g.FormatFloat(mc.P95, 4)},
			{fmt.Sprintf("CI %.0f%%", mc.ConfidenceLevel*100),
				fmt.Sprintf("%.4f - %.4f", mc.CiLow, mc.CiHigh)},
		})
	}

	// Анализ чувствительности
	if len(sd.Sensitivity) > 0 {
		m.AddRow(8)
		g.addSubSection(m, "Sensitivity Analysis")
		g.addSensitivityTable(m, sd.Sensitivity)
	}

	// Устойчивость
	if sd.Resilience != nil {
		m.AddRow(8)
		g.addSubSection(m, "Resilience Analysis")
		r := sd.Resilience

		g.addMetricCards(m, []metricCard{
			{Label: "Overall Score", Value: g.FormatFloat(r.OverallScore, 2), Highlight: true},
			{Label: "N-1 Feasible", Value: fmt.Sprintf("%v", r.NMinusOneFeasible)},
		})

		m.AddRow(5)
		g.addKeyValueTable(m, []keyValue{
			{"Single Points of Failure", fmt.Sprintf("%d", r.SinglePointsOfFailure)},
			{"Worst Case Flow Reduction", g.FormatPercent(r.WorstCaseFlowReduction)},
		})
	}
}

func (g *PDFGenerator) addSummaryContent(m core.Maroto, data *ReportData) {
	if data.FlowResult != nil {
		g.addFlowContent(m, data)
	}

	if data.AnalyticsData != nil {
		m.AddRow(10)
		g.addAnalyticsContent(m, data)
	}

	if data.SimulationData != nil {
		m.AddRow(10)
		g.addSimulationContent(m, data)
	}
}

func (g *PDFGenerator) addComparisonContent(m core.Maroto, data *ReportData) {
	if len(data.ComparisonData) == 0 {
		g.addSection(m, "No Comparison Data")
		return
	}

	g.addSection(m, "Scenario Comparison")

	// Основная таблица сравнения
	g.addComparisonTable(m, data.ComparisonData)

	// Нахождение лучшего сценария
	m.AddRow(10)
	best := g.findBestScenario(data.ComparisonData)
	if best != nil {
		m.AddRow(8,
			text.NewCol(12, fmt.Sprintf("Best scenario by flow: %s (%.4f)", best.Name, best.MaxFlow), boldStyle),
		)
	}

	// Детальные метрики (если есть)
	if len(data.ComparisonData) > 0 && len(data.ComparisonData[0].Metrics) > 0 {
		m.AddRow(10)
		g.addSubSection(m, "Detailed Metrics")
		g.addDetailedMetricsTable(m, data.ComparisonData)
	}
}

func (g *PDFGenerator) addHistoryContent(m core.Maroto, data *ReportData) {
	g.addSection(m, "Calculation History")
	m.AddRow(8,
		text.NewCol(12, "History report content", normalStyle),
	)
}

// === Вспомогательные методы ===

type metricCard struct {
	Label     string
	Value     string
	Highlight bool
}

func (g *PDFGenerator) addMetricCards(m core.Maroto, cards []metricCard) {
	if len(cards) == 0 {
		return
	}

	colSize := 12 / len(cards)
	if colSize < 2 {
		colSize = 2
	}

	var cols []core.Col
	for _, card := range cards {
		valueStyle := metricValueStyle
		if !card.Highlight {
			valueStyle.Size = 14
		}

		cols = append(cols,
			col.New(colSize).Add(
				text.New(card.Value, valueStyle),
				text.New(card.Label, metricLabelStyle),
			),
		)
	}

	m.AddRow(20, cols...)
}

type keyValue struct {
	Key   string
	Value string
}

func (g *PDFGenerator) addKeyValueTable(m core.Maroto, items []keyValue) {
	for _, item := range items {
		m.AddRow(6,
			text.NewCol(6, item.Key, boldStyle),
			text.NewCol(6, item.Value, normalStyle),
		)
	}
}

func (g *PDFGenerator) addSection(m core.Maroto, title string) {
	m.AddRow(10,
		text.NewCol(12, title, h2Style),
	)
	m.AddRow(2,
		line.NewCol(12, props.Line{Color: primaryColor}),
	)
	m.AddRow(5)
}

func (g *PDFGenerator) addSubSection(m core.Maroto, title string) {
	m.AddRow(8,
		text.NewCol(12, title, h3Style),
	)
}

func (g *PDFGenerator) addEdgeFlowsTable(m core.Maroto, edges []*EdgeFlowData) {
	// Заголовок
	m.AddRow(8,
		text.NewCol(2, "From", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "To", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Flow", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Capacity", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Cost", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Utilization", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
	)

	// Данные (ограничиваем количество для PDF)
	maxRows := 30
	count := 0
	for _, edge := range edges {
		if edge.Flow <= 0.001 {
			continue
		}
		if count >= maxRows {
			m.AddRow(6,
				text.NewCol(12, fmt.Sprintf("... and %d more rows", len(edges)-maxRows), smallStyle),
			)
			break
		}

		m.AddRow(6,
			text.NewCol(2, fmt.Sprintf("%d", edge.From), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, fmt.Sprintf("%d", edge.To), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatFloat(edge.Flow, 4), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatFloat(edge.Capacity, 4), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatFloat(edge.Cost, 4), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatPercent(edge.Utilization), tableCellTextStyle).WithStyle(tableCellStyle),
		)
		count++
	}
}

func (g *PDFGenerator) addBottlenecksTable(m core.Maroto, bottlenecks []*BottleneckData) {
	// Заголовок
	m.AddRow(8,
		text.NewCol(2, "From", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "To", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Utilization", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Impact", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Severity", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
	)

	for _, bn := range bottlenecks {
		severityStyle := tableCellTextStyle
		switch bn.Severity {
		case "BOTTLENECK_SEVERITY_HIGH", "high", "HIGH":
			severityStyle.Color = dangerColor
		case "BOTTLENECK_SEVERITY_MEDIUM", "medium", "MEDIUM":
			severityStyle.Color = warningColor
		case "BOTTLENECK_SEVERITY_LOW", "low", "LOW":
			severityStyle.Color = successColor
		}

		m.AddRow(6,
			text.NewCol(2, fmt.Sprintf("%d", bn.From), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, fmt.Sprintf("%d", bn.To), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, g.FormatPercent(bn.Utilization), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, g.FormatFloat(bn.ImpactScore, 2), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, bn.Severity, severityStyle).WithStyle(tableCellStyle),
		)
	}
}

func (g *PDFGenerator) addRecommendation(m core.Maroto, num int, rec *RecommendationData) {
	m.AddRow(8,
		text.NewCol(12, fmt.Sprintf("%d. %s", num, rec.Type), boldStyle),
	)

	m.AddRow(6,
		text.NewCol(12, rec.Description, normalStyle),
	)

	if rec.EstimatedImprovement > 0 || rec.EstimatedCost > 0 {
		details := ""
		if rec.EstimatedImprovement > 0 {
			details += fmt.Sprintf("Expected improvement: %s", g.FormatPercent(rec.EstimatedImprovement))
		}
		if rec.EstimatedCost > 0 {
			if details != "" {
				details += " | "
			}
			details += fmt.Sprintf("Estimated cost: %.2f", rec.EstimatedCost)
		}
		m.AddRow(5,
			text.NewCol(12, details, smallStyle),
		)
	}

	m.AddRow(3)
}

func (g *PDFGenerator) addScenariosTable(m core.Maroto, scenarios []*ScenarioData) {
	// Заголовок
	m.AddRow(8,
		text.NewCol(3, "Name", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Max Flow", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Cost", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Change %", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Impact", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
	)

	for _, sc := range scenarios {
		impactStyle := tableCellTextStyle
		switch sc.ImpactLevel {
		case "IMPACT_LEVEL_HIGH", "high", "HIGH":
			impactStyle.Color = dangerColor
		case "IMPACT_LEVEL_MEDIUM", "medium", "MEDIUM":
			impactStyle.Color = warningColor
		case "IMPACT_LEVEL_LOW", "low", "LOW":
			impactStyle.Color = successColor
		}

		m.AddRow(6,
			text.NewCol(3, sc.Name, tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatFloat(sc.MaxFlow, 4), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatFloat(sc.TotalCost, 2), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, fmt.Sprintf("%.1f%%", sc.FlowChangePercent), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, sc.ImpactLevel, impactStyle).WithStyle(tableCellStyle),
		)
	}
}

func (g *PDFGenerator) addSensitivityTable(m core.Maroto, params []*SensitivityData) {
	// Заголовок
	m.AddRow(8,
		text.NewCol(4, "Parameter", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Elasticity", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Index", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Level", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
	)

	for _, p := range params {
		m.AddRow(6,
			text.NewCol(4, p.ParameterId, tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, g.FormatFloat(p.Elasticity, 3), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, g.FormatFloat(p.SensitivityIndex, 3), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, p.Level, tableCellTextStyle).WithStyle(tableCellStyle),
		)
	}
}

func (g *PDFGenerator) addComparisonTable(m core.Maroto, items []*ComparisonItemData) {
	// Заголовок
	m.AddRow(8,
		text.NewCol(4, "Scenario", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Max Flow", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(3, "Total Cost", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		text.NewCol(2, "Efficiency", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
	)

	for _, item := range items {
		m.AddRow(6,
			text.NewCol(4, item.Name, tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, g.FormatFloat(item.MaxFlow, 4), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(3, g.FormatFloat(item.TotalCost, 2), tableCellTextStyle).WithStyle(tableCellStyle),
			text.NewCol(2, g.FormatPercent(item.Efficiency), tableCellTextStyle).WithStyle(tableCellStyle),
		)
	}
}

func (g *PDFGenerator) addDetailedMetricsTable(m core.Maroto, items []*ComparisonItemData) {
	if len(items) == 0 || len(items[0].Metrics) == 0 {
		return
	}

	// Собираем ключи метрик
	var keys []string
	for k := range items[0].Metrics {
		keys = append(keys, k)
	}

	// Ограничиваем количество колонок
	maxCols := 5
	scenarioCount := len(items)
	if scenarioCount > maxCols {
		scenarioCount = maxCols
	}

	// Вычисляем размер колонок
	metricColSize := 4
	valueColSize := (12 - metricColSize) / scenarioCount

	// Заголовок
	headerCols := []core.Col{
		text.NewCol(metricColSize, "Metric", tableHeaderTextStyle).WithStyle(tableHeaderStyle),
	}
	for i := 0; i < scenarioCount; i++ {
		headerCols = append(headerCols,
			text.NewCol(valueColSize, items[i].Name, tableHeaderTextStyle).WithStyle(tableHeaderStyle),
		)
	}
	m.AddRow(8, headerCols...)

	// Данные
	for _, key := range keys {
		dataCols := []core.Col{
			text.NewCol(metricColSize, key, tableCellTextStyle).WithStyle(tableCellStyle),
		}
		for i := 0; i < scenarioCount; i++ {
			val := items[i].Metrics[key]
			dataCols = append(dataCols,
				text.NewCol(valueColSize, g.FormatFloat(val, 2), tableCellTextStyle).WithStyle(tableCellStyle),
			)
		}
		m.AddRow(6, dataCols...)
	}
}

func (g *PDFGenerator) findBestScenario(items []*ComparisonItemData) *ComparisonItemData {
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

func (g *PDFGenerator) addFooter(m core.Maroto) {
	m.AddRow(10)
	m.AddRow(2,
		line.NewCol(12, props.Line{Color: lightGrayColor}),
	)
	m.AddRow(6,
		text.NewCol(12,
			fmt.Sprintf("Generated by Logistics Platform | %s", time.Now().Format("2006-01-02 15:04:05")),
			props.Text{Size: 8, Color: darkGrayColor, Align: align.Center},
		),
	)
}
