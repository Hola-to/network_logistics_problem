//go:build integration

package pkg_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"logistics/pkg/config"
	"logistics/pkg/server"
	"logistics/tests/integration/testutil"
)

func TestGRPCServer_StartStop(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	port := testutil.FreePort(t)

	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test-server",
			Version:     "1.0.0",
			Environment: "test",
		},
		GRPC: config.GRPCConfig{
			Port:           port,
			MaxRecvMsgSize: 4 * 1024 * 1024,
			MaxSendMsgSize: 4 * 1024 * 1024,
			KeepAlive: config.KeepAliveConfig{
				MaxConnectionIdle: 5 * time.Minute,
				Time:              1 * time.Minute,
				Timeout:           20 * time.Second,
			},
		},
		Metrics:   config.MetricsConfig{Enabled: false},
		Tracing:   config.TracingConfig{Enabled: false},
		Swagger:   config.SwaggerConfig{Enabled: false},
		Audit:     config.AuditConfig{Enabled: false},
		RateLimit: config.RateLimitConfig{Enabled: false},
	}

	srv := server.New(cfg)

	// Start in background
	go func() {
		_ = srv.Run()
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Connect using NewClient (non-deprecated)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer conn.Close()

	// Check health
	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "test-server",
	})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("status = %v, want SERVING", resp.Status)
	}

	// Stop
	srv.GracefulStop()
}

func TestGRPCServer_HealthCheck(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	port := testutil.FreePort(t)

	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "health-test",
			Version:     "1.0.0",
			Environment: "test",
		},
		GRPC: config.GRPCConfig{
			Port:           port,
			MaxRecvMsgSize: 4 * 1024 * 1024,
			MaxSendMsgSize: 4 * 1024 * 1024,
			KeepAlive: config.KeepAliveConfig{
				MaxConnectionIdle: 5 * time.Minute,
				Time:              1 * time.Minute,
				Timeout:           20 * time.Second,
			},
		},
		Metrics:   config.MetricsConfig{Enabled: false},
		Tracing:   config.TracingConfig{Enabled: false},
		Swagger:   config.SwaggerConfig{Enabled: false},
		Audit:     config.AuditConfig{Enabled: false},
		RateLimit: config.RateLimitConfig{Enabled: false},
	}

	srv := server.New(cfg)

	go func() {
		_ = srv.Run()
	}()
	defer srv.GracefulStop()

	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Watch health
	stream, err := healthClient.Watch(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "health-test",
	})
	if err != nil {
		t.Fatalf("watch failed: %v", err)
	}

	// Should receive initial status
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("initial status = %v, want SERVING", resp.Status)
	}
}

func TestGRPCServer_WithRateLimit(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	addr := testutil.RequireRedis(t)
	port := testutil.FreePort(t)

	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "ratelimit-test",
			Version:     "1.0.0",
			Environment: "test",
		},
		GRPC: config.GRPCConfig{
			Port:           port,
			MaxRecvMsgSize: 4 * 1024 * 1024,
			MaxSendMsgSize: 4 * 1024 * 1024,
			KeepAlive: config.KeepAliveConfig{
				MaxConnectionIdle: 5 * time.Minute,
				Time:              1 * time.Minute,
				Timeout:           20 * time.Second,
			},
		},
		Metrics: config.MetricsConfig{Enabled: false},
		Tracing: config.TracingConfig{Enabled: false},
		Swagger: config.SwaggerConfig{Enabled: false},
		Audit:   config.AuditConfig{Enabled: false},
		RateLimit: config.RateLimitConfig{
			Enabled:   true,
			Requests:  100,
			Window:    time.Minute,
			Backend:   "redis",
			RedisAddr: addr,
		},
	}

	srv := server.New(cfg)

	go func() {
		srv.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Server should be running with rate limiting
	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("status = %v, want SERVING", resp.Status)
	}
}
