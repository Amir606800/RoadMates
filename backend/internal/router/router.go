package router

import (
	"log/slog"
	"net/http"
	"strings"

	"roadmates/backend/internal/api/v1"
	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/operational"
	"roadmates/backend/internal/service"
)

func New(cfg config.Config, logger *slog.Logger, ops operational.Handlers, authService *service.Auth, tokens *auth.Tokens, applications *service.DriverApplications, districts *service.Districts, rides *service.Rides, limiter service.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", getOnly(ops.Health))
	mux.Handle("/readyz", getOnly(ops.Ready))
	mux.Handle("/api/version", getOnly(ops.AppVersion))
	mux.Handle("/api/v1", http.RedirectHandler("/api/v1/", http.StatusPermanentRedirect))
	mux.Handle("/api/v1/", deprecated(cfg, http.StripPrefix("/api/v1", v1.NewRouterV1(authService, tokens, cfg, applications, districts, rides, limiter))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		segment := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/"), "/")[0]
		if strings.HasPrefix(r.URL.Path, "/api/") && strings.HasPrefix(segment, "v") {
			apperrors.Write(w, r, apperrors.ErrAPIVersionNotFound)
			return
		}
		apperrors.Write(w, r, apperrors.ErrNotFound)
	})
	return observe(logger, recoverPanics(logger, enforceAppVersion(cfg.MinSupportedAppVersion, mux)))
}

func getOnly(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			apperrors.Write(w, r, apperrors.ErrMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}
