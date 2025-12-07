// services/report-svc/internal/generator/generator.go
package generator

import (
	"context"
	"fmt"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

// ReportData данные для генерации отчёта
type ReportData struct {
	Type    reportv1.ReportType
	Options *reportv1.ReportOptions

	// Граф
	Graph      *commonv1.Graph
	FlowResult *commonv1.FlowResult

	// Данные по типам отчётов (внутренние типы)
	FlowData       *FlowReportData
	AnalyticsData  *AnalyticsReportData
	SimulationData *SimulationReportData
	ComparisonData []*ComparisonItemData

	// Дополнительные данные для конвертации в генераторах
	FlowEdges []*EdgeFlowData
}

// Generator интерфейс генератора отчётов
type Generator interface {
	Generate(ctx context.Context, data *ReportData) ([]byte, error)
	Format() reportv1.ReportFormat
}

// BaseGenerator базовые утилиты для генераторов
type BaseGenerator struct{}

// GetTitle возвращает заголовок отчёта
func (b *BaseGenerator) GetTitle(data *ReportData) string {
	if data.Options != nil && data.Options.Title != "" {
		return data.Options.Title
	}
	switch data.Type {
	case reportv1.ReportType_REPORT_TYPE_FLOW:
		return "Flow Optimization Report"
	case reportv1.ReportType_REPORT_TYPE_ANALYTICS:
		return "Analytics Report"
	case reportv1.ReportType_REPORT_TYPE_SIMULATION:
		return "Simulation Report"
	case reportv1.ReportType_REPORT_TYPE_SUMMARY:
		return "Summary Report"
	case reportv1.ReportType_REPORT_TYPE_COMPARISON:
		return "Comparison Report"
	case reportv1.ReportType_REPORT_TYPE_HISTORY:
		return "History Report"
	default:
		return "Logistics Report"
	}
}

// GetAuthor возвращает автора отчёта
func (b *BaseGenerator) GetAuthor(data *ReportData) string {
	if data.Options != nil && data.Options.Author != "" {
		return data.Options.Author
	}
	return "Logistics System"
}

// GetDescription возвращает описание
func (b *BaseGenerator) GetDescription(data *ReportData) string {
	if data.Options != nil && data.Options.Description != "" {
		return data.Options.Description
	}
	return ""
}

// GetLanguage возвращает язык
func (b *BaseGenerator) GetLanguage(data *ReportData) string {
	if data.Options != nil && data.Options.Language != "" {
		return data.Options.Language
	}
	return "en"
}

// ShouldIncludeRawData проверяет нужно ли включать сырые данные
func (b *BaseGenerator) ShouldIncludeRawData(data *ReportData) bool {
	if data.Options == nil {
		return true
	}
	return data.Options.IncludeRawData
}

// ShouldIncludeRecommendations проверяет нужно ли включать рекомендации
func (b *BaseGenerator) ShouldIncludeRecommendations(data *ReportData) bool {
	if data.Options == nil {
		return true
	}
	return data.Options.IncludeRecommendations
}

// FormatFloat форматирует число с заданной точностью
func (b *BaseGenerator) FormatFloat(v float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, v)
}

// FormatPercent форматирует процент
func (b *BaseGenerator) FormatPercent(v float64) string {
	return fmt.Sprintf("%.2f%%", v*100)
}

// FormatDuration форматирует длительность
func (b *BaseGenerator) FormatDuration(ms float64) string {
	if ms < 1000 {
		return fmt.Sprintf("%.2f ms", ms)
	}
	return fmt.Sprintf("%.2f s", ms/1000)
}

// FormatTimestamp форматирует время
func (b *BaseGenerator) FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// ColName преобразует индекс колонки в буквенное обозначение (0 -> A, 25 -> Z, 26 -> AA)
func ColName(index int) string {
	result := ""
	for {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	return result
}

// Cell возвращает адрес ячейки
func Cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}

// CellByIndex возвращает адрес ячейки по индексам
func CellByIndex(colIndex, rowIndex int) string {
	return fmt.Sprintf("%s%d", ColName(colIndex), rowIndex)
}
