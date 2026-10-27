package v1

import (
	"net/http"
	"net/http/httptest"
	"roadmates/backend/internal/ratelimit"
	"strconv"
	"strings"
	"testing"
)

func TestRateLimitHTTPAndTrustedProxy(t *testing.T) {
	for _, trust := range []bool{false, true} {
		limiter := &authLimiter{limiter: ratelimit.NewMemory(), trustProxy: trust}
		h := limiter.wrap("login", 2, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
		for i := 0; i < 4; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			if i < 3 {
				req.Header.Set("X-Real-IP", "192.0.2.2")
			} else {
				req.Header.Set("X-Real-IP", "192.0.2.3")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			want := 204
			if i >= 2 && !(trust && i == 3) {
				want = 429
			}
			if rec.Code != want {
				t.Fatal("IP trust or rate enforcement mismatch")
			}
			if want == 429 {
				seconds, _ := strconv.Atoi(rec.Header().Get("Retry-After"))
				if seconds < 1 || !strings.Contains(rec.Body.String(), `"code":"RATE_LIMITED"`) {
					t.Fatal("missing retry or error contract")
				}
			}
		}
	}
}
