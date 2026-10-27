package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"roadmates/backend/internal/domain"
)

type DistrictStore interface {
	List(context.Context, string) ([]domain.District, error)
}

type Districts struct {
	store   DistrictStore
	queries *cachedQueries
}

func NewDistricts(store DistrictStore, cache Cache, logger *slog.Logger) *Districts {
	return &Districts{store, newCachedQueries(cache, logger)}
}
func (s *Districts) List(ctx context.Context, query string) ([]domain.District, error) {
	folded := FoldDistrict(query)
	key := fmt.Sprintf("roadmates:v1:districts:%x", sha256.Sum256([]byte(folded)))
	return cached(ctx, s.queries, key, 24*time.Hour, func(ctx context.Context) ([]domain.District, error) { return s.store.List(ctx, folded) })
}

// Fold the same alphabet used by the seed migration, including uppercase input.
func FoldDistrict(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 'ə', 'Ə':
			return 'e'
		case 'ı', 'İ':
			return 'i'
		case 'ö', 'Ö':
			return 'o'
		case 'ü', 'Ü':
			return 'u'
		case 'ç', 'Ç':
			return 'c'
		case 'ş', 'Ş':
			return 's'
		case 'ğ', 'Ğ':
			return 'g'
		default:
			return unicode.ToLower(r)
		}
	}, strings.TrimSpace(value))
}
