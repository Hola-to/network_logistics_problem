// pkg/config/config.go
package config

import (
	"fmt"
	"strings"
	"time"
)

// Config - главная структура конфигурации
type Config struct {
	App       AppConfig       `koanf:"app"`
	GRPC      GRPCConfig      `koanf:"grpc"`
	HTTP      HTTPConfig      `koanf:"http"`
	Log       LogConfig       `koanf:"log"`
	Metrics   MetricsConfig   `koanf:"metrics"`
	Tracing   TracingConfig   `koanf:"tracing"`
	Services  ServicesConfig  `koanf:"services"`
	Database  DatabaseConfig  `koanf:"database"`
	Cache     CacheConfig     `koanf:"cache"`
	RateLimit RateLimitConfig `koanf:"rate_limit"`
	Audit     AuditConfig     `koanf:"audit"`
	Swagger   SwaggerConfig   `koanf:"swagger"`
	Retry     RetryConfig     `koanf:"retry"`
	Report    ReportConfig    `koanf:"report"`
}

// AppConfig - общие настройки приложения
type AppConfig struct {
	Name        string `koanf:"name"`
	Version     string `koanf:"version"`
	Environment string `koanf:"environment"` // development, staging, production
	Debug       bool   `koanf:"debug"`
}

// GRPCConfig - настройки gRPC сервера
type GRPCConfig struct {
	Port              int             `koanf:"port"`
	MaxRecvMsgSize    int             `koanf:"max_recv_msg_size"` // bytes
	MaxSendMsgSize    int             `koanf:"max_send_msg_size"` // bytes
	MaxConcurrentConn int             `koanf:"max_concurrent_conn"`
	KeepAlive         KeepAliveConfig `koanf:"keepalive"`
	TLS               TLSConfig       `koanf:"tls"`
}

// KeepAliveConfig - настройки keep-alive
type KeepAliveConfig struct {
	MaxConnectionIdle     time.Duration `koanf:"max_connection_idle"`
	MaxConnectionAge      time.Duration `koanf:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `koanf:"max_connection_age_grace"`
	Time                  time.Duration `koanf:"time"`
	Timeout               time.Duration `koanf:"timeout"`
}

// TLSConfig - настройки TLS
type TLSConfig struct {
	Enabled  bool   `koanf:"enabled"`
	CertFile string `koanf:"cert_file"`
	KeyFile  string `koanf:"key_file"`
	CAFile   string `koanf:"ca_file"`
}

// HTTPConfig - настройки HTTP сервера (для gateway)
type HTTPConfig struct {
	Port            int           `koanf:"port"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
	CORS            CORSConfig    `koanf:"cors"`
}

// CORSConfig - настройки CORS
type CORSConfig struct {
	Enabled          bool     `koanf:"enabled"`
	AllowedOrigins   []string `koanf:"allowed_origins"`
	AllowedMethods   []string `koanf:"allowed_methods"`
	AllowedHeaders   []string `koanf:"allowed_headers"`
	AllowCredentials bool     `koanf:"allow_credentials"`
	MaxAge           int      `koanf:"max_age"`
}

// LogConfig - настройки логирования
type LogConfig struct {
	Level      string `koanf:"level"`       // debug, info, warn, error
	Format     string `koanf:"format"`      // json, text
	Output     string `koanf:"output"`      // stdout, stderr, file
	FilePath   string `koanf:"file_path"`   // путь к файлу логов
	MaxSize    int    `koanf:"max_size"`    // MB
	MaxBackups int    `koanf:"max_backups"` // количество бэкапов
	MaxAge     int    `koanf:"max_age"`     // дней
	Compress   bool   `koanf:"compress"`
}

// MetricsConfig - настройки Prometheus метрик
type MetricsConfig struct {
	Enabled   bool   `koanf:"enabled"`
	Port      int    `koanf:"port"`
	Path      string `koanf:"path"`
	Namespace string `koanf:"namespace"`
	Subsystem string `koanf:"subsystem"`
}

// TracingConfig - настройки OpenTelemetry
type TracingConfig struct {
	Enabled     bool    `koanf:"enabled"`
	Endpoint    string  `koanf:"endpoint"`
	ServiceName string  `koanf:"service_name"`
	SampleRate  float64 `koanf:"sample_rate"`
}

