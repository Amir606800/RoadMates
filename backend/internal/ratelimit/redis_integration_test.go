package ratelimit

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestRedisSharedAtomicLimit(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("set TEST_REDIS_URL for Redis integration tests")
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal("invalid Redis test URL")
	}
	options.MaxRetries = -1
	options.ContextTimeoutEnabled = true
	client := redis.NewClient(options)
	defer client.Close()
	ctx := context.Background()
	if client.Ping(ctx).Err() != nil {
		t.Fatal("test Redis unavailable")
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	first, second := NewRedis(NewRedisCounter(client), logger), NewRedis(NewRedisCounter(client), logger)
	group := "test-" + uuid.NewString()
	key, _ := windowKey(group, "192.0.2.1", time.Hour, time.Now())
	defer client.Del(ctx, key)
	var allowed atomic.Int32
	var workers sync.WaitGroup
	for i := 0; i < 40; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			limiter := first
			if i%2 == 0 {
				limiter = second
			}
			if ok, _ := limiter.Allow(ctx, group, "192.0.2.1", 10, time.Hour); ok {
				allowed.Add(1)
			}
		}(i)
	}
	workers.Wait()
	if allowed.Load() != 10 {
		t.Fatal("two instances did not enforce one shared atomic Redis limit")
	}
	if ttl := client.TTL(ctx, key).Val(); ttl <= 0 || ttl > time.Hour+time.Second {
		t.Fatal("rate counter missing expiry")
	}
}
