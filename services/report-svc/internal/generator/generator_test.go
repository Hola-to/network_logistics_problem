// services/report-svc/internal/generator/generator_test.go

package generator

import (
	"testing"
	"time"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

func TestBaseGenerator_GetTitle(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		name     string
		data     *ReportData
		expected string
	}{
		{
			name: "custom title",
			data: &ReportData{
				Options: &reportv1.ReportOptions{Title: "Custom Title"},
			},
			expected: "Custom Title",
		},
		{
			name: "flow type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_FLOW,
			},
			expected: "Flow Optimization Report",
		},
		{
			name: "analytics type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
			},
			expected: "Analytics Report",
		},
		{
			name: "simulation type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_SIMULATION,
			},
			expected: "Simulation Report",
		},
		{
			name: "summary type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_SUMMARY,
			},
			expected: "Summary Report",
		},
		{
			name: "comparison type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_COMPARISON,
			},
			expected: "Comparison Report",
		},
		{
			name: "history type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_HISTORY,
			},
			expected: "History Report",
		},
		{
			name: "unspecified type",
			data: &ReportData{
				Type: reportv1.ReportType_REPORT_TYPE_UNSPECIFIED,
			},
			expected: "Logistics Report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bg.GetTitle(tt.data)
			if result != tt.expected {
				t.Errorf("GetTitle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseGenerator_GetAuthor(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		name     string
		data     *ReportData
		expected string
	}{
		{
			name: "custom author",
			data: &ReportData{
				Options: &reportv1.ReportOptions{Author: "John Doe"},
			},
			expected: "John Doe",
		},
		{
			name:     "default author",
			data:     &ReportData{},
			expected: "Logistics System",
		},
		{
			name:     "nil options",
			data:     &ReportData{Options: nil},
			expected: "Logistics System",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bg.GetAuthor(tt.data)
			if result != tt.expected {
				t.Errorf("GetAuthor() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseGenerator_GetDescription(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		name     string
		data     *ReportData
		expected string
	}{
		{
			name: "with description",
			data: &ReportData{
				Options: &reportv1.ReportOptions{Description: "Test description"},
			},
			expected: "Test description",
		},
		{
			name:     "empty description",
			data:     &ReportData{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bg.GetDescription(tt.data)
			if result != tt.expected {
				t.Errorf("GetDescription() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseGenerator_GetLanguage(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		name     string
		data     *ReportData
		expected string
	}{
		{
			name: "russian",
			data: &ReportData{
				Options: &reportv1.ReportOptions{Language: "ru"},
			},
			expected: "ru",
		},
		{
			name:     "default english",
			data:     &ReportData{},
			expected: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bg.GetLanguage(tt.data)
			if result != tt.expected {
				t.Errorf("GetLanguage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseGenerator_ShouldIncludeRawData(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		name     string
		data     *ReportData
		expected bool
	}{
		{
			name:     "nil options - include by default",
			data:     &ReportData{},
			expected: true,
		},
		{
			name: "explicitly include",
			data: &ReportData{
				Options: &reportv1.ReportOptions{IncludeRawData: true},
			},
			expected: true,
		},
		{
			name: "explicitly exclude",
			data: &ReportData{
				Options: &reportv1.ReportOptions{IncludeRawData: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bg.ShouldIncludeRawData(tt.data)
			if result != tt.expected {
				t.Errorf("ShouldIncludeRawData() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseGenerator_ShouldIncludeRecommendations(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		name     string
		data     *ReportData
		expected bool
	}{
		{
			name:     "nil options - include by default",
			data:     &ReportData{},
			expected: true,
		},
		{
			name: "explicitly include",
			data: &ReportData{
				Options: &reportv1.ReportOptions{IncludeRecommendations: true},
			},
			expected: true,
		},
		{
			name: "explicitly exclude",
			data: &ReportData{
				Options: &reportv1.ReportOptions{IncludeRecommendations: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bg.ShouldIncludeRecommendations(tt.data)
			if result != tt.expected {
				t.Errorf("ShouldIncludeRecommendations() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseGenerator_FormatFloat(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		value     float64
		precision int
		expected  string
	}{
		{123.456789, 2, "123.46"},
		{123.456789, 4, "123.4568"},
		{100.0, 0, "100"},
		{0.123, 3, "0.123"},
		{-50.5, 1, "-50.5"},
	}

	for _, tt := range tests {
		result := bg.FormatFloat(tt.value, tt.precision)
		if result != tt.expected {
			t.Errorf("FormatFloat(%v, %d) = %v, want %v", tt.value, tt.precision, result, tt.expected)
		}
	}
}

func TestBaseGenerator_FormatPercent(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		value    float64
		expected string
	}{
		{0.5, "50.00%"},
		{1.0, "100.00%"},
		{0.123, "12.30%"},
		{0.0, "0.00%"},
	}

	for _, tt := range tests {
		result := bg.FormatPercent(tt.value)
		if result != tt.expected {
			t.Errorf("FormatPercent(%v) = %v, want %v", tt.value, result, tt.expected)
		}
	}
}

func TestBaseGenerator_FormatDuration(t *testing.T) {
	bg := &BaseGenerator{}

	tests := []struct {
		ms       float64
		expected string
	}{
		{100.5, "100.50 ms"},
		{999.0, "999.00 ms"},
		{1000.0, "1.00 s"},
		{2500.0, "2.50 s"},
	}

	for _, tt := range tests {
		result := bg.FormatDuration(tt.ms)
		if result != tt.expected {
			t.Errorf("FormatDuration(%v) = %v, want %v", tt.ms, result, tt.expected)
		}
	}
}

func TestBaseGenerator_FormatTimestamp(t *testing.T) {
	bg := &BaseGenerator{}

	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	expected := "2024-01-15 14:30:45"

	result := bg.FormatTimestamp(tm)
	if result != expected {
		t.Errorf("FormatTimestamp() = %v, want %v", result, expected)
	}
}

func TestColName(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
	}

	for _, tt := range tests {
		result := ColName(tt.index)
		if result != tt.expected {
			t.Errorf("ColName(%d) = %v, want %v", tt.index, result, tt.expected)
		}
	}
}

func TestCell(t *testing.T) {
	tests := []struct {
		col      string
		row      int
		expected string
	}{
		{"A", 1, "A1"},
		{"B", 10, "B10"},
		{"AA", 100, "AA100"},
	}

	for _, tt := range tests {
		result := Cell(tt.col, tt.row)
		if result != tt.expected {
			t.Errorf("Cell(%q, %d) = %v, want %v", tt.col, tt.row, result, tt.expected)
		}
	}
}

func TestCellByIndex(t *testing.T) {
	tests := []struct {
		colIndex int
		rowIndex int
		expected string
	}{
		{0, 1, "A1"},
		{1, 5, "B5"},
		{26, 10, "AA10"},
	}

	for _, tt := range tests {
		result := CellByIndex(tt.colIndex, tt.rowIndex)
		if result != tt.expected {
			t.Errorf("CellByIndex(%d, %d) = %v, want %v", tt.colIndex, tt.rowIndex, result, tt.expected)
		}
	}
}
