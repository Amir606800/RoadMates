package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var ErrCacheMiss = errors.New("cache miss")

// Only public district data and ride-search IDs pass through this interface.
// Auth, user profiles, application data, PINs, and ride notes are never cached.
type Cache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	Delete(context.Context, string) error
	NamespaceVersion(context.Context, string) (string, error)
	BumpNamespace(context.Context, string) error
}

type NoopCache struct{}

func (NoopCache) Get(context.Context, string) ([]byte, error)              { return nil, ErrCacheMiss }
func (NoopCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (NoopCache) Delete(context.Context, string) error                     { return nil }
func (NoopCache) NamespaceVersion(context.Context, string) (string, error) { return "0", nil }
func (NoopCache) BumpNamespace(context.Context, string) error              { return nil }

type cachedQueries struct {
	cache       Cache
	logger      *slog.Logger
	flights     singleflight.Group
	mu          sync.Mutex
	nextWarning time.Time
}

func newCachedQueries(cache Cache, logger *slog.Logger) *cachedQueries {
	if cache == nil {
		cache = NoopCache{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &cachedQueries{cache: cache, logger: logger}
}
func (q *cachedQueries) warn() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if time.Now().Before(q.nextWarning) {
		return
	}
	q.nextWarning = time.Now().Add(30 * time.Second)
	// Do not log raw errors, connection URLs, cache keys, or payloads.
	q.logger.Warn("cache unavailable or invalid; using PostgreSQL")
}

func cached[T any](ctx context.Context, q *cachedQueries, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	result := q.flights.DoChan(key, func() (any, error) {
		// A cancelled caller must not cancel the shared database read for others.
		work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		data, err := q.cache.Get(work, key)
		var value T
		if err == nil && json.Unmarshal(data, &value) == nil {
			return value, nil
		}
		writable := err == nil || errors.Is(err, ErrCacheMiss)
		if !errors.Is(err, ErrCacheMiss) {
			q.warn()
		}
		value, err = load(work)
		if err != nil {
			return zero, err
		}
		if writable {
			if data, encodeErr := json.Marshal(value); encodeErr == nil {
				if q.cache.Set(work, key, data, ttl) != nil {
					q.warn()
				}
			}
		}
		return value, nil
	})
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case result := <-result:
		if result.Err != nil {
			return zero, result.Err
		}
		return result.Val.(T), nil
	}
}

type RateLimiter interface {
	Allow(context.Context, string, string, int, time.Duration) (bool, time.Duration)
}
