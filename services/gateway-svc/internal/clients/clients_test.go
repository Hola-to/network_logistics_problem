// services/gateway-svc/internal/clients/clients_test.go

package clients

import (
	"testing"

	"logistics/pkg/config"
)

func TestDialOptions(t *testing.T) {
	endpoint := config.ServiceEndpoint{
		Host: "localhost",
		Port: 50051,
	}

	opts := dialOptions(endpoint)

	if len(opts) == 0 {
		t.Error("dialOptions should return at least one option")
	}
}

func TestServiceEndpointAddress(t *testing.T) {
	endpoint := config.ServiceEndpoint{
		Host: "localhost",
		Port: 50051,
	}

	expected := "localhost:50051"
	if endpoint.Address() != expected {
		t.Errorf("Address() = %v, want %v", endpoint.Address(), expected)
	}
}

func TestManagerConfig(t *testing.T) {
	cfg := &Config{
		Auth:       config.ServiceEndpoint{Host: "auth", Port: 50055},
		Solver:     config.ServiceEndpoint{Host: "solver", Port: 50051},
		Analytics:  config.ServiceEndpoint{Host: "analytics", Port: 50053},
		Validation: config.ServiceEndpoint{Host: "validation", Port: 50052},
		Simulation: config.ServiceEndpoint{Host: "simulation", Port: 50054},
		History:    config.ServiceEndpoint{Host: "history", Port: 50056},
		Report:     config.ServiceEndpoint{Host: "report", Port: 50058},
		Audit:      config.ServiceEndpoint{Host: "audit", Port: 50057},
	}

	if cfg.Auth.Address() != "auth:50055" {
		t.Errorf("Auth address = %v, want auth:50055", cfg.Auth.Address())
	}

	if cfg.Solver.Address() != "solver:50051" {
		t.Errorf("Solver address = %v, want solver:50051", cfg.Solver.Address())
	}
}

func TestSolveResult_Fields(t *testing.T) {
	result := &SolveResult{
		Success:      true,
		ErrorMessage: "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	result.Success = false
	result.ErrorMessage = "test error"

	if result.Success {
		t.Error("Success should be false")
	}
	if result.ErrorMessage != "test error" {
		t.Errorf("ErrorMessage = %v, want 'test error'", result.ErrorMessage)
	}
}
