package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator интерфейс для валидируемых сообщений
type Validator interface {
	Validate() error
}

// ValidationInterceptor валидирует входящие запросы
func ValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Проверяем, реализует ли запрос интерфейс Validator
		if v, ok := req.(Validator); ok {
			if err := v.Validate(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
			}
		}

		return handler(ctx, req)
	}
}
