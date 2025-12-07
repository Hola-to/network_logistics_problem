// services/gateway-svc/internal/metrics/metrics_test.go

package metrics

import (
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	m := Init()
	if m == nil {
		t.Fatal("Init() should not return nil")
	}

	// Calling Init again should return the same instance
	m2 := Init()
	if m != m2 {
		t.Error("Init() should return the same instance (singleton)")
	}
}

func TestGet(t *testing.T) {
	m := Get()
	if m == nil {
		t.Fatal("Get() should not return nil")
	}
}

func TestGatewayMetrics_RecordRequest(t *testing.T) {
	m := Get()

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordRequest panicked: %v", r)
		}
	}()

	m.RecordRequest("auth", "Login", "success", 100*time.Millisecond)
	m.RecordRequest("solver", "Solve", "error", 500*time.Millisecond)
	m.RecordRequest("analytics", "Analyze", "success", 50*time.Millisecond)
}

func TestGatewayMetrics_RecordBackendRequest(t *testing.T) {
	m := Get()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordBackendRequest panicked: %v", r)
		}
	}()

	m.RecordBackendRequest("auth-svc", "Login", "success", 50*time.Millisecond)
	m.RecordBackendRequest("solver-svc", "Solve", "error", 200*time.Millisecond)
}

func TestGatewayMetrics_ActiveRequests(t *testing.T) {
	m := Get()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Active requests operations panicked: %v", r)
		}
	}()

	m.IncActiveRequests()
	m.IncActiveRequests()
	m.DecActiveRequests()
}

func TestGatewayMetrics_RecordBackendHealth(t *testing.T) {
	m := Get()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordBackendHealth panicked: %v", r)
		}
	}()

	m.RecordBackendHealth("auth-svc", true)
	m.RecordBackendHealth("solver-svc", false)
	m.RecordBackendHealth("analytics-svc", true)
}

func TestGatewayMetrics_RecordError(t *testing.T) {
	m := Get()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordError panicked: %v", r)
		}
	}()

	m.RecordError("timeout", "solver-svc")
	m.RecordError("connection", "auth-svc")
	m.RecordError("validation", "gateway")
}

func TestGatewayMetrics_Counters(t *testing.T) {
	m := Get()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Counter operations panicked: %v", r)
		}
	}()

	// Rate limiting
	m.RateLimitHits.Inc()
	m.RateLimitPassed.Inc()

	// Auth
	m.AuthSuccessful.Inc()
	m.AuthFailed.Inc()

	// Cache
	m.CacheHits.Inc()
	m.CacheMisses.Inc()
}

func TestGatewayMetrics_AllMetricsInitialized(t *testing.T) {
	m := Get()

	if m.RequestsTotal == nil {
		t.Error("RequestsTotal should not be nil")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration should not be nil")
	}
	if m.BackendRequests == nil {
		t.Error("BackendRequests should not be nil")
	}
	if m.BackendDuration == nil {
		t.Error("BackendDuration should not be nil")
	}
	if m.ActiveRequests == nil {
		t.Error("ActiveRequests should not be nil")
	}
	if m.RequestsByCategory == nil {
		t.Error("RequestsByCategory should not be nil")
	}
	if m.ResponseTimeByCategory == nil {
		t.Error("ResponseTimeByCategory should not be nil")
	}
	if m.BackendConnections == nil {
		t.Error("BackendConnections should not be nil")
	}
	if m.BackendHealth == nil {
		t.Error("BackendHealth should not be nil")
	}
	if m.ErrorsByType == nil {
		t.Error("ErrorsByType should not be nil")
	}
	if m.RequestSize == nil {
		t.Error("RequestSize should not be nil")
	}
	if m.ResponseSize == nil {
		t.Error("ResponseSize should not be nil")
	}
	if m.RateLimitHits == nil {
		t.Error("RateLimitHits should not be nil")
	}
	if m.RateLimitPassed == nil {
		t.Error("RateLimitPassed should not be nil")
	}
	if m.AuthSuccessful == nil {
		t.Error("AuthSuccessful should not be nil")
	}
	if m.AuthFailed == nil {
		t.Error("AuthFailed should not be nil")
	}
	if m.CacheHits == nil {
		t.Error("CacheHits should not be nil")
	}
	if m.CacheMisses == nil {
		t.Error("CacheMisses should not be nil")
	}
}

func TestGatewayMetrics_RecordRequestWithVariousStatuses(t *testing.T) {
	m := Get()

	statuses := []string{"success", "error", "timeout", "canceled", "invalid_argument"}
	categories := []string{"auth", "solver", "analytics", "validation", "simulation"}

	for _, cat := range categories {
		for _, status := range statuses {
			m.RecordRequest(cat, "TestMethod", status, 10*time.Millisecond)
		}
	}
}
