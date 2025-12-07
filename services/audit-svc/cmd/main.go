package main

import (
	"context"
	"log"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	"logistics/migrations"
	"logistics/pkg/config"
	"logistics/pkg/database"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/audit-svc/internal/repository"
	"logistics/services/audit-svc/internal/service"
)

func main() {
	cfg, err := config.LoadWithServiceDefaults("audit-svc", 50057)
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
				if err := tp.Shutdown(context.Background()); err != nil {
					logger.Log.Warn("Failed to shutdown telemetry", "error", err)
				}
			}()
			logger.Log.Info("Telemetry initialized", "endpoint", cfg.Tracing.Endpoint)
		}
	}

	metrics.InitMetrics(cfg.Metrics.Namespace, cfg.App.Name)

	// Подключение к PostgreSQL
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

	// Создаём репозиторий и сервис
	repo := repository.NewPostgresAuditRepository(db)
	auditService := service.NewAuditService(repo, cfg.App.Version)

	// Создаём gRPC сервер (без аудита для самого audit-svc чтобы избежать рекурсии)
	srv := server.NewWithOptions(cfg, &server.ServerOptions{
		AuditExcludeMethods: []string{
			"/logistics.audit.v1.AuditService/LogEvent",
			"/logistics.audit.v1.AuditService/LogEventBatch",
		},
	})

	auditv1.RegisterAuditServiceServer(srv.GetEngine(), auditService)

	logger.Info("Starting audit service",
		"port", cfg.GRPC.Port,
		"environment", cfg.App.Environment,
		"version", cfg.App.Version,
	)

	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}
