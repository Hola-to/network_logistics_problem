package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"logistics/pkg/audit"
	"logistics/pkg/logger"
)

// AuditConfig конфигурация аудит интерсептора
type AuditConfig struct {
	ServiceName    string
	ExcludeMethods map[string]bool
	Logger         audit.Logger
}

// AuditInterceptor создаёт интерсептор для аудит логирования
func AuditInterceptor(cfg *AuditConfig) grpc.UnaryServerInterceptor {
	if cfg.Logger == nil {
		cfg.Logger = audit.Get()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Пропускаем исключённые методы
		if cfg.ExcludeMethods != nil && cfg.ExcludeMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		start := time.Now()

		// Извлекаем информацию о клиенте
		clientIP := extractClientIP(ctx)
		userID, username := extractUserInfo(ctx)
		requestID := extractRequestID(ctx)

		// Выполняем handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Строим аудит запись
		builder := audit.NewEntry().
			Service(cfg.ServiceName).
			Method(info.FullMethod).
			Action(methodToAction(info.FullMethod)).
			User(userID, username).
			Client(clientIP, "").
			RequestID(requestID).
			Duration(duration)

		if err != nil {
			st, _ := status.FromError(err)
			builder.Outcome(audit.OutcomeFailure).
				Error(st.Code().String(), st.Message())
		} else {
			builder.Outcome(audit.OutcomeSuccess)
		}

		entry := builder.Build()

		// Асинхронно логируем
		go func() {
			if logErr := cfg.Logger.Log(context.Background(), entry); logErr != nil {
				logger.Log.Warn("Failed to write audit log", "error", logErr)
			}
		}()

		return resp, err
	}
}

// StreamAuditInterceptor для streaming
func StreamAuditInterceptor(cfg *AuditConfig) grpc.StreamServerInterceptor {
	if cfg.Logger == nil {
		cfg.Logger = audit.Get()
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if cfg.ExcludeMethods != nil && cfg.ExcludeMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		start := time.Now()
		ctx := ss.Context()

		clientIP := extractClientIP(ctx)
		userID, username := extractUserInfo(ctx)
		requestID := extractRequestID(ctx)

		err := handler(srv, ss)

		duration := time.Since(start)

		builder := audit.NewEntry().
			Service(cfg.ServiceName).
			Method(info.FullMethod).
			Action(audit.ActionRead).
			User(userID, username).
			Client(clientIP, "").
			RequestID(requestID).
			Duration(duration).
			Meta("stream", true)

		if err != nil {
			st, _ := status.FromError(err)
			builder.Outcome(audit.OutcomeFailure).
				Error(st.Code().String(), st.Message())
		} else {
			builder.Outcome(audit.OutcomeSuccess)
		}

		go func() {
			if logErr := cfg.Logger.Log(context.Background(), builder.Build()); logErr != nil {
				logger.Log.Warn("Failed to write audit log", "error", logErr)
			}
		}()

		return err
	}
}

func extractClientIP(ctx context.Context) string {
	// Из metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			return xff[0]
		}
		if xri := md.Get("x-real-ip"); len(xri) > 0 {
			return xri[0]
		}
	}

	// Из peer
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	return "unknown"
}

func extractUserInfo(ctx context.Context) (userID, username string) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if uid := md.Get("x-user-id"); len(uid) > 0 {
			userID = uid[0]
		}
		if uname := md.Get("x-username"); len(uname) > 0 {
			username = uname[0]
		}
	}
	return
}

func extractRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if rid := md.Get("x-request-id"); len(rid) > 0 {
			return rid[0]
		}
	}
	return ""
}

func methodToAction(method string) audit.Action {
	// Простое определение действия по имени метода
	switch {
	case contains(method, "Create") || contains(method, "Save") || contains(method, "Register"):
		return audit.ActionCreate
	case contains(method, "Get") || contains(method, "List") || contains(method, "Find"):
		return audit.ActionRead
	case contains(method, "Update") || contains(method, "Refresh"):
		return audit.ActionUpdate
	case contains(method, "Delete") || contains(method, "Remove"):
		return audit.ActionDelete
	case contains(method, "Login"):
		return audit.ActionLogin
	case contains(method, "Logout"):
		return audit.ActionLogout
	case contains(method, "Solve"):
		return audit.ActionSolve
	case contains(method, "Analyze"):
		return audit.ActionAnalyze
	default:
		return audit.ActionRead
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
