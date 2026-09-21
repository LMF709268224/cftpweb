package server

import (
	"context"
	"sync"
	"time"
)

const (
	userULIDCacheTTL        = 5 * time.Minute
	userULIDCacheMaxEntries = 10000
)

// TODO: Use Redis for this cache when the shared cache infrastructure is available,
// so multiple adminbff instances can share identity mappings and TTLs.

type userULIDCacheEntry struct {
	value     string
	expiresAt time.Time
}

// userULIDCache caches only successful identity mappings.
type userULIDCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	maxSize int
	entries map[string]userULIDCacheEntry
}

func newUserULIDCache(ttl time.Duration, maxSize int) *userULIDCache {
	return &userULIDCache{
		ttl:     ttl,
		maxSize: maxSize,
		entries: make(map[string]userULIDCacheEntry),
	}
}

func (c *userULIDCache) resolve(ctx context.Context, key string, load func(context.Context) (string, error)) (string, error) {
	now := time.Now()

	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		if now.Before(entry.expiresAt) {
			c.mu.Unlock()
			return entry.value, nil
		}
		delete(c.entries, key)
	}
	c.mu.Unlock()

	value, err := load(ctx)

	if err == nil && value != "" {
		c.mu.Lock()
		if c.maxSize > 0 && len(c.entries) >= c.maxSize {
			for oldKey := range c.entries {
				delete(c.entries, oldKey)
				break
			}
		}
		c.entries[key] = userULIDCacheEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
		c.mu.Unlock()
	}

	return value, err
}
