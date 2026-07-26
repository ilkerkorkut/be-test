package cache

import (
	"context"
	"sync"
	"time"
)

type Loader[V any] func() (V, error)

type Store[V any] interface {
	GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader Loader[V]) (V, error)
}

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]entry[V]
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		entries: make(map[K]entry[V]),
	}
}

type memoryStore[V any] struct {
	cache *Cache[string, V]
}

func NewMemory[V any]() Store[V] {
	return &memoryStore[V]{
		cache: New[string, V](),
	}
}

func (c *memoryStore[V]) GetOrLoad(_ context.Context, key string, ttl time.Duration, loader Loader[V]) (V, error) {
	return c.cache.GetOrLoad(key, ttl, loader)
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			c.Delete(key)
		}
		var zero V
		return zero, false
	}

	return entry.value, true
}

func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = entry[V]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

func (c *Cache[K, V]) GetOrLoad(key K, ttl time.Duration, loader Loader[V]) (V, error) {
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok && !time.Now().After(entry.expiresAt) {
		return entry.value, nil
	}

	value, err := loader()
	if err != nil {
		var zero V
		return zero, err
	}

	c.entries[key] = entry[V]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return value, nil
}
