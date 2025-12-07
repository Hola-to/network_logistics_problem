package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Стандартные ошибки
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrLimiterClosed     = errors.New("limiter is closed")
)

// Limiter интерфейс ограничителя запросов
type Limiter interface {
	// Allow проверяет, разрешён ли запрос
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN проверяет, разрешены ли n запросов
	AllowN(ctx context.Context, key string, n int) (bool, error)

	// Wait блокирует до получения разрешения
	Wait(ctx context.Context, key string) error

	// Reset сбрасывает лимит для ключа
	Reset(ctx context.Context, key string) error

	// GetInfo возвращает информацию о текущем состоянии
	GetInfo(ctx context.Context, key string) (*LimitInfo, error)

	// Close закрывает лимитер
	Close() error
}

// LimitInfo информация о состоянии лимита
type LimitInfo struct {
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	ResetAt    time.Time     `json:"reset_at"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// Config конфигурация rate limiter
type Config struct {
	// Requests количество запросов
	Requests int `koanf:"requests"`

	// Window временное окно
	Window time.Duration `koanf:"window"`

	// Strategy стратегия (sliding_window, token_bucket, fixed_window)
	Strategy string `koanf:"strategy"`

	// KeyFunc функция извлечения ключа (ip, user, method)
	KeyFunc string `koanf:"key_func"`

	// Backend хранилище (memory, redis)
	Backend string `koanf:"backend"`

	// BurstSize размер burst для token bucket
	BurstSize int `koanf:"burst_size"`

	// CleanupInterval интервал очистки для in-memory
	CleanupInterval time.Duration `koanf:"cleanup_interval"`

	// Redis настройки Redis
	RedisAddr     string `koanf:"redis_addr"`
	RedisPassword string `koanf:"redis_password"`
	RedisDB       int    `koanf:"redis_db"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Requests:        100,
		Window:          time.Minute,
		Strategy:        "sliding_window",
		KeyFunc:         "ip",
		Backend:         "memory",
		BurstSize:       10,
		CleanupInterval: 5 * time.Minute,
	}
}

// New создаёт лимитер на основе конфигурации
func New(cfg *Config) (Limiter, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	switch cfg.Backend {
	case "redis":
		return NewRedisLimiter(cfg)
	case "memory", "":
		return NewMemoryLimiter(cfg), nil
	default:
		return NewMemoryLimiter(cfg), nil
	}
}

// KeyExtractor функция извлечения ключа
type KeyExtractor func(ctx context.Context, method string, metadata map[string]string) string

// DefaultKeyExtractor извлекает ключ по IP
func DefaultKeyExtractor(_ context.Context, _ string, metadata map[string]string) string {
	if ip, ok := metadata["x-forwarded-for"]; ok && ip != "" {
		return ip
	}
	if ip, ok := metadata["x-real-ip"]; ok && ip != "" {
		return ip
	}
	if peer, ok := metadata[":authority"]; ok {
		return peer
	}
	return "unknown"
}

// MethodKeyExtractor извлекает ключ по методу
func MethodKeyExtractor(_ context.Context, method string, _ map[string]string) string {
	return method
}

// UserKeyExtractor извлекает ключ по пользователю
func UserKeyExtractor(ctx context.Context, method string, metadata map[string]string) string {
	if userID, ok := metadata["x-user-id"]; ok && userID != "" {
		return userID
	}
	return DefaultKeyExtractor(ctx, method, metadata)
}

// CompositeKeyExtractor комбинирует несколько ключей
func CompositeKeyExtractor(extractors ...KeyExtractor) KeyExtractor {
	return func(ctx context.Context, method string, metadata map[string]string) string {
		var key string
		for _, ext := range extractors {
			key += ext(ctx, method, metadata) + ":"
		}
		return key
	}
}

// RateLimitedMethods методы с rate limiting
type RateLimitedMethods struct {
	mu            sync.RWMutex
	methods       map[string]*Config
	defaultConfig *Config
}

// NewRateLimitedMethods создаёт конфигурацию методов
func NewRateLimitedMethods(defaultCfg *Config) *RateLimitedMethods {
	if defaultCfg == nil {
		defaultCfg = DefaultConfig()
	}
	return &RateLimitedMethods{
		methods:       make(map[string]*Config),
		defaultConfig: defaultCfg,
	}
}

// Set устанавливает лимит для метода
func (r *RateLimitedMethods) Set(method string, cfg *Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.methods[method] = cfg
}

// Get возвращает конфигурацию для метода
func (r *RateLimitedMethods) Get(method string) *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cfg, ok := r.methods[method]; ok {
		return cfg
	}
	return r.defaultConfig
}
