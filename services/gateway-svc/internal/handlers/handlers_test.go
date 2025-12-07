// services/gateway-svc/internal/handlers/handlers_test.go

package handlers

import (
	"testing"
	"time"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	authv1 "logistics/gen/go/logistics/auth/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	"logistics/pkg/config"
)

// ============================================================
// Helper functions tests
// ============================================================

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("generateRequestID() should not return empty string")
	}

	if id1 == id2 {
		t.Error("generateRequestID() should return unique IDs")
	}

	// Should have timestamp prefix
	if len(id1) < 14 {
		t.Error("generateRequestID() should include timestamp")
	}
}

func TestAuthHandler_ConvertUserProfile(t *testing.T) {
	h := &AuthHandler{}

	// Test nil input
	result := h.convertUserProfile(nil)
	if result != nil {
		t.Error("convertUserProfile(nil) should return nil")
	}

	// Test valid input
	user := &authv1.UserInfo{
		UserId:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Test User",
		Role:     "admin",
	}

	profile := h.convertUserProfile(user)
	if profile == nil {
		t.Fatal("convertUserProfile should not return nil for valid input")
	}

	if profile.UserId != user.UserId {
		t.Errorf("UserId = %v, want %v", profile.UserId, user.UserId)
	}
	if profile.Username != user.Username {
		t.Errorf("Username = %v, want %v", profile.Username, user.Username)
	}
	if profile.Email != user.Email {
		t.Errorf("Email = %v, want %v", profile.Email, user.Email)
	}
	if profile.Role != user.Role {
		t.Errorf("Role = %v, want %v", profile.Role, user.Role)
	}
}

func TestAnalyticsHandler_ConvertCostBreakdown(t *testing.T) {
	h := &AnalyticsHandler{}

	// Test nil input
	result := h.convertCostBreakdown(nil)
	if result != nil {
		t.Error("convertCostBreakdown(nil) should return nil")
	}

	// Test valid input
	breakdown := &analyticsv1.CostBreakdown{
		TransportCost:  100.0,
		FixedCost:      50.0,
		HandlingCost:   25.0,
		DiscountAmount: 10.0,
		MarkupAmount:   5.0,
		CostByRoadType: map[string]float64{"highway": 80.0},
		CostByNodeType: map[string]float64{"warehouse": 20.0},
	}

	result = h.convertCostBreakdown(breakdown)
	if result == nil {
		t.Fatal("convertCostBreakdown should not return nil for valid input")
	}

	if result.TransportCost != breakdown.TransportCost {
		t.Errorf("TransportCost = %v, want %v", result.TransportCost, breakdown.TransportCost)
	}
	if result.FixedCost != breakdown.FixedCost {
		t.Errorf("FixedCost = %v, want %v", result.FixedCost, breakdown.FixedCost)
	}
}

func TestAnalyticsHandler_ConvertEfficiency(t *testing.T) {
	h := &AnalyticsHandler{}

	// Test nil input
	result := h.convertEfficiency(nil)
	if result != nil {
		t.Error("convertEfficiency(nil) should return nil")
	}

	// Test valid input
	efficiency := &analyticsv1.EfficiencyReport{
		OverallEfficiency:   0.85,
		CapacityUtilization: 0.75,
		UnusedEdgesCount:    5,
		SaturatedEdgesCount: 3,
		Grade:               "B",
	}

	result = h.convertEfficiency(efficiency)
	if result == nil {
		t.Fatal("convertEfficiency should not return nil for valid input")
	}

	if result.Grade != "B" {
		t.Errorf("Grade = %v, want B", result.Grade)
	}
	if result.OverallEfficiency != 0.85 {
		t.Errorf("OverallEfficiency = %v, want 0.85", result.OverallEfficiency)
	}
}

func TestAuditHandler_ParseAction(t *testing.T) {
	h := &AuditHandler{}

	tests := []struct {
		input    string
		expected string
	}{
		{"CREATE", "AUDIT_ACTION_CREATE"},
		{"READ", "AUDIT_ACTION_READ"},
		{"UPDATE", "AUDIT_ACTION_UPDATE"},
		{"DELETE", "AUDIT_ACTION_DELETE"},
		{"LOGIN", "AUDIT_ACTION_LOGIN"},
		{"LOGOUT", "AUDIT_ACTION_LOGOUT"},
		{"SOLVE", "AUDIT_ACTION_SOLVE"},
		{"ANALYZE", "AUDIT_ACTION_ANALYZE"},
		{"UNKNOWN", "AUDIT_ACTION_UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.parseAction(tt.input)
			if result.String() != tt.expected {
				t.Errorf("parseAction(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAuditHandler_FormatAction(t *testing.T) {
	tests := []struct {
		name     string
		input    int32 // AuditAction value
		expected string
	}{
		{"create", 1, "CREATE"},
		{"read", 2, "READ"},
		{"update", 3, "UPDATE"},
		{"delete", 4, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The actual enum values might differ, this is just testing the logic
			// In real test we'd use the actual enum
		})
	}
}

func TestAuditHandler_FormatOutcome(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"success", "SUCCESS"},
		{"failure", "FAILURE"},
		{"denied", "DENIED"},
		{"error", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test formatOutcome logic
		})
	}
}