// ServicesConfig - адреса других сервисов
type ServicesConfig struct {
	Solver     ServiceEndpoint `koanf:"solver"`
	Analytics  ServiceEndpoint `koanf:"analytics"`
	Validation ServiceEndpoint `koanf:"validation"`
	History    ServiceEndpoint `koanf:"history"`
	Auth       ServiceEndpoint `koanf:"auth"`
	Audit      ServiceEndpoint `koanf:"audit"`
	Simulation ServiceEndpoint `koanf:"simulation"`
	Report     ServiceEndpoint `koanf:"report"`
}

// ServiceEndpoint - конфигурация подключения к сервису
type ServiceEndpoint struct {
	Host            string        `koanf:"host"`
	Port            int           `koanf:"port"`
	Timeout         time.Duration `koanf:"timeout"`
	MaxRetries      int           `koanf:"max_retries"`
	RetryBackoff    time.Duration `koanf:"retry_backoff"`
	TLS             bool          `koanf:"tls"`
	LoadBalancing   string        `koanf:"load_balancing"` // round_robin, pick_first
	HealthCheckPath string        `koanf:"health_check_path"`
}

// Address возвращает полный адрес сервиса
func (s ServiceEndpoint) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig - настройки базы данных
type DatabaseConfig struct {
	Driver          string        `koanf:"driver"` // postgres, mysql, sqlite
	Host            string        `koanf:"host"`
	Port            int           `koanf:"port"`
	Database        string        `koanf:"database"`
	Username        string        `koanf:"username"`
	Password        string        `koanf:"password"`
	SSLMode         string        `koanf:"ssl_mode"`
	MaxOpenConns    int           `koanf:"max_open_conns"`
	MaxIdleConns    int           `koanf:"max_idle_conns"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time"`
	MigrationsPath  string        `koanf:"migrations_path"`
	AutoMigrate     bool          `koanf:"auto_migrate"`
}

// DSN возвращает строку подключения
func (d DatabaseConfig) DSN() string {
	switch strings.ToLower(d.Driver) {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			d.Host, d.Port, d.Username, d.Password, d.Database, d.SSLMode,
		)
	case "mysql":
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true",
			d.Username, d.Password, d.Host, d.Port, d.Database,
		)
	case "sqlite":
		return d.Database
	default:
		return ""
	}
}

// CacheConfig - настройки кэширования
type CacheConfig struct {
	Enabled    bool          `koanf:"enabled"`
	Driver     string        `koanf:"driver"` // redis, memory
	Host       string        `koanf:"host"`
	Port       int           `koanf:"port"`
	Password   string        `koanf:"password"`
	DB         int           `koanf:"db"`
	DefaultTTL time.Duration `koanf:"default_ttl"`
	MaxEntries int           `koanf:"max_entries"` // для in-memory
}

