package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStore[V any] struct {
	client   redis.UniversalClient
	prefix   string
	fallback Store[V]
	mu       sync.Mutex
}

const (
	redisLockTTL      = 5 * time.Second
	redisLockWaitTime = 2 * time.Second
	redisLockPollTime = 50 * time.Millisecond
)

func NewRedisClient(redisURL string) (redis.UniversalClient, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	return redis.NewClient(options), nil
}

func NewRedis[V any](client redis.UniversalClient, prefix string) Store[V] {
	return NewRedisWithFallback(client, prefix, NewMemory[V]())
}

func NewRedisWithFallback[V any](client redis.UniversalClient, prefix string, fallback Store[V]) Store[V] {
	return &redisStore[V]{
		client:   client,
		prefix:   prefix,
		fallback: fallback,
	}
}

func (c *redisStore[V]) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader Loader[V]) (V, error) {
	value, found, err := c.get(ctx, key)
	if err != nil {
		return c.loadFromFallback(ctx, key, ttl, loader)
	}
	if found {
		return value, nil
	}

	locked, err := c.client.SetNX(ctx, c.lockKey(key), "1", redisLockTTL).Result()
	if err != nil {
		return c.loadFromFallback(ctx, key, ttl, loader)
	}
	if !locked {
		return c.waitForValue(ctx, key, ttl, loader)
	}
	defer c.client.Del(context.WithoutCancel(ctx), c.lockKey(key))

	c.mu.Lock()
	defer c.mu.Unlock()

	value, found, err = c.get(ctx, key)
	if err != nil {
		return c.loadFromFallback(ctx, key, ttl, loader)
	}
	if found {
		return value, nil
	}

	value, err = loader()
	if err != nil {
		var zero V
		return zero, err
	}
	if err := c.set(ctx, key, value, ttl); err != nil {
		return c.loadFromFallback(ctx, key, ttl, func() (V, error) { return value, nil })
	}

	return value, nil
}

func (c *redisStore[V]) waitForValue(ctx context.Context, key string, ttl time.Duration, loader Loader[V]) (V, error) {
	timer := time.NewTimer(redisLockWaitTime)
	defer timer.Stop()
	ticker := time.NewTicker(redisLockPollTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		case <-timer.C:
			return c.loadFromFallback(ctx, key, ttl, loader)
		case <-ticker.C:
			value, found, err := c.get(ctx, key)
			if err != nil {
				return c.loadFromFallback(ctx, key, ttl, loader)
			}
			if found {
				return value, nil
			}
		}
	}
}

func (c *redisStore[V]) loadFromFallback(ctx context.Context, key string, ttl time.Duration, loader Loader[V]) (V, error) {
	return c.fallback.GetOrLoad(ctx, key, ttl, loader)
}

func (c *redisStore[V]) get(ctx context.Context, key string) (V, bool, error) {
	value, err := c.client.Get(ctx, c.prefixedKey(key)).Bytes()
	if err == redis.Nil {
		var zero V
		return zero, false, nil
	}
	if err != nil {
		var zero V
		return zero, false, fmt.Errorf("get redis cache value: %w", err)
	}

	var result V
	if err := json.Unmarshal(value, &result); err != nil {
		var zero V
		return zero, false, fmt.Errorf("decode redis cache value: %w", err)
	}

	return result, true, nil
}

func (c *redisStore[V]) set(ctx context.Context, key string, value V, ttl time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode redis cache value: %w", err)
	}
	if err := c.client.Set(ctx, c.prefixedKey(key), encoded, ttl).Err(); err != nil {
		return fmt.Errorf("set redis cache value: %w", err)
	}

	return nil
}

func (c *redisStore[V]) prefixedKey(key string) string {
	return c.prefix + ":" + key
}

func (c *redisStore[V]) lockKey(key string) string {
	return c.prefix + ":lock:" + key
}
