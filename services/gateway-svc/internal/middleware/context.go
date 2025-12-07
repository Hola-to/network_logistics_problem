package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"

	authv1 "logistics/gen/go/logistics/auth/v1"
)

// Context keys
type contextKey string

const (
	userIDKey    contextKey = "user_id"
	userInfoKey  contextKey = "user_info"
	requestIDKey contextKey = "request_id"
)

// GetUserID извлекает user_id из контекста
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// GetUserInfo извлекает информацию о пользователе
func GetUserInfo(ctx context.Context) *authv1.UserInfo {
	if v, ok := ctx.Value(userInfoKey).(*authv1.UserInfo); ok {
		return v
	}
	return nil
}

// GetRequestID извлекает request_id из контекста
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// WithUserID добавляет user_id в контекст
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithUserInfo добавляет информацию о пользователе в контекст
func WithUserInfo(ctx context.Context, user *authv1.UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey, user)
}

// WithRequestID добавляет request_id в контекст
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GenerateRequestID генерирует уникальный ID запроса
func GenerateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		// Fallback: return empty string, caller should handle
		return "00000000"
	}
	return hex.EncodeToString(bytes)
}
