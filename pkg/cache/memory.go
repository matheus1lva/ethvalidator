package cache

import (
	"sync"
	"time"
)

type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]cacheItem
	ttl      time.Duration
	maxSize  int
	stopChan chan struct{}
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

func NewMemoryCache(ttl time.Duration, maxSize int) *MemoryCache {
	c := &MemoryCache{
		items:    make(map[string]cacheItem),
		ttl:      ttl,
		maxSize:  maxSize,
		stopChan: make(chan struct{}),
	}

	go c.cleanupExpired()

	return c
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

func (c *MemoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}

func (c *MemoryCache) Close() {
	close(c.stopChan)
}

func (c *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopChan:
			return
		}
	}
}

func (c *MemoryCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiration) {
			delete(c.items, key)
		}
	}
}

func (c *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestTime.IsZero() || item.expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.expiration
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}
