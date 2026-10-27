package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/operational"
	"roadmates/backend/internal/ratelimit"
	"roadmates/backend/internal/repository"
	"roadmates/backend/internal/router"
	"roadmates/backend/internal/service"
)

// Forces both service prechecks to see no active ride before either insert.
type simultaneousRideStore struct {
	service.RideStore
	ready sync.WaitGroup
}

func (s *simultaneousRideStore) Active(ctx context.Context, id string) (domain.Ride, error) {
	ride, err := s.RideStore.Active(ctx, id)
	s.ready.Done()
	s.ready.Wait()
	return ride, err
}

func TestDistrictsAndRides(t *testing.T) {
	pool := authDatabase(t)
	ctx := context.Background()
	tokens := auth.NewTokens(strings.Repeat("test", 12), 15*time.Minute)
	store := repository.NewRides(pool)
	rides := service.NewRides(store, nil, nil)
	server := httptest.NewServer(router.New(config.Config{MinSupportedAppVersion: "0.0.0"}, slog.New(slog.NewJSONHandler(io.Discard, nil)), operational.Handlers{}, nil, tokens, nil, service.NewDistricts(repository.NewDistricts(pool), nil, nil), rides, ratelimit.NewMemory()))
	defer server.Close()
	newUser := func(roles ...string) (string, string) {
		t.Helper()
		id := uuid.NewString()
		_, err := pool.Exec(ctx, `INSERT INTO users(id,email,password_hash,first_name,last_name,phone) VALUES($1,$2,'unused','Test','Driver','+994501112233')`, id, id+"@example.test")
		if err != nil {
			t.Fatal("could not create ride test user")
		}
		for _, role := range roles {
			if _, err = pool.Exec(ctx, `INSERT INTO user_roles(user_id,role) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, role); err != nil {
				t.Fatal("could not assign test role")
			}
		}
		token, err := tokens.Access(domain.User{ID: id, Roles: append([]string{"client"}, roles...)})
		if err != nil {
			t.Fatal("could not issue test credential")
		}
		return id, token
	}
	call := func(method, path, bearer string, body any, expected int) []byte {
		t.Helper()
		var payload io.Reader
		if body != nil {
			data, _ := json.Marshal(body)
			payload = bytes.NewReader(data)
		}
		req, _ := http.NewRequest(method, server.URL+"/api/v1"+path, payload)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal("ride HTTP request failed")
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != expected {
			t.Fatalf("%s %s status=%d, want %d; body=%s", method, path, res.StatusCode, expected, data)
		}
		return data
	}
	decodeRide := func(data []byte) string {
		var body struct{ ID string }
		if json.Unmarshal(data, &body) != nil || body.ID == "" {
			t.Fatal("missing ride id")
		}
		return body.ID
	}
	input := func() map[string]any {
		return map[string]any{"from_district_id": 9, "to_district_id": 48, "departure_at": time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339), "seats_total": 3, "price_azn": 20.25, "notes": "  Meeting point\nBring a small bag.  "}
	}

	t.Run("public database districts and folded literal search", func(t *testing.T) {
		var all struct {
			Districts []struct {
				ID     int
				NameAZ string `json:"name_az"`
				Permit bool   `json:"requires_permit"`
			}
		}
		json.Unmarshal(call("GET", "/districts", "", nil, 200), &all)
		if len(all.Districts) != 63 {
			t.Fatal("expected 63 seeded districts")
		}
		permits := 0
		for i, d := range all.Districts {
			if d.ID != i+1 {
				t.Fatal("district seed ordering changed")
			}
			if d.Permit {
				permits++
			}
		}
		if permits != 9 {
			t.Fatal("expected nine selectable permit districts")
		}
		for _, query := range []string{"samaxi", " ŞAMAXI ", "Şamaxı"} {
			var result struct{ Districts []struct{ ID int } }
			json.Unmarshal(call("GET", "/districts?q="+url.QueryEscape(query), "", nil, 200), &result)
			if len(result.Districts) != 1 || result.Districts[0].ID != 9 {
				t.Fatal("folded search did not find district")
			}
		}
		if !bytes.Contains(call("GET", "/districts?q=%25", "", nil, 200), []byte(`"districts":[]`)) {
			t.Fatal("percent query became a wildcard")
		}
		if _, err := pool.Exec(ctx, `INSERT INTO districts(name_az,slug,search_key,sort_order) VALUES('New district','new-district','new district',64)`); err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(call("GET", "/districts?q=new", "", nil, 200), []byte("New district")) {
			t.Fatal("new database district missing")
		}
		if _, err := pool.Exec(ctx, `UPDATE districts SET is_active=false WHERE slug='new-district'`); err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(call("GET", "/districts?q=new", "", nil, 200), []byte(`"districts":[]`)) {
			t.Fatal("inactive district returned")
		}
	})

	_, client := newUser()
	driverID, driver := newUser("driver")
	_, otherDriver := newUser("driver")
	t.Run("roles and input validation", func(t *testing.T) {
		call("POST", "/rides", "", input(), 401)
		call("POST", "/rides", client, input(), 403)
		call("GET", "/rides/me/active", client, nil, 403)
		call("GET", "/rides/me/active", driver, nil, 404)
		for key, value := range map[string]any{"from_district_id": 48, "to_district_id": 32000, "departure_at": time.Now().Add(-time.Hour).Format(time.RFC3339), "seats_total": 9, "price_azn": 20.255, "notes": strings.Repeat("x", 301), "driver_id": uuid.NewString()} {
			body := input()
			body[key] = value
			call("POST", "/rides", driver, body, 400)
		}
		for _, query := range []string{"?from_district_id=-1", "?from_district_id=9&to_district_id=9", "?date=2030-02-30", "?date=0000-01-01", "?to_district_id=32768"} {
			call("GET", "/rides"+query, "", nil, 400)
		}
		if _, err := pool.Exec(ctx, `UPDATE districts SET is_active=false WHERE id=10`); err != nil {
			t.Fatal(err)
		}
		body := input()
		body["to_district_id"] = 10
		call("POST", "/rides", driver, body, 400)
	})
	t.Run("publish search ownership complete and cancel", func(t *testing.T) {
		data := call("POST", "/rides", driver, input(), 201)
		id := decodeRide(data)
		if !bytes.Contains(data, []byte("Şamaxı")) || !bytes.Contains(data, []byte("Şuşa")) || !bytes.Contains(data, []byte(`"price_azn":20.25`)) {
			t.Fatal("ride response missing real route or exact price")
		}
		for _, private := range []string{"password", "pin_code", "contact_phone", "email"} {
			if bytes.Contains(data, []byte(private)) {
				t.Fatal("private data in public ride response")
			}
		}
		if decodeRide(call("GET", "/rides/me/active", driver, nil, 200)) != id {
			t.Fatal("wrong active ride")
		}
		call("POST", "/rides", driver, input(), 409)
		if !bytes.Contains(call("POST", "/rides", driver, input(), 409), []byte(`"code":"DRIVER_HAS_ACTIVE_RIDE"`)) {
			t.Fatal("missing stable conflict code")
		}
		call("GET", "/rides/"+id, "", nil, 200)
		if !bytes.Contains(call("GET", "/rides?from_district_id=9&to_district_id=48", "", nil, 200), []byte(id)) {
			t.Fatal("published ride missing from search")
		}
		call("PATCH", "/rides/"+id+"/cancel", otherDriver, nil, 403)
		call("PATCH", "/rides/"+id+"/complete", client, nil, 403)
		call("PATCH", "/rides/"+id+"/complete", driver, nil, 200)
		call("PATCH", "/rides/"+id+"/cancel", driver, nil, 409)
		call("GET", "/rides/me/active", driver, nil, 404)
		if bytes.Contains(call("GET", "/rides", "", nil, 200), []byte(id)) {
			t.Fatal("completed ride remains searchable")
		}
		id = decodeRide(call("POST", "/rides", driver, input(), 201))
		call("PATCH", "/rides/"+id+"/cancel", driver, nil, 200)
		call("GET", "/rides/me/active", driver, nil, 404)
		id = decodeRide(call("POST", "/rides", driver, input(), 201))
		if _, err := pool.Exec(ctx, `UPDATE rides SET status='in_progress' WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
		call("POST", "/rides", driver, input(), 409)
		if decodeRide(call("GET", "/rides/me/active", driver, nil, 200)) != id {
			t.Fatal("in-progress ride not active")
		}
		call("PATCH", "/rides/"+id+"/complete", driver, nil, 200)
	})
	t.Run("concurrent prechecks rely on the unique index", func(t *testing.T) {
		barrier := &simultaneousRideStore{RideStore: store}
		barrier.ready.Add(2)
		concurrent := service.NewRides(barrier, nil, nil)
		result := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				_, err := concurrent.Publish(ctx, domain.Ride{DriverID: driverID, From: domain.District{ID: 9}, To: domain.District{ID: 48}, DepartureAt: time.Now().Add(time.Hour), SeatsTotal: 3, PriceCents: 2025})
				result <- err
			}()
		}
		wins, conflicts := 0, 0
		for i := 0; i < 2; i++ {
			err := <-result
			if err == nil {
				wins++
			} else if errors.Is(err, apperrors.ErrDriverHasActiveRide) {
				conflicts++
			} else {
				t.Fatal("unexpected concurrent publish error")
			}
		}
		if wins != 1 || conflicts != 1 {
			t.Fatal("concurrent publishing did not yield one success and one conflict")
		}
		var count int
		pool.QueryRow(ctx, `SELECT count(*) FROM rides WHERE driver_id=$1 AND status IN ('published','in_progress')`, driverID).Scan(&count)
		if count != 1 {
			t.Fatal("database accepted two active rides")
		}
	})
	t.Run("Azerbaijan date boundaries and future availability", func(t *testing.T) {
		baku := time.FixedZone("AZT", 4*60*60)
		now := time.Now().In(baku).AddDate(0, 0, 3)
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, baku)
		insert := func(at time.Time, status string, seats int) string {
			owner, _ := newUser("driver")
			var id string
			err := pool.QueryRow(ctx, `INSERT INTO rides(driver_id,from_district_id,to_district_id,departure_at,seats_total,seats_available,price_azn,status) VALUES($1,1,2,$2,3,$3,1,$4) RETURNING id::text`, owner, at, seats, status).Scan(&id)
			if err != nil {
				t.Fatal("could not insert search test ride")
			}
			return id
		}
		first := insert(start, "published", 3)
		last := insert(start.Add(24*time.Hour-time.Second), "published", 1)
		insert(start.Add(-time.Second), "published", 3)
		insert(start.Add(24*time.Hour), "published", 3)
		insert(start.Add(time.Hour), "in_progress", 3)
		insert(start.Add(time.Hour), "cancelled", 3)
		insert(start.Add(time.Hour), "completed", 3)
		insert(start.Add(time.Hour), "published", 0)
		insert(time.Now().Add(-time.Hour), "published", 3)
		var found struct{ Rides []struct{ ID string } }
		json.Unmarshal(call("GET", "/rides?from_district_id=1&to_district_id=2&date="+start.Format("2006-01-02"), "", nil, 200), &found)
		if len(found.Rides) != 2 || found.Rides[0].ID != first || found.Rides[1].ID != last {
			t.Fatal("search date bounds, status, availability, or ordering incorrect")
		}
	})
}
