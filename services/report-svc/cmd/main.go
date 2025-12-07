package main

import (
	"context"
	"log"
	"time"

	reportv1 "logistics/gen/go/logistics/report/v1"
	"logistics/migrations"
	"logistics/pkg/config"
	"logistics/pkg/database"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/report-svc/internal/repository"
	"logistics/services/report-svc/internal/service"
)

func main() {
	cfg, err := config.LoadWithServiceDefaults("report-svc", 50059)
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
		}
	}

	metrics.InitMetrics(cfg.Metrics.Namespace, cfg.App.Name)

	// Инициализируем хранилище
	var store repository.Repository
	var db *database.PostgresDB

	if cfg.Database.Driver == "postgres" {
		db, err = database.NewPostgresDB(ctx, &cfg.Database)
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
		store = repository.NewPostgresRepository(db)
		logger.Info("Storage initialized", "driver", cfg.Database.Driver)

		// Запускаем cleanup горутину
		go runCleanup(ctx, store, cfg.Report.CleanupInterval)

	} else {
		logger.Log.Warn("Database not configured or driver is not 'postgres', running without persistence")
	}

	srv := server.New(cfg)

	// Настройки сервиса
	svcConfig := service.ServiceConfig{
		Version:       cfg.App.Version,
		DefaultTTL:    cfg.Report.DefaultTTL,
		SaveToStorage: store != nil,
	}

	reportService := service.NewReportService(svcConfig, store)
	reportv1.RegisterReportServiceServer(srv.GetEngine(), reportService)

	logger.Info("Starting report service",
		"port", cfg.GRPC.Port,
		"storage_enabled", store != nil,
	)

	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}

// runCleanup периодически удаляет устаревшие отчёты
func runCleanup(ctx context.Context, store repository.Repository, interval time.Duration) {
	if interval == 0 {
		interval = 1 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Expired reports cleanup worker started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping cleanup worker")
			return
		case <-ticker.C:
			deleted, err := store.DeleteExpired(ctx)
			if err != nil {
				logger.Log.Error("Failed to cleanup expired reports", "error", err)
			} else if deleted > 0 {
				logger.Info("Cleaned up expired reports", "count", deleted)
			}
		}
	}
}
