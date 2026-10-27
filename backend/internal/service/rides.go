package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/domain"
)

type RideStore interface {
	Active(context.Context, string) (domain.Ride, error)
	Create(context.Context, domain.Ride) (domain.Ride, error)
	Search(context.Context, domain.RideSearch) ([]domain.Ride, error)
	SearchIDs(context.Context, domain.RideSearch) ([]string, error)
	SearchByIDs(context.Context, []string) ([]domain.Ride, error)
	ByID(context.Context, string) (domain.Ride, error)
	Finish(context.Context, string, string, string) (domain.Ride, error)
}

type Rides struct {
	store         RideStore
	queries       *cachedQueries
	invalidations atomic.Uint64
	synced        atomic.Uint64
	syncMu        sync.Mutex
}

func NewRides(store RideStore, cache Cache, logger *slog.Logger) *Rides {
	return &Rides{store: store, queries: newCachedQueries(cache, logger)}
}
func (s *Rides) Publish(ctx context.Context, ride domain.Ride) (domain.Ride, error) {
	_, err := s.store.Active(ctx, ride.DriverID)
	if err == nil {
		return domain.Ride{}, apperrors.ErrDriverHasActiveRide
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		return domain.Ride{}, err
	}
	// The partial unique index remains the guarantee when two checks race.
	created, err := s.store.Create(ctx, ride)
	if err == nil {
		s.invalidate(ctx)
	}
	return created, err
}
func (s *Rides) Active(ctx context.Context, driverID string) (domain.Ride, error) {
	return s.store.Active(ctx, driverID)
}
func (s *Rides) Search(ctx context.Context, filter domain.RideSearch) ([]domain.Ride, error) {
	if !s.syncNamespace(ctx) {
		return s.store.Search(ctx, filter)
	}
	version, err := s.queries.cache.NamespaceVersion(ctx, "rides")
	if err != nil {
		s.queries.warn()
		return s.store.Search(ctx, filter)
	}
	query := fmt.Sprintf("%d:%d:%s", filter.FromID, filter.ToID, filter.Date)
	key := fmt.Sprintf("roadmates:v1:rides:search:%s:%x", version, sha256.Sum256([]byte(query)))
	ids, err := cached(ctx, s.queries, key, 30*time.Second+time.Duration(rand.IntN(30001))*time.Millisecond, func(ctx context.Context) ([]string, error) { return s.store.SearchIDs(ctx, filter) })
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if uuid.Validate(id) != nil {
			s.queries.warn()
			s.queries.cache.Delete(ctx, key)
			return s.store.Search(ctx, filter)
		}
	}
	// Hydrate personal fields from Postgres, and recheck status/seats/departure
	// even on hits. Redis contains only matching ride IDs, never names or notes.
	return s.store.SearchByIDs(ctx, ids)
}
func (s *Rides) Detail(ctx context.Context, id string) (domain.Ride, error) {
	return s.store.ByID(ctx, id)
}
func (s *Rides) Finish(ctx context.Context, id, driverID, status string) (domain.Ride, error) {
	ride, err := s.store.Finish(ctx, id, driverID, status)
	if err == nil {
		s.invalidate(ctx)
	}
	return ride, err
}

func (s *Rides) invalidate(ctx context.Context) {
	s.invalidations.Add(1)
	// A client disconnect after commit must not skip cache invalidation.
	work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 200*time.Millisecond)
	defer cancel()
	s.syncNamespace(work)
}
func (s *Rides) syncNamespace(ctx context.Context) bool {
	if s.invalidations.Load() == s.synced.Load() {
		return true
	}
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	target := s.invalidations.Load()
	if target == s.synced.Load() {
		return true
	}
	if s.queries.cache.BumpNamespace(ctx, "rides") != nil {
		s.queries.warn()
		return false
	}
	s.synced.Store(target)
	return true
}
