// Package cache provides a flexible caching interface and implementations
// for in-memory and Redis-backed caches.
package cache

import (
	"context"
	"errors"
	"time"

	"logistics/pkg/config"
)

// Backend types for cache implementations.
const (
	// BackendMemory specifies an in-memory cache backend.
	BackendMemory = "memory"
	// BackendRedis specifies a Redis cache backend.
	BackendRedis = "redis"
)

// Standard errors returned by cache operations.
var (
	// ErrKeyNotFound is returned when a requested key does not exist in the cache.
	ErrKeyNotFound = errors.New("key not found")
	// ErrCacheClosed is returned when an operation is attempted on a closed cache.
	ErrCacheClosed = errors.New("cache is closed")
)

// Cache is an interface that defines common operations for various cache implementations.
type Cache interface {
	// Basic operations

	// Get retrieves the value associated with the given key.
	// Returns ErrKeyNotFound if the key does not exist.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores a value for the given key with a specified time-to-live (TTL).
	// If the key already exists, its value and TTL are updated.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Delete removes the key-value pair from the cache.
	// Returns nil if the key was not found or successfully deleted.
	Delete(ctx context.Context, key string) error
	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// Operations with TTL

	// GetWithTTL retrieves the value and its remaining TTL for the given key.
	// Returns ErrKeyNotFound if the key does not exist.
	GetWithTTL(ctx context.Context, key string) (value []byte, ttl time.Duration, err error)

	// Multiple operations

	// MGet retrieves multiple values for the given keys.
	// It returns a map of existing keys to their values. Keys not found will not be in the map.
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	// MSet stores multiple key-value pairs with a specified TTL.
	MSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) error
	// MDelete removes multiple key-value pairs from the cache.
	// Returns the number of keys that were actually deleted.
	MDelete(ctx context.Context, keys []string) (int64, error)

	// Pattern-based operations

	// Keys returns all keys matching a given pattern.
	// Note: Use with caution on large caches as it can be resource-intensive.
	Keys(ctx context.Context, pattern string) ([]string, error)
	// DeleteByPattern removes all keys matching a given pattern.
	// Returns the number of keys that were deleted.
	// Note: Use with caution on large caches as it can be resource-intensive.
	DeleteByPattern(ctx context.Context, pattern string) (int64, error)

	// Management operations

	// Stats returns statistics about the cache.
	Stats(ctx context.Context) (*Stats, error)
	// Clear removes all keys from the cache.
	Clear(ctx context.Context) error
	// Close shuts down the cache and releases any underlying resources.
	Close() error
}

// Stats holds various statistics about a cache's performance and state.
type Stats struct {
	TotalKeys    int64            // Total number of keys currently in the cache.
	Hits         int64            // Number of successful cache retrievals.
	Misses       int64            // Number of failed cache retrievals.
	HitRate      float64          // Ratio of hits to total lookups.
	MemoryBytes  int64            // Current memory consumption of the cache in bytes.
	KeysByPrefix map[string]int64 // Optional: Number of keys grouped by common prefixes.
	Backend      string           // The name of the cache backend (e.g., "memory", "redis").
}

// Options contains configuration parameters for creating a Cache instance.
type Options struct {
	Backend    string        // The desired cache backend: BackendMemory or BackendRedis.
	DefaultTTL time.Duration // The default time-to-live for cache entries if not specified per operation.

	// Memory cache specific options
	MaxEntries      int           // Maximum number of entries for the memory cache.
	MaxMemoryBytes  int64         // Maximum memory in bytes for the memory cache.
	CleanupInterval time.Duration // Interval for background cleanup of expired entries in memory cache.

	// Redis cache specific options
	RedisAddr     string // Address of the Redis server (e.g., "localhost:6379").
	RedisPassword string // Password for Redis authentication.
	RedisDB       int    // Redis database number to use.
	RedisPoolSize int    // Maximum number of connections in the Redis client pool.
}

// DefaultOptions returns a new Options struct with sensible default values.
func DefaultOptions() *Options {
	return &Options{
		Backend:         BackendMemory,
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      100000,
		MaxMemoryBytes:  256 * 1024 * 1024,
		CleanupInterval: 1 * time.Minute,
		RedisAddr:       "localhost:6379",
		RedisDB:         0,
		RedisPoolSize:   10,
	}
}

// FromConfig создаёт опции из конфигурации
func FromConfig(cfg *config.CacheConfig) *Options {
	return &Options{
		Backend:       cfg.Driver,
		DefaultTTL:    cfg.DefaultTTL,
		MaxEntries:    cfg.MaxEntries,
		RedisAddr:     cfg.Address(),
		RedisPassword: cfg.Password,
		RedisDB:       cfg.DB,
		RedisPoolSize: 10,
	}
}

// New создаёт кэш на основе опций
func New(opts *Options) (Cache, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	switch opts.Backend {
	case BackendRedis:
		return NewRedisCache(opts)
	case BackendMemory, "":
		return NewMemoryCache(opts), nil
	default:
		return NewMemoryCache(opts), nil
	}
}

// MustNew создаёт кэш или паникует
func MustNew(opts *Options) Cache {
	c, err := New(opts)
	if err != nil {
		panic(err)
	}
	return c
}
