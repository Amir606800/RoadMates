package ratelimit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type counter struct {
	mu     sync.Mutex
	counts map[string]int64
	down   atomic.Bool
}

func (c *counter) Increment(_ context.Context, key string, _ time.Duration) (int64, error) {
	if c.down.Load() {
		return 0, errors.New("unavailable")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[key]++
	return c.counts[key], nil
}
func TestSharedLimitAndOutageFallback(t *testing.T) {
	backend := &counter{counts: make(map[string]int64)}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	first, second := NewRedis(backend, logger), NewRedis(backend, logger)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		limiter := first
		if i%2 != 0 {
			limiter = second
		}
		ok, _ := limiter.Allow(ctx, "login", "192.0.2.1", 10, time.Minute)
		if !ok {
			t.Fatal("shared allowance rejected early")
		}
	}
	if ok, retry := second.Allow(ctx, "login", "192.0.2.1", 10, time.Minute); ok || retry <= 0 || retry > time.Minute {
		t.Fatal("instances did not share a fixed-window limit")
	}
	if ok, _ := first.Allow(ctx, "register", "192.0.2.1", 5, time.Minute); !ok {
		t.Fatal("route groups share a quota")
	}
	if ok, _ := first.Allow(ctx, "login", "192.0.2.2", 10, time.Minute); !ok {
		t.Fatal("IP addresses share a quota")
	}
	for i := 0; i < 3; i++ {
		first.Allow(ctx, "fallback", "192.0.2.1", 3, time.Minute)
	}
	backend.down.Store(true)
	if ok, retry := first.Allow(ctx, "fallback", "192.0.2.1", 3, time.Minute); ok || retry <= 0 {
		t.Fatal("outage reset the locally observed count")
	}
	for i := 0; i < 2; i++ {
		if ok, _ := second.Allow(ctx, "fallback-new", "192.0.2.1", 2, time.Minute); !ok {
			t.Fatal("outage blocked allowed request")
		}
	}
	if ok, _ := second.Allow(ctx, "fallback-new", "192.0.2.1", 2, time.Minute); ok {
		t.Fatal("outage disabled rate limiting")
	}
}
func TestMemoryWindowRolloverAndConcurrentLimit(t *testing.T) {
	limiter := NewMemory()
	start := time.Unix(100000, 0).Truncate(time.Minute)
	for i := 0; i < 2; i++ {
		if ok, _ := limiter.allow("login", "192.0.2.1", 2, time.Minute, start); !ok {
			t.Fatal("request rejected early")
		}
	}
	if ok, _ := limiter.allow("login", "192.0.2.1", 2, time.Minute, start); ok {
		t.Fatal("limit exceeded")
	}
	if ok, _ := limiter.allow("login", "192.0.2.1", 2, time.Minute, start.Add(time.Minute)); !ok {
		t.Fatal("new window did not reset")
	}
	var wins atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 100; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if ok, _ := limiter.Allow(context.Background(), "concurrent", "192.0.2.2", 7, time.Minute); ok {
				wins.Add(1)
			}
		}()
	}
	group.Wait()
	if wins.Load() != 7 {
		t.Fatal("concurrent memory limit was not atomic")
	}
}
