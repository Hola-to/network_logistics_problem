package main

import (
	"context"
	"log"
	"time"

	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/pkg/config"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/validation-svc/internal/service"
)

func main() {
	cfg, err := config.LoadWithServiceDefaults("validation-svc", 50054)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger.InitWithConfig(logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		FilePath:   cfg.Log.FilePath,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})

	// Инициализация телеметрии
	if cfg.Tracing.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tp, err := telemetry.Init(ctx, telemetry.Config{
			Enabled:     cfg.Tracing.Enabled,
			Endpoint:    cfg.Tracing.Endpoint,
			ServiceName: cfg.App.Name,
			Version:     cfg.App.Version,
			Environment: cfg.App.Environment,
			SampleRate:  cfg.Tracing.SampleRate,
		})
		if err != nil {
			logger.Log.Warn("Failed to init telemetry", "error", err)
		} else {
			defer func() {
				if err := tp.Shutdown(context.Background()); err != nil {
					logger.Log.Warn("Failed to shutdown telemetry", "error", err)
				}
			}()
			logger.Log.Info("Telemetry initialized", "endpoint", cfg.Tracing.Endpoint)
		}
	}

	metrics.InitMetrics(cfg.Metrics.Namespace, cfg.App.Name)

	srv := server.New(cfg)

	impl := service.NewValidationService(cfg.App.Version)
	validationv1.RegisterValidationServiceServer(srv.GetEngine(), impl)

	logger.Info("Starting validation service",
		"port", cfg.GRPC.Port,
		"environment", cfg.App.Environment,
		"version", cfg.App.Version,
		"tracing_enabled", cfg.Tracing.Enabled,
	)

	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}
