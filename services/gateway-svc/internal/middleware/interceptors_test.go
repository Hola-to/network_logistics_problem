// services/gateway-svc/internal/middleware/interceptors_test.go

package middleware

import (
	"testing"

	"logistics/pkg/config"
)

func TestNewLoggingInterceptor_Creation(t *testing.T) {
	interceptor := NewLoggingInterceptor()
	if interceptor == nil {
		t.Error("NewLoggingInterceptor should not return nil")
	}
}

func TestNewMetricsInterceptor_Creation(t *testing.T) {
	interceptor := NewMetricsInterceptor()
	if interceptor == nil {
		t.Error("NewMetricsInterceptor should not return nil")
	}
}

func TestNewRateLimitInterceptor_Disabled(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled: false,
	}

	interceptor := NewRateLimitInterceptor(cfg)
	if interceptor == nil {
		t.Error("NewRateLimitInterceptor should not return nil even when disabled")
	}
}

func TestNewStreamLoggingInterceptor_Creation(t *testing.T) {
	interceptor := NewStreamLoggingInterceptor()
	if interceptor == nil {
		t.Error("NewStreamLoggingInterceptor should not return nil")
	}
}