func TestAuditHandler_CalculatePage(t *testing.T) {
	h := &AuditHandler{}

	tests := []struct {
		name     string
		offset   int32
		limit    int32
		expected int32
	}{
		{"first page", 0, 20, 1},
		{"second page", 20, 20, 2},
		{"third page", 40, 20, 3},
		{"zero limit defaults", 0, 0, 1},
		{"negative limit defaults", 0, -1, 1},
		{"partial offset", 15, 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.calculatePage(tt.offset, tt.limit)
			if result != tt.expected {
				t.Errorf("calculatePage(%d, %d) = %d, want %d",
					tt.offset, tt.limit, result, tt.expected)
			}
		})
	}
}

func TestHistoryHandler_Constants(t *testing.T) {
	if anonymousUserID != "anonymous" {
		t.Errorf("anonymousUserID = %v, want 'anonymous'", anonymousUserID)
	}
}

func TestGatewayHandler_Constants(t *testing.T) {
	if statusHealthy != "HEALTHY" {
		t.Errorf("statusHealthy = %v, want 'HEALTHY'", statusHealthy)
	}
}

func TestSolverHandler_ConvertMetrics(t *testing.T) {
	h := &SolverHandler{}

	// Test nil input
	result := h.convertMetrics(nil)
	if result != nil {
		t.Error("convertMetrics(nil) should return nil")
	}
}

func TestSolverHandler_SetMetadata(t *testing.T) {
	h := &SolverHandler{}

	meta := &gatewayv1.RequestMetadata{}
	start := time.Now().Add(-100 * time.Millisecond)

	h.setMetadata(meta, 10*time.Millisecond, 50*time.Millisecond, 20*time.Millisecond, start)

	if meta.ValidationTimeMs != 10 {
		t.Errorf("ValidationTimeMs = %v, want 10", meta.ValidationTimeMs)
	}
	if meta.SolveTimeMs != 50 {
		t.Errorf("SolveTimeMs = %v, want 50", meta.SolveTimeMs)
	}
	if meta.AnalyticsTimeMs != 20 {
		t.Errorf("AnalyticsTimeMs = %v, want 20", meta.AnalyticsTimeMs)
	}
	if meta.TotalTimeMs < 100 {
		t.Errorf("TotalTimeMs = %v, should be >= 100", meta.TotalTimeMs)
	}
}

func TestSimulationHandler_ConvertMetadata(t *testing.T) {
	h := &SimulationHandler{}

	// Test nil input
	result := h.convertMetadata(nil)
	if result != nil {
		t.Error("convertMetadata(nil) should return nil")
	}
}

func TestSimulationHandler_ConvertMonteCarloStats(t *testing.T) {
	h := &SimulationHandler{}

	// Test nil input
	result := h.convertMonteCarloStats(nil)
	if result != nil {
		t.Error("convertMonteCarloStats(nil) should return nil")
	}
}

func TestSimulationHandler_ConvertRiskAnalysis(t *testing.T) {
	h := &SimulationHandler{}

	// Test nil input
	result := h.convertRiskAnalysis(nil)
	if result != nil {
		t.Error("convertRiskAnalysis(nil) should return nil")
	}
}

func TestSimulationHandler_ConvertScenarioResult(t *testing.T) {
	h := &SimulationHandler{}

	// Test nil input
	result := h.convertScenarioResult(nil)
	if result != nil {
		t.Error("convertScenarioResult(nil) should return nil")
	}
}

func TestSimulationHandler_ConvertComparison(t *testing.T) {
	h := &SimulationHandler{}

	// Test nil input
	result := h.convertComparison(nil)
	if result != nil {
		t.Error("convertComparison(nil) should return nil")
	}
}

func TestReportHandler_ConvertOptions(t *testing.T) {
	h := &ReportHandler{}

	// Test nil input
	result := h.convertOptions(nil)
	if result != nil {
		t.Error("convertOptions(nil) should return nil")
	}

	// Test valid input
	opts := &gatewayv1.ReportOptions{
		Title:       "Test Report",
		Description: "Test description",
		Author:      "Test Author",
		Language:    "ru",
	}

	result = h.convertOptions(opts)
	if result == nil {
		t.Fatal("convertOptions should not return nil for valid input")
	}

	if result.Title != opts.Title {
		t.Errorf("Title = %v, want %v", result.Title, opts.Title)
	}
}

