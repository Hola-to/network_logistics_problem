package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"logistics/gen/openapi"
	"logistics/pkg/audit"
	"logistics/pkg/config"
	"logistics/pkg/interceptors"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/ratelimit"
	"logistics/pkg/swagger"
	"logistics/pkg/telemetry"
)

// GRPCServer обёртка над grpc.Server
type GRPCServer struct {
	server      *grpc.Server
	health      *health.Server
	serviceName string
	config      *config.Config
	telemetry   *telemetry.Provider
	rateLimiter ratelimit.Limiter
	auditLogger audit.Logger
}

// New создаёт новый gRPC сервер
func New(cfg *config.Config) *GRPCServer {
	return NewWithOptions(cfg, nil)
}

// ServerOptions дополнительные опции сервера
type ServerOptions struct {
	RateLimiter         ratelimit.Limiter
	AuditLogger         audit.Logger
	AuditExcludeMethods []string
	KeyExtractor        ratelimit.KeyExtractor
}

// NewWithOptions создаёт сервер с дополнительными опциями
func NewWithOptions(cfg *config.Config, opts *ServerOptions) *GRPCServer {
	if opts == nil {
		opts = &ServerOptions{}
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     cfg.GRPC.KeepAlive.MaxConnectionIdle,
		MaxConnectionAge:      cfg.GRPC.KeepAlive.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.GRPC.KeepAlive.MaxConnectionAgeGrace,
		Time:                  cfg.GRPC.KeepAlive.Time,
		Timeout:               cfg.GRPC.KeepAlive.Timeout,
	}

	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	rateLimiter := opts.RateLimiter
	if rateLimiter == nil && cfg.RateLimit.Enabled {
		var err error
		rateLimiter, err = ratelimit.New(&ratelimit.Config{
			Requests:        cfg.RateLimit.Requests,
			Window:          cfg.RateLimit.Window,
			Strategy:        cfg.RateLimit.Strategy,
			Backend:         cfg.RateLimit.Backend,
			BurstSize:       cfg.RateLimit.BurstSize,
			CleanupInterval: cfg.RateLimit.CleanupInterval,
			RedisAddr:       cfg.RateLimit.RedisAddr,
		})
		if err != nil {
			logger.Log.Warn("Failed to create rate limiter, continuing without it", "error", err)
			rateLimiter = nil
		} else {
			logger.Log.Info("Rate limiter initialized",
				"requests", cfg.RateLimit.Requests,
				"window", cfg.RateLimit.Window,
				"strategy", cfg.RateLimit.Strategy,
			)
		}
	}

	auditLogger := opts.AuditLogger
	if auditLogger == nil && cfg.Audit.Enabled {
		var err error
		auditLogger, err = audit.New(&audit.Config{
			Enabled:         cfg.Audit.Enabled,
			Backend:         cfg.Audit.Backend,
			FilePath:        cfg.Audit.FilePath,
			BufferSize:      cfg.Audit.BufferSize,
			FlushPeriod:     cfg.Audit.FlushPeriod,
			ExcludeMethods:  cfg.Audit.ExcludeMethods,
			IncludeRequest:  cfg.Audit.IncludeRequest,
			IncludeResponse: cfg.Audit.IncludeResponse,
		})
		if err != nil {
			logger.Log.Warn("Failed to create audit logger, continuing without it", "error", err)
			auditLogger = nil
		} else {
			audit.SetGlobal(auditLogger)
			logger.Log.Info("Audit logger initialized", "backend", cfg.Audit.Backend)
		}
	}

	auditExclude := make(map[string]bool)
	for _, method := range opts.AuditExcludeMethods {
		auditExclude[method] = true
	}
	for _, method := range cfg.Audit.ExcludeMethods {
		auditExclude[method] = true
	}
	auditExclude["/grpc.health.v1.Health/Check"] = true
	auditExclude["/grpc.health.v1.Health/Watch"] = true

	interceptorCfg := &interceptors.ServerConfig{
		ServiceName:   cfg.App.Name,
		EnableTracing: cfg.Tracing.Enabled,
		EnableAudit:   cfg.Audit.Enabled && auditLogger != nil,
		RateLimiter:   rateLimiter,
		AuditLogger:   auditLogger,
		AuditExclude:  auditExclude,
		KeyExtractor:  opts.KeyExtractor,
	}

	serverOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.GRPC.MaxSendMsgSize),
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentConn)),
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaPolicy),
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptors(interceptorCfg)),
		grpc.StreamInterceptor(interceptors.StreamServerInterceptors(interceptorCfg)),
	}

	if cfg.GRPC.TLS.Enabled {
		logger.Log.Warn("TLS is enabled but not implemented yet")
	}

	s := grpc.NewServer(serverOpts...)

	h := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, h)

	if cfg.IsDevelopment() {
		reflection.Register(s)
		logger.Log.Debug("gRPC reflection enabled")
	}

	return &GRPCServer{
		server:      s,
		health:      h,
		serviceName: cfg.App.Name,
		config:      cfg,
		rateLimiter: rateLimiter,
		auditLogger: auditLogger,
	}
}

