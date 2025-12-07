package main

import (
	"context"
	"log"
	"time"

	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/cache"
	"logistics/pkg/config"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/solver-svc/internal/service"
)

func main() {
	cfg, err := config.LoadWithServiceDefaults("solver-svc", 50054)
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

	ctx := context.Background()

	// Инициализация телеметрии
	if cfg.Tracing.Enabled {
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
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tp.Shutdown(shutdownCtx); err != nil {
					logger.Log.Warn("Failed to shutdown telemetry", "error", err)
				}
			}()
			logger.Log.Info("Telemetry initialized", "endpoint", cfg.Tracing.Endpoint)
		}
	}

	metrics.InitMetrics(cfg.Metrics.Namespace, cfg.App.Name)

	// Инициализация кэша для solver
	var solverCache *cache.SolverCache
	if cfg.Cache.Enabled {
		// Создаём опции из конфигурации
		cacheOpts := cache.FromConfig(&cfg.Cache)

		// Создаём базовый кэш
		baseCache, err := cache.New(cacheOpts)
		if err != nil {
			logger.Log.Warn("Failed to create cache, continuing without cache", "error", err)
		} else {
			solverCache = cache.NewSolverCache(baseCache, cfg.Cache.DefaultTTL)
			logger.Log.Info("Solver cache initialized",
				"driver", cfg.Cache.Driver,
				"ttl", cfg.Cache.DefaultTTL,
			)
		}
	}

	srv := server.New(cfg)

	solverService := service.NewSolverService(cfg.App.Version, solverCache)
	optimizationv1.RegisterSolverServiceServer(srv.GetEngine(), solverService)

	logger.Info("Starting solver service",
		"port", cfg.GRPC.Port,
		"environment", cfg.App.Environment,
		"version", cfg.App.Version,
		"cache_enabled", solverCache != nil,
	)

	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}
