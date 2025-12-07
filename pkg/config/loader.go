// pkg/config/loader.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	envPrefix    = "LOGISTICS_"
	configEnvVar = "CONFIG_PATH"
)

// Loader загружает конфигурацию из разных источников
type Loader struct {
	k           *koanf.Koanf
	configPaths []string
	envPrefix   string
}

// NewLoader создаёт новый загрузчик конфигурации
func NewLoader(opts ...LoaderOption) *Loader {
	l := &Loader{
		k: koanf.New("."),
		configPaths: []string{
			"config.yaml",
			"config/config.yaml",
			"/etc/logistics/config.yaml",
		},
		envPrefix: envPrefix,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LoaderOption - опция для конфигурации загрузчика
type LoaderOption func(*Loader)

// WithConfigPaths устанавливает пути поиска конфигурации
func WithConfigPaths(paths ...string) LoaderOption {
	return func(l *Loader) {
		l.configPaths = paths
	}
}

// WithEnvPrefix устанавливает префикс переменных окружения
func WithEnvPrefix(prefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = prefix
	}
}

// Load загружает конфигурацию с приоритетом:
// 1. Defaults (самый низкий)
// 2. Config file (yaml)
// 3. Environment variables (самый высокий)
func (l *Loader) Load() (*Config, error) {
	// 1. Загружаем значения по умолчанию
	if err := l.loadDefaults(); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	// 2. Загружаем из файла конфигурации
	if err := l.loadConfigFile(); err != nil {
		// Файл не обязателен, логируем warning
		fmt.Printf("Warning: %v\n", err)
	}

	// 3. Загружаем из переменных окружения (перезаписывают файл)
	if err := l.loadEnv(); err != nil {
		return nil, fmt.Errorf("failed to load env: %w", err)
	}

	// 4. Распаковываем в структуру
	var cfg Config
	if err := l.k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 5. Валидируем
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// loadDefaults загружает значения по умолчанию
func (l *Loader) loadDefaults() error {
	defaults := map[string]any{
		// App
		"app.name":        "logistics-service",
		"app.version":     "1.0.0",
		"app.environment": "development",
		"app.debug":       false,

		// GRPC
		"grpc.port":                               50051,
		"grpc.max_recv_msg_size":                  16 * 1024 * 1024, // 16MB
		"grpc.max_send_msg_size":                  16 * 1024 * 1024,
		"grpc.max_concurrent_conn":                1000,
		"grpc.keepalive.max_connection_idle":      15 * time.Minute,
		"grpc.keepalive.max_connection_age":       30 * time.Minute,
		"grpc.keepalive.max_connection_age_grace": 5 * time.Minute,
		"grpc.keepalive.time":                     5 * time.Minute,
		"grpc.keepalive.timeout":                  20 * time.Second,
		"grpc.tls.enabled":                        false,

		// HTTP
		"http.port":                   8080,
		"http.read_timeout":           30 * time.Second,
		"http.write_timeout":          30 * time.Second,
		"http.shutdown_timeout":       10 * time.Second,
		"http.cors.enabled":           true,
		"http.cors.allowed_origins":   []string{"*"},
		"http.cors.allowed_methods":   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		"http.cors.allowed_headers":   []string{"*"},
		"http.cors.allow_credentials": false,
		"http.cors.max_age":           86400,

		// Log
		"log.level":       "info",
		"log.format":      "json",
		"log.output":      "stdout",
		"log.max_size":    100,
		"log.max_backups": 3,
		"log.max_age":     7,
		"log.compress":    true,

		// Metrics
		"metrics.enabled":   true,
		"metrics.port":      9090,
		"metrics.path":      "/metrics",
		"metrics.namespace": "logistics",
		"metrics.subsystem": "",

		// Tracing
		"tracing.enabled":      false,
		"tracing.endpoint":     "localhost:4317",
		"tracing.service_name": "logistics-service",
		"tracing.sample_rate":  0.1,

		// Services - Solver
		"services.solver.host":           "localhost",
		"services.solver.port":           50052,
		"services.solver.timeout":        30 * time.Second,
		"services.solver.max_retries":    3,
		"services.solver.retry_backoff":  100 * time.Millisecond,
		"services.solver.load_balancing": "round_robin",

		// Services - Analytics
		"services.analytics.host":           "localhost",
		"services.analytics.port":           50053,
		"services.analytics.timeout":        30 * time.Second,
		"services.analytics.max_retries":    3,
		"services.analytics.retry_backoff":  100 * time.Millisecond,
		"services.analytics.load_balancing": "round_robin",

		// Services - Validation
		"services.validation.host":           "localhost",
		"services.validation.port":           50054,
		"services.validation.timeout":        30 * time.Second,
		"services.validation.max_retries":    3,
		"services.validation.retry_backoff":  100 * time.Millisecond,
		"services.validation.load_balancing": "round_robin",

		// Services - History
		"services.history.host":           "localhost",
		"services.history.port":           50055,
		"services.history.timeout":        30 * time.Second,
		"services.history.max_retries":    3,
		"services.history.retry_backoff":  100 * time.Millisecond,
		"services.history.load_balancing": "round_robin",

		// Services - Auth
		"services.auth.host":           "localhost",
		"services.auth.port":           50056,
		"services.auth.timeout":        10 * time.Second,
		"services.auth.max_retries":    3,
		"services.auth.retry_backoff":  100 * time.Millisecond,
		"services.auth.load_balancing": "round_robin",

		// Services - Audit
		"services.audit.host":           "localhost",
		"services.audit.port":           50057,
		"services.audit.timeout":        10 * time.Second,
		"services.audit.max_retries":    3,
		"services.audit.retry_backoff":  100 * time.Millisecond,
		"services.audit.load_balancing": "round_robin",

		// Services - Simulation
		"services.simulation.host":           "localhost",
		"services.simulation.port":           50058,
		"services.simulation.timeout":        600 * time.Second,
		"services.simulation.max_retries":    3,
		"services.simulation.retry_backoff":  100 * time.Millisecond,
		"services.simulation.load_balancing": "round_robin",

		// Services - Report
		"services.report.host":           "localhost",
		"services.report.port":           50059,
		"services.report.timeout":        60 * time.Second,
		"services.report.max_retries":    3,
		"services.report.retry_backoff":  100 * time.Millisecond,
		"services.report.load_balancing": "round_robin",

		// Database
		"database.driver":             "postgres",
		"database.host":               "localhost",
		"database.port":               5432,
		"database.database":           "logistics",
		"database.username":           "postgres",
		"database.password":           "",
		"database.ssl_mode":           "disable",
		"database.max_open_conns":     25,
		"database.max_idle_conns":     5,
		"database.conn_max_lifetime":  5 * time.Minute,
		"database.conn_max_idle_time": 5 * time.Minute,
		"database.auto_migrate":       true,

		// Cache
		"cache.enabled":     false,
		"cache.driver":      "memory",
		"cache.host":        "localhost",
		"cache.port":        6379,
		"cache.db":          0,
		"cache.default_ttl": 5 * time.Minute,
		"cache.max_entries": 10000,

		// Rate Limit
		"rate_limit.enabled":          true,
		"rate_limit.requests":         100,
		"rate_limit.window":           time.Minute,
		"rate_limit.strategy":         "sliding_window",
		"rate_limit.backend":          "memory",
		"rate_limit.burst_size":       10,
		"rate_limit.cleanup_interval": 5 * time.Minute,

		// Audit
		"audit.enabled":      true,
		"audit.backend":      "stdout",
		"audit.buffer_size":  1000,
		"audit.flush_period": 5 * time.Second,

		// Swagger
		"swagger.enabled": true,
		"swagger.port":    8081,
		"swagger.title":   "Logistics API",

		// Retry
		"retry.max_attempts":       3,
		"retry.initial_backoff":    100 * time.Millisecond,
		"retry.max_backoff":        10 * time.Second,
		"retry.backoff_multiplier": 2.0,

		// Report - Storage
		"report.save_to_storage":       true,
		"report.default_ttl":           30 * 24 * time.Hour,     // 30 дней
		"report.max_report_size_bytes": 50 * 1024 * 1024,        // 50 MB
		"report.max_storage_bytes":     10 * 1024 * 1024 * 1024, // 10 GB
		"report.max_reports_per_user":  1000,

		// Report - Generation
		"report.default_language":         "en",
		"report.default_currency":         "USD",
		"report.default_theme":            "light",
		"report.max_edges_in_table":       50,
		"report.max_paths_in_table":       20,
		"report.include_raw_data_default": true,

		// Report - Cleanup
		"report.cleanup_interval":   1 * time.Hour,
		"report.retention_period":   7 * 24 * time.Hour, // 7 дней для soft-deleted
		"report.cleanup_batch_size": 100,

		// Report - Branding
		"report.default_company_name": "Logistics Platform",
		"report.default_logo_url":     "",

		// Report - PDF
		"report.pdf.page_size":           "A4",
		"report.pdf.orientation":         "portrait",
		"report.pdf.margin_top":          15.0,
		"report.pdf.margin_bottom":       15.0,
		"report.pdf.margin_left":         15.0,
		"report.pdf.margin_right":        15.0,
		"report.pdf.font_family":         "Arial",
		"report.pdf.font_size":           10.0,
		"report.pdf.header_font_size":    14.0,
		"report.pdf.enable_page_numbers": true,
		"report.pdf.enable_watermark":    false,
		"report.pdf.watermark_text":      "CONFIDENTIAL",
	}

	return l.k.Load(confmap.Provider(defaults, "."), nil)
}

// loadConfigFile загружает конфигурацию из файла
func (l *Loader) loadConfigFile() error {
	// Сначала проверяем переменную окружения
	if configPath := os.Getenv(configEnvVar); configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return l.k.Load(file.Provider(configPath), yaml.Parser())
		}
	}

	// Ищем файл по списку путей
	for _, path := range l.configPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		if _, err := os.Stat(absPath); err == nil {
			return l.k.Load(file.Provider(absPath), yaml.Parser())
		}
	}

	return fmt.Errorf("config file not found in paths: %v", l.configPaths)
}

// loadEnv загружает конфигурацию из переменных окружения
func (l *Loader) loadEnv() error {
	return l.k.Load(env.Provider(l.envPrefix, ".", func(s string) string {
		// LOGISTICS_GRPC_PORT -> grpc.port
		return strings.ReplaceAll(
			strings.ToLower(
				strings.TrimPrefix(s, l.envPrefix),
			),
			"_", ".",
		)
	}), nil)
}

// MustLoad загружает конфигурацию или паникует
func MustLoad(opts ...LoaderOption) *Config {
	cfg, err := NewLoader(opts...).Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// Load - удобная функция для загрузки с дефолтными настройками
func Load() (*Config, error) {
	return NewLoader().Load()
}

// LoadWithServiceDefaults загружает конфигурацию с переопределением для конкретного сервиса
func LoadWithServiceDefaults(serviceName string, defaultPort int) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Если порт не задан явно, используем дефолтный для сервиса
	if cfg.GRPC.Port == 50051 && defaultPort != 0 {
		cfg.GRPC.Port = defaultPort
	}

	// Обновляем имя сервиса
	if cfg.App.Name == "logistics-service" {
		cfg.App.Name = serviceName
	}

	return cfg, nil
}
