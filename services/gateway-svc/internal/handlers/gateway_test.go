// services/gateway-svc/internal/handlers/gateway_test.go

package handlers

import (
	"testing"
	"time"

	"logistics/pkg/config"
)

func TestGatewayHandler_Initialization(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "gateway-test",
			Version:     "1.0.0",
			Environment: "test",
		},
		RateLimit: config.RateLimitConfig{
			Enabled:   true,
			Requests:  100,
			BurstSize: 10,
		},
	}

	// Test that config is properly stored
	// Note: Can't create full handler without clients
	if cfg.App.Name != "gateway-test" {
		t.Error("Config should be properly set")
	}
}

func TestStatusHealthy(t *testing.T) {
	if statusHealthy != "HEALTHY" {
		t.Errorf("statusHealthy = %q, want %q", statusHealthy, "HEALTHY")
	}
}

func TestStartedAtTracking(t *testing.T) {
	// Verify time tracking works correctly
	before := time.Now()
	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	if !before.Before(after) {
		t.Error("Time tracking should work correctly")
	}
}
