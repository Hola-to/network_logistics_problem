package interceptors

import (
	"context"
	"time"

	"logistics/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor логирует gRPC запросы
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// Выполняем handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Извлекаем код статуса
		st, _ := status.FromError(err)
		code := st.Code().String()

		// Логируем
		if err != nil {
			logger.Log.Error("gRPC request failed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"code", code,
				"error", err.Error(),
			)
		} else {
			logger.Log.Info("gRPC request completed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"code", code,
			)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor логирует streaming запросы
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)

		if err != nil {
			logger.Log.Error("gRPC stream failed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"error", err.Error(),
			)
		} else {
			logger.Log.Info("gRPC stream completed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
			)
		}

		return err
	}
}
