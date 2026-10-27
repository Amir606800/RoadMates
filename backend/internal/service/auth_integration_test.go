package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/buildinfo"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/operational"
	"roadmates/backend/internal/pii"
	"roadmates/backend/internal/ratelimit"
	"roadmates/backend/internal/repository"
	"roadmates/backend/internal/router"
	"roadmates/backend/internal/service"
)

// TEST_DATABASE_URL opts in. Every run gets a unique schema; no application
// tables or accounts are reused or changed.
func authDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL auth integration checks")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("could not initialize integration database connection")
	}
	schema := "auth_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		admin.Close()
		t.Fatal("could not create isolated test schema")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("invalid integration database URL")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("could not initialize test schema pool")
	}
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE"); admin.Close() })
	sql, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))
	if err != nil {
		t.Fatal("could not read initial schema migration")
	}
	if _, err = pool.Exec(ctx, string(sql)); err != nil {
		t.Fatal("could not migrate isolated test schema")
	}
	seed, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000002_seed_districts.up.sql"))
	if err != nil {
		t.Fatal("could not read district seed migration")
	}
	if _, err = pool.Exec(ctx, string(seed)); err != nil {
		t.Fatal("could not seed isolated test schema")
	}
	return pool
}

type sessionBody struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID, Email string
		FirstName string `json:"first_name"`
		Roles     []string
	} `json:"user"`
	Code string `json:"code"`
}

