package analysis

import (
	"testing"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		graph        *commonv1.Graph
		options      *analyticsv1.CostOptions
		expectedCost float64
		checkDetails func(t *testing.T, result *analyticsv1.CalculateCostResponse)
	}{
		{
			name: "simple cost calculation",
			graph: &commonv1.Graph{
				Nodes: []*commonv1.Node{
					{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
					{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
				},
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 10, Cost: 5, Capacity: 100},
				},
			},
			options:      nil,
			expectedCost: 50, // 10 * 5
		},
		{
			name: "ignores virtual nodes",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: -1, To: 1, CurrentFlow: 10, Cost: 5, Capacity: 100},
					{From: 1, To: 2, CurrentFlow: 10, Cost: 5, Capacity: 100},
					{From: 2, To: -2, CurrentFlow: 10, Cost: 5, Capacity: 100},
				},
			},
			options:      nil,
			expectedCost: 50, // only 1->2
		},
		{
			name: "ignores zero flow edges",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 0, Cost: 5, Capacity: 100},
					{From: 2, To: 3, CurrentFlow: 10, Cost: 3, Capacity: 100},
				},
			},
			options:      nil,
			expectedCost: 30, // only 2->3
		},
		{
			name: "with cost multiplier",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 10, Cost: 5, Capacity: 100,
						RoadType: commonv1.RoadType_ROAD_TYPE_HIGHWAY},
				},
			},
			options: &analyticsv1.CostOptions{
				Currency: "RUB",
				Mode:     analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE,
				CostMultipliers: map[string]float64{
					"ROAD_TYPE_HIGHWAY": 2.0,
				},
			},
			expectedCost: 100, // 10 * 5 * 2.0
		},
		{
			name: "with discount",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 10, Cost: 10, Capacity: 100},
				},
			},
			options: &analyticsv1.CostOptions{
				Currency:        "RUB",
				Mode:            analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE,
				DiscountPercent: 10,
			},
			expectedCost: 90, // 100 - 10%
		},
		{
			name: "with markup",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 10, Cost: 10, Capacity: 100},
				},
			},
			options: &analyticsv1.CostOptions{
				Currency:      "RUB",
				Mode:          analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE,
				MarkupPercent: 20,
			},
			expectedCost: 120, // 100 + 20%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCost(tt.graph, tt.options)

			if !floatEquals(result.TotalCost, tt.expectedCost, 0.01) {
				t.Errorf("CalculateCost() total = %v, want %v",
					result.TotalCost, tt.expectedCost)
			}

			if tt.checkDetails != nil {
				tt.checkDetails(t, result)
			}
		})
	}
}

func TestCalculateCostSimple(t *testing.T) {
	tests := []struct {
		name     string
		graph    *commonv1.Graph
		expected float64
	}{
		{
			name: "basic calculation",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 10, Cost: 5},
					{From: 2, To: 3, CurrentFlow: 20, Cost: 3},
				},
			},
			expected: 110, // 10*5 + 20*3
		},
		{
			name: "empty graph",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{},
			},
			expected: 0,
		},
		{
			name: "all virtual edges",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: -1, To: 1, CurrentFlow: 10, Cost: 5},
					{From: 2, To: -2, CurrentFlow: 10, Cost: 5},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCostSimple(tt.graph)
			if !floatEquals(result, tt.expected, 0.01) {
				t.Errorf("CalculateCostSimple() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCostOptionsBuilder(t *testing.T) {
	builder := CreateCostOptions()
	opts := builder.
		Currency("USD").
		IncludeFixed(true).
		Mode(analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_FULL).
		WarehouseCost(500).
		DeliveryPointCost(50).
		PerUnitHandlingCost(0.5).
		BaseOperationCost(100).
		PerEdgeCost(10).
		RoadMultiplier("HIGHWAY", 1.5).
		RoadBaseCost("HIGHWAY", 5).
		Discount(5).
		Markup(10).
		Build()

	if opts.Currency != "USD" {
		t.Errorf("Currency = %v, want USD", opts.Currency)
	}
	if !opts.IncludeFixedCosts {
		t.Error("IncludeFixedCosts should be true")
	}
	if opts.Mode != analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_FULL {
		t.Errorf("Mode = %v, want FULL", opts.Mode)
	}
	if opts.FixedCosts.WarehouseCost != 500 {
		t.Errorf("WarehouseCost = %v, want 500", opts.FixedCosts.WarehouseCost)
	}
	if opts.CostMultipliers["HIGHWAY"] != 1.5 {
		t.Errorf("RoadMultiplier[HIGHWAY] = %v, want 1.5", opts.CostMultipliers["HIGHWAY"])
	}
	if opts.DiscountPercent != 5 {
		t.Errorf("DiscountPercent = %v, want 5", opts.DiscountPercent)
	}
	if opts.MarkupPercent != 10 {
		t.Errorf("MarkupPercent = %v, want 10", opts.MarkupPercent)
	}
}

func TestCalculateCostWithModes(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 10, Cost: 10, Capacity: 100},
		},
	}

	tests := []struct {
		name string
		mode analyticsv1.CostCalculationMode
	}{
		{
			name: "simple mode",
			mode: analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE,
		},
		{
			name: "with fixed mode",
			mode: analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_WITH_FIXED,
		},
		{
			name: "full mode",
			mode: analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_FULL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &analyticsv1.CostOptions{
				Currency: "RUB",
				Mode:     tt.mode,
				FixedCosts: &analyticsv1.FixedCostConfig{
					WarehouseCost:     100,
					DeliveryPointCost: 50,
				},
			}

			result := CalculateCost(graph, opts)

			// В простом режиме фиксированные затраты должны быть 0
			if tt.mode == analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE {
				if result.Breakdown.FixedCost != 0 {
					t.Errorf("FixedCost in SIMPLE mode = %v, want 0",
						result.Breakdown.FixedCost)
				}
			}

			// Breakdown должен быть заполнен
			if result.Breakdown == nil {
				t.Error("Breakdown should not be nil")
			}
		})
	}
}

