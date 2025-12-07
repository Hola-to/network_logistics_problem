package interceptors

import (
	"google.golang.org/grpc"

	"logistics/pkg/audit"
	"logistics/pkg/ratelimit"
	"logistics/pkg/telemetry"
)

// ServerConfig конфигурация серверных интерсепторов
type ServerConfig struct {
	ServiceName   string
	EnableTracing bool
	EnableAudit   bool
	RateLimiter   ratelimit.Limiter
	AuditLogger   audit.Logger
	AuditExclude  map[string]bool
	KeyExtractor  ratelimit.KeyExtractor
}

// UnaryServerInterceptors возвращает цепочку unary интерсепторов
func UnaryServerInterceptors(cfg *ServerConfig) grpc.UnaryServerInterceptor {
	interceptors := []grpc.UnaryServerInterceptor{
		RecoveryInterceptor(),
	}

	// Rate Limiting (первым после recovery)
	if cfg.RateLimiter != nil {
		interceptors = append(interceptors, RateLimitInterceptor(cfg.RateLimiter, cfg.KeyExtractor))
	}

	// Tracing
	if cfg.EnableTracing {
		interceptors = append(interceptors, telemetry.UnaryServerInterceptor())
	}

	// Metrics
	interceptors = append(interceptors, MetricsInterceptor(cfg.ServiceName))

	// Logging
	interceptors = append(interceptors, LoggingInterceptor())

	// Validation
	interceptors = append(interceptors, ValidationInterceptor())

	// Audit (последним, чтобы логировать результат)
	if cfg.EnableAudit && cfg.AuditLogger != nil {
		interceptors = append(interceptors, AuditInterceptor(&AuditConfig{
			ServiceName:    cfg.ServiceName,
			ExcludeMethods: cfg.AuditExclude,
			Logger:         cfg.AuditLogger,
		}))
	}

	return chainUnaryInterceptors(interceptors...)
}

// StreamServerInterceptors возвращает цепочку stream интерсепторов
func StreamServerInterceptors(cfg *ServerConfig) grpc.StreamServerInterceptor {
	interceptors := []grpc.StreamServerInterceptor{
		StreamRecoveryInterceptor(),
	}

	// Rate Limiting
	if cfg.RateLimiter != nil {
		interceptors = append(interceptors, StreamRateLimitInterceptor(cfg.RateLimiter, cfg.KeyExtractor))
	}

	// Tracing
	if cfg.EnableTracing {
		interceptors = append(interceptors, telemetry.StreamServerInterceptor())
	}

	// Metrics & Logging
	interceptors = append(interceptors,
		StreamMetricsInterceptor(cfg.ServiceName),
		StreamLoggingInterceptor(),
	)

	// Audit
	if cfg.EnableAudit && cfg.AuditLogger != nil {
		interceptors = append(interceptors, StreamAuditInterceptor(&AuditConfig{
			ServiceName:    cfg.ServiceName,
			ExcludeMethods: cfg.AuditExclude,
			Logger:         cfg.AuditLogger,
		}))
	}

	return chainStreamInterceptors(interceptors...)
}

// Legacy functions for backward compatibility

func UnaryServerInterceptorsLegacy(serviceName string, enableTracing bool) grpc.UnaryServerInterceptor {
	return UnaryServerInterceptors(&ServerConfig{
		ServiceName:   serviceName,
		EnableTracing: enableTracing,
	})
}

func StreamServerInterceptorsLegacy(serviceName string, enableTracing bool) grpc.StreamServerInterceptor {
	return StreamServerInterceptors(&ServerConfig{
		ServiceName:   serviceName,
		EnableTracing: enableTracing,
	})
}
