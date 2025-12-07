package validators

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestValidateForAlgorithm(t *testing.T) {
	tests := []struct {
		name           string
		graph          *commonv1.Graph
		algorithm      commonv1.Algorithm
		wantCompatible bool
		wantIssues     int
	}{
		{
			name:           "edmonds_karp_valid",
			graph:          createValidGraph(),
			algorithm:      commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			wantCompatible: true,
			wantIssues:     0,
		},
		{
			name:           "dinic_valid",
			graph:          createValidGraph(),
			algorithm:      commonv1.Algorithm_ALGORITHM_DINIC,
			wantCompatible: true,
			wantIssues:     0,
		},
		{
			name:           "min_cost_valid",
			graph:          createValidGraphWithCosts(),
			algorithm:      commonv1.Algorithm_ALGORITHM_MIN_COST,
			wantCompatible: true,
			wantIssues:     0,
		},
		{
			name:           "push_relabel_valid",
			graph:          createValidGraph(),
			algorithm:      commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
			wantCompatible: true,
			wantIssues:     0,
		},
		{
			name:           "ford_fulkerson_valid",
			graph:          createValidGraph(),
			algorithm:      commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
			wantCompatible: true,
			wantIssues:     0,
		},
		{
			name:           "negative_capacity",
			graph:          createGraphWithNegativeCapacity(),
			algorithm:      commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			wantCompatible: false,
			wantIssues:     1,
		},
		{
			name:           "unknown_algorithm",
			graph:          createValidGraph(),
			algorithm:      commonv1.Algorithm(999),
			wantCompatible: false,
			wantIssues:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateForAlgorithm(tt.graph, tt.algorithm)

			if result.IsCompatible != tt.wantCompatible {
				t.Errorf("IsCompatible = %v, want %v", result.IsCompatible, tt.wantCompatible)
			}

			if len(result.Issues) != tt.wantIssues {
				t.Errorf("Issues count = %d, want %d: %v", len(result.Issues), tt.wantIssues, result.Issues)
			}
		})
	}
}

func TestValidateForAlgorithm_Complexity(t *testing.T) {
	graph := createValidGraph()

	tests := []struct {
		name      string
		algorithm commonv1.Algorithm
	}{
		{"edmonds_karp", commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{"dinic", commonv1.Algorithm_ALGORITHM_DINIC},
		{"min_cost", commonv1.Algorithm_ALGORITHM_MIN_COST},
		{"push_relabel", commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
		{"ford_fulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateForAlgorithm(graph, tt.algorithm)

			if result.Complexity == nil {
				t.Error("Complexity should not be nil for known algorithms")
			}

			if result.Complexity != nil {
				if result.Complexity.TimeComplexity == "" {
					t.Error("TimeComplexity should not be empty")
				}
				if result.Complexity.SpaceComplexity == "" {
					t.Error("SpaceComplexity should not be empty")
				}
			}
		})
	}
}

func TestCheckNonNegativeCapacity(t *testing.T) {
	tests := []struct {
		name       string
		graph      *commonv1.Graph
		wantIssues int
	}{
		{
			name:       "all_positive",
			graph:      createValidGraph(),
			wantIssues: 0,
		},
		{
			name:       "negative_capacity",
			graph:      createGraphWithNegativeCapacity(),
			wantIssues: 1,
		},
		{
			name:       "multiple_negative",
			graph:      createGraphWithMultipleNegativeCapacities(),
			wantIssues: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateForAlgorithm(tt.graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)

			issueCount := 0
			for _, issue := range result.Issues {
				if issue != "" {
					issueCount++
				}
			}

			if !result.IsCompatible && tt.wantIssues == 0 {
				t.Errorf("Expected compatible but got issues: %v", result.Issues)
			}
		})
	}
}

func TestMinCostAlgorithmValidation(t *testing.T) {
	t.Run("zero_costs_warning", func(t *testing.T) {
		graph := createValidGraph() // No costs set
		result := ValidateForAlgorithm(graph, commonv1.Algorithm_ALGORITHM_MIN_COST)

		if len(result.Recommendations) == 0 {
			t.Error("Expected recommendation for zero costs")
		}
	})

	t.Run("with_costs", func(t *testing.T) {
		graph := createValidGraphWithCosts()
		result := ValidateForAlgorithm(graph, commonv1.Algorithm_ALGORITHM_MIN_COST)

		if !result.IsCompatible {
			t.Error("Expected compatible for graph with costs")
		}
	})
}

func TestFordFulkersonValidation(t *testing.T) {
	t.Run("integer_capacities", func(t *testing.T) {
		graph := createValidGraph() // Integer capacities
		result := ValidateForAlgorithm(graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)

		if !result.IsCompatible {
			t.Error("Expected compatible for integer capacities")
		}
	})

	t.Run("non_integer_capacities", func(t *testing.T) {
		graph := createGraphWithNonIntegerCapacities()
		result := ValidateForAlgorithm(graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)

		// Should still be compatible but with warning
		if len(result.Recommendations) == 0 {
			t.Error("Expected recommendation for non-integer capacities")
		}
	})
}

// Helper functions
func createValidGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 0},
			{From: 2, To: 3, Capacity: 10, Cost: 0},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createValidGraphWithCosts() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 5},
			{From: 2, To: 3, Capacity: 10, Cost: 3},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createGraphWithNegativeCapacity() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: -10, Cost: 0},
		},
		SourceId: 1,
		SinkId:   2,
	}
}

func createGraphWithMultipleNegativeCapacities() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: -10, Cost: 0},
			{From: 2, To: 3, Capacity: -5, Cost: 0},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createGraphWithNonIntegerCapacities() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10.5, Cost: 0},
		},
		SourceId: 1,
		SinkId:   2,
	}
}
