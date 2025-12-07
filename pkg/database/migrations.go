package database

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"logistics/pkg/config"
	"logistics/pkg/logger"
)

// Migrator управляет миграциями
type Migrator struct {
	pool       *pgxpool.Pool
	migrations embed.FS
	dir        string
}

// NewMigrator создаёт новый мигратор
func NewMigrator(pool *pgxpool.Pool, migrations embed.FS, dir string) *Migrator {
	return &Migrator{
		pool:       pool,
		migrations: migrations,
		dir:        dir,
	}
}

// Up применяет все миграции
func (m *Migrator) Up(ctx context.Context) error {
	db := stdlib.OpenDBFromPool(m.pool)
	defer db.Close()

	goose.SetBaseFS(m.migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, m.dir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Log.Info("Migrations applied successfully")
	return nil
}

// Down откатывает последнюю миграцию
func (m *Migrator) Down(ctx context.Context) error {
	db := stdlib.OpenDBFromPool(m.pool)
	defer db.Close()

	goose.SetBaseFS(m.migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.DownContext(ctx, db, m.dir); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	logger.Log.Info("Migration rolled back successfully")
	return nil
}

// Status показывает статус миграций
func (m *Migrator) Status(ctx context.Context) error {
	db := stdlib.OpenDBFromPool(m.pool)
	defer db.Close()

	goose.SetBaseFS(m.migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	return goose.StatusContext(ctx, db, m.dir)
}

// RunMigrations запускает миграции если включено в конфигурации
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, cfg *config.DatabaseConfig, migrations embed.FS, dir string) error {
	if !cfg.AutoMigrate {
		logger.Log.Info("Auto-migration is disabled")
		return nil
	}

	migrator := NewMigrator(pool, migrations, dir)
	return migrator.Up(ctx)
}
