// services/gateway-svc/internal/middleware/metrics_test.go

package middleware

import (
	"testing"
)

func TestMetricsInterceptor_Creation(t *testing.T) {
	interceptor := MetricsInterceptor()
	if interceptor == nil {
		t.Error("MetricsInterceptor should not return nil")
	}
}

func TestStreamMetricsInterceptor_Creation(t *testing.T) {
	interceptor := StreamMetricsInterceptor()
	if interceptor == nil {
		t.Error("StreamMetricsInterceptor should not return nil")
	}
}
