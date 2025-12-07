package repository

import (
	"context"
	"fmt"

	"logistics/pkg/config"
	"logistics/pkg/database"
)

// RepositoryType тип репозитория
type RepositoryType string

const (
	RepositoryTypeMemory   RepositoryType = "memory"
	RepositoryTypePostgres RepositoryType = "postgres"
)

// Repositories контейнер репозиториев
type Repositories struct {
	Users     UserRepository
	Blacklist TokenBlacklist
	db        *database.PostgresDB // Для закрытия при shutdown
}

// Close закрывает соединения
func (r *Repositories) Close() {
	if r.db != nil {
		r.db.Close()
	}
}

// NewRepositories создаёт репозитории на основе конфигурации
func NewRepositories(ctx context.Context, cfg *config.DatabaseConfig) (*Repositories, error) {
	repoType := RepositoryType(cfg.Driver)

	switch repoType {
	case RepositoryTypeMemory, "":
		return newMemoryRepositories(), nil

	case RepositoryTypePostgres, "postgresql":
		return newPostgresRepositories(ctx, cfg)

	default:
		return nil, fmt.Errorf("unsupported repository type: %s", cfg.Driver)
	}
}

func newMemoryRepositories() *Repositories {
	return &Repositories{
		Users:     NewMemoryUserRepository(),
		Blacklist: NewMemoryTokenBlacklist(),
		db:        nil,
	}
}

func newPostgresRepositories(ctx context.Context, cfg *config.DatabaseConfig) (*Repositories, error) {
	db, err := database.NewPostgresDB(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return &Repositories{
		Users:     NewPostgresUserRepository(db),
		Blacklist: NewPostgresTokenBlacklist(db),
		db:        db,
	}, nil
}
