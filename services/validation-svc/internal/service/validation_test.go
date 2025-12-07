package service

import (
	"context"
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
)

func TestNewValidationService(t *testing.T) {
	svc := NewValidationService("1.0.0")

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.version != "1.0.0" {
		t.Errorf("version = %s, want 1.0.0", svc.version)
	}
}

func TestValidationService_ValidateGraph(t *testing.T) {
	svc := NewValidationService("1.0.0")
	ctx := context.Background()

	tests := []struct {
		name      string
		request   *validationv1.ValidateGraphRequest
		wantValid bool
	}{
		{
			name: "valid_graph",
			request: &validationv1.ValidateGraphRequest{
				Graph: createTestGraph(),
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
			},
			wantValid: true,
		},
		{
			name: "empty_graph",
			request: &validationv1.ValidateGraphRequest{
				Graph: &commonv1.Graph{},
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
			},
			wantValid: false,
		},
		{
			name: "invalid_source",
			request: &validationv1.ValidateGraphRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{
						{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
						{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
					},
					Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
					SourceId: 999,
					SinkId:   2,
				},
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
			},
			wantValid: false,
		},
		{
			name: "full_validation",
			request: &validationv1.ValidateGraphRequest{
				Graph:              createTestGraph(),
				Level:              validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
				CheckConnectivity:  true,
				CheckBusinessRules: true,
				CheckTopology:      true,
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.ValidateGraph(ctx, tt.request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Result.IsValid != tt.wantValid {
				t.Errorf("IsValid = %v, want %v, errors: %+v", resp.Result.IsValid, tt.wantValid, resp.Result.Errors)
			}

			if resp.Metrics == nil {
				t.Error("expected metrics to be set")
			}
		})
	}
}

func TestValidationService_ValidateFlow(t *testing.T) {
	svc := NewValidationService("1.0.0")
	ctx := context.Background()

	tests := []struct {
		name      string
		request   *validationv1.ValidateFlowRequest
		wantValid bool
	}{
		{
			name: "valid_flow",
			request: &validationv1.ValidateFlowRequest{
				Graph: createFlowTestGraph(5, 5), // Equal flows
			},
			wantValid: true,
		},
		{
			name: "invalid_flow_overflow",
			request: &validationv1.ValidateFlowRequest{
				Graph: createFlowTestGraph(15, 15), // Overflow
			},
			wantValid: false,
		},
		{
			name: "expected_max_flow_match",
			request: &validationv1.ValidateFlowRequest{
				Graph:           createFlowTestGraph(5, 5),
				ExpectedMaxFlow: 5,
			},
			wantValid: true,
		},
		{
			name: "expected_max_flow_mismatch",
			request: &validationv1.ValidateFlowRequest{
				Graph:           createFlowTestGraph(5, 5),
				ExpectedMaxFlow: 10,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.ValidateFlow(ctx, tt.request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.IsValid != tt.wantValid {
				t.Errorf("IsValid = %v, want %v, violations: %+v", resp.IsValid, tt.wantValid, resp.Violations)
			}

			if resp.Summary == nil {
				t.Error("expected summary to be set")
			}
		})
	}
}

func TestValidationService_ValidateForAlgorithm(t *testing.T) {
	svc := NewValidationService("1.0.0")
	ctx := context.Background()

	tests := []struct {
		name           string
		request        *validationv1.ValidateForAlgorithmRequest
		wantCompatible bool
	}{
		{
			name: "edmonds_karp_compatible",
			request: &validationv1.ValidateForAlgorithmRequest{
				Graph:     createTestGraph(),
				Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			},
			wantCompatible: true,
		},
		{
			name: "dinic_compatible",
			request: &validationv1.ValidateForAlgorithmRequest{
				Graph:     createTestGraph(),
				Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
			},
			wantCompatible: true,
		},
		{
			name: "negative_capacity",
			request: &validationv1.ValidateForAlgorithmRequest{
				Graph: &commonv1.Graph{
					Nodes: []*commonv1.Node{
						{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
						{Id: 2, Type: commonv1.NodeType_NODE_TYPE_SINK},
					},
					Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: -10}},
					SourceId: 1,
					SinkId:   2,
				},
				Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			},
			wantCompatible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.ValidateForAlgorithm(ctx, tt.request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.IsCompatible != tt.wantCompatible {
				t.Errorf("IsCompatible = %v, want %v", resp.IsCompatible, tt.wantCompatible)
			}
		})
	}
}

func TestValidationService_ValidateAll(t *testing.T) {
	svc := NewValidationService("1.0.0")
	ctx := context.Background()

	tests := []struct {
		name      string
		request   *validationv1.ValidateAllRequest
		wantValid bool
	}{
		{
			name: "all_valid",
			request: &validationv1.ValidateAllRequest{
				Graph:     createTestGraph(),
				Level:     validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
				Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
			},
			wantValid: true,
		},
		{
			name: "without_algorithm",
			request: &validationv1.ValidateAllRequest{
				Graph: createTestGraph(),
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
			},
			wantValid: true,
		},
		{
			name: "invalid_graph",
			request: &validationv1.ValidateAllRequest{
				Graph: &commonv1.Graph{},
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.ValidateAll(ctx, tt.request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.IsValid != tt.wantValid {
				t.Errorf("IsValid = %v, want %v", resp.IsValid, tt.wantValid)
			}

			if resp.GraphValidation == nil {
				t.Error("expected GraphValidation to be set")
			}

			if resp.FlowValidation == nil {
				t.Error("expected FlowValidation to be set")
			}

			if resp.Metrics == nil {
				t.Error("expected Metrics to be set")
			}
		})
	}
}

func TestValidationService_Health(t *testing.T) {
	svc := NewValidationService("1.0.0")
	ctx := context.Background()

	resp, err := svc.Health(ctx, &validationv1.HealthRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "SERVING" {
		t.Errorf("Status = %s, want SERVING", resp.Status)
	}

	if resp.Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", resp.Version)
	}

	if resp.UptimeSeconds < 0 {
		t.Error("UptimeSeconds should be non-negative")
	}
}

// Helper functions
func createTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 2, To: 3, Capacity: 10, Cost: 1},
		},
		SourceId: 1,
		SinkId:   3,
	}
}

func createFlowTestGraph(flow1, flow2 float64) *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: flow1},
			{From: 2, To: 3, Capacity: 10, CurrentFlow: flow2},
		},
		SourceId: 1,
		SinkId:   3,
	}
}
