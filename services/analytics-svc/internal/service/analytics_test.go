package service

import (
	"context"
	"testing"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestNewAnalyticsService(t *testing.T) {
	svc := NewAnalyticsService()
	if svc == nil {
		t.Error("NewAnalyticsService() returned nil")
	}
}

func TestAnalyticsService_CalculateCost(t *testing.T) {
	svc := NewAnalyticsService()
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *analyticsv1.CalculateCostRequest
		wantErr     bool
		checkResult func(t *testing.T, resp *analyticsv1.CalculateCostResponse)
	}{
		{
			name: "valid request",
			request: &analyticsv1.CalculateCostRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
					Edges: []*commonv1.Edge{
						{From: 1, To: 2, CurrentFlow: 10, Cost: 5, Capacity: 100},
					},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *analyticsv1.CalculateCostResponse) {
				if resp.TotalCost != 50 {
					t.Errorf("TotalCost = %v, want 50", resp.TotalCost)
				}
			},
		},
		{
			name: "nil graph",
			request: &analyticsv1.CalculateCostRequest{
				Graph: nil,
			},
			wantErr: true,
		},
		{
			name: "empty graph",
			request: &analyticsv1.CalculateCostRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{},
				},
			},
			wantErr: true,
		},
		{
			name: "with options",
			request: &analyticsv1.CalculateCostRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
					Edges: []*commonv1.Edge{
						{From: 1, To: 2, CurrentFlow: 10, Cost: 10, Capacity: 100},
					},
				},
				Options: &analyticsv1.CostOptions{
					Currency:        "USD",
					DiscountPercent: 10,
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *analyticsv1.CalculateCostResponse) {
				if resp.Currency != "USD" {
					t.Errorf("Currency = %v, want USD", resp.Currency)
				}
				// 100 - 10% = 90
				if resp.TotalCost != 90 {
					t.Errorf("TotalCost = %v, want 90", resp.TotalCost)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CalculateCost(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateCost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, resp)
			}
		})
	}
}

func TestAnalyticsService_FindBottlenecks(t *testing.T) {
	svc := NewAnalyticsService()
	ctx := context.Background()

	tests := []struct {
		name          string
		request       *analyticsv1.FindBottlenecksRequest
		wantErr       bool
		expectedCount int
	}{
		{
			name: "finds bottlenecks",
			request: &analyticsv1.FindBottlenecksRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
					Edges: []*commonv1.Edge{
						{From: 1, To: 2, CurrentFlow: 95, Capacity: 100, Cost: 1},
					},
				},
				UtilizationThreshold: 0.9,
				TopN:                 10,
			},
			wantErr:       false,
			expectedCount: 1,
		},
		{
			name: "default threshold",
			request: &analyticsv1.FindBottlenecksRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
					Edges: []*commonv1.Edge{
						{From: 1, To: 2, CurrentFlow: 95, Capacity: 100, Cost: 1},
					},
				},
				// threshold = 0 -> uses default 0.9
			},
			wantErr:       false,
			expectedCount: 1,
		},
		{
			name: "nil graph",
			request: &analyticsv1.FindBottlenecksRequest{
				Graph: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.FindBottlenecks(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindBottlenecks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(resp.Bottlenecks) != tt.expectedCount {
				t.Errorf("Bottlenecks count = %d, want %d",
					len(resp.Bottlenecks), tt.expectedCount)
			}
		})
	}
}

func TestAnalyticsService_AnalyzeFlow(t *testing.T) {
	svc := NewAnalyticsService()
	ctx := context.Background()

	tests := []struct {
		name    string
		request *analyticsv1.AnalyzeFlowRequest
		wantErr bool
	}{
		{
			name: "full analysis with default options",
			request: &analyticsv1.AnalyzeFlowRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{
						{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
						{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
					},
					Edges: []*commonv1.Edge{
						{From: 1, To: 2, CurrentFlow: 50, Cost: 5, Capacity: 100},
					},
					SourceId: 1,
					SinkId:   2,
				},
			},
			wantErr: false,
		},
		{
			name: "custom options",
			request: &analyticsv1.AnalyzeFlowRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
					Edges: []*commonv1.Edge{
						{From: 1, To: 2, CurrentFlow: 50, Cost: 5, Capacity: 100},
					},
				},
				Options: &analyticsv1.AnalysisOptions{
					AnalyzeCosts:        false,
					FindBottlenecks:     true,
					CalculateStatistics: true,
				},
			},
			wantErr: false,
		},
		{
			name: "nil graph",
			request: &analyticsv1.AnalyzeFlowRequest{
				Graph: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.AnalyzeFlow(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeFlow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify response structure
				if resp.Efficiency == nil {
					t.Error("Efficiency should not be nil")
				}
			}
		})
	}
}

