package auth

import (
	"context"
	"net/http"
	"strings"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/requestmeta"
)

type principalKey struct{}
type Principal struct {
	UserID string
	Roles  []string
}

func Current(ctx context.Context) Principal {
	value, _ := ctx.Value(principalKey{}).(Principal)
	return value
}

func (t *Tokens) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 8192 {
			apperrors.Write(w, r, apperrors.ErrUnauthorized)
			return
		}
		claims, err := t.Validate(parts[1])
		if err != nil {
			apperrors.Write(w, r, apperrors.ErrUnauthorized)
			return
		}
		requestmeta.From(r.Context()).UserID = claims.Subject
		ctx := context.WithValue(r.Context(), principalKey{}, Principal{claims.Subject, claims.Roles})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole accepts any of the supplied roles. Always place it after Authenticate.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := Current(r.Context())
			if principal.UserID == "" {
				apperrors.Write(w, r, apperrors.ErrUnauthorized)
				return
			}
			for _, wanted := range roles {
				for _, granted := range principal.Roles {
					if wanted == granted {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			apperrors.Write(w, r, apperrors.ErrForbidden)
		})
	}
}
