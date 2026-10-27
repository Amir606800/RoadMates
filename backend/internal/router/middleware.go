package router

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/appversion"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/requestmeta"
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func observe(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID.MatchString(id) {
			var bytes [16]byte
			if _, err := rand.Read(bytes[:]); err != nil {
				logger.Error("request ID generation failed")
				apperrors.Write(w, r, apperrors.ErrInternal)
				return
			}
			id = hex.EncodeToString(bytes[:])
		}
		meta := &requestmeta.Metadata{ID: id}
		r = r.WithContext(requestmeta.With(r.Context(), meta))
		w.Header().Set("X-API-Version", "1")
		w.Header().Set("X-Request-ID", id)
		wrapped := &responseWriter{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(wrapped, r)
		status := wrapped.status
		if status == 0 {
			status = http.StatusOK
		}
		logger.Info("http_request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", status,
			"duration_ms", float64(time.Since(start).Microseconds())/1000, "user_id", meta.UserID,
			"app_version", r.Header.Get("X-App-Version"), "client", r.Header.Get("X-Client"))
	})
}

func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				logger.Error("handler panic", "request_id", requestmeta.From(r.Context()).ID)
				apperrors.Write(w, r, apperrors.ErrInternal)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func enforceAppVersion(minVersion string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := r.Header.Get("X-App-Version")
		if value != "" && strings.HasPrefix(r.URL.Path, "/api/") {
			if !appversion.Valid(value) {
				apperrors.Write(w, r, apperrors.ErrAppVersionInvalid)
				return
			}
			if appversion.Compare(value, minVersion) < 0 {
				apperrors.Write(w, r, apperrors.ErrAppVersionUnsupported)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func deprecated(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.V1Deprecated {
			w.Header().Set("Deprecation", "@"+strconv.FormatInt(cfg.V1DeprecationDate.Unix(), 10))
			w.Header().Set("Sunset", cfg.V1SunsetDate.UTC().Format(http.TimeFormat))
		}
		next.ServeHTTP(w, r)
	})
}