// Address возвращает адрес кэша
func (c CacheConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// RateLimitConfig конфигурация rate limiting
type RateLimitConfig struct {
	Enabled         bool          `koanf:"enabled"`
	Requests        int           `koanf:"requests"`
	Window          time.Duration `koanf:"window"`
	Strategy        string        `koanf:"strategy"`
	Backend         string        `koanf:"backend"`
	BurstSize       int           `koanf:"burst_size"`
	CleanupInterval time.Duration `koanf:"cleanup_interval"`
	RedisAddr       string        `koanf:"redis_addr"`
}

// AuditConfig конфигурация аудит лога
type AuditConfig struct {
	Enabled         bool          `koanf:"enabled"`
	Backend         string        `koanf:"backend"`
	FilePath        string        `koanf:"file_path"`
	BufferSize      int           `koanf:"buffer_size"`
	FlushPeriod     time.Duration `koanf:"flush_period"`
	ExcludeMethods  []string      `koanf:"exclude_methods"`
	IncludeRequest  bool          `koanf:"include_request"`
	IncludeResponse bool          `koanf:"include_response"`
}

// SwaggerConfig конфигурация Swagger UI
type SwaggerConfig struct {
	Enabled bool   `koanf:"enabled"`
	Port    int    `koanf:"port"`
	Title   string `koanf:"title"`
}

// RetryConfig конфигурация retry
type RetryConfig struct {
	MaxAttempts       int           `koanf:"max_attempts"`
	InitialBackoff    time.Duration `koanf:"initial_backoff"`
	MaxBackoff        time.Duration `koanf:"max_backoff"`
	BackoffMultiplier float64       `koanf:"backoff_multiplier"`
}

// ReportConfig конфигурация сервиса отчётов
type ReportConfig struct {
	// Хранилище
	SaveToStorage bool          `koanf:"save_to_storage"` // Сохранять отчёты в БД
	DefaultTTL    time.Duration `koanf:"default_ttl"`     // TTL по умолчанию для отчётов

	// Лимиты
	MaxReportSizeBytes int64 `koanf:"max_report_size_bytes"` // Максимальный размер отчёта
	MaxStorageBytes    int64 `koanf:"max_storage_bytes"`     // Максимальный общий размер хранилища
	MaxReportsPerUser  int   `koanf:"max_reports_per_user"`  // Максимум отчётов на пользователя

	// Генерация
	DefaultLanguage       string `koanf:"default_language"`         // Язык по умолчанию (en, ru)
	DefaultCurrency       string `koanf:"default_currency"`         // Валюта по умолчанию
	DefaultTheme          string `koanf:"default_theme"`            // Тема по умолчанию (light, dark, corporate)
	MaxEdgesInTable       int    `koanf:"max_edges_in_table"`       // Максимум рёбер в таблице отчёта
	MaxPathsInTable       int    `koanf:"max_paths_in_table"`       // Максимум путей в таблице отчёта
	IncludeRawDataDefault bool   `koanf:"include_raw_data_default"` // Включать сырые данные по умолчанию

	// Очистка
	CleanupInterval  time.Duration `koanf:"cleanup_interval"`   // Интервал очистки устаревших отчётов
	RetentionPeriod  time.Duration `koanf:"retention_period"`   // Период хранения удалённых отчётов
	CleanupBatchSize int           `koanf:"cleanup_batch_size"` // Размер батча при очистке

	// PDF генерация
	PDF PDFConfig `koanf:"pdf"`

	// Брендинг по умолчанию
	DefaultCompanyName string `koanf:"default_company_name"`
	DefaultLogoURL     string `koanf:"default_logo_url"`
}

// PDFConfig конфигурация PDF генератора
type PDFConfig struct {
	PageSize          string  `koanf:"page_size"`        // A4, Letter, Legal
	Orientation       string  `koanf:"orientation"`      // portrait, landscape
	MarginTop         float64 `koanf:"margin_top"`       // mm
	MarginBottom      float64 `koanf:"margin_bottom"`    // mm
	MarginLeft        float64 `koanf:"margin_left"`      // mm
	MarginRight       float64 `koanf:"margin_right"`     // mm
	FontFamily        string  `koanf:"font_family"`      // Arial, Helvetica, etc.
	FontSize          float64 `koanf:"font_size"`        // pt
	HeaderFontSize    float64 `koanf:"header_font_size"` // pt
	EnablePageNumbers bool    `koanf:"enable_page_numbers"`
	EnableWatermark   bool    `koanf:"enable_watermark"`
	WatermarkText     string  `koanf:"watermark_text"`
}

// Validate проверяет конфигурацию
func (c *Config) Validate() error {
	var errs []string

	if c.App.Name == "" {
		errs = append(errs, "app.name is required")
	}

	if c.GRPC.Port <= 0 || c.GRPC.Port > 65535 {
		errs = append(errs, fmt.Sprintf("grpc.port must be between 1 and 65535, got %d", c.GRPC.Port))
	}

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.Log.Level)] {
		errs = append(errs, fmt.Sprintf("log.level must be one of: debug, info, warn, error, got %s", c.Log.Level))
	}

	// Валидация Report config
	if c.Report.MaxReportSizeBytes < 0 {
		errs = append(errs, "report.max_report_size_bytes must be non-negative")
	}

	validThemes := map[string]bool{"light": true, "dark": true, "corporate": true}
	if c.Report.DefaultTheme != "" && !validThemes[c.Report.DefaultTheme] {
		errs = append(errs, fmt.Sprintf("report.default_theme must be one of: light, dark, corporate, got %s", c.Report.DefaultTheme))
	}

	validPageSizes := map[string]bool{"A4": true, "Letter": true, "Legal": true, "A3": true}
	if c.Report.PDF.PageSize != "" && !validPageSizes[c.Report.PDF.PageSize] {
		errs = append(errs, fmt.Sprintf("report.pdf.page_size must be one of: A4, Letter, Legal, A3, got %s", c.Report.PDF.PageSize))
	}

	validOrientations := map[string]bool{"portrait": true, "landscape": true}
	if c.Report.PDF.Orientation != "" && !validOrientations[c.Report.PDF.Orientation] {
		errs = append(errs, fmt.Sprintf("report.pdf.orientation must be one of: portrait, landscape, got %s", c.Report.PDF.Orientation))
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

// IsDevelopment проверяет режим разработки
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development" || c.App.Environment == "dev"
}

// IsProduction проверяет продакшн режим
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production" || c.App.Environment == "prod"
}
