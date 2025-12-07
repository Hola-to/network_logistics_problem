// services/gateway-svc/internal/middleware/ratelimit_test.go

package middleware

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestDefaultKeyExtractor(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		method   string
		wantKey  string
	}{
		{
			name: "with user id",
			setupCtx: func() context.Context {
				return WithUserID(context.Background(), "user-123")
			},
			method:  "/test/Method",
			wantKey: "user:user-123",
		},
		{
			name: "with x-forwarded-for",
			setupCtx: func() context.Context {
				md := metadata.Pairs("x-forwarded-for", "192.168.1.1")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			method:  "/test/Method",
			wantKey: "ip:192.168.1.1",
		},
		{
			name: "with x-real-ip",
			setupCtx: func() context.Context {
				md := metadata.Pairs("x-real-ip", "10.0.0.1")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			method:  "/test/Method",
			wantKey: "ip:10.0.0.1",
		},
		{
			name: "no identifiers",
			setupCtx: func() context.Context {
				return context.Background()
			},
			method:  "/test/Method",
			wantKey: "unknown",
		},
		{
			name: "user id takes priority over ip",
			setupCtx: func() context.Context {
				ctx := WithUserID(context.Background(), "user-456")
				md := metadata.Pairs("x-forwarded-for", "192.168.1.1")
				return metadata.NewIncomingContext(ctx, md)
			},
			method:  "/test/Method",
			wantKey: "user:user-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			key := DefaultKeyExtractor(ctx, tt.method)

			if key != tt.wantKey {
				t.Errorf("DefaultKeyExtractor() = %v, want %v", key, tt.wantKey)
			}
		})
	}
}

func TestMethodCategoryExtractor(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		// Optimization
		{"/gateway.v1.GatewayService/SolveGraph", "optimization"},
		{"/gateway.v1.GatewayService/CalculateLogistics", "optimization"},
		{"/gateway.v1.GatewayService/BatchSolve", "optimization"},

		// Validation
		{"/gateway.v1.GatewayService/ValidateGraph", "validation"},

		// Analytics
		{"/gateway.v1.GatewayService/AnalyzeGraph", "analytics"},
		{"/gateway.v1.GatewayService/GetBottlenecks", "analytics"},
		{"/gateway.v1.GatewayService/CompareScenarios", "analytics"},
		{"/gateway.v1.GatewayService/CalculateCost", "analytics"},

		// Simulation
		{"/gateway.v1.GatewayService/RunMonteCarlo", "simulation"},
		{"/gateway.v1.GatewayService/RunWhatIf", "simulation"},
		{"/gateway.v1.GatewayService/AnalyzeResilience", "simulation"},
		{"/gateway.v1.GatewayService/SimulateFailures", "simulation"},
		{"/gateway.v1.GatewayService/FindCriticalElements", "simulation"},
		{"/gateway.v1.GatewayService/AnalyzeSensitivity", "simulation"},

		// History
		{"/gateway.v1.GatewayService/GetCalculation", "history"},
		{"/gateway.v1.GatewayService/ListCalculations", "history"},
		{"/gateway.v1.GatewayService/GetStatistics", "history"},

		// Report
		{"/gateway.v1.GatewayService/GenerateReport", "report"},
		{"/gateway.v1.GatewayService/DownloadReport", "report"},

		// Audit
		{"/gateway.v1.GatewayService/GetAuditLogs", "audit"},

		// Auth
		{"/gateway.v1.GatewayService/Login", "auth"},
		{"/gateway.v1.GatewayService/Register", "auth"},
		{"/gateway.v1.GatewayService/RefreshToken", "auth"},
		{"/gateway.v1.GatewayService/GetProfile", "auth"},

		// General (unknown)
		{"/gateway.v1.GatewayService/Health", "general"},
		{"/gateway.v1.GatewayService/Info", "general"},
		{"/gateway.v1.GatewayService/Unknown", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := MethodCategoryExtractor(tt.method)
			if result != tt.expected {
				t.Errorf("MethodCategoryExtractor(%s) = %v, want %v",
					tt.method, result, tt.expected)
			}
		})
	}
}

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"SolveGraph", "Solve", true},
		{"CalculateCost", "Cost", true},
		{"GetProfile", "Profile", true},
		{"Health", "Solve", false},
		{"", "test", false},
		{"test", "", true},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := containsSubstring(tt.s, tt.substr)
			if result != tt.want {
				t.Errorf("containsSubstring(%s, %s) = %v, want %v",
					tt.s, tt.substr, result, tt.want)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello", "hello", 0},
		{"hello", "ell", 1},
		{"hello", "xyz", -1},
		{"", "test", -1},
		{"test", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := findSubstring(tt.s, tt.substr)
			if result != tt.want {
				t.Errorf("findSubstring(%s, %s) = %v, want %v",
					tt.s, tt.substr, result, tt.want)
			}
		})
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{12345, "12345"},
		{999999, "999999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := formatInt(tt.n)
			if result != tt.want {
				t.Errorf("formatInt(%d) = %v, want %v", tt.n, result, tt.want)
			}
		})
	}
}

func TestRateLimitConfig(t *testing.T) {
	cfg := &RateLimitConfig{
		ExcludeMethods: map[string]bool{
			"/test/Health": true,
		},
		CategoryLimits: map[string]*CategoryLimit{
			"optimization": {Requests: 10, Window: 60},
			"analytics":    {Requests: 50, Window: 60},
		},
	}

	if !cfg.ExcludeMethods["/test/Health"] {
		t.Error("Health should be excluded")
	}

	if cfg.CategoryLimits["optimization"].Requests != 10 {
		t.Error("Optimization limit should be 10")
	}
}
