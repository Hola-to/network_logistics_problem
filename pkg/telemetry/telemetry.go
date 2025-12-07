package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config конфигурация телеметрии
type Config struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	Version     string
	Environment string
	SampleRate  float64
}

// Provider обёртка над TracerProvider
type Provider struct {
	tp     *sdktrace.TracerProvider
	tracer trace.Tracer
}

var globalProvider *Provider

// Init инициализирует телеметрию
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		// Возвращаем noop provider
		return &Provider{
			tracer: otel.Tracer(cfg.ServiceName),
		}, nil
	}

	// Создаём OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(), // Для dev окружения
	)
	if err != nil {
		return nil, err
	}

	// Resource с метаданными сервиса
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Sampler
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Устанавливаем глобально
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	provider := &Provider{
		tp:     tp,
		tracer: tp.Tracer(cfg.ServiceName),
	}

	globalProvider = provider
	return provider, nil
}

// Shutdown завершает работу телеметрии
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp != nil {
		return p.tp.Shutdown(ctx)
	}
	return nil
}

// Tracer возвращает tracer
func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

// Get возвращает глобальный provider
func Get() *Provider {
	if globalProvider == nil {
		return &Provider{
			tracer: otel.Tracer("default"),
		}
	}
	return globalProvider
}

// StartSpan начинает новый span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Get().tracer.Start(ctx, name, opts...)
}

// SpanFromContext получает span из контекста
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent добавляет событие в текущий span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetError помечает span как ошибочный
func SetError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// RecordError записывает ошибку в span без изменения статуса
// Используется для записи некритичных ошибок
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
}

// SetAttributes устанавливает атрибуты span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// WithAttributes создаёт SpanStartOption с атрибутами
func WithAttributes(attrs ...attribute.KeyValue) trace.SpanStartOption {
	return trace.WithAttributes(attrs...)
}
