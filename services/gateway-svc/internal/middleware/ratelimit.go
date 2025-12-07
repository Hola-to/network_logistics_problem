package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"logistics/pkg/logger"
	"logistics/pkg/ratelimit"
	gatewaymetrics "logistics/services/gateway-svc/internal/metrics"
)

// RateLimitConfig конфигурация rate limiting
type RateLimitConfig struct {
	Limiter        ratelimit.Limiter
	KeyExtractor   KeyExtractor
	ExcludeMethods map[string]bool

	// Лимиты по категориям методов
	CategoryLimits map[string]*CategoryLimit
}

// CategoryLimit лимит для категории
type CategoryLimit struct {
	Requests int
	Window   time.Duration
}

// KeyExtractor функция извлечения ключа
type KeyExtractor func(ctx context.Context, method string) string

// DefaultKeyExtractor извлекает ключ по IP и user_id
func DefaultKeyExtractor(ctx context.Context, _ string) string {
	// Сначала пробуем user_id
	if userID := GetUserID(ctx); userID != "" {
		return "user:" + userID
	}

	// Затем IP
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			return "ip:" + xff[0]
		}
		if xri := md.Get("x-real-ip"); len(xri) > 0 {
			return "ip:" + xri[0]
		}
	}

	return "unknown"
}

// MethodCategoryExtractor извлекает категорию метода
func MethodCategoryExtractor(method string) string {
	// Order matters: more specific keywords must come first
	// This ensures "AnalyzeResilience" matches "Resilience" (simulation)
	// before matching "Analyze" (analytics)
	categories := []struct {
		keyword  string
		category string
	}{
		// Simulation - specific keywords that might combine with "Analyze"
		{"MonteCarlo", "simulation"},
		{"WhatIf", "simulation"},
		{"Resilience", "simulation"},
		{"Failure", "simulation"},
		{"Critical", "simulation"},
		{"Sensitivity", "simulation"},
		{"Simulation", "simulation"},

		// Analytics - "Cost" before "Calculate" for "CalculateCost"
		{"Cost", "analytics"},
		{"Bottleneck", "analytics"},
		{"Compare", "analytics"},
		{"Analyze", "analytics"},

		// Optimization
		{"Solve", "optimization"},
		{"Calculate", "optimization"},
		{"Batch", "optimization"},

		// Validation
		{"Validate", "validation"},

		// History
		{"Calculation", "history"},
		{"History", "history"},
		{"Statistics", "history"},

		// Report
		{"Report", "report"},
		{"Download", "report"},

		// Audit
		{"Audit", "audit"},

		// Auth
		{"Login", "auth"},
		{"Register", "auth"},
		{"Token", "auth"},
		{"Profile", "auth"},
		{"Auth", "auth"},
	}

	for _, c := range categories {
		if containsSubstring(method, c.keyword) {
			return c.category
		}
	}
	return "general"
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RateLimitInterceptor создаёт interceptor для rate limiting
func RateLimitInterceptor(cfg *RateLimitConfig) grpc.UnaryServerInterceptor {
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = DefaultKeyExtractor
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Проверяем исключения
		if cfg.ExcludeMethods != nil && cfg.ExcludeMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Получаем ключ
		key := cfg.KeyExtractor(ctx, info.FullMethod)

		// Добавляем категорию к ключу
		category := MethodCategoryExtractor(info.FullMethod)
		fullKey := category + ":" + key

		// Проверяем лимит
		allowed, err := cfg.Limiter.Allow(ctx, fullKey)
		if err != nil {
			logger.Log.Warn("Rate limit check failed", "error", err, "key", fullKey)
			// При ошибке пропускаем (fail open)
			return handler(ctx, req)
		}

		if !allowed {
			gatewaymetrics.Get().RateLimitHits.Inc()

			limitInfo, infoErr := cfg.Limiter.GetInfo(ctx, fullKey)
			if infoErr != nil {
				logger.Log.Warn("Failed to get rate limit info", "error", infoErr, "key", fullKey)
				// Используем дефолтные значения
				limitInfo = &ratelimit.LimitInfo{
					Limit:   0,
					ResetAt: time.Now().Add(time.Minute),
				}
			}

			logger.Log.Warn("Rate limit exceeded",
				"key", fullKey,
				"category", category,
				"limit", limitInfo.Limit,
			)

			// Добавляем заголовки с информацией о лимите
			header := metadata.Pairs(
				"x-ratelimit-limit", formatInt(limitInfo.Limit),
				"x-ratelimit-remaining", "0",
				"x-ratelimit-reset", limitInfo.ResetAt.Format(time.RFC3339),
				"x-ratelimit-category", category,
			)
			if err := grpc.SetHeader(ctx, header); err != nil {
				logger.Log.Warn("Failed to set rate limit headers", "error", err)
			}

			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded for category %s: retry after %v", category, time.Until(limitInfo.ResetAt))
		}

		gatewaymetrics.Get().RateLimitPassed.Inc()

		return handler(ctx, req)
	}
}

// StreamRateLimitInterceptor для streaming
func StreamRateLimitInterceptor(cfg *RateLimitConfig) grpc.StreamServerInterceptor {
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = DefaultKeyExtractor
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if cfg.ExcludeMethods != nil && cfg.ExcludeMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		key := cfg.KeyExtractor(ctx, info.FullMethod)
		category := MethodCategoryExtractor(info.FullMethod)
		fullKey := category + ":" + key

		allowed, err := cfg.Limiter.Allow(ctx, fullKey)
		if err != nil {
			return handler(srv, ss)
		}

		if !allowed {
			gatewaymetrics.Get().RateLimitHits.Inc()
			return status.Errorf(codes.ResourceExhausted, "rate limit exceeded for category %s", category)
		}

		gatewaymetrics.Get().RateLimitPassed.Inc()
		return handler(srv, ss)
	}
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}

	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}
