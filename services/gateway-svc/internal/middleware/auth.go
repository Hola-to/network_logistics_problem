package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "logistics/gen/go/logistics/auth/v1"
	"logistics/pkg/logger"
	gatewaymetrics "logistics/services/gateway-svc/internal/metrics"
)

// AuthClient интерфейс для auth клиента
type AuthClient interface {
	ValidateToken(ctx context.Context, token string) (*authv1.ValidateTokenResponse, error)
}

// AuthConfig конфигурация auth middleware
type AuthConfig struct {
	Client        AuthClient
	PublicMethods map[string]bool
}

// AuthInterceptor создаёт interceptor для проверки авторизации
func AuthInterceptor(cfg *AuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Проверяем, является ли метод публичным
		if cfg.PublicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Извлекаем токен
		token, err := extractToken(ctx)
		if err != nil {
			gatewaymetrics.Get().AuthFailed.Inc()
			return nil, err
		}

		// Валидируем токен
		resp, err := cfg.Client.ValidateToken(ctx, token)
		if err != nil {
			gatewaymetrics.Get().AuthFailed.Inc()
			logger.Log.Warn("Token validation failed", "error", err)
			return nil, status.Error(codes.Unauthenticated, "failed to validate token")
		}

		if !resp.Valid {
			gatewaymetrics.Get().AuthFailed.Inc()
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		gatewaymetrics.Get().AuthSuccessful.Inc()

		// Добавляем информацию о пользователе в контекст
		ctx = context.WithValue(ctx, userIDKey, resp.UserId)
		ctx = context.WithValue(ctx, userInfoKey, resp.User)

		return handler(ctx, req)
	}
}

// StreamAuthInterceptor для streaming методов
func StreamAuthInterceptor(cfg *AuthConfig) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if cfg.PublicMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		token, err := extractToken(ctx)
		if err != nil {
			gatewaymetrics.Get().AuthFailed.Inc()
			return err
		}

		resp, err := cfg.Client.ValidateToken(ctx, token)
		if err != nil {
			gatewaymetrics.Get().AuthFailed.Inc()
			return status.Error(codes.Unauthenticated, "failed to validate token")
		}

		if !resp.Valid {
			gatewaymetrics.Get().AuthFailed.Inc()
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		gatewaymetrics.Get().AuthSuccessful.Inc()

		wrappedStream := &authServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(context.WithValue(ctx, userIDKey, resp.UserId), userInfoKey, resp.User),
		}

		return handler(srv, wrappedStream)
	}
}

type authServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authServerStream) Context() context.Context {
	return s.ctx
}

func extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "no authorization header")
	}

	token := values[0]
	token = strings.TrimPrefix(token, "Bearer ")

	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty token")
	}

	return token, nil
}

// PublicMethods возвращает список публичных методов
func PublicMethods() map[string]bool {
	return map[string]bool{
		"/logistics.gateway.v1.GatewayService/Health":         true,
		"/logistics.gateway.v1.GatewayService/ReadinessCheck": true,
		"/logistics.gateway.v1.GatewayService/Info":           true,
		"/logistics.gateway.v1.GatewayService/GetAlgorithms":  true,
		"/logistics.gateway.v1.GatewayService/Login":          true,
		"/logistics.gateway.v1.GatewayService/Register":       true,
		"/logistics.gateway.v1.GatewayService/RefreshToken":   true,
		"/grpc.health.v1.Health/Check":                        true,
		"/grpc.health.v1.Health/Watch":                        true,
	}
}
