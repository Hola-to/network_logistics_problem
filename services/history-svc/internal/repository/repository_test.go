// services/history-svc/internal/repository/repository_test.go

package repository

import (
	"testing"
	"time"
)

func TestCalculation_Fields(t *testing.T) {
	now := time.Now()
	calc := &Calculation{
		ID:                "calc-123",
		UserID:            "user-456",
		Name:              "Test Calculation",
		Algorithm:         "ALGORITHM_DINIC",
		MaxFlow:           100.5,
		TotalCost:         500.25,
		ComputationTimeMs: 150.5,
		NodeCount:         10,
		EdgeCount:         20,
		RequestData:       []byte(`{"test": "request"}`),
		ResponseData:      []byte(`{"test": "response"}`),
		Tags:              []string{"tag1", "tag2"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if calc.ID != "calc-123" {
		t.Errorf("ID = %v, want calc-123", calc.ID)
	}
	if calc.MaxFlow != 100.5 {
		t.Errorf("MaxFlow = %v, want 100.5", calc.MaxFlow)
	}
	if len(calc.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(calc.Tags))
	}
}

func TestCalculationSummary_Fields(t *testing.T) {
	summary := &CalculationSummary{
		ID:                "calc-123",
		Name:              "Summary Test",
		Algorithm:         "ALGORITHM_EDMONDS_KARP",
		MaxFlow:           200.0,
		TotalCost:         1000.0,
		ComputationTimeMs: 250.0,
		NodeCount:         50,
		EdgeCount:         100,
		Tags:              []string{"production"},
		CreatedAt:         time.Now(),
	}

	if summary.NodeCount != 50 {
		t.Errorf("NodeCount = %d, want 50", summary.NodeCount)
	}
	if summary.EdgeCount != 100 {
		t.Errorf("EdgeCount = %d, want 100", summary.EdgeCount)
	}
}

func TestListFilter_Fields(t *testing.T) {
	minFlow := 10.0
	maxFlow := 100.0
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	filter := &ListFilter{
		Algorithm: "ALGORITHM_DINIC",
		Tags:      []string{"tag1", "tag2"},
		MinFlow:   &minFlow,
		MaxFlow:   &maxFlow,
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	if filter.Algorithm != "ALGORITHM_DINIC" {
		t.Errorf("Algorithm = %v, want ALGORITHM_DINIC", filter.Algorithm)
	}
	if *filter.MinFlow != 10.0 {
		t.Errorf("MinFlow = %v, want 10.0", *filter.MinFlow)
	}
	if len(filter.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(filter.Tags))
	}
}

func TestSortOrder_Values(t *testing.T) {
	tests := []struct {
		order    SortOrder
		expected string
	}{
		{SortByCreatedDesc, "created_desc"},
		{SortByCreatedAsc, "created_asc"},
		{SortByMaxFlowDesc, "max_flow_desc"},
		{SortByTotalCostDesc, "cost_desc"},
	}

	for _, tt := range tests {
		if string(tt.order) != tt.expected {
			t.Errorf("SortOrder = %v, want %v", tt.order, tt.expected)
		}
	}
}

func TestListOptions_Defaults(t *testing.T) {
	opts := &ListOptions{}

	if opts.Limit != 0 {
		t.Errorf("Default Limit = %d, want 0", opts.Limit)
	}
	if opts.Offset != 0 {
		t.Errorf("Default Offset = %d, want 0", opts.Offset)
	}
	if opts.Sort != "" {
		t.Errorf("Default Sort = %v, want empty", opts.Sort)
	}
}

func TestUserStatistics_Fields(t *testing.T) {
	stats := &UserStatistics{
		TotalCalculations:        100,
		AverageMaxFlow:           150.5,
		AverageTotalCost:         750.25,
		AverageComputationTimeMs: 200.0,
		CalculationsByAlgorithm:  map[string]int{"DINIC": 60, "EDMONDS_KARP": 40},
		DailyStats: []DailyStats{
			{Date: "2024-01-15", Count: 10, TotalFlow: 1500.0},
			{Date: "2024-01-14", Count: 8, TotalFlow: 1200.0},
		},
	}

	if stats.TotalCalculations != 100 {
		t.Errorf("TotalCalculations = %d, want 100", stats.TotalCalculations)
	}
	if stats.CalculationsByAlgorithm["DINIC"] != 60 {
		t.Errorf("DINIC count = %d, want 60", stats.CalculationsByAlgorithm["DINIC"])
	}
	if len(stats.DailyStats) != 2 {
		t.Errorf("DailyStats length = %d, want 2", len(stats.DailyStats))
	}
}

func TestDailyStats_Fields(t *testing.T) {
	ds := DailyStats{
		Date:      "2024-01-15",
		Count:     25,
		TotalFlow: 5000.0,
	}

	if ds.Date != "2024-01-15" {
		t.Errorf("Date = %v, want 2024-01-15", ds.Date)
	}
	if ds.Count != 25 {
		t.Errorf("Count = %d, want 25", ds.Count)
	}
	if ds.TotalFlow != 5000.0 {
		t.Errorf("TotalFlow = %v, want 5000.0", ds.TotalFlow)
	}
}

func TestErrors(t *testing.T) {
	if ErrCalculationNotFound.Error() != "calculation not found" {
		t.Errorf("ErrCalculationNotFound = %v, want 'calculation not found'", ErrCalculationNotFound)
	}
	if ErrAccessDenied.Error() != "access denied" {
		t.Errorf("ErrAccessDenied = %v, want 'access denied'", ErrAccessDenied)
	}
}
