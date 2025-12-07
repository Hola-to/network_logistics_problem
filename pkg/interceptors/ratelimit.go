package interceptors

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"logistics/pkg/logger"
	"logistics/pkg/ratelimit"
)

// RateLimitInterceptor создаёт интерсептор для rate limiting
func RateLimitInterceptor(limiter ratelimit.Limiter, keyExtractor ratelimit.KeyExtractor) grpc.UnaryServerInterceptor {
	if keyExtractor == nil {
		keyExtractor = ratelimit.DefaultKeyExtractor
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Извлекаем метаданные
		md, _ := metadata.FromIncomingContext(ctx)
		metadataMap := make(map[string]string)
		for k, v := range md {
			if len(v) > 0 {
				metadataMap[k] = v[0]
			}
		}

		// Получаем ключ
		key := keyExtractor(ctx, info.FullMethod, metadataMap)

		// Проверяем лимит
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			logger.Log.Warn("Rate limit check failed", "error", err, "key", key)
			// При ошибке пропускаем (fail open)
			return handler(ctx, req)
		}

		if !allowed {
			limitInfo, infoErr := limiter.GetInfo(ctx, key)
			if infoErr != nil {
				logger.Log.Warn("Failed to get rate limit info", "error", infoErr, "key", key)
				limitInfo = &ratelimit.LimitInfo{
					Limit:   0,
					ResetAt: time.Now().Add(time.Minute),
				}
			}

			logger.Log.Warn("Rate limit exceeded",
				"key", key,
				"limit", limitInfo.Limit,
			)

			// Добавляем заголовки с информацией о лимите
			header := metadata.Pairs(
				"x-ratelimit-limit", fmt.Sprintf("%d", limitInfo.Limit),
				"x-ratelimit-remaining", "0",
				"x-ratelimit-reset", limitInfo.ResetAt.Format(time.RFC3339),
			)
			if err := grpc.SetHeader(ctx, header); err != nil {
				logger.Log.Debug("Failed to set rate limit headers", "error", err)
			}

			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded: %d requests per %v", limitInfo.Limit, time.Until(limitInfo.ResetAt))
		}

		return handler(ctx, req)
	}
}

// StreamRateLimitInterceptor для streaming
func StreamRateLimitInterceptor(limiter ratelimit.Limiter, keyExtractor ratelimit.KeyExtractor) grpc.StreamServerInterceptor {
	if keyExtractor == nil {
		keyExtractor = ratelimit.DefaultKeyExtractor
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		md, _ := metadata.FromIncomingContext(ctx)
		metadataMap := make(map[string]string)
		for k, v := range md {
			if len(v) > 0 {
				metadataMap[k] = v[0]
			}
		}

		key := keyExtractor(ctx, info.FullMethod, metadataMap)

		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			return handler(srv, ss)
		}

		if !allowed {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}
