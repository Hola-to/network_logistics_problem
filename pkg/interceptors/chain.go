package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

// chainUnaryInterceptors объединяет unary интерсепторы
func chainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildUnaryChain(interceptors[i], chain, info)
		}
		return chain(ctx, req)
	}
}

func buildUnaryChain(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler, info *grpc.UnaryServerInfo) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		return current(ctx, req, info, next)
	}
}

// chainStreamInterceptors объединяет stream интерсепторы
func chainStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildStreamChain(interceptors[i], chain, info)
		}
		return chain(srv, ss)
	}
}

func buildStreamChain(current grpc.StreamServerInterceptor, next grpc.StreamHandler, info *grpc.StreamServerInfo) grpc.StreamHandler {
	return func(srv any, ss grpc.ServerStream) error {
		return current(srv, ss, info, next)
	}
}