// GetEngine возвращает *grpc.Server для регистрации сервисов
func (s *GRPCServer) GetEngine() *grpc.Server {
	return s.server
}

// GetAuditLogger возвращает audit logger
func (s *GRPCServer) GetAuditLogger() audit.Logger {
	return s.auditLogger
}

// Run запускает сервер
func (s *GRPCServer) Run() error {
	ctx := context.Background()

	if s.config.Tracing.Enabled {
		tp, err := telemetry.Init(ctx, telemetry.Config{
			Enabled:     s.config.Tracing.Enabled,
			Endpoint:    s.config.Tracing.Endpoint,
			ServiceName: s.config.Tracing.ServiceName,
			Version:     s.config.App.Version,
			Environment: s.config.App.Environment,
			SampleRate:  s.config.Tracing.SampleRate,
		})
		if err != nil {
			logger.Log.Warn("Failed to init telemetry", "error", err)
		} else {
			s.telemetry = tp
			logger.Log.Info("Telemetry initialized",
				"endpoint", s.config.Tracing.Endpoint,
				"sample_rate", s.config.Tracing.SampleRate,
			)
		}
	}

	if s.config.Metrics.Enabled {
		go func() {
			logger.Log.Info("Starting metrics server",
				"port", s.config.Metrics.Port,
				"path", s.config.Metrics.Path,
			)
			if err := metrics.StartMetricsServer(s.config.Metrics.Port); err != nil {
				logger.Log.Error("Metrics server failed", "error", err)
			}
		}()
	}

	if s.config.Swagger.Enabled {
		go func() {
			spec, err := openapi.GetSpec()
			if err != nil {
				logger.Log.Error("Failed to load OpenAPI spec", "error", err)
				return
			}

			swaggerCfg := &swagger.Config{
				Title:    s.config.Swagger.Title,
				BasePath: "/swagger",
			}

			server := swagger.NewServer(swaggerCfg, spec)
			if err := server.Start(s.config.Swagger.Port); err != nil {
				logger.Log.Error("Swagger server failed", "error", err)
			}
		}()
		logger.Log.Info("Swagger UI started", "port", s.config.Swagger.Port)
	}

	// Используем ListenConfig с контекстом вместо net.Listen
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.health.SetServingStatus(s.serviceName, grpc_health_v1.HealthCheckResponse_SERVING)

	errCh := make(chan error, 1)

	go func() {
		logger.Log.Info("Starting gRPC server",
			"service", s.serviceName,
			"port", s.config.GRPC.Port,
			"environment", s.config.App.Environment,
			"version", s.config.App.Version,
		)
		if err := s.server.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	if m := metrics.Get(); m != nil {
		m.SetServiceInfo(s.config.App.Version, s.config.App.Environment)
	}

	// Логируем аудит событие старта сервиса
	if s.auditLogger != nil {
		entry := audit.NewEntry().
			Service(s.serviceName).
			Method("server.Start").
			Action(audit.ActionCreate).
			Outcome(audit.OutcomeSuccess).
			Meta("port", s.config.GRPC.Port).
			Meta("version", s.config.App.Version).
			Meta("environment", s.config.App.Environment).
			Build()
		if err := s.auditLogger.Log(ctx, entry); err != nil {
			logger.Log.Warn("Failed to log audit entry", "error", err)
		}
	}

	return s.waitForShutdown(errCh)
}

func (s *GRPCServer) waitForShutdown(errCh chan error) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		logger.Log.Info("Received shutdown signal", "signal", sig)
	}

	// Логируем аудит событие остановки
	if s.auditLogger != nil {
		entry := audit.NewEntry().
			Service(s.serviceName).
			Method("server.Shutdown").
			Action(audit.ActionUpdate).
			Outcome(audit.OutcomeSuccess).
			Meta("reason", "signal").
			Build()
		if err := s.auditLogger.Log(context.Background(), entry); err != nil {
			logger.Log.Warn("Failed to log audit entry", "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.health.SetServingStatus(s.serviceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	if s.telemetry != nil {
		if err := s.telemetry.Shutdown(ctx); err != nil {
			logger.Log.Warn("Failed to shutdown telemetry", "error", err)
		}
	}

	if s.rateLimiter != nil {
		if err := s.rateLimiter.Close(); err != nil {
			logger.Log.Warn("Failed to close rate limiter", "error", err)
		}
	}

	if s.auditLogger != nil {
		if err := s.auditLogger.Close(); err != nil {
			logger.Log.Warn("Failed to close audit logger", "error", err)
		}
	}

	time.Sleep(2 * time.Second)

	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		logger.Log.Info("Server stopped gracefully")
	case <-ctx.Done():
		logger.Log.Warn("Forcing server stop")
		s.server.Stop()
	}

	return nil
}

// SetServingStatus устанавливает статус сервиса
func (s *GRPCServer) SetServingStatus(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.health.SetServingStatus(s.serviceName, status)
}

// Stop останавливает сервер немедленно
func (s *GRPCServer) Stop() {
	s.server.Stop()
}

// GracefulStop останавливает сервер gracefully
func (s *GRPCServer) GracefulStop() {
	s.server.GracefulStop()
}
