package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/cache"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/service"
)

type cacheProbe struct {
	service.Cache
	down     atomic.Bool
	failSet  atomic.Bool
	mu       sync.Mutex
	keys     []string
	payloads [][]byte
	ttls     []time.Duration
}

func (c *cacheProbe) Get(ctx context.Context, key string) ([]byte, error) {
	if c.down.Load() {
		return nil, errors.New("unavailable")
	}
	return c.Cache.Get(ctx, key)
}
func (c *cacheProbe) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c.down.Load() || c.failSet.Load() {
		return errors.New("unavailable")
	}
	c.mu.Lock()
	c.keys = append(c.keys, key)
	c.payloads = append(c.payloads, append([]byte(nil), value...))
	c.ttls = append(c.ttls, ttl)
	c.mu.Unlock()
	return c.Cache.Set(ctx, key, value, ttl)
}
func (c *cacheProbe) NamespaceVersion(ctx context.Context, namespace string) (string, error) {
	if c.down.Load() {
		return "", errors.New("unavailable")
	}
	return c.Cache.NamespaceVersion(ctx, namespace)
}
func (c *cacheProbe) BumpNamespace(ctx context.Context, namespace string) error {
	if c.down.Load() {
		return errors.New("unavailable")
	}
	return c.Cache.BumpNamespace(ctx, namespace)
}

type districtCounter struct {
	calls            atomic.Int32
	entered, release chan struct{}
}

func (d *districtCounter) List(ctx context.Context, q string) ([]domain.District, error) {
	d.calls.Add(1)
	if d.entered != nil {
		close(d.entered)
		select {
		case <-d.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return []domain.District{{ID: 9, NameAZ: "Şamaxı", Slug: "samaxi"}}, nil
}
func quietLogger() *slog.Logger { return slog.New(slog.NewJSONHandler(io.Discard, nil)) }
func TestDistrictCache(t *testing.T) {
	ctx := context.Background()
	probe := &cacheProbe{Cache: cache.NewMemory()}
	store := &districtCounter{}
	svc := service.NewDistricts(store, probe, quietLogger())
	for _, query := range []string{"samaxi", " ŞAMAXI ", "Şamaxı"} {
		items, err := svc.List(ctx, query)
		if err != nil || len(items) != 1 {
			t.Fatal("district lookup failed")
		}
	}
	if store.calls.Load() != 1 || len(probe.ttls) != 1 || probe.ttls[0] != 24*time.Hour {
		t.Fatal("district cache did not normalize searches or use a 24 hour TTL")
	}
	probe.Cache.Set(ctx, probe.keys[0], []byte("invalid JSON"), time.Hour)
	if _, err := svc.List(ctx, "samaxi"); err != nil || store.calls.Load() != 2 {
		t.Fatal("invalid cache data did not fall back")
	}
	probe.down.Store(true)
	if _, err := svc.List(ctx, "samaxi"); err != nil || store.calls.Load() != 3 {
		t.Fatal("cache outage became a district error")
	}
	probe.down.Store(false)
	probe.failSet.Store(true)
	if _, err := svc.List(ctx, "new search"); err != nil {
		t.Fatal("cache write failure became a district error")
	}
}
func TestDistrictSingleflightSurvivesCallerCancellation(t *testing.T) {
	store := &districtCounter{entered: make(chan struct{}), release: make(chan struct{})}
	svc := service.NewDistricts(store, cache.NewMemory(), quietLogger())
	first, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() { _, err := svc.List(first, "samaxi"); firstResult <- err }()
	<-store.entered
	var group sync.WaitGroup
	results := make(chan error, 50)
	for i := 0; i < 50; i++ {
		group.Add(1)
		go func() { defer group.Done(); _, err := svc.List(context.Background(), "Şamaxı"); results <- err }()
	}
	cancel()
	if !errors.Is(<-firstResult, context.Canceled) {
		t.Fatal("cancelled caller did not return")
	}
	close(store.release)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal("one cancellation broke shared readers")
		}
	}
	if store.calls.Load() != 1 {
		t.Fatal("concurrent cache miss hit the store multiple times")
	}
}

type rideCacheStore struct {
	mu                             sync.Mutex
	items                          []domain.Ride
	searches, idsReads, hydrations int
}

