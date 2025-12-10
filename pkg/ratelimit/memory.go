// pkg/ratelimit/memory.go

package ratelimit

import (
	"context"
	"sync"
	"time"
)

type MemoryLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	config  *Config
	stopCh  chan struct{}
	closed  bool
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
	requests  []time.Time
}

func NewMemoryLimiter(cfg *Config) *MemoryLimiter {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// ✅ Нормализуем конфигурацию ДО использования
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 5 * time.Minute
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.Requests <= 0 {
		cfg.Requests = 100
	}
	if cfg.BurstSize <= 0 {
		cfg.BurstSize = 10
	}

	l := &MemoryLimiter{
		buckets: make(map[string]*bucket),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}

	go l.cleanup()

	return l
}

func (l *MemoryLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

func (l *MemoryLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if l.closed {
		return false, ErrLimiterClosed
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{
			tokens:    float64(l.config.Requests + l.config.BurstSize),
			lastCheck: time.Now(),
			requests:  make([]time.Time, 0),
		}
		l.buckets[key] = b
	}

	switch l.config.Strategy {
	case "token_bucket":
		return l.allowTokenBucket(b, n), nil
	case "sliding_window":
		return l.allowSlidingWindow(b, n), nil
	default:
		return l.allowSlidingWindow(b, n), nil
	}
}

func (l *MemoryLimiter) allowTokenBucket(b *bucket, n int) bool {
	now := time.Now()
	elapsed := now.Sub(b.lastCheck)
	b.lastCheck = now

	rate := float64(l.config.Requests) / l.config.Window.Seconds()
	b.tokens += elapsed.Seconds() * rate

	maxTokens := float64(l.config.Requests + l.config.BurstSize)
	if b.tokens > maxTokens {
		b.tokens = maxTokens
	}

	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true
	}

	return false
}

func (l *MemoryLimiter) allowSlidingWindow(b *bucket, n int) bool {
	now := time.Now()
	windowStart := now.Add(-l.config.Window)

	validRequests := make([]time.Time, 0, len(b.requests))
	for _, t := range b.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	b.requests = validRequests

	if len(b.requests)+n <= l.config.Requests {
		for i := 0; i < n; i++ {
			b.requests = append(b.requests, now)
		}
		return true
	}

	return false
}

func (l *MemoryLimiter) Wait(ctx context.Context, key string) error {
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

func (l *MemoryLimiter) Reset(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.buckets, key)
	return nil
}

func (l *MemoryLimiter) GetInfo(ctx context.Context, key string) (*LimitInfo, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	b, ok := l.buckets[key]
	if !ok {
		return &LimitInfo{
			Limit:     l.config.Requests,
			Remaining: l.config.Requests,
			ResetAt:   time.Now().Add(l.config.Window),
		}, nil
	}

	var remaining int
	switch l.config.Strategy {
	case "token_bucket":
		remaining = int(b.tokens)
	case "sliding_window":
		windowStart := time.Now().Add(-l.config.Window)
		count := 0
		for _, t := range b.requests {
			if t.After(windowStart) {
				count++
			}
		}
		remaining = l.config.Requests - count
	}

	if remaining < 0 {
		remaining = 0
	}

	return &LimitInfo{
		Limit:     l.config.Requests,
		Remaining: remaining,
		ResetAt:   time.Now().Add(l.config.Window),
	}, nil
}

func (l *MemoryLimiter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	close(l.stopCh)
	l.buckets = nil

	return nil
}

// ✅ ИСПРАВЛЕННАЯ функция cleanup
func (l *MemoryLimiter) cleanup() {
	// Защита от нулевого интервала
	interval := l.config.CleanupInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			l.doCleanup()
		}
	}
}

func (l *MemoryLimiter) doCleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.config.Window * 2)

	for key, b := range l.buckets {
		if len(b.requests) == 0 && b.lastCheck.Before(windowStart) {
			delete(l.buckets, key)
			continue
		}

		validRequests := make([]time.Time, 0)
		for _, t := range b.requests {
			if t.After(windowStart) {
				validRequests = append(validRequests, t)
			}
		}
		b.requests = validRequests
	}
}
