package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestCacheExpiresEntries(t *testing.T) {
	cache := New[string, int]()
	cache.Set("value", 42, -time.Second)

	if _, ok := cache.Get("value"); ok {
		t.Fatal("expired cache entry was returned")
	}
}

func TestCacheGetOrLoadDeduplicatesParallelLoads(t *testing.T) {
	cache := New[string, int]()
	var loads atomic.Int32
	var waitGroup sync.WaitGroup
	errors := make(chan error, 50)

	for index := 0; index < cap(errors); index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			value, err := cache.GetOrLoad("shared", time.Minute, func() (int, error) {
				loads.Add(1)
				return 42, nil
			})
			if err != nil {
				errors <- err
				return
			}
			if value != 42 {
				errors <- &unexpectedValueError{value: value}
			}
		}()
	}

	waitGroup.Wait()
	close(errors)
	for err := range errors {
		t.Fatalf("cache load: %v", err)
	}
	if loads.Load() != 1 {
		t.Fatalf("loader calls = %d, want 1", loads.Load())
	}
}

func TestRedisCacheFallsBackWhenRedisIsUnavailable(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 10 * time.Millisecond,
		ReadTimeout: 10 * time.Millisecond,
		MaxRetries:  0,
	})
	defer client.Close()

	value, err := NewRedis[string](client, "test").GetOrLoad(context.Background(), "sports", time.Minute, func() (string, error) {
		return "provider-value", nil
	})
	if err != nil {
		t.Fatalf("load through unavailable redis: %v", err)
	}
	if value != "provider-value" {
		t.Fatalf("value = %q, want provider-value", value)
	}
}

type unexpectedValueError struct {
	value int
}

func (e *unexpectedValueError) Error() string {
	return "unexpected cache value"
}
