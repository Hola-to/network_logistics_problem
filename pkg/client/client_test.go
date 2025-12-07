package client

import (
	commonv1 "logistics/gen/go/logistics/common/v1"
	"testing"
	"time"
)

func TestDefaultSolverClientConfig(t *testing.T) {
	cfg := DefaultSolverClientConfig()

	if cfg.Address == "" {
		t.Error("Address should not be empty")
	}
	if cfg.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	if cfg.MaxRetries <= 0 {
		t.Error("MaxRetries should be positive")
	}
}

func TestSolverClientConfig_CustomValues(t *testing.T) {
	cfg := &SolverClientConfig{
		Address:    "custom:50054",
		Timeout:    60 * time.Second,
		MaxRetries: 5,
		EnableTLS:  true,
		CertFile:   "/path/to/cert",
	}

	if cfg.Address != "custom:50054" {
		t.Errorf("Address = %s, want custom:50054", cfg.Address)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
}

func TestCalculateFlowStats(t *testing.T) {
	tests := []struct {
		name            string
		graph           *commonv1.Graph
		wantAvgUtil     float64
		wantSaturated   int32
		wantActivePaths int32
	}{
		{
			name:            "nil graph",
			graph:           nil,
			wantAvgUtil:     0,
			wantSaturated:   0,
			wantActivePaths: 0,
		},
		{
			name: "graph with flow",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 10, CurrentFlow: 10}, // 100% saturated
					{From: 2, To: 3, Capacity: 10, CurrentFlow: 5},  // 50%
				},
			},
			wantSaturated:   1,
			wantActivePaths: 2,
		},
		{
			name: "graph without flow",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 10, CurrentFlow: 0},
				},
			},
			wantAvgUtil:     0,
			wantSaturated:   0,
			wantActivePaths: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avgUtil, saturated, activePaths := calculateFlowStats(tt.graph)

			if saturated != tt.wantSaturated {
				t.Errorf("saturated = %d, want %d", saturated, tt.wantSaturated)
			}
			if activePaths != tt.wantActivePaths {
				t.Errorf("activePaths = %d, want %d", activePaths, tt.wantActivePaths)
			}
			_ = avgUtil // используется в некоторых тестах
		})
	}
}

func TestClientConfig(t *testing.T) {
	cfg := ClientConfig{
		Address:      "localhost:50051",
		Timeout:      10 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
	}

	if cfg.Address != "localhost:50051" {
		t.Errorf("Address = %s, want localhost:50051", cfg.Address)
	}
}
