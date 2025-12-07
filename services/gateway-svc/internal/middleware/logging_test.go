// services/gateway-svc/internal/middleware/logging_test.go

package middleware

import (
	"testing"
)

func TestLoggingInterceptor_Creation(t *testing.T) {
	interceptor := LoggingInterceptor()
	if interceptor == nil {
		t.Error("LoggingInterceptor should not return nil")
	}
}

func TestStreamLoggingInterceptor_Creation(t *testing.T) {
	interceptor := StreamLoggingInterceptor()
	if interceptor == nil {
		t.Error("StreamLoggingInterceptor should not return nil")
	}
}
