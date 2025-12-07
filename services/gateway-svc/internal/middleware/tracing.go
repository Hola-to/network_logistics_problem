package middleware

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const tracerName = "gateway-svc"

// TracingInterceptor создаёт interceptor для tracing
func TracingInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(tracerName)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Извлекаем parent span из metadata
		ctx = extractTraceContext(ctx)

		// Создаём span
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Добавляем атрибуты
		category := MethodCategoryExtractor(info.FullMethod)
		userID := GetUserID(ctx)

		span.SetAttributes(
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.category", category),
			attribute.String("user.id", userID),
		)

		// Выполняем handler
		resp, err := handler(ctx, req)

		if err != nil {
			st, _ := status.FromError(err)
			span.SetStatus(codes.Error, st.Message())
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", st.Code().String()),
			)
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

// StreamTracingInterceptor для streaming
func StreamTracingInterceptor() grpc.StreamServerInterceptor {
	tracer := otel.Tracer(tracerName)

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := extractTraceContext(ss.Context())

		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		span.SetAttributes(
			attribute.String("rpc.method", info.FullMethod),
			attribute.Bool("rpc.stream", true),
		)

		wrappedStream := &tracedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		err := handler(srv, wrappedStream)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}

		return err
	}
}

type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}

func extractTraceContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	// Извлекаем trace headers
	carrier := make(map[string]string)
	for key, values := range md {
		if len(values) > 0 {
			carrier[key] = values[0]
		}
	}

	// Propagator извлечёт trace context
	return otel.GetTextMapPropagator().Extract(ctx, propagatorCarrier(carrier))
}

type propagatorCarrier map[string]string

func (c propagatorCarrier) Get(key string) string {
	return c[key]
}

func (c propagatorCarrier) Set(key, value string) {
	c[key] = value
}

func (c propagatorCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