func (s *rideCacheStore) Active(context.Context, string) (domain.Ride, error) {
	return domain.Ride{}, apperrors.ErrNotFound
}
func (s *rideCacheStore) Create(_ context.Context, ride domain.Ride) (domain.Ride, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ride.ID = uuid.NewString()
	ride.Status = "published"
	ride.SeatsAvailable = ride.SeatsTotal
	s.items = append(s.items, ride)
	return ride, nil
}
func (s *rideCacheStore) Search(context.Context, domain.RideSearch) ([]domain.Ride, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searches++
	return s.current(nil), nil
}
func (s *rideCacheStore) SearchIDs(context.Context, domain.RideSearch) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idsReads++
	ids := make([]string, 0)
	for _, r := range s.current(nil) {
		ids = append(ids, r.ID)
	}
	return ids, nil
}
func (s *rideCacheStore) SearchByIDs(_ context.Context, ids []string) ([]domain.Ride, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hydrations++
	return s.current(ids), nil
}
func (s *rideCacheStore) current(ids []string) []domain.Ride {
	items := make([]domain.Ride, 0)
	for _, r := range s.items {
		if r.Status != "published" || r.SeatsAvailable < 1 || !r.DepartureAt.After(time.Now()) {
			continue
		}
		if ids == nil {
			items = append(items, r)
			continue
		}
		for _, id := range ids {
			if id == r.ID {
				items = append(items, r)
				break
			}
		}
	}
	return items
}
func (s *rideCacheStore) ByID(context.Context, string) (domain.Ride, error) {
	return domain.Ride{}, apperrors.ErrNotFound
}
func (s *rideCacheStore) Finish(_ context.Context, id, owner, status string) (domain.Ride, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.items {
		if r.ID == id {
			s.items[i].Status = status
			return s.items[i], nil
		}
	}
	return domain.Ride{}, apperrors.ErrNotFound
}

func TestRideCacheInvalidationAndRecovery(t *testing.T) {
	ctx := context.Background()
	probe := &cacheProbe{Cache: cache.NewMemory()}
	store := &rideCacheStore{}
	svc := service.NewRides(store, probe, quietLogger())
	filter := domain.RideSearch{}
	check := func(want int) {
		t.Helper()
		items, err := svc.Search(ctx, filter)
		if err != nil || len(items) != want {
			t.Fatalf("search found %d rides, want %d", len(items), want)
		}
	}
	check(0)
	check(0)
	if store.idsReads != 1 {
		t.Fatal("empty search was not cached")
	}
	input := domain.Ride{DriverID: uuid.NewString(), DriverFirstName: "PrivateName", Notes: "Private notes", SeatsTotal: 3, DepartureAt: time.Now().Add(time.Hour)}
	ride, err := svc.Publish(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	check(1)
	check(1)
	if store.idsReads != 2 {
		t.Fatal("publish did not invalidate the search")
	}
	// A cache hit always obtains current personal fields and availability.
	store.mu.Lock()
	store.items[0].DriverFirstName = "ChangedName"
	store.mu.Unlock()
	items, _ := svc.Search(ctx, filter)
	if items[0].DriverFirstName != "ChangedName" {
		t.Fatal("personal fields were cached")
	}
	if _, err = svc.Finish(ctx, ride.ID, input.DriverID, "cancelled"); err != nil {
		t.Fatal(err)
	}
	check(0)
	ride, _ = svc.Publish(ctx, input)
	check(1)
	if _, err = svc.Finish(ctx, ride.ID, input.DriverID, "completed"); err != nil {
		t.Fatal(err)
	}
	check(0)
	// Commit while Redis is down, then repair the missed invalidation before
	// consulting the previously cached empty result when Redis returns.
	probe.down.Store(true)
	ride, err = svc.Publish(ctx, input)
	if err != nil {
		t.Fatal("cache outage rejected a successful publish")
	}
	check(1)
	probe.down.Store(false)
	check(1)
	if store.searches != 1 {
		t.Fatal("outage did not fall through to the uncached store")
	}
	for i, raw := range probe.payloads {
		var ids []string
		if json.Unmarshal(raw, &ids) != nil {
			t.Fatal("ride cache contained more than IDs")
		}
		for _, id := range ids {
			if uuid.Validate(id) != nil {
				t.Fatal("non-ID value in ride cache")
			}
		}
		if probe.ttls[i] < 30*time.Second || probe.ttls[i] > 60*time.Second {
			t.Fatal("search TTL outside jitter range")
		}
	}
	key := probe.keys[len(probe.keys)-1]
	probe.Cache.Set(ctx, key, []byte(`["invalid-uuid"]`), time.Hour)
	check(1)
	if store.searches != 2 {
		t.Fatal("invalid cached IDs reached the database hydration query")
	}
}

func TestCachedRideRechecksAvailability(t *testing.T) {
	ctx := context.Background()
	store := &rideCacheStore{}
	svc := service.NewRides(store, cache.NewMemory(), quietLogger())
	ride, _ := svc.Publish(ctx, domain.Ride{SeatsTotal: 1, DepartureAt: time.Now().Add(time.Hour)})
	items, _ := svc.Search(ctx, domain.RideSearch{})
	if len(items) != 1 {
		t.Fatal("ride missing")
	}
	store.mu.Lock()
	store.items[0].DepartureAt = time.Now().Add(-time.Second)
	store.mu.Unlock()
	items, err := svc.Search(ctx, domain.RideSearch{})
	if err != nil || len(items) != 0 {
		t.Fatal("cached ride remained visible after departure")
	}
	_ = ride
}