func TestPostgresAuthentication(t *testing.T) {
	pool := authDatabase(t)
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tokens := auth.NewTokens(strings.Repeat("s", 48), 15*time.Minute)
	store := repository.NewAuth(pool)
	svc, err := service.NewAuth(store, tokens, 30*24*time.Hour, logger)
	if err != nil {
		t.Fatal("auth initialization failed")
	}
	cfg := config.Config{MinSupportedAppVersion: "0.0.0", RefreshTokenTTL: 30 * 24 * time.Hour, AllowedOrigin: "http://localhost:8081"}
	healthy := func(context.Context) error { return nil }
	piiCipher, _ := pii.New([]byte(strings.Repeat("k", 32)))
	applications := service.NewDriverApplications(repository.NewDriverApplications(pool), piiCipher)
	server := httptest.NewServer(router.New(cfg, logger, operational.Handlers{Postgres: healthy, Redis: healthy, Version: buildinfo.Current("0.0.0")}, svc, tokens, applications, service.NewDistricts(repository.NewDistricts(pool), nil, logger), service.NewRides(repository.NewRides(pool), nil, logger), ratelimit.NewMemory()))
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	call := func(client *http.Client, method, path string, body any, bearer, rawCookie, origin, csrfHeader string, expected int) (sessionBody, *http.Response) {
		t.Helper()
		var payload io.Reader
		if body != nil {
			encoded, _ := json.Marshal(body)
			payload = bytes.NewReader(encoded)
		}
		r, _ := http.NewRequest(method, server.URL+"/api/v1"+path, payload)
		if body != nil {
			r.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		if rawCookie != "" {
			r.AddCookie(&http.Cookie{Name: "roadmates_refresh", Value: rawCookie})
		}
		r.Header.Set("Origin", origin)
		r.Header.Set("X-Requested-With", csrfHeader)
		response, err := client.Do(r)
		if err != nil {
			t.Fatal("auth HTTP request failed")
		}
		defer response.Body.Close()
		if response.StatusCode != expected {
			t.Fatalf("%s %s: status %d, expected %d", method, path, response.StatusCode, expected)
		}
		var result sessionBody
		if expected != 204 {
			if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
				t.Fatal("invalid JSON auth response")
			}
		}
		if response.Header.Get("X-Request-ID") == "" {
			t.Fatal("request ID missing")
		}
		return result, response
	}
	password := "Integration-roadmates!1"
	input := map[string]string{"email": "  AUTH@example.com ", "password": password, "first_name": "  Şəms ", "last_name": " Aliyeva ", "phone": " +994 50 111 22 33 "}
	registered, response := call(client, "POST", "/auth/register", input, "", "", "", "", 201)
	if registered.User.Email != "auth@example.com" || registered.User.FirstName != "Şəms" || len(registered.User.Roles) != 1 || registered.User.Roles[0] != "client" {
		t.Fatal("registration normalization/roles failed")
	}
	initialCookie := response.Cookies()[0]
	if !initialCookie.HttpOnly || initialCookie.Path != "/api/v1/auth" || initialCookie.SameSite != http.SameSiteLaxMode || initialCookie.MaxAge != 30*24*60*60 {
		t.Fatal("refresh cookie attributes incorrect")
	}
	var passwordHash, tokenHash string
	if err := pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1", registered.User.ID).Scan(&passwordHash); err != nil {
		t.Fatal("could not inspect stored password")
	}
	if cost, _ := bcrypt.Cost([]byte(passwordHash)); cost != 12 {
		t.Fatal("incorrect bcrypt cost")
	}
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM refresh_tokens WHERE user_id=$1", registered.User.ID).Scan(&tokenHash); err != nil || tokenHash != auth.HashToken(initialCookie.Value) {
		t.Fatal("refresh token hashing failed")
	}
	call(client, "POST", "/auth/register", input, "", "", "", "", 409)
	injected := map[string]any{"email": "evil@example.com", "password": password, "first_name": "Test", "last_name": "User", "phone": "+994501112233", "roles": []string{"admin"}}
	call(client, "POST", "/auth/register", injected, "", "", "", "", 400)
	call(client, "GET", "/auth/me", nil, "", "", "", "", 401)
	call(client, "GET", "/auth/me", nil, registered.AccessToken, "", "", "", 200)
	unknown, _ := call(client, "POST", "/auth/login", map[string]string{"email": "missing@example.com", "password": password}, "", "", "", "", 401)
	wrong, _ := call(client, "POST", "/auth/login", map[string]string{"email": "auth@example.com", "password": "Wrong-password!1"}, "", "", "", "", 401)
	if unknown.Code != "INVALID_CREDENTIALS" || wrong.Code != unknown.Code {
		t.Fatal("login failure reveals account existence")
	}
	// A second login creates an independent family that replay must not revoke.
	otherJar, _ := cookiejar.New(nil)
	otherClient := &http.Client{Jar: otherJar}
	_, otherLogin := call(otherClient, "POST", "/auth/login", map[string]string{"email": "AUTH@example.com", "password": password}, "", "", "", "", 200)
	otherRaw := otherLogin.Cookies()[0].Value
	call(client, "POST", "/auth/refresh", nil, "", "", "", "roadmates", 403)
	call(client, "POST", "/auth/refresh", nil, "", "", cfg.AllowedOrigin, "", 403)
	call(client, "POST", "/auth/refresh", nil, "", "", "https://evil.example", "roadmates", 403)
	if _, err := pool.Exec(ctx, "INSERT INTO user_roles(user_id,role) VALUES($1,'driver')", registered.User.ID); err != nil {
		t.Fatal("could not arrange role refresh test")
	}
	rotated, rotation := call(client, "POST", "/auth/refresh", nil, "", "", cfg.AllowedOrigin, "roadmates", 200)
	if rotation.Cookies()[0].Value == initialCookie.Value || len(rotated.User.Roles) != 2 {
		t.Fatal("rotation or role refresh failed")
	}
	claims, err := tokens.Validate(rotated.AccessToken)
	if err != nil || len(claims.Roles) != 2 {
		t.Fatal("refreshed JWT roles incorrect")
	}
	var replaced bool
	if err := pool.QueryRow(ctx, "SELECT revoked_at IS NOT NULL AND replaced_by IS NOT NULL FROM refresh_tokens WHERE token_hash=$1", auth.HashToken(initialCookie.Value)).Scan(&replaced); err != nil || !replaced {
		t.Fatal("rotation was not recorded")
	}
	replay, _ := call(&http.Client{}, "POST", "/auth/refresh", nil, "", initialCookie.Value, cfg.AllowedOrigin, "roadmates", 401)
	if replay.Code != "REFRESH_TOKEN_REUSED" {
		t.Fatal("replay not detected")
	}
	call(client, "POST", "/auth/refresh", nil, "", "", cfg.AllowedOrigin, "roadmates", 401)
	call(otherClient, "POST", "/auth/refresh", nil, "", "", cfg.AllowedOrigin, "roadmates", 200)
	// Direct concurrent service calls exercise PostgreSQL locking independently
	// of the browser's single-flight protection.
	concurrent, err := svc.Login(ctx, "auth@example.com", password, service.ClientMetadata{})
	if err != nil {
		t.Fatal("could not arrange concurrent refresh test")
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Refresh(ctx, concurrent.RefreshToken, service.ClientMetadata{})
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	winners, reuses := 0, 0
	for err := range results {
		if err == nil {
			winners++
		} else if errors.Is(err, apperrors.ErrRefreshReused) {
			reuses++
		} else {
			t.Fatal("unexpected concurrent refresh error")
		}
	}
	if winners != 1 || reuses != 1 {
		t.Fatal("concurrent refresh did not rotate once and detect reuse")
	}
	var active int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM refresh_tokens WHERE family_id=(SELECT family_id FROM refresh_tokens WHERE token_hash=$1) AND revoked_at IS NULL", auth.HashToken(concurrent.RefreshToken)).Scan(&active); err != nil || active != 0 {
		t.Fatal("reuse revocation did not commit")
	}
	call(otherClient, "POST", "/auth/logout", nil, "", "", "https://evil.example", "roadmates", 403)
	_, logout := call(otherClient, "POST", "/auth/logout", nil, "", "", cfg.AllowedOrigin, "roadmates", 204)
	if logout.Cookies()[0].MaxAge != -1 || logout.Cookies()[0].Path != "/api/v1/auth" {
		t.Fatal("logout did not clear refresh cookie")
	}
	call(&http.Client{}, "POST", "/auth/refresh", nil, "", otherRaw, cfg.AllowedOrigin, "roadmates", 401)
	call(otherClient, "POST", "/auth/logout", nil, "", "", cfg.AllowedOrigin, "roadmates", 204)
	// Bootstrap grants admin only to a newly created account.
	if err := svc.Bootstrap(ctx, "auth@example.com", password); err != nil {
		t.Fatal("existing account bootstrap failed")
	}
	user, _ := store.UserByID(ctx, registered.User.ID)
	for _, role := range user.Roles {
		if role == "admin" {
			t.Fatal("bootstrap promoted an existing account")
		}
	}
	if err := svc.Bootstrap(ctx, "admin@example.com", password); err != nil {
		t.Fatal("admin bootstrap failed")
	}
	if err := svc.Bootstrap(ctx, "admin@example.com", password); err != nil {
		t.Fatal("admin bootstrap is not idempotent")
	}
	admin, _ := store.UserByEmail(ctx, "admin@example.com")
	if len(admin.Roles) != 2 || admin.Roles[0] != "admin" || admin.Roles[1] != "client" {
		t.Fatal("admin roles incorrect")
	}
	// Cleanup retains recently expired tokens and removes only older ones.
	for _, days := range []int{8, 1} {
		_, err := pool.Exec(ctx, "INSERT INTO refresh_tokens(user_id,family_id,token_hash,created_at,expires_at) VALUES($1,$2,$3,now()-interval '12 days',now()-$4*interval '1 day')", user.ID, uuid.NewString(), auth.HashToken(uuid.NewString()), days)
		if err != nil {
			t.Fatal("could not arrange cleanup test")
		}
	}
	if err := store.CleanupRefresh(ctx); err != nil {
		t.Fatal("cleanup failed")
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM refresh_tokens WHERE expires_at < now()").Scan(&active); err != nil || active != 1 {
		t.Fatal("cleanup retention incorrect")
	}
}
