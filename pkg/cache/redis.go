package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis реализация кэша
type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewRedisCache создаёт новый Redis кэш
func NewRedisCache(opts *Options) (*RedisCache, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	poolSize := opts.RedisPoolSize
	if poolSize <= 0 {
		poolSize = 10
	}

	client := redis.NewClient(&redis.Options{
		Addr:     opts.RedisAddr,
		Password: opts.RedisPassword,
		DB:       opts.RedisDB,
		PoolSize: poolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisCache{
		client:     client,
		defaultTTL: opts.DefaultTTL,
	}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return val, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (c *RedisCache) GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error) {
	pipe := c.client.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, 0, err
	}

	val, err := getCmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, 0, ErrKeyNotFound
		}
		return nil, 0, err
	}

	ttl := ttlCmd.Val()
	if ttl < 0 {
		ttl = 0
	}

	return val, ttl, nil
}

func (c *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(vals))
	for i, val := range vals {
		if val != nil {
			if str, ok := val.(string); ok {
				result[keys[i]] = []byte(str)
			}
		}
	}

	return result, nil
}

func (c *RedisCache) MSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	if len(entries) == 0 {
		return nil
	}

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	pipe := c.client.Pipeline()
	for key, value := range entries {
		pipe.Set(ctx, key, value, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisCache) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return c.client.Del(ctx, keys...).Result()
}

func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

func (c *RedisCache) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	return c.client.Del(ctx, keys...).Result()
}

func (c *RedisCache) Stats(ctx context.Context) (*Stats, error) {
	info, err := c.client.Info(ctx, "stats", "memory", "keyspace").Result()
	if err != nil {
		return nil, err
	}

	stats := &Stats{
		KeysByPrefix: make(map[string]int64),
		Backend:      "redis",
	}

	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "keyspace_hits:"):
			parseStatLine(line, "keyspace_hits:%d", &stats.Hits)
		case strings.HasPrefix(line, "keyspace_misses:"):
			parseStatLine(line, "keyspace_misses:%d", &stats.Misses)
		case strings.HasPrefix(line, "used_memory:"):
			parseStatLine(line, "used_memory:%d", &stats.MemoryBytes)
		}
	}

	dbSize, err := c.client.DBSize(ctx).Result()
	if err == nil {
		stats.TotalKeys = dbSize
	}

	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats, nil
}

// parseStatLine парсит строку статистики Redis (best-effort, ошибки игнорируются)
func parseStatLine(line, format string, target *int64) {
	// Best-effort parsing - ошибки игнорируются для статистики
	if _, err := fmt.Sscanf(line, format, target); err != nil {
		// Статистика не критична, продолжаем с нулевым значением
		return
	}
}

func (c *RedisCache) Clear(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
