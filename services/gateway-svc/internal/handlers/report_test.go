// services/gateway-svc/internal/handlers/report_test.go

package handlers

import (
	"testing"

	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestReportHandler_ConvertOptions_Valid(t *testing.T) {
	h := &ReportHandler{}

	opts := &gatewayv1.ReportOptions{
		Title:                  "Test Report",
		Description:            "Description",
		Author:                 "Author",
		Language:               "ru",
		Timezone:               "Europe/Moscow",
		IncludeGraphDetails:    true,
		IncludeEdgeList:        true,
		IncludePathDetails:     false,
		IncludeRecommendations: true,
		IncludeCharts:          true,
		CompanyName:            "Test Company",
		LogoUrl:                "http://example.com/logo.png",
		Theme:                  "dark",
		Currency:               "RUB",
		Tags:                   []string{"tag1", "tag2"},
		TtlSeconds:             3600,
		SaveToStorage:          true,
	}

	result := h.convertOptions(opts)

	if result.Title != opts.Title {
		t.Errorf("Title = %v, want %v", result.Title, opts.Title)
	}
	if result.Language != "ru" {
		t.Errorf("Language = %v, want ru", result.Language)
	}
	if !result.IncludeCharts {
		t.Error("IncludeCharts should be true")
	}
	if result.Currency != "RUB" {
		t.Errorf("Currency = %v, want RUB", result.Currency)
	}
}

func TestReportHandler_ConvertReportInfo_Valid(t *testing.T) {
	h := &ReportHandler{}

	metadata := &reportv1.ReportMetadata{
		ReportId:         "report-123",
		Title:            "Test Report",
		Type:             reportv1.ReportType_REPORT_TYPE_FLOW,
		Format:           reportv1.ReportFormat_REPORT_FORMAT_PDF,
		SizeBytes:        1024,
		GenerationTimeMs: 150,
		Filename:         "report.pdf",
	}

	result := h.convertReportInfo(metadata)

	if result.ReportId != "report-123" {
		t.Errorf("ReportId = %v, want report-123", result.ReportId)
	}
	if result.SizeBytes != 1024 {
		t.Errorf("SizeBytes = %d, want 1024", result.SizeBytes)
	}
}
