package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"roadmates/backend/internal/buildinfo"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/operational"
	"roadmates/backend/internal/ratelimit"
)

func TestAPIVersionCompatibilityAndTracing(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := config.Config{MinSupportedAppVersion: "0.2.0", V1Deprecated: true, V1DeprecationDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), V1SunsetDate: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	server := New(cfg, logger, operational.Handlers{Version: buildinfo.Current("0.2.0")}, nil, nil, nil, nil, nil, ratelimit.NewMemory())
	for _, tc := range []struct {
		path, version string
		status        int
		code          string
	}{
		{"/api/v9/rides", "", 404, "API_VERSION_NOT_FOUND"}, {"/api/v1/", "0.1.0", 426, "APP_VERSION_UNSUPPORTED"},
		{"/api/v1/", "invalid", 400, "APP_VERSION_INVALID"}, {"/api/v1/", "", 200, ""}, {"/api/v1/", "0.2.0", 200, ""},
		{"/api/v1/", "0.2.0-rc.1", 426, "APP_VERSION_UNSUPPORTED"}, {"/api/v1/", "1.0.0", 200, ""},
		{"/healthz", "0.0.1", 200, ""}, {"/api/v1/missing", "", 404, "NOT_FOUND"},
	} {
		req := httptest.NewRequest("GET", tc.path, nil)
		req.Header.Set("X-App-Version", tc.version)
		req.Header.Set("X-Request-ID", "phase5-request")
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s status %d, want %d", tc.path, rec.Code, tc.status)
		}
		if rec.Header().Get("X-API-Version") != "1" || rec.Header().Get("X-Request-ID") != "phase5-request" {
			t.Fatal("missing trace/version headers")
		}
		if tc.code != "" {
			var body struct {
				Code      string
				RequestID string `json:"request_id"`
			}
			json.Unmarshal(rec.Body.Bytes(), &body)
			if body.Code != tc.code || body.RequestID != "phase5-request" {
				t.Fatal("error contract mismatch")
			}
		}
		if tc.path == "/api/v1/" && tc.status == 200 && (rec.Header().Get("Deprecation") == "" || rec.Header().Get("Sunset") == "") {
			t.Fatal("configured deprecation headers missing")
		}
	}
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "invalid request id")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if !validRequestID.MatchString(rec.Header().Get("X-Request-ID")) {
		t.Fatal("invalid request id was not replaced")
	}
}

func TestReadiness(t *testing.T) {
	for _, tc := range []struct {
		postgres, redis bool
		status          int
		state           string
	}{
		{true, true, 200, "ok"}, {true, false, 200, "degraded"}, {false, true, 503, "not_ready"}, {false, false, 503, "not_ready"},
	} {
		check := func(up bool) operational.Check {
			return func(context.Context) error {
				if up {
					return nil
				}
				return errors.New("unavailable")
			}
		}
		ops := operational.Handlers{Postgres: check(tc.postgres), Redis: check(tc.redis)}
		server := New(config.Config{MinSupportedAppVersion: "0.0.0"}, slog.New(slog.NewJSONHandler(io.Discard, nil)), ops, nil, nil, nil, nil, nil, ratelimit.NewMemory())
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
		var body struct{ Status, Postgres, Redis string }
		json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != tc.status || body.Status != tc.state || (body.Postgres == "up") != tc.postgres || (body.Redis == "up") != tc.redis {
			t.Fatal("readiness did not reflect dependencies")
		}
		rec = httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != 200 {
			t.Fatal("dependency outage broke liveness")
		}
	}
}
