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

type userULIDCacheEntry struct {
	value     string
	expiresAt time.Time
}

type userULIDLookup struct {
	done  chan struct{}
	value string
	err   error
}

// userULIDCache caches only successful identity mappings. In-flight lookups for
// the same user share one gmid request during login bursts or page fan-out.
type userULIDCache struct {
	mu       sync.Mutex
	ttl      time.Duration
	maxSize  int
	entries  map[string]userULIDCacheEntry
	inflight map[string]*userULIDLookup
}

func newUserULIDCache(ttl time.Duration, maxSize int) *userULIDCache {
	return &userULIDCache{
		ttl:      ttl,
		maxSize:  maxSize,
		entries:  make(map[string]userULIDCacheEntry),
		inflight: make(map[string]*userULIDLookup),
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

	if pending, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-pending.done:
			return pending.value, pending.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	pending := &userULIDLookup{done: make(chan struct{})}
	c.inflight[key] = pending
	c.mu.Unlock()

	value, err := load(ctx)

	c.mu.Lock()
	if err == nil && value != "" {
		if c.maxSize > 0 && len(c.entries) >= c.maxSize {
			for oldKey := range c.entries {
				delete(c.entries, oldKey)
				break
			}
		}
		c.entries[key] = userULIDCacheEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
	}
	pending.value = value
	pending.err = err
	delete(c.inflight, key)
	close(pending.done)
	c.mu.Unlock()

	return value, err
}
