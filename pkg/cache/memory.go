package cache

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCache in-memory реализация кэша с LRU eviction
type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]*cacheItem
	defaultTTL time.Duration
	maxEntries int

	// Статистика
	hits   atomic.Int64
	misses atomic.Int64

	// Lifecycle
	closed atomic.Bool
	stopCh chan struct{}
	wg     sync.WaitGroup
}

type cacheItem struct {
	value      []byte
	expiresAt  time.Time
	accessedAt time.Time
	size       int64
}

func (i *cacheItem) isExpired() bool {
	if i.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.expiresAt)
}

func (i *cacheItem) ttl() time.Duration {
	if i.expiresAt.IsZero() {
		return -1 // Бессрочный
	}
	ttl := time.Until(i.expiresAt)
	if ttl < 0 {
		return 0
	}
	return ttl
}

// NewMemoryCache создаёт новый in-memory кэш

func NewMemoryCache(opts *Options) *MemoryCache {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Валидация параметров
	maxEntries := opts.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 100000
	}

	cleanupInterval := opts.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Minute
	}

	c := &MemoryCache{
		items:      make(map[string]*cacheItem),
		defaultTTL: opts.DefaultTTL,
		maxEntries: maxEntries,
		stopCh:     make(chan struct{}),
	}

	// Запускаем фоновую очистку
	c.wg.Add(1)
	go c.cleanupLoop(cleanupInterval)

	return c
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	if c.closed.Load() {
		return nil, ErrCacheClosed
	}

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		c.misses.Add(1)
		return nil, ErrKeyNotFound
	}

	c.hits.Add(1)

	// Обновляем время доступа (для LRU)
	c.mu.Lock()
	item.accessedAt = time.Now()
	c.mu.Unlock()

	// Возвращаем копию
	result := make([]byte, len(item.value))
	copy(result, item.value)
	return result, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Копируем значение
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Eviction если нужно
	for len(c.items) >= c.maxEntries {
		c.evictLRU()
	}

	c.items[key] = &cacheItem{
		value:      valueCopy,
		expiresAt:  expiresAt,
		accessedAt: now,
		size:       int64(len(valueCopy)),
	}

	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()

	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if c.closed.Load() {
		return false, ErrCacheClosed
	}

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	return ok && !item.isExpired(), nil
}

func (c *MemoryCache) GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error) {
	if c.closed.Load() {
		return nil, 0, ErrCacheClosed
	}

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		c.misses.Add(1)
		return nil, 0, ErrKeyNotFound
	}

	c.hits.Add(1)

	c.mu.Lock()
	item.accessedAt = time.Now()
	c.mu.Unlock()

	result := make([]byte, len(item.value))
	copy(result, item.value)
	return result, item.ttl(), nil
}

func (c *MemoryCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if c.closed.Load() {
		return nil, ErrCacheClosed
	}

	result := make(map[string][]byte, len(keys))
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		if item, ok := c.items[key]; ok && !item.isExpired() {
			c.hits.Add(1)
			item.accessedAt = now

			valueCopy := make([]byte, len(item.value))
			copy(valueCopy, item.value)
			result[key] = valueCopy
		} else {
			c.misses.Add(1)
		}
	}

	return result, nil
}

func (c *MemoryCache) MSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range entries {
		for len(c.items) >= c.maxEntries {
			c.evictLRU()
		}

		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		c.items[key] = &cacheItem{
			value:      valueCopy,
			expiresAt:  expiresAt,
			accessedAt: now,
			size:       int64(len(valueCopy)),
		}
	}

	return nil
}

func (c *MemoryCache) MDelete(ctx context.Context, keys []string) (int64, error) {
	if c.closed.Load() {
		return 0, ErrCacheClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var count int64
	for _, key := range keys {
		if _, ok := c.items[key]; ok {
			delete(c.items, key)
			count++
		}
	}

	return count, nil
}

func (c *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if c.closed.Load() {
		return nil, ErrCacheClosed
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var keys []string
	for key, item := range c.items {
		if !item.isExpired() && matchPattern(pattern, key) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (c *MemoryCache) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	if c.closed.Load() {
		return 0, ErrCacheClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var count int64
	for key := range c.items {
		if matchPattern(pattern, key) {
			delete(c.items, key)
			count++
		}
	}

	return count, nil
}

func (c *MemoryCache) Stats(ctx context.Context) (*Stats, error) {
	if c.closed.Load() {
		return nil, ErrCacheClosed
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &Stats{
		TotalKeys:    int64(len(c.items)),
		Hits:         c.hits.Load(),
		Misses:       c.misses.Load(),
		KeysByPrefix: make(map[string]int64),
		Backend:      "memory",
	}

	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	for key, item := range c.items {
		if !item.isExpired() {
			stats.MemoryBytes += item.size
			prefix := extractPrefix(key)
			stats.KeysByPrefix[prefix]++
		}
	}

	return stats, nil
}

func (c *MemoryCache) Clear(ctx context.Context) error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	c.mu.Lock()
	c.items = make(map[string]*cacheItem)
	c.mu.Unlock()

	return nil
}

func (c *MemoryCache) Close() error {
	if c.closed.Swap(true) {
		return nil // Уже закрыт
	}

	close(c.stopCh)
	c.wg.Wait()

	c.mu.Lock()
	c.items = nil
	c.mu.Unlock()

	return nil
}

func (c *MemoryCache) cleanupLoop(interval time.Duration) {
	defer c.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.isExpired() {
			delete(c.items, key)
		}
	}
}

func (c *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestAccess time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.accessedAt.Before(oldestAccess) {
			oldestKey = key
			oldestAccess = item.accessedAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// matchPattern проверяет соответствие ключа паттерну
// Поддерживает:
//   - "*" — любой ключ
//   - "prefix*" — ключи, начинающиеся с prefix
//   - "*suffix" — ключи, заканчивающиеся на suffix
//   - "prefix*suffix" — ключи, начинающиеся с prefix и заканчивающиеся на suffix
func matchPattern(pattern, key string) bool {
	if pattern == "*" {
		return true
	}

	// Проверяем наличие wildcard
	starIndex := strings.Index(pattern, "*")
	if starIndex == -1 {
		// Нет wildcard — точное совпадение
		return pattern == key
	}

	// Разбиваем на prefix и suffix
	prefix := pattern[:starIndex]
	suffix := pattern[starIndex+1:]

	// Проверяем что ключ начинается с prefix и заканчивается на suffix
	// и что prefix + suffix не длиннее ключа
	if len(key) < len(prefix)+len(suffix) {
		return false
	}

	return strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix)
}

// extractPrefix извлекает префикс ключа
func extractPrefix(key string) string {
	if idx := strings.Index(key, ":"); idx > 0 {
		return key[:idx]
	}
	return "other"
}
