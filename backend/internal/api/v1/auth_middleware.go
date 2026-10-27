package v1

import (
	"net/http"
	"strconv"
	"time"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/service"
)

func csrf(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Requested-With") != "roadmates" || r.Header.Get("Origin") != origin {
			apperrors.Write(w, r, apperrors.ErrCSRF)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func onlyMethod(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method && !(method == http.MethodGet && r.Method == http.MethodHead) {
			allow := method
			if method == http.MethodGet {
				allow += ", HEAD"
			}
			w.Header().Set("Allow", allow)
			apperrors.Write(w, r, apperrors.ErrMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type authLimiter struct {
	limiter    service.RateLimiter
	trustProxy bool
}

func (l *authLimiter) wrap(group string, limit int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retry := l.limiter.Allow(r.Context(), group, clientIP(r, l.trustProxy), limit, time.Minute)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int((retry+time.Second-1)/time.Second))))
			apperrors.Write(w, r, apperrors.ErrRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}
