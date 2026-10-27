package operational

import (
	"context"
	"net/http"
	"time"

	"roadmates/backend/internal/buildinfo"
	"roadmates/backend/internal/httpjson"
)

type Check func(context.Context) error

type Handlers struct {
	Postgres Check
	Redis    Check
	Version  buildinfo.Info
}

type readinessResponse struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

func (h Handlers) Health(w http.ResponseWriter, r *http.Request) {
	httpjson.Write(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{"ok"})
}

func (h Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	redisResult := make(chan error, 1)
	go func() { redisResult <- h.Redis(ctx) }()
	pgErr, redisErr := h.Postgres(ctx), <-redisResult
	status := http.StatusOK
	result := readinessResponse{"ok", "up", "up"}
	if redisErr != nil {
		result.Status, result.Redis = "degraded", "down"
	}
	if pgErr != nil {
		status, result.Status, result.Postgres = http.StatusServiceUnavailable, "not_ready", "down"
	}
	httpjson.Write(w, status, result)
}

func (h Handlers) AppVersion(w http.ResponseWriter, r *http.Request) {
	httpjson.Write(w, http.StatusOK, h.Version)
}
