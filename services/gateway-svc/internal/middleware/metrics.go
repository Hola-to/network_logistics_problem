package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	gwmetrics "logistics/services/gateway-svc/internal/metrics"
)

// MetricsInterceptor записывает метрики запросов
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)

		// Записываем метрики
		gwmetrics.Get().RecordBackendRequest("gateway", info.FullMethod, st.Code().String(), duration)

		return resp, err
	}
}

// StreamMetricsInterceptor для streaming
func StreamMetricsInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		statusStr := "OK"
		if err != nil {
			st, _ := status.FromError(err)
			statusStr = st.Code().String()
		}

		gwmetrics.Get().RecordBackendRequest("gateway", info.FullMethod, statusStr, duration)

		return err
	}
}
