package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor создаёт gRPC interceptor для трейсинга
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx, span := StartSpan(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Добавляем метод как атрибут
		span.SetAttributes(attribute.String("rpc.method", info.FullMethod))

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

// StreamServerInterceptor создаёт stream interceptor
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx, span := StartSpan(ss.Context(), info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		span.SetAttributes(
			attribute.String("rpc.method", info.FullMethod),
			attribute.Bool("rpc.stream", true),
		)

		// Оборачиваем stream с новым контекстом
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
