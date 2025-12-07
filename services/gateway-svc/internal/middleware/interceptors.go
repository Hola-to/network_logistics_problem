package middleware

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"logistics/pkg/config"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/ratelimit"
	"logistics/services/gateway-svc/internal/clients"
)

// LoggingInterceptor логирует запросы
func NewLoggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Генерируем request ID
			requestID := GenerateRequestID()
			ctx = WithRequestID(ctx, requestID)

			start := time.Now()
			procedure := req.Spec().Procedure

			resp, err := next(ctx, req)

			duration := time.Since(start)

			if err != nil {
				logger.Log.Error("Request failed",
					"request_id", requestID,
					"method", procedure,
					"duration_ms", duration.Milliseconds(),
					"error", err,
				)
			} else {
				logger.Log.Info("Request completed",
					"request_id", requestID,
					"method", procedure,
					"duration_ms", duration.Milliseconds(),
				)
			}

			return resp, err
		}
	}
}

// AuthInterceptor проверяет авторизацию
func NewAuthInterceptor(authClient *clients.AuthClient) connect.UnaryInterceptorFunc {
	// Публичные методы без авторизации
	publicMethods := map[string]bool{
		"/logistics.gateway.v1.GatewayService/Health":         true,
		"/logistics.gateway.v1.GatewayService/ReadinessCheck": true,
		"/logistics.gateway.v1.GatewayService/Info":           true,
		"/logistics.gateway.v1.GatewayService/GetAlgorithms":  true,
		"/logistics.gateway.v1.GatewayService/Login":          true,
		"/logistics.gateway.v1.GatewayService/Register":       true,
		"/logistics.gateway.v1.GatewayService/RefreshToken":   true,
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Пропускаем публичные методы
			if publicMethods[procedure] {
				return next(ctx, req)
			}

			// Извлекаем токен
			token := req.Header().Get("Authorization")
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing authorization header"))
			}

			// Убираем "Bearer " prefix
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}

			// Валидируем токен
			resp, err := authClient.ValidateToken(ctx, token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("token validation failed"))
			}

			if !resp.Valid {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
			}

			// Добавляем user info в контекст
			ctx = WithUserID(ctx, resp.UserId)
			ctx = WithUserInfo(ctx, resp.User)

			return next(ctx, req)
		}
	}
}

// RateLimitInterceptor ограничивает частоту запросов
func NewRateLimitInterceptor(cfg config.RateLimitConfig) connect.UnaryInterceptorFunc {
	if !cfg.Enabled {
		return func(next connect.UnaryFunc) connect.UnaryFunc {
			return next
		}
	}

	limiter, err := ratelimit.New(&ratelimit.Config{
		Requests:  cfg.Requests,
		Window:    cfg.Window,
		Strategy:  cfg.Strategy,
		Backend:   cfg.Backend,
		BurstSize: cfg.BurstSize,
	})
	if err != nil {
		logger.Log.Warn("Failed to create rate limiter", "error", err)
		return func(next connect.UnaryFunc) connect.UnaryFunc {
			return next
		}
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Ключ по user_id или IP
			key := GetUserID(ctx)
			if key == "" {
				key = req.Peer().Addr
			}

			allowed, err := limiter.Allow(ctx, key)
			if err != nil {
				logger.Log.Warn("Rate limit check failed", "error", err)
				return next(ctx, req)
			}

			if !allowed {
				return nil, connect.NewError(
					connect.CodeResourceExhausted,
					fmt.Errorf("rate limit exceeded"),
				)
			}

			return next(ctx, req)
		}
	}
}

// MetricsInterceptor собирает метрики
func NewMetricsInterceptor() connect.UnaryInterceptorFunc {
	m := metrics.Get()

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()

			resp, err := next(ctx, req)

			duration := time.Since(start)
			status := "OK"
			if err != nil {
				status = connect.CodeOf(err).String()
			}

			m.RecordGRPCRequest(req.Spec().Procedure, status, duration)

			return resp, err
		}
	}
}

// NewStreamLoggingInterceptor для streaming
func NewStreamLoggingInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			requestID := GenerateRequestID()
			ctx = WithRequestID(ctx, requestID)

			resp, err := next(ctx, req)

			logger.Log.Info("Stream/Unary completed",
				"request_id", requestID,
				"method", req.Spec().Procedure,
				"duration_ms", time.Since(start).Milliseconds(),
			)

			return resp, err
		}
	})
}
