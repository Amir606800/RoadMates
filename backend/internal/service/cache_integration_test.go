package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"roadmates/backend/internal/cache"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/repository"
	"roadmates/backend/internal/service"
)

type countingRideStore struct {
	service.RideStore
	calls atomic.Int32
}

func (s *countingRideStore) SearchIDs(ctx context.Context, filter domain.RideSearch) ([]string, error) {
	s.calls.Add(1)
	return s.RideStore.SearchIDs(ctx, filter)
}
func TestPostgresRedisRideSearch(t *testing.T) {
	pool := authDatabase(t)
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
	owner := uuid.NewString()
	if _, err = pool.Exec(ctx, `INSERT INTO users(id,email,password_hash,first_name,last_name,phone) VALUES($1,$2,'unused','PersonalName','Test','+994501112233')`, owner, owner+"@example.test"); err != nil {
		t.Fatal("could not create test user")
	}
	store := &countingRideStore{RideStore: repository.NewRides(pool)}
	adapter := cache.NewRedis(client)
	svc := service.NewRides(store, adapter, quietLogger())
	filter := domain.RideSearch{FromID: 9, ToID: 48}
	// Give this test a fresh search namespace; no global flush or enumeration.
	if adapter.BumpNamespace(ctx, "rides") != nil {
		t.Fatal("could not initialize namespace")
	}
	query := fmt.Sprintf("%d:%d:%s", filter.FromID, filter.ToID, filter.Date)
	keys := make([]string, 0)
	check := func(want int) {
		t.Helper()
		items, err := svc.Search(ctx, filter)
		if err != nil || len(items) != want {
			t.Fatalf("search returned %d rides, want %d", len(items), want)
		}
		version, _ := adapter.NamespaceVersion(ctx, "rides")
		key := fmt.Sprintf("roadmates:v1:rides:search:%s:%x", version, sha256.Sum256([]byte(query)))
		keys = append(keys, key)
	}
	defer func() {
		for _, key := range keys {
			client.Del(ctx, key)
		}
	}()
	check(0)
	check(0)
	if store.calls.Load() != 1 {
		t.Fatal("Redis did not cache the empty search")
	}
	input := domain.Ride{DriverID: owner, From: domain.District{ID: 9}, To: domain.District{ID: 48}, DepartureAt: time.Now().Add(time.Hour), SeatsTotal: 3, PriceCents: 2025, Notes: "Personal meeting details"}
	ride, err := svc.Publish(ctx, input)
	if err != nil {
		t.Fatal("publish failed")
	}
	check(1)
	check(1)
	if store.calls.Load() != 2 {
		t.Fatal("publish did not invalidate the Redis search")
	}
	key := keys[len(keys)-1]
	raw, err := adapter.Get(ctx, key)
	if err != nil {
		t.Fatal("cached search missing")
	}
	var ids []string
	if json.Unmarshal(raw, &ids) != nil || len(ids) != 1 || ids[0] != ride.ID {
		t.Fatal("cached payload is not an ID-only result")
	}
	if ttl := client.PTTL(ctx, key).Val(); ttl < 29*time.Second || ttl > 60*time.Second {
		t.Fatal("Redis search TTL not jittered within range")
	}
	if _, err = svc.Finish(ctx, ride.ID, owner, "cancelled"); err != nil {
		t.Fatal("cancel failed")
	}
	check(0)
	ride, err = svc.Publish(ctx, input)
	if err != nil {
		t.Fatal("republish failed")
	}
	check(1)
	if _, err = svc.Finish(ctx, ride.ID, owner, "completed"); err != nil {
		t.Fatal("complete failed")
	}
	check(0)
}
