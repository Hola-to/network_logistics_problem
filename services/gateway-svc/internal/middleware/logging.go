package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"logistics/pkg/logger"
)

// LoggingInterceptor логирует запросы с дополнительной информацией
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// Извлекаем user_id если есть
		userID := GetUserID(ctx)

		// Выполняем handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)

		// Логируем
		logFields := []any{
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"code", st.Code().String(),
		}

		if userID != "" {
			logFields = append(logFields, "user_id", userID)
		}

		if err != nil {
			logFields = append(logFields, "error", err.Error())
			logger.Log.Error("Gateway request failed", logFields...)
		} else {
			logger.Log.Info("Gateway request completed", logFields...)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor для streaming
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		userID := GetUserID(ss.Context())

		err := handler(srv, ss)

		duration := time.Since(start)

		logFields := []any{
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"stream", true,
		}

		if userID != "" {
			logFields = append(logFields, "user_id", userID)
		}

		if err != nil {
			logFields = append(logFields, "error", err.Error())
			logger.Log.Error("Gateway stream failed", logFields...)
		} else {
			logger.Log.Info("Gateway stream completed", logFields...)
		}

		return err
	}
}
