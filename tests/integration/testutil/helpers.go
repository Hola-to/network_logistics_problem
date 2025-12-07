package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"logistics/pkg/config"
)

// Environment variables
const (
	EnvIntegrationTests = "INTEGRATION_TESTS"
	EnvRedisAddr        = "REDIS_TEST_ADDR"
	EnvPostgresDSN      = "POSTGRES_TEST_DSN"
)

// SkipIfNotIntegration пропускает тест если не integration mode
func SkipIfNotIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv(EnvIntegrationTests) != "1" {
		t.Skip("skipping integration test; set INTEGRATION_TESTS=1 to run")
	}
}

// RequireRedis проверяет доступность Redis и возвращает адрес
func RequireRedis(t *testing.T) string {
	t.Helper()
	SkipIfNotIntegration(t)

	addr := os.Getenv(EnvRedisAddr)
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set")
	}

	// Проверяем доступность с контекстом
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
	}
	conn.Close()

	return addr
}

// RequirePostgres проверяет доступность PostgreSQL и возвращает DSN
func RequirePostgres(t *testing.T) string {
	t.Helper()
	SkipIfNotIntegration(t)

	dsn := os.Getenv(EnvPostgresDSN)
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set")
	}

	return dsn
}

// PostgresConfig возвращает конфигурацию для PostgreSQL
func PostgresConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Driver:          "postgres",
		Host:            getEnvOrDefault("POSTGRES_HOST", "localhost"),
		Port:            getEnvIntOrDefault("POSTGRES_PORT", 5433),
		Database:        getEnvOrDefault("POSTGRES_DB", "logistics_test"),
		Username:        getEnvOrDefault("POSTGRES_USER", "postgres"),
		Password:        getEnvOrDefault("POSTGRES_PASSWORD", "postgres"),
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// RequireService проверяет доступность сервиса
func RequireService(t *testing.T, envVar, defaultAddr string) string {
	t.Helper()
	SkipIfNotIntegration(t)

	addr := os.Getenv(envVar)
	if addr == "" {
		addr = defaultAddr
	}

	// Проверяем доступность с контекстом
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Skipf("Service not available at %s: %v", addr, err)
	}
	conn.Close()

	return addr
}

// Context возвращает контекст с таймаутом для тестов
func Context(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// ContextWithDuration возвращает контекст с указанным таймаутом
func ContextWithDuration(t *testing.T, d time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), d)
}

// Cleanup регистрирует функцию очистки
func Cleanup(t *testing.T, fn func()) {
	t.Helper()
	t.Cleanup(fn)
}

// RandomString генерирует случайную строку заданной длины
func RandomString(n int) string {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "fallback" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)[:n]
}

// UniqueKey генерирует уникальный ключ для теста
func UniqueKey(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s:%s:%s", prefix, t.Name(), RandomString(8))
}

// FreePort находит свободный порт
func FreePort(t *testing.T) int {
	t.Helper()

	// Используем ListenConfig для совместимости с noctx
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer lis.Close()
	return lis.Addr().(*net.TCPAddr).Port
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}
