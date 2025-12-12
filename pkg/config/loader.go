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
		"grpc.max_recv_msg_size":                  16 * 1024 * 1024,
		"grpc.max_send_msg_size":                  16 * 1024 * 1024,
		"grpc.max_concurrent_conn":                1000,
		"grpc.keepalive.max_connection_idle":      15 * time.Minute,
		"grpc.keepalive.max_connection_age":       30 * time.Minute,
		"grpc.keepalive.max_connection_age_grace": 5 * time.Minute,
		"grpc.keepalive.time":                     5 * time.Minute,
		"grpc.keepalive.timeout":                  20 * time.Second,
		"grpc.tls.enabled":                        false,

		// HTTP
		"http.port":             8080,
		"http.read_timeout":     30 * time.Second,
		"http.write_timeout":    30 * time.Second,
		"http.shutdown_timeout": 10 * time.Second,
		// CORS - явно указываем Authorization!
		"http.cors.enabled":           true,
		"http.cors.allowed_origins":   []string{"*"},
		"http.cors.allowed_methods":   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		"http.cors.allowed_headers":   []string{"Content-Type", "Authorization", "Accept", "Origin", "X-Requested-With", "X-Grpc-Web", "Grpc-Timeout", "Grpc-Metadata-*"},
		"http.cors.exposed_headers":   []string{"Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin"},
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
	if configPath := os.Getenv(configEnvVar); configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return l.k.Load(file.Provider(configPath), yaml.Parser())
		}
	}

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
// Использует умную трансформацию ключей для полей с подчёркиванием
func (l *Loader) loadEnv() error {
	return l.k.Load(env.ProviderWithValue(l.envPrefix, ".", func(envKey string, value string) (string, interface{}) {
		// Убираем префикс и приводим к нижнему регистру
		key := strings.ToLower(strings.TrimPrefix(envKey, l.envPrefix))

		// Маппинг для полей с подчёркиванием в именах
		if mappedKey, ok := envKeyMappings[key]; ok {
			key = mappedKey
		} else {
			// По умолчанию заменяем все подчёркивания на точки
			key = strings.ReplaceAll(key, "_", ".")
		}

		// Для slice-полей разбиваем по запятой
		if isSliceField(key) {
			return key, splitAndTrim(value)
		}

		return key, value
	}), nil)
}