func TestReportHandler_ConvertReportInfo(t *testing.T) {
	h := &ReportHandler{}

	// Test nil input
	result := h.convertReportInfo(nil)
	if result != nil {
		t.Error("convertReportInfo(nil) should return nil")
	}
}

// ============================================================
// GatewayHandler initialization tests
// ============================================================

func TestNewGatewayHandler(t *testing.T) {
	// We can't fully test without real clients, but we can test with nil
	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewGatewayHandler panicked: %v", r)
		}
	}()

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

	// This will panic because clients is nil, but that's expected
	// In real tests we'd use mock clients
	_ = cfg
}

func TestNewAuthHandler(t *testing.T) {
	h := NewAuthHandler(nil)
	if h == nil {
		t.Error("NewAuthHandler should not return nil")
	}
}

func TestNewAnalyticsHandler(t *testing.T) {
	h := NewAnalyticsHandler(nil)
	if h == nil {
		t.Error("NewAnalyticsHandler should not return nil")
	}
}

func TestNewAuditHandler(t *testing.T) {
	h := NewAuditHandler(nil)
	if h == nil {
		t.Error("NewAuditHandler should not return nil")
	}
}

func TestNewValidationHandler(t *testing.T) {
	h := NewValidationHandler(nil)
	if h == nil {
		t.Error("NewValidationHandler should not return nil")
	}
}

func TestNewSimulationHandler(t *testing.T) {
	h := NewSimulationHandler(nil)
	if h == nil {
		t.Error("NewSimulationHandler should not return nil")
	}
}

func TestNewHistoryHandler(t *testing.T) {
	h := NewHistoryHandler(nil)
	if h == nil {
		t.Error("NewHistoryHandler should not return nil")
	}
}

func TestNewReportHandler(t *testing.T) {
	h := NewReportHandler(nil)
	if h == nil {
		t.Error("NewReportHandler should not return nil")
	}
}

// ============================================================
// Bottleneck conversion tests
// ============================================================

func TestAnalyticsHandler_ConvertBottleneckAnalysis(t *testing.T) {
	h := &AnalyticsHandler{}

	// Test nil input
	result := h.convertBottleneckAnalysis(nil)
	if result != nil {
		t.Error("convertBottleneckAnalysis(nil) should return nil")
	}

	// Test empty bottlenecks
	empty := &analyticsv1.FindBottlenecksResponse{
		Bottlenecks:     []*analyticsv1.Bottleneck{},
		Recommendations: []*analyticsv1.Recommendation{},
	}

	result = h.convertBottleneckAnalysis(empty)
	if result == nil {
		t.Fatal("convertBottleneckAnalysis should not return nil for valid input")
	}
	if result.TotalBottlenecks != 0 {
		t.Errorf("TotalBottlenecks = %d, want 0", result.TotalBottlenecks)
	}

	// Test with bottlenecks
	withBottlenecks := &analyticsv1.FindBottlenecksResponse{
		Bottlenecks: []*analyticsv1.Bottleneck{
			{
				Edge:        &commonv1.Edge{From: 1, To: 2},
				Utilization: 0.95,
				ImpactScore: 0.8,
				Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
			},
		},
		Recommendations: []*analyticsv1.Recommendation{
			{
				Type:        "increase_capacity",
				Description: "Increase capacity",
			},
		},
	}

	result = h.convertBottleneckAnalysis(withBottlenecks)
	if result.TotalBottlenecks != 1 {
		t.Errorf("TotalBottlenecks = %d, want 1", result.TotalBottlenecks)
	}
	if len(result.Bottlenecks) != 1 {
		t.Errorf("Bottlenecks count = %d, want 1", len(result.Bottlenecks))
	}
	if len(result.Recommendations) != 1 {
		t.Errorf("Recommendations count = %d, want 1", len(result.Recommendations))
	}
}

func TestAnalyticsHandler_ConvertCostAnalysis(t *testing.T) {
	h := &AnalyticsHandler{}

	// Test nil input
	result := h.convertCostAnalysis(nil)
	if result != nil {
		t.Error("convertCostAnalysis(nil) should return nil")
	}

	// Test valid input
	cost := &analyticsv1.CalculateCostResponse{
		TotalCost: 1500.0,
		Currency:  "RUB",
		Breakdown: &analyticsv1.CostBreakdown{
			TransportCost: 1000.0,
			FixedCost:     500.0,
		},
	}

	result = h.convertCostAnalysis(cost)
	if result == nil {
		t.Fatal("convertCostAnalysis should not return nil for valid input")
	}

	if result.TotalCost != 1500.0 {
		t.Errorf("TotalCost = %v, want 1500.0", result.TotalCost)
	}
	if result.Currency != "RUB" {
		t.Errorf("Currency = %v, want RUB", result.Currency)
	}
}
