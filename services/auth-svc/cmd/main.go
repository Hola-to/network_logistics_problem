package main

import (
	"context"
	"log"
	"os"
	"time"

	authv1 "logistics/gen/go/logistics/auth/v1"
	"logistics/migrations"
	"logistics/pkg/config"
	"logistics/pkg/database"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/passhash"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/auth-svc/internal/repository"
	"logistics/services/auth-svc/internal/service"
	"logistics/services/auth-svc/internal/token"
)

func main() {
	cfg, err := config.LoadWithServiceDefaults("auth-svc", 50055)
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

	// Инициализация репозиториев
	repos, err := repository.NewRepositories(ctx, &cfg.Database)
	if err != nil {
		logger.Fatal("failed to create repositories", "error", err)
	}
	defer repos.Close()

	// Запускаем миграции для PostgreSQL
	if cfg.Database.Driver == "postgres" || cfg.Database.Driver == "postgresql" {
		if err := runMigrations(ctx, cfg); err != nil {
			logger.Fatal("failed to run migrations", "error", err)
		}
	}

	// Инициализация менеджера токенов
	tokenManager := token.NewManager(&token.Config{
		SecretKey:          getEnv("JWT_SECRET", "super-secret-key-change-in-production"),
		AccessTokenExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m")),
		RefreshTokenExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h")), // 7 days
		Issuer:             "logistics-auth",
	})

	// Создаём сервис
	authService := service.NewAuthService(repos.Users, repos.Blacklist, tokenManager)

	// Создаём и запускаем gRPC сервер
	srv := server.New(cfg)
	authv1.RegisterAuthServiceServer(srv.GetEngine(), authService)

	logger.Info("Starting auth service",
		"port", cfg.GRPC.Port,
		"environment", cfg.App.Environment,
		"version", cfg.App.Version,
		"database", cfg.Database.Driver,
	)

	// Создаём тестового пользователя для разработки
	if cfg.IsDevelopment() {
		createTestUser(ctx, repos.Users)
	}

	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}

func runMigrations(ctx context.Context, cfg *config.Config) error {
	db, err := database.NewPostgresDB(ctx, &cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	return database.RunMigrations(
		ctx,
		db.Pool(),
		&cfg.Database,
		migrations.PostgresMigrations,
		"postgres",
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func createTestUser(ctx context.Context, repo repository.UserRepository) {
	passwordHash, err := passhash.HashPassword("password123")
	if err != nil {
		logger.Log.Warn("Failed to hash password for test user", "error", err)
		return
	}

	testUser := &repository.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		FullName:     "Test User",
		Role:         "admin",
	}

	if err := repo.Create(ctx, testUser); err != nil {
		// Игнорируем ошибку если пользователь уже существует
		if err != repository.ErrUserAlreadyExists {
			logger.Log.Warn("Failed to create test user", "error", err)
		}
		return
	}

	logger.Log.Info("Test user created",
		"username", testUser.Username,
		"password", "password123",
	)
}
