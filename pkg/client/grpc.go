package client

import (
	"context"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientConfig struct {
	Address      string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
}

// NewGRPCClient создает соединение с Retry и Timeout
func NewGRPCClient(_ context.Context, cfg ClientConfig) (*grpc.ClientConn, error) {
	opts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(cfg.RetryBackoff)),
		grpc_retry.WithCodes(codes.Unavailable, codes.Aborted, codes.DeadlineExceeded),
		grpc_retry.WithMax(uint(cfg.MaxRetries)),
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpc_retry.UnaryClientInterceptor(opts...),
		),
		grpc.WithChainStreamInterceptor(
			grpc_retry.StreamClientInterceptor(opts...),
		),
	}

	return grpc.NewClient(cfg.Address, dialOpts...)
}
