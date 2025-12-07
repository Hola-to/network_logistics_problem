// services/gateway-svc/internal/middleware/tracing_test.go

package middleware

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestPropagatorCarrier(t *testing.T) {
	carrier := propagatorCarrier{
		"traceparent": "00-trace-id-span-id-01",
		"tracestate":  "key=value",
	}

	// Test Get
	if carrier.Get("traceparent") != "00-trace-id-span-id-01" {
		t.Error("Get should return correct value")
	}

	if carrier.Get("nonexistent") != "" {
		t.Error("Get for nonexistent key should return empty string")
	}

	// Test Set
	carrier.Set("newkey", "newvalue")
	if carrier.Get("newkey") != "newvalue" {
		t.Error("Set should add new key")
	}

	// Test Keys
	keys := carrier.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() length = %d, want 3", len(keys))
	}
}

func TestExtractTraceContext(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
	}{
		{
			name: "with trace headers",
			setupCtx: func() context.Context {
				md := metadata.Pairs(
					"traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
				)
				return metadata.NewIncomingContext(context.Background(), md)
			},
		},
		{
			name: "without metadata",
			setupCtx: func() context.Context {
				return context.Background()
			},
		},
		{
			name: "with empty metadata",
			setupCtx: func() context.Context {
				md := metadata.New(nil)
				return metadata.NewIncomingContext(context.Background(), md)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			// Should not panic
			result := extractTraceContext(ctx)

			if result == nil {
				t.Error("extractTraceContext should not return nil")
			}
		})
	}
}

func TestTracedServerStream(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "trace-123")

	stream := &tracedServerStream{
		ctx: ctx,
	}

	resultCtx := stream.Context()

	if GetRequestID(resultCtx) != "trace-123" {
		t.Error("tracedServerStream.Context() should preserve context values")
	}
}
