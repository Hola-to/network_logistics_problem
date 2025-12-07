package interceptors

import (
	"context"
	"time"

	"logistics/pkg/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// MetricsInterceptor записывает метрики запросов
func MetricsInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	m := metrics.Get()
	tracker := metrics.NewRequestTracker(m.GRPCRequestsInFlight)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tracker.Start(info.FullMethod)
		defer tracker.End(info.FullMethod)

		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		st, _ := status.FromError(err)
		statusStr := st.Code().String()

		m.RecordGRPCRequest(info.FullMethod, statusStr, duration)

		return resp, err
	}
}

// StreamMetricsInterceptor записывает метрики stream запросов
func StreamMetricsInterceptor(serviceName string) grpc.StreamServerInterceptor {
	m := metrics.Get()
	tracker := metrics.NewRequestTracker(m.GRPCRequestsInFlight)

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		tracker.Start(info.FullMethod)
		defer tracker.End(info.FullMethod)

		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)

		statusStr := "OK"
		if err != nil {
			st, _ := status.FromError(err)
			statusStr = st.Code().String()
		}

		m.RecordGRPCRequest(info.FullMethod, statusStr, duration)

		return err
	}
}