func TestAnalyticsService_CompareScenarios(t *testing.T) {
	svc := NewAnalyticsService()
	ctx := context.Background()

	baseline := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 50, Cost: 5, Capacity: 100},
		},
		SourceId: 1,
		SinkId:   2,
	}

	scenario := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 80, Cost: 5, Capacity: 100},
		},
		SourceId: 1,
		SinkId:   2,
	}

	tests := []struct {
		name    string
		request *analyticsv1.CompareScenariosRequest
		wantErr bool
	}{
		{
			name: "compare with baseline",
			request: &analyticsv1.CompareScenariosRequest{
				Baseline:      baseline,
				Scenarios:     []*commonv1.Graph{scenario},
				ScenarioNames: []string{"Improved"},
			},
			wantErr: false,
		},
		{
			name: "multiple scenarios",
			request: &analyticsv1.CompareScenariosRequest{
				Baseline:      baseline,
				Scenarios:     []*commonv1.Graph{scenario, scenario},
				ScenarioNames: []string{"A", "B"},
			},
			wantErr: false,
		},
		{
			name: "nil baseline",
			request: &analyticsv1.CompareScenariosRequest{
				Baseline:  nil,
				Scenarios: []*commonv1.Graph{scenario},
			},
			wantErr: true,
		},
		{
			name: "empty scenarios",
			request: &analyticsv1.CompareScenariosRequest{
				Baseline:  baseline,
				Scenarios: []*commonv1.Graph{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CompareScenarios(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CompareScenarios() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(resp.Results) != len(tt.request.Scenarios) {
					t.Errorf("Results count = %d, want %d",
						len(resp.Results), len(tt.request.Scenarios))
				}
			}
		})
	}
}

func TestCalculateEfficiency(t *testing.T) {
	tests := []struct {
		name          string
		graph         *commonv1.Graph
		expectedGrade string
	}{
		{
			name: "grade A - high utilization",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 90, Capacity: 100},
				},
			},
			expectedGrade: "A",
		},
		{
			name: "grade B - medium-high utilization",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 70, Capacity: 100},
				},
			},
			expectedGrade: "B",
		},
		{
			name: "grade C - medium utilization",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 50, Capacity: 100},
				},
			},
			expectedGrade: "C",
		},
		{
			name: "grade D - low utilization",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 30, Capacity: 100},
				},
			},
			expectedGrade: "D",
		},
		{
			name: "grade F - very low utilization",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 10, Capacity: 100},
				},
			},
			expectedGrade: "F",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateEfficiency(tt.graph, nil)
			if result.Grade != tt.expectedGrade {
				t.Errorf("Grade = %v, want %v", result.Grade, tt.expectedGrade)
			}
		})
	}
}

func TestAnalyticsService_AnalyzeFlow_AllOptionsDisabled(t *testing.T) {
	svc := NewAnalyticsService()
	ctx := context.Background()

	request := &analyticsv1.AnalyzeFlowRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, CurrentFlow: 50, Cost: 5, Capacity: 100},
			},
		},
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        false,
			FindBottlenecks:     false,
			CalculateStatistics: false,
		},
	}

	resp, err := svc.AnalyzeFlow(ctx, request)
	if err != nil {
		t.Fatalf("AnalyzeFlow() error = %v", err)
	}

	// С отключёнными опциями статистика не должна вычисляться
	if resp.FlowStats != nil {
		t.Error("FlowStats should be nil when CalculateStatistics is false")
	}
	if resp.Cost != nil {
		t.Error("Cost should be nil when AnalyzeCosts is false")
	}
	if resp.Bottlenecks != nil {
		t.Error("Bottlenecks should be nil when FindBottlenecks is false")
	}

	// Но Efficiency всегда вычисляется
	if resp.Efficiency == nil {
		t.Error("Efficiency should not be nil")
	}
}

func TestAnalyticsService_CompareScenarios_NoImprovement(t *testing.T) {
	svc := NewAnalyticsService()
	ctx := context.Background()

	// Baseline лучше всех сценариев
	baseline := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 100, Cost: 5, Capacity: 100},
		},
		SourceId: 1,
		SinkId:   2,
	}

	worseScenario := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 50, Cost: 5, Capacity: 100},
		},
		SourceId: 1,
		SinkId:   2,
	}

	resp, err := svc.CompareScenarios(ctx, &analyticsv1.CompareScenariosRequest{
		Baseline:      baseline,
		Scenarios:     []*commonv1.Graph{worseScenario},
		ScenarioNames: []string{"Worse"},
	})

	if err != nil {
		t.Fatalf("CompareScenarios() error = %v", err)
	}

	if resp.BestScenario != "" {
		t.Errorf("BestScenario should be empty when all scenarios are worse, got %s", resp.BestScenario)
	}

	if resp.Results[0].ImprovementVsBaseline >= 0 {
		t.Error("Improvement should be negative for worse scenario")
	}
}

func TestGenerateComparisonSummary_EmptyResults(t *testing.T) {
	baseStats := &commonv1.FlowStatistics{TotalFlow: 100}
	baseCost := &analyticsv1.CalculateCostResponse{TotalCost: 500}

	summary := generateComparisonSummary([]*analyticsv1.ScenarioResult{}, baseStats, baseCost)

	if summary != "Нет сценариев для сравнения" {
		t.Errorf("Unexpected summary: %s", summary)
	}
}
