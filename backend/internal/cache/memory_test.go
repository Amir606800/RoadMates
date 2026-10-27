package cache

import (
	"context"
	"errors"
	"roadmates/backend/internal/service"
	"testing"
	"time"
)

func TestMemoryCacheContract(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()
	if _, err := c.Get(ctx, "missing"); !errors.Is(err, service.ErrCacheMiss) {
		t.Fatal("missing key did not miss")
	}
	input := []byte("safe")
	c.Set(ctx, "key", input, time.Minute)
	input[0] = 'X'
	output, _ := c.Get(ctx, "key")
	if string(output) != "safe" {
		t.Fatal("cache borrowed caller bytes")
	}
	output[0] = 'Y'
	output, _ = c.Get(ctx, "key")
	if string(output) != "safe" {
		t.Fatal("caller modified cached bytes")
	}
	c.Set(ctx, "expired", []byte("old"), -time.Second)
	if _, err := c.Get(ctx, "expired"); !errors.Is(err, service.ErrCacheMiss) {
		t.Fatal("expired item returned")
	}
	c.Delete(ctx, "key")
	if _, err := c.Get(ctx, "key"); !errors.Is(err, service.ErrCacheMiss) {
		t.Fatal("delete failed")
	}
	before, _ := c.NamespaceVersion(ctx, "rides")
	c.BumpNamespace(ctx, "rides")
	after, _ := c.NamespaceVersion(ctx, "rides")
	if before == after {
		t.Fatal("namespace did not change")
	}
}
