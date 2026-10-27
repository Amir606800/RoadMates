package cache

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"roadmates/backend/internal/service"
)

func TestRedisCacheContract(t *testing.T) {
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
	c := NewRedis(client)
	namespace := "test-" + uuid.NewString()
	key := "roadmates:v1:" + namespace + ":data"
	versionKey := "roadmates:v1:" + namespace + ":version"
	defer client.Del(ctx, key, versionKey)
	if _, err := c.Get(ctx, key); !errors.Is(err, service.ErrCacheMiss) {
		t.Fatal("missing Redis key did not miss")
	}
	if c.Set(ctx, key, []byte("value"), time.Minute) != nil {
		t.Fatal("Redis write failed")
	}
	if got, err := c.Get(ctx, key); err != nil || string(got) != "value" {
		t.Fatal("Redis round trip failed")
	}
	if ttl := client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatal("Redis value has no correct expiry")
	}
	before, err := c.NamespaceVersion(ctx, namespace)
	if err != nil {
		t.Fatal("namespace lookup failed")
	}
	if c.BumpNamespace(ctx, namespace) != nil {
		t.Fatal("namespace bump failed")
	}
	after, _ := c.NamespaceVersion(ctx, namespace)
	if before == after {
		t.Fatal("bump reused a namespace")
	}
	client.Del(ctx, versionKey)
	recreated, err := c.NamespaceVersion(ctx, namespace)
	if err != nil || recreated == before || recreated == after {
		t.Fatal("evicted namespace reused old search keys")
	}
	if c.Delete(ctx, key) != nil {
		t.Fatal("Redis delete failed")
	}
	if _, err := c.Get(ctx, key); !errors.Is(err, service.ErrCacheMiss) {
		t.Fatal("deleted key returned")
	}
}
func TestRedisConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	client := redis.NewClient(&redis.Options{Addr: address, MaxRetries: -1, ContextTimeoutEnabled: true, DialTimeout: 100 * time.Millisecond})
	defer client.Close()
	c := NewRedis(client)
	ctx := context.Background()
	if _, err = c.Get(ctx, "roadmates:v1:test:down"); err == nil {
		t.Fatal("closed Redis endpoint succeeded")
	}
	if c.Set(ctx, "roadmates:v1:test:down", []byte("value"), time.Minute) == nil {
		t.Fatal("closed Redis write succeeded")
	}
	if _, err = c.NamespaceVersion(ctx, "test-down"); err == nil {
		t.Fatal("closed Redis namespace succeeded")
	}
}
