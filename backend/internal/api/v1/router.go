package v1

import (
	"net/http"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/service"
)

// The parent router strips /api/v1 before dispatching here.
func NewRouterV1(authService *service.Auth, tokens *auth.Tokens, cfg config.Config, applications *service.DriverApplications, districts *service.Districts, rides *service.Rides, rateLimiter service.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", index)
	h := authHandlers{authService, cfg}
	limiter := &authLimiter{limiter: rateLimiter, trustProxy: cfg.TrustProxyHeaders}
	mux.Handle("/auth/register", onlyMethod("POST", limiter.wrap("register", 5, http.HandlerFunc(h.register))))
	mux.Handle("/auth/login", onlyMethod("POST", limiter.wrap("login", 10, http.HandlerFunc(h.login))))
	mux.Handle("/auth/refresh", onlyMethod("POST", limiter.wrap("refresh", 60, csrf(cfg.AllowedOrigin, http.HandlerFunc(h.refresh)))))
	mux.Handle("/auth/logout", onlyMethod("POST", csrf(cfg.AllowedOrigin, http.HandlerFunc(h.logout))))
	mux.Handle("/auth/me", onlyMethod("GET", tokens.Authenticate(auth.RequireRole("client")(http.HandlerFunc(h.me)))))
	apps := driverApplicationHandlers{applications}
	client := func(next http.Handler) http.Handler { return tokens.Authenticate(auth.RequireRole("client")(next)) }
	admin := func(next http.Handler) http.Handler { return tokens.Authenticate(auth.RequireRole("admin")(next)) }
	mux.Handle("/driver-applications", onlyMethod("POST", client(limiter.wrap("driver-applications", 10, http.HandlerFunc(apps.apply)))))
	mux.Handle("/driver-applications/me", onlyMethod("GET", client(http.HandlerFunc(apps.mine))))
	mux.Handle("/admin/driver-applications", onlyMethod("GET", admin(http.HandlerFunc(apps.list))))
	mux.Handle("/admin/driver-applications/{id}", admin(http.HandlerFunc(apps.adminApplication)))
	driver := func(next http.Handler) http.Handler { return tokens.Authenticate(auth.RequireRole("driver")(next)) }
	rh := rideHandlers{rides: rides, districts: districts}
	mux.Handle("/districts", onlyMethod("GET", http.HandlerFunc(rh.listDistricts)))
	mux.Handle("/rides", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET", "HEAD":
			rh.search(w, r)
		case "POST":
			driver(http.HandlerFunc(rh.publish)).ServeHTTP(w, r)
		default:
			w.Header().Set("Allow", "GET, HEAD, POST")
			apperrors.Write(w, r, apperrors.ErrMethodNotAllowed)
		}
	}))
	mux.Handle("/rides/me/active", onlyMethod("GET", driver(http.HandlerFunc(rh.mine))))
	mux.Handle("/rides/{id}", onlyMethod("GET", http.HandlerFunc(rh.detail)))
	mux.Handle("/rides/{id}/cancel", onlyMethod("PATCH", driver(rh.finish("cancelled"))))
	mux.Handle("/rides/{id}/complete", onlyMethod("PATCH", driver(rh.finish("completed"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { apperrors.Write(w, r, apperrors.ErrNotFound) })
	return mux
}
