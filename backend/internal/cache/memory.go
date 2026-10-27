package cache

import (
	"context"
	"roadmates/backend/internal/service"
	"strconv"
	"sync"
	"time"
)

type entry struct {
	data    []byte
	expires time.Time
}

// Memory implements the same contract for deterministic, Redis-free tests.
type Memory struct {
	mu       sync.Mutex
	entries  map[string]entry
	versions map[string]uint64
}

func NewMemory() *Memory {
	return &Memory{entries: make(map[string]entry), versions: make(map[string]uint64)}
}
func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.entries[key]
	if !ok || !item.expires.After(time.Now()) {
		delete(m.entries, key)
		return nil, service.ErrCacheMiss
	}
	return append([]byte(nil), item.data...), nil
}
func (m *Memory) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = entry{append([]byte(nil), data...), time.Now().Add(ttl)}
	return nil
}
func (m *Memory) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
	return nil
}
func (m *Memory) NamespaceVersion(ctx context.Context, namespace string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return strconv.FormatUint(m.versions[namespace], 10), nil
}
func (m *Memory) BumpNamespace(ctx context.Context, namespace string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions[namespace]++
	return nil
}