// envKeyMappings - маппинг переменных окружения на ключи конфига
// Необходим для полей, содержащих подчёркивания в именах
var envKeyMappings = map[string]string{
	// HTTP CORS
	"http_cors_enabled":           "http.cors.enabled",
	"http_cors_allowed_origins":   "http.cors.allowed_origins",
	"http_cors_allowed_methods":   "http.cors.allowed_methods",
	"http_cors_allowed_headers":   "http.cors.allowed_headers",
	"http_cors_exposed_headers":   "http.cors.exposed_headers",
	"http_cors_allow_credentials": "http.cors.allow_credentials",
	"http_cors_max_age":           "http.cors.max_age",

	// HTTP
	"http_port":             "http.port",
	"http_read_timeout":     "http.read_timeout",
	"http_write_timeout":    "http.write_timeout",
	"http_shutdown_timeout": "http.shutdown_timeout",

	// Database
	"database_driver":             "database.driver",
	"database_host":               "database.host",
	"database_port":               "database.port",
	"database_database":           "database.database",
	"database_username":           "database.username",
	"database_password":           "database.password",
	"database_ssl_mode":           "database.ssl_mode",
	"database_max_open_conns":     "database.max_open_conns",
	"database_max_idle_conns":     "database.max_idle_conns",
	"database_conn_max_lifetime":  "database.conn_max_lifetime",
	"database_conn_max_idle_time": "database.conn_max_idle_time",
	"database_migrations_path":    "database.migrations_path",
	"database_auto_migrate":       "database.auto_migrate",

	// Cache
	"cache_enabled":     "cache.enabled",
	"cache_driver":      "cache.driver",
	"cache_host":        "cache.host",
	"cache_port":        "cache.port",
	"cache_password":    "cache.password",
	"cache_db":          "cache.db",
	"cache_default_ttl": "cache.default_ttl",
	"cache_max_entries": "cache.max_entries",

	// Rate limit
	"rate_limit_enabled":          "rate_limit.enabled",
	"rate_limit_requests":         "rate_limit.requests",
	"rate_limit_window":           "rate_limit.window",
	"rate_limit_strategy":         "rate_limit.strategy",
	"rate_limit_backend":          "rate_limit.backend",
	"rate_limit_burst_size":       "rate_limit.burst_size",
	"rate_limit_cleanup_interval": "rate_limit.cleanup_interval",
	"rate_limit_redis_addr":       "rate_limit.redis_addr",

	// Audit
	"audit_enabled":          "audit.enabled",
	"audit_backend":          "audit.backend",
	"audit_file_path":        "audit.file_path",
	"audit_buffer_size":      "audit.buffer_size",
	"audit_flush_period":     "audit.flush_period",
	"audit_exclude_methods":  "audit.exclude_methods",
	"audit_include_request":  "audit.include_request",
	"audit_include_response": "audit.include_response",

	// GRPC
	"grpc_port":                               "grpc.port",
	"grpc_max_recv_msg_size":                  "grpc.max_recv_msg_size",
	"grpc_max_send_msg_size":                  "grpc.max_send_msg_size",
	"grpc_max_concurrent_conn":                "grpc.max_concurrent_conn",
	"grpc_keepalive_max_connection_idle":      "grpc.keepalive.max_connection_idle",
	"grpc_keepalive_max_connection_age":       "grpc.keepalive.max_connection_age",
	"grpc_keepalive_max_connection_age_grace": "grpc.keepalive.max_connection_age_grace",
	"grpc_keepalive_time":                     "grpc.keepalive.time",
	"grpc_keepalive_timeout":                  "grpc.keepalive.timeout",
	"grpc_tls_enabled":                        "grpc.tls.enabled",
	"grpc_tls_cert_file":                      "grpc.tls.cert_file",
	"grpc_tls_key_file":                       "grpc.tls.key_file",
	"grpc_tls_ca_file":                        "grpc.tls.ca_file",

	// Log
	"log_level":       "log.level",
	"log_format":      "log.format",
	"log_output":      "log.output",
	"log_file_path":   "log.file_path",
	"log_max_size":    "log.max_size",
	"log_max_backups": "log.max_backups",
	"log_max_age":     "log.max_age",
	"log_compress":    "log.compress",

	// Services (примеры)
	"services_solver_host":     "services.solver.host",
	"services_solver_port":     "services.solver.port",
	"services_solver_timeout":  "services.solver.timeout",
	"services_analytics_host":  "services.analytics.host",
	"services_analytics_port":  "services.analytics.port",
	"services_validation_host": "services.validation.host",
	"services_validation_port": "services.validation.port",
	"services_history_host":    "services.history.host",
	"services_history_port":    "services.history.port",
	"services_auth_host":       "services.auth.host",
	"services_auth_port":       "services.auth.port",
	"services_audit_host":      "services.audit.host",
	"services_audit_port":      "services.audit.port",
	"services_simulation_host": "services.simulation.host",
	"services_simulation_port": "services.simulation.port",
	"services_report_host":     "services.report.host",
	"services_report_port":     "services.report.port",

	// Report
	"report_save_to_storage":       "report.save_to_storage",
	"report_default_ttl":           "report.default_ttl",
	"report_max_report_size_bytes": "report.max_report_size_bytes",
	"report_cleanup_interval":      "report.cleanup_interval",
	"report_default_language":      "report.default_language",
	"report_default_theme":         "report.default_theme",
}

// sliceFields - поля, которые должны парситься как слайсы
var sliceFields = map[string]bool{
	"http.cors.allowed_origins": true,
	"http.cors.allowed_methods": true,
	"http.cors.allowed_headers": true,
	"http.cors.exposed_headers": true,
	"audit.exclude_methods":     true,
}

func isSliceField(key string) bool {
	return sliceFields[key]
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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

	if cfg.GRPC.Port == 50051 && defaultPort != 0 {
		cfg.GRPC.Port = defaultPort
	}

	if cfg.App.Name == "logistics-service" {
		cfg.App.Name = serviceName
	}

	return cfg, nil
}
