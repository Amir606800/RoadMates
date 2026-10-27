package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

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

func TestDriverVerification(t *testing.T) {
	pool := authDatabase(t)
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tokens := auth.NewTokens(strings.Repeat("s", 32), 15*time.Minute)
	authStore := repository.NewAuth(pool)
	sessions, err := service.NewAuth(authStore, tokens, 30*24*time.Hour, logger)
	if err != nil {
		t.Fatal("could not initialize auth")
	}
	if err = sessions.Bootstrap(ctx, "reviewer@example.test", "Driver-reviewer!1"); err != nil {
		t.Fatal("admin bootstrap failed")
	}
	admin, err := sessions.Login(ctx, "reviewer@example.test", "Driver-reviewer!1", service.ClientMetadata{})
	if err != nil {
		t.Fatal("could not sign in reviewer")
	}
	register := func(email string) string {
		t.Helper()
		session, err := sessions.Register(ctx, service.Registration{Email: email, Password: "Driver-applicant!1", FirstName: "Test", LastName: "Applicant", Phone: "+994501112233"}, service.ClientMetadata{})
		if err != nil {
			t.Fatal("could not register applicant")
		}
		return session.AccessToken
	}
	clientToken := register("driver@example.test")
	otherToken := register("other@example.test")
	rollbackToken := register("rollback@example.test")
	cipher, _ := pii.New(bytes.Repeat([]byte{7}, 32))
	applications := service.NewDriverApplications(repository.NewDriverApplications(pool), cipher)
	cfg := config.Config{MinSupportedAppVersion: "0.0.0", AllowedOrigin: "http://localhost:8081", RefreshTokenTTL: 30 * 24 * time.Hour}
	healthy := func(context.Context) error { return nil }
	server := httptest.NewServer(router.New(cfg, logger, operational.Handlers{Postgres: healthy, Redis: healthy, Version: buildinfo.Current("0.0.0")}, sessions, tokens, applications, service.NewDistricts(repository.NewDistricts(pool), nil, logger), service.NewRides(repository.NewRides(pool), nil, logger), ratelimit.NewMemory()))
	defer server.Close()
	client := &http.Client{Timeout: 10 * time.Second}
	type result struct {
		status int
		body   []byte
		header http.Header
	}
	call := func(method, path, bearer string, body any) result {
		var payload io.Reader
		if body != nil {
			encoded, _ := json.Marshal(body)
			payload = bytes.NewReader(encoded)
		}
		r, _ := http.NewRequest(method, server.URL+"/api/v1"+path, payload)
		r.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		response, err := client.Do(r)
		if err != nil {
			return result{status: 0}
		}
		defer response.Body.Close()
		data, _ := io.ReadAll(response.Body)
		return result{response.StatusCode, data, response.Header}
	}
	check := func(result result, expected int) map[string]any {
		t.Helper()
		if result.status != expected {
			var e struct{ Code string }
			_ = json.Unmarshal(result.body, &e)
			t.Fatalf("unexpected status: %d, expected %d (code %s)", result.status, expected, e.Code)
		}
		var body map[string]any
		if json.Unmarshal(result.body, &body) != nil {
			t.Fatal("invalid application response JSON")
		}
		if result.header.Get("Cache-Control") != "no-store" {
			t.Fatal("personal response must not be cached")
		}
		return body
	}
	noPIN := func(result result) {
		t.Helper()
		if bytes.Contains(result.body, []byte("ABC1234")) || bytes.Contains(result.body, []byte("pin_code")) || bytes.Contains(result.body, []byte("PINCodeEncrypted")) {
			t.Fatal("a public/list/review response exposed PIN data")
		}
	}
	input := map[string]any{"contact_phone": " +994 50 111 22 33 ", "first_name": " Şəms ", "last_name": " Əliyeva ", "pin_code": "abc1234", "car_make": " Toyota ", "car_model": " Corolla ", "car_year": 2022, "car_plate": "10-rm-245", "car_color": "Silver", "seats": 4}
	check(call("POST", "/driver-applications", "", input), 401)
	check(call("GET", "/admin/driver-applications", clientToken, nil), 403)
	check(call("GET", "/driver-applications/me", clientToken, nil), 404)
	bad := map[string]any{}
	for key, value := range input {
		bad[key] = value
	}
	bad["pin_code"] = "invalid!"
	check(call("POST", "/driver-applications", clientToken, bad), 400)
	// Two concurrent submissions must yield exactly one pending application.
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- call("POST", "/driver-applications", clientToken, input) }()
	}
	close(start)
	wg.Wait()
	close(results)
	created, conflicts := 0, 0
	var application map[string]any
	for response := range results {
		noPIN(response)
		if response.status == 201 {
			created++
			application = check(response, 201)
		} else {
			conflicts++
			body := check(response, 409)
			if body["code"] != "APPLICATION_PENDING" {
				t.Fatal("duplicate application code incorrect")
			}
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatal("concurrent applications were not serialized")
	}
	id := application["id"].(string)
	userID := application["user_id"].(string)
	if application["status"] != "pending" || application["first_name"] != "Şəms" || application["car_plate"] != "10-RM-245" {
		t.Fatal("application normalization failed")
	}
	var encrypted []byte
	if err := pool.QueryRow(ctx, "SELECT pin_code_encrypted FROM driver_applications WHERE id=$1", id).Scan(&encrypted); err != nil || bytes.Contains(encrypted, []byte("ABC1234")) {
		t.Fatal("PIN was not encrypted in storage")
	}
	if plain, err := cipher.DecryptPIN(encrypted, userID, id); err != nil || plain != "ABC1234" {
		t.Fatal("stored PIN cannot be authenticated/decrypted")
	}
	mine := call("GET", "/driver-applications/me", clientToken, nil)
	check(mine, 200)
	noPIN(mine)
	check(call("GET", "/driver-applications/me", otherToken, nil), 404)
	check(call("GET", "/admin/driver-applications/"+id, clientToken, nil), 403)
	check(call("GET", "/admin/driver-applications/not-a-uuid", admin.AccessToken, nil), 400)
	check(call("GET", "/admin/driver-applications/"+uuid.NewString(), admin.AccessToken, nil), 404)
	listing := call("GET", "/admin/driver-applications?status=pending&limit=1", admin.AccessToken, nil)
	check(listing, 200)
	noPIN(listing)
	detail := check(call("GET", "/admin/driver-applications/"+id, admin.AccessToken, nil), 200)
	if detail["pin_code"] != "ABC1234" {
		t.Fatal("admin detail did not decrypt the PIN")
	}
	check(call("GET", "/admin/driver-applications?status=unknown", admin.AccessToken, nil), 400)
	check(call("PATCH", "/admin/driver-applications/"+id, clientToken, map[string]string{"action": "approve"}), 403)
	approval := call("PATCH", "/admin/driver-applications/"+id, admin.AccessToken, map[string]string{"action": "approve"})
	approved := check(approval, 200)
	noPIN(approval)
	if approved["status"] != "approved" || approved["reviewed_by"] != admin.User.ID || approved["reviewed_at"] == nil {
		t.Fatal("approval audit information incorrect")
	}
	user, _ := authStore.UserByID(ctx, userID)
	if strings.Join(user.Roles, ",") != "client,driver" {
		t.Fatal("approval failed to retain client and grant driver")
	}
	oldClaims, _ := tokens.Validate(clientToken)
	if len(oldClaims.Roles) != 1 {
		t.Fatal("existing access token changed unexpectedly")
	}
	login, err := sessions.Login(ctx, "driver@example.test", "Driver-applicant!1", service.ClientMetadata{})
	if err != nil {
		t.Fatal("could not arrange token refresh")
	}
	refreshed, err := sessions.Refresh(ctx, login.RefreshToken, service.ClientMetadata{})
	if err != nil || strings.Join(refreshed.User.Roles, ",") != "client,driver" {
		t.Fatal("refresh did not obtain new roles")
	}
	stale := check(call("POST", "/driver-applications", clientToken, input), 409)
	if stale["code"] != "DRIVER_ALREADY_VERIFIED" {
		t.Fatal("stale client JWT bypassed the driver check")
	}
	check(call("PATCH", "/admin/driver-applications/"+id, admin.AccessToken, map[string]string{"action": "reject"}), 409)
	second := check(call("POST", "/driver-applications", otherToken, input), 201)
	secondID := second["id"].(string)
	rejected := check(call("PATCH", "/admin/driver-applications/"+secondID, admin.AccessToken, map[string]string{"action": "reject", "reason": "  Please correct the plate.  "}), 200)
	if rejected["status"] != "rejected" || rejected["reject_reason"] != "Please correct the plate." {
		t.Fatal("rejection reason not recorded")
	}
	other, _ := authStore.UserByID(ctx, second["user_id"].(string))
	if strings.Join(other.Roles, ",") != "client" {
		t.Fatal("rejection granted a driver role")
	}
	resubmission := check(call("POST", "/driver-applications", otherToken, input), 201)
	latest := check(call("GET", "/driver-applications/me", otherToken, nil), 200)
	if latest["id"] != resubmission["id"] {
		t.Fatal("latest application did not select resubmission")
	}
	// Conflicting administrator decisions have a single winner.
	start = make(chan struct{})
	results = make(chan result, 2)
	for _, action := range []string{"approve", "reject"} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			<-start
			results <- call("PATCH", "/admin/driver-applications/"+resubmission["id"].(string), admin.AccessToken, map[string]string{"action": action})
		}(action)
	}
	close(start)
	wg.Wait()
	close(results)
	winners, losers := 0, 0
	for response := range results {
		if response.status == 200 {
			winners++
		} else if response.status == 409 {
			losers++
		} else {
			check(response, 200)
		}
	}
	if winners != 1 || losers != 1 {
		t.Fatal("concurrent reviews did not have exactly one winner")
	}
	// Fail the status update after the role insert. Both writes must roll back.
	rollback := check(call("POST", "/driver-applications", rollbackToken, input), 201)
	_, err = pool.Exec(ctx, `CREATE FUNCTION reject_application_review_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='approved' THEN RAISE EXCEPTION 'test failure'; END IF; RETURN NEW; END; $$; CREATE TRIGGER reject_application_review_test BEFORE UPDATE ON driver_applications FOR EACH ROW EXECUTE FUNCTION reject_application_review_test();`)
	if err != nil {
		t.Fatal("could not arrange transaction rollback test")
	}
	check(call("PATCH", "/admin/driver-applications/"+rollback["id"].(string), admin.AccessToken, map[string]string{"action": "approve"}), 500)
	unchanged := check(call("GET", "/driver-applications/me", rollbackToken, nil), 200)
	if unchanged["status"] != "pending" || unchanged["reviewed_at"] != nil {
		t.Fatal("approval was partially committed")
	}
	rollbackUser, _ := authStore.UserByID(ctx, rollback["user_id"].(string))
	if strings.Join(rollbackUser.Roles, ",") != "client" {
		t.Fatal("failed review left a driver role behind")
	}
	_, err = pool.Exec(ctx, `DROP TRIGGER reject_application_review_test ON driver_applications; DROP FUNCTION reject_application_review_test();`)
	if err != nil {
		t.Fatal("could not clear rollback test trigger")
	}
	check(call("PATCH", "/admin/driver-applications/"+rollback["id"].(string), admin.AccessToken, map[string]string{"action": "approve"}), 200)
}
