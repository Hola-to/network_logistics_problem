package main

import (
	"context"
	"log"

	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/migrations"
	"logistics/pkg/client"
	"logistics/pkg/config"
	"logistics/pkg/database"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/simulation-svc/internal/repository"
	"logistics/services/simulation-svc/internal/service"
)

func main() {
	// Загружаем конфиг с дефолтным портом 50058
	cfg, err := config.LoadWithServiceDefaults("simulation-svc", 50058)
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

	// Телеметрия
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
				if err := tp.Shutdown(context.Background()); err != nil {
					logger.Log.Warn("Failed to shutdown telemetry", "error", err)
				}
			}()
			logger.Log.Info("Telemetry initialized", "endpoint", cfg.Tracing.Endpoint)
		}
	}

	metrics.InitMetrics(cfg.Metrics.Namespace, cfg.App.Name)

	// База данных
	db, err := database.NewPostgresDB(ctx, &cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to database", "error", err)
	}
	defer db.Close()

	// Миграции
	if cfg.Database.AutoMigrate {
		if err := database.RunMigrations(
			ctx,
			db.Pool(),
			&cfg.Database,
			migrations.PostgresMigrations,
			"postgres",
		); err != nil {
			logger.Fatal("failed to run migrations", "error", err)
		}
	}

	// Клиент к Solver Service (используем настройки из cfg)
	solverConfig := &client.SolverClientConfig{
		Address:    cfg.Services.Solver.Address(),
		Timeout:    cfg.Services.Solver.Timeout,
		MaxRetries: cfg.Services.Solver.MaxRetries,
		EnableTLS:  cfg.Services.Solver.TLS,
	}

	solverClient, err := client.NewSolverClient(solverConfig)
	if err != nil {
		logger.Fatal("failed to create solver client", "error", err)
	}
	defer solverClient.Close()

	// Инициализация слоев
	repo := repository.NewPostgresSimulationRepository(db)

	simulationService := service.NewSimulationService(repo, solverClient, cfg.App.Version)

	// gRPC сервер
	srv := server.New(cfg)
	simulationv1.RegisterSimulationServiceServer(srv.GetEngine(), simulationService)

	logger.Info("Starting simulation service",
		"port", cfg.GRPC.Port,
		"solver_addr", solverConfig.Address,
		"environment", cfg.App.Environment,
		"version", cfg.App.Version,
	)

	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}
