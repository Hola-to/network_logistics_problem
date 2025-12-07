package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"logistics/pkg/config"
	"logistics/pkg/logger"
)

// DB интерфейс для работы с базой данных
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Close()
	Ping(ctx context.Context) error
}

// PostgresDB обёртка над pgxpool.Pool
type PostgresDB struct {
	pool *pgxpool.Pool
	cfg  *config.DatabaseConfig
}

// NewPostgresDB создаёт новое подключение к PostgreSQL
func NewPostgresDB(ctx context.Context, cfg *config.DatabaseConfig) (*PostgresDB, error) {
	connString := buildConnectionString(cfg)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Настройки пула
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	// Настройки подключения
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Connected to PostgreSQL",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"max_conns", cfg.MaxOpenConns,
	)

	return &PostgresDB{
		pool: pool,
		cfg:  cfg,
	}, nil
}

func buildConnectionString(cfg *config.DatabaseConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)
}

// Exec выполняет запрос без возврата результатов
func (db *PostgresDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// Query выполняет запрос с возвратом строк
func (db *PostgresDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow выполняет запрос с возвратом одной строки
func (db *PostgresDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// BeginTx начинает транзакцию
func (db *PostgresDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return db.pool.BeginTx(ctx, txOptions)
}

// Close закрывает пул соединений
func (db *PostgresDB) Close() {
	db.pool.Close()
	logger.Log.Info("PostgreSQL connection pool closed")
}

// Ping проверяет соединение
func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Pool возвращает базовый пул (для особых случаев)
func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}

// Stats возвращает статистику пула
func (db *PostgresDB) Stats() *pgxpool.Stat {
	return db.pool.Stat()
}

// HealthCheck проверяет здоровье подключения
func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var result int
	err := db.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}
