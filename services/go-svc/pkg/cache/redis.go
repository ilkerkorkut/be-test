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
	client redis.UniversalClient
	prefix string
	mu     sync.Mutex
}

func NewRedisClient(redisURL string) (redis.UniversalClient, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	return redis.NewClient(options), nil
}

func NewRedis[V any](client redis.UniversalClient, prefix string) Store[V] {
	return &redisStore[V]{
		client: client,
		prefix: prefix,
	}
}

func (c *redisStore[V]) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader Loader[V]) (V, error) {
	value, found, err := c.get(ctx, key)
	if err != nil {
		var zero V
		return zero, err
	}
	if found {
		return value, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	value, found, err = c.get(ctx, key)
	if err != nil {
		var zero V
		return zero, err
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
		var zero V
		return zero, err
	}

	return value, nil
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
