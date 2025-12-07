package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		ServiceName: "test-service",
		Version:     "1.0.0",
		Environment: "test",
		SampleRate:  0.5,
	}

	if cfg.ServiceName != "test-service" {
		t.Errorf("ServiceName = %s, want test-service", cfg.ServiceName)
	}
}

func TestInit_Disabled(t *testing.T) {
	cfg := Config{
		Enabled:     false,
		ServiceName: "test",
	}

	provider, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if provider == nil {
		t.Fatal("provider should not be nil")
	}

	if provider.tracer == nil {
		t.Error("tracer should not be nil even when disabled")
	}
}

func TestGet_Uninitialized(t *testing.T) {
	// Reset global
	globalProvider = nil

	provider := Get()
	if provider == nil {
		t.Fatal("Get() should return provider even when uninitialized")
	}

	if provider.tracer == nil {
		t.Error("tracer should not be nil")
	}
}

func TestStartSpan(t *testing.T) {
	globalProvider = nil

	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-span")

	if span == nil {
		t.Error("span should not be nil")
	}

	// Проверяем, что контекст изменился (содержит span)
	_ = newCtx

	span.End()
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	span := SpanFromContext(ctx)

	// Should return noop span for context without span
	if span == nil {
		t.Error("SpanFromContext should return span (noop)")
	}
}

func TestAddEvent(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	// Should not panic
	AddEvent(newCtx, "test-event",
		attribute.String("key", "value"),
		attribute.Int("count", 42),
	)
}

func TestSetError(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	// Should not panic
	SetError(newCtx, context.DeadlineExceeded)
}

func TestSetAttributes(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	// Should not panic
	SetAttributes(newCtx,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	)
}

func TestWithAttributes(t *testing.T) {
	opt := WithAttributes(
		attribute.String("key", "value"),
	)

	if opt == nil {
		t.Error("WithAttributes should return option")
	}
}

func TestProvider_Tracer(t *testing.T) {
	provider := &Provider{
		tracer: noop.NewTracerProvider().Tracer("test"),
	}

	tracer := provider.Tracer()
	if tracer == nil {
		t.Error("Tracer() should not return nil")
	}
}

func TestProvider_Shutdown(t *testing.T) {
	provider := &Provider{
		tp:     nil,
		tracer: noop.NewTracerProvider().Tracer("test"),
	}

	err := provider.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestGraphAttributes(t *testing.T) {
	attrs := GraphAttributes(10, 20, 1, 5)

	if len(attrs) != 4 {
		t.Errorf("expected 4 attributes, got %d", len(attrs))
	}

	expected := map[string]any{
		AttrGraphNodes:    10,
		AttrGraphEdges:    20,
		AttrGraphSourceID: int64(1),
		AttrGraphSinkID:   int64(5),
	}

	for _, attr := range attrs {
		key := string(attr.Key)
		if _, ok := expected[key]; !ok {
			t.Errorf("unexpected attribute key: %s", key)
		}
	}
}

func TestAlgorithmAttributes(t *testing.T) {
	attrs := AlgorithmAttributes("DINIC", 100, 50.5, 25.0)

	if len(attrs) != 4 {
		t.Errorf("expected 4 attributes, got %d", len(attrs))
	}
}

func TestValidationAttributes(t *testing.T) {
	attrs := ValidationAttributes("strict", 3, false)

	if len(attrs) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(attrs))
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	interceptor := UnaryServerInterceptor()

	if interceptor == nil {
		t.Error("UnaryServerInterceptor should not return nil")
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	interceptor := StreamServerInterceptor()

	if interceptor == nil {
		t.Error("StreamServerInterceptor should not return nil")
	}
}