func TestGetActiveNodes(t *testing.T) {
	graph := &commonv1.Graph{
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 10},
			{From: 2, To: 3, CurrentFlow: 5},
			{From: 4, To: 5, CurrentFlow: 0}, // no flow
		},
	}

	active := getActiveNodes(graph)

	// Nodes 1, 2, 3 should be active
	if !active[1] || !active[2] || !active[3] {
		t.Error("Nodes 1, 2, 3 should be active")
	}

	// Nodes 4, 5 should not be active (zero flow)
	if active[4] || active[5] {
		t.Error("Nodes 4, 5 should not be active")
	}
}

func TestCalculateCost_WithFullMode(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 10, Cost: 5, Capacity: 100, Length: 50,
				RoadType: commonv1.RoadType_ROAD_TYPE_HIGHWAY},
		},
	}

	opts := CreateCostOptions().
		Mode(analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_FULL).
		WarehouseCost(1000).
		DeliveryPointCost(100).
		PerUnitHandlingCost(0.5).
		RoadBaseCost("ROAD_TYPE_HIGHWAY", 2.0).
		Build()

	result := CalculateCost(graph, opts)

	if result.Breakdown == nil {
		t.Fatal("Breakdown should not be nil")
	}

	// Transport: 10 * 5 = 50
	if result.Breakdown.TransportCost != 50 {
		t.Errorf("TransportCost = %v, want 50", result.Breakdown.TransportCost)
	}

	// Handling: 10 * 0.5 = 5
	if result.Breakdown.HandlingCost != 5 {
		t.Errorf("HandlingCost = %v, want 5", result.Breakdown.HandlingCost)
	}

	// RoadBase: 50 (length) * 2.0 = 100
	if result.Breakdown.RoadBaseCost != 100 {
		t.Errorf("RoadBaseCost = %v, want 100", result.Breakdown.RoadBaseCost)
	}
}

func TestCalculateCost_DiscountAndMarkupCombined(t *testing.T) {
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, CurrentFlow: 10, Cost: 10, Capacity: 100},
		},
	}

	opts := &analyticsv1.CostOptions{
		Currency:        "RUB",
		Mode:            analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE,
		DiscountPercent: 20,
		MarkupPercent:   10,
	}

	result := CalculateCost(graph, opts)

	// Base: 100
	// After discount (20%): 100 - 20 = 80
	// After markup (10% of 80): 80 + 8 = 88
	expected := 88.0
	if !floatEquals(result.TotalCost, expected, 0.01) {
		t.Errorf("TotalCost = %v, want %v", result.TotalCost, expected)
	}
}

func TestNormalizeOptions_NilOptions(t *testing.T) {
	opts := normalizeOptions(nil)

	if opts == nil {
		t.Fatal("normalizeOptions(nil) should return non-nil options")
	}
	if opts.Currency != "RUB" {
		t.Errorf("Default currency = %v, want RUB", opts.Currency)
	}
	if opts.Mode != analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE {
		t.Errorf("Default mode should be SIMPLE")
	}
}
