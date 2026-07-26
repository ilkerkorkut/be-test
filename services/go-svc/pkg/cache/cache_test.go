package cache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

type unexpectedValueError struct {
	value int
}

func (e *unexpectedValueError) Error() string {
	return "unexpected cache value"
}
