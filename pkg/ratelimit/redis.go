package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter Redis-based rate limiter
type RedisLimiter struct {
	client *redis.Client
	config *Config
	script *redis.Script
}

// NewRedisLimiter создаёт Redis rate limiter
func NewRedisLimiter(cfg *Config) (*RedisLimiter, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	// Lua скрипт для атомарной проверки и инкремента
	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local count = tonumber(ARGV[4])

		-- Удаляем устаревшие записи
		redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

		-- Считаем текущие запросы
		local current = redis.call('ZCARD', key)

		if current + count <= limit then
			-- Добавляем новые запросы
			for i = 1, count do
				redis.call('ZADD', key, now, now .. ':' .. i .. ':' .. math.random())
			end
			redis.call('EXPIRE', key, window / 1000 + 1)
			return {1, limit - current - count}
		end

		return {0, 0}
	`)

	return &RedisLimiter{
		client: client,
		config: cfg,
		script: script,
	}, nil
}

func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

func (l *RedisLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	now := time.Now().UnixMilli()
	window := l.config.Window.Milliseconds()

	result, err := l.script.Run(ctx, l.client, []string{redisKey},
		l.config.Requests, window, now, n).Slice()
	if err != nil {
		return false, fmt.Errorf("redis script error: %w", err)
	}

	if len(result) == 0 {
		return false, fmt.Errorf("unexpected empty result from redis script")
	}

	allowed, ok := result[0].(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from redis script")
	}

	return allowed == 1, nil
}

func (l *RedisLimiter) Wait(ctx context.Context, key string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			allowed, err := l.Allow(ctx, key)
			if err != nil {
				return err
			}
			if allowed {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

func (l *RedisLimiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	return l.client.Del(ctx, redisKey).Err()
}

func (l *RedisLimiter) GetInfo(ctx context.Context, key string) (*LimitInfo, error) {
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	now := time.Now()
	windowStart := now.Add(-l.config.Window).UnixMilli()

	count, err := l.client.ZCount(ctx, redisKey, strconv.FormatInt(windowStart, 10), "+inf").Result()
	if err != nil {
		return nil, err
	}

	remaining := l.config.Requests - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return &LimitInfo{
		Limit:     l.config.Requests,
		Remaining: remaining,
		ResetAt:   now.Add(l.config.Window),
	}, nil
}

func (l *RedisLimiter) Close() error {
	return l.client.Close()
}
