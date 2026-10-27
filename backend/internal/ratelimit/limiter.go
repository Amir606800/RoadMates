package ratelimit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type window struct {
	count int
	end   time.Time
}
type Memory struct {
	mu          sync.Mutex
	windows     map[string]window
	nextCleanup time.Time
}

func NewMemory() *Memory { return &Memory{windows: make(map[string]window)} }
func windowKey(group, ip string, period time.Duration, now time.Time) (string, time.Time) {
	start := now.Truncate(period)
	return fmt.Sprintf("roadmates:v1:ratelimit:%s:%x:%d", group, sha256.Sum256([]byte(ip)), start.UnixNano()), start.Add(period)
}
func (m *Memory) Allow(ctx context.Context, group, ip string, limit int, period time.Duration) (bool, time.Duration) {
	if ctx.Err() != nil {
		return false, time.Second
	}
	return m.allow(group, ip, limit, period, time.Now())
}
func (m *Memory) allow(group, ip string, limit int, period time.Duration, now time.Time) (bool, time.Duration) {
	key, end := windowKey(group, ip, period, now)
	m.mu.Lock()
	defer m.mu.Unlock()
	if !now.Before(m.nextCleanup) {
		for key, w := range m.windows {
			if !w.end.After(now) {
				delete(m.windows, key)
			}
		}
		m.nextCleanup = now.Add(time.Minute)
	}
	w, exists := m.windows[key]
	if !exists && len(m.windows) >= 10000 {
		return false, end.Sub(now)
	}
	w.end = end
	w.count++
	m.windows[key] = w
	return w.count <= limit, end.Sub(now)
}

type Counter interface {
	Increment(context.Context, string, time.Duration) (int64, error)
}
type RedisCounter struct{ client *redis.Client }

func NewRedisCounter(client *redis.Client) *RedisCounter { return &RedisCounter{client} }

// INCR and expiry must be atomic: a dropped connection cannot leave a counter
// without an expiry. Compatible with the project's Redis 7 image.
var incrementScript = redis.NewScript(`
local count = redis.call('INCR',KEYS[1])
if count == 1 then redis.call('EXPIRE',KEYS[1],ARGV[1]) end
return count`)

func (r *RedisCounter) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	seconds := int64((ttl + time.Second - 1) / time.Second)
	return incrementScript.Run(ctx, r.client, []string{key}, seconds).Int64()
}

type Redis struct {
	counter     Counter
	fallback    *Memory
	logger      *slog.Logger
	mu          sync.Mutex
	nextWarning time.Time
}

func NewRedis(counter Counter, logger *slog.Logger) *Redis {
	if logger == nil {
		logger = slog.Default()
	}
	return &Redis{counter: counter, fallback: NewMemory(), logger: logger}
}
func (r *Redis) Allow(ctx context.Context, group, ip string, limit int, period time.Duration) (bool, time.Duration) {
	now := time.Now()
	key, end := windowKey(group, ip, period, now)
	// Shadow all attempts locally, so an outage mid-window does not reset the
	// requests this process already observed while Redis was healthy.
	local, retry := r.fallback.allow(group, ip, limit, period, now)
	count, err := r.counter.Increment(ctx, key, end.Sub(now)+time.Second)
	if err == nil {
		return count <= int64(limit), end.Sub(now)
	}
	r.mu.Lock()
	if !now.Before(r.nextWarning) {
		r.logger.Warn("Redis rate limiting unavailable; using per-process limits")
		r.nextWarning = now.Add(30 * time.Second)
	}
	r.mu.Unlock()
	return local, retry
}
