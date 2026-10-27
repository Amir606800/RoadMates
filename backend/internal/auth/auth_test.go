package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"roadmates/backend/internal/domain"
)

func TestPasswordPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, password string
		valid          bool
	}{
		{"valid", "Roadmates!1", true}, {"Unicode", "ŞamaxıÖ!2", true}, {"short", "Aa!1234", false},
		{"no upper", "roadmates!1", false}, {"no lower", "ROADMATES!1", false}, {"no special", "Roadmates12", false},
		{"only special", "!!!!!!!!", false}, {"space", "Road mates!1", false}, {"Unicode space", "Road\u00a0mates!1", false},
		{"control", "Roadmates!\n", false}, {"72 bytes", "Aa!" + strings.Repeat("b", 69), true},
		{"over 72 bytes", "Aa!" + strings.Repeat("b", 70), false}, {"UTF8 bytes", "Aa!" + strings.Repeat("ə", 35), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (ValidatePassword(tc.password) == nil) != tc.valid {
				t.Fatal("unexpected password policy result")
			}
		})
	}
}

func TestJWTValidation(t *testing.T) {
	secret := strings.Repeat("s", 32)
	tokens := NewTokens(secret, 15*time.Minute)
	user := domain.User{ID: uuid.NewString(), Roles: []string{"client", "driver"}}
	raw, err := tokens.Access(user)
	if err != nil {
		t.Fatal("access signing failed")
	}
	claims, err := tokens.Validate(raw)
	if err != nil || claims.Subject != user.ID || len(claims.Roles) != 2 || claims.ID == "" {
		t.Fatal("valid access token rejected")
	}
	for _, tc := range []struct {
		name    string
		method  jwt.SigningMethod
		expires time.Time
		issued  *jwt.NumericDate
		key     any
	}{
		{"expired", jwt.SigningMethodHS256, time.Now().Add(-time.Minute), jwt.NewNumericDate(time.Now().Add(-time.Hour)), []byte(secret)},
		{"HS384", jwt.SigningMethodHS384, time.Now().Add(time.Hour), jwt.NewNumericDate(time.Now()), []byte(secret)},
		{"none", jwt.SigningMethodNone, time.Now().Add(time.Hour), jwt.NewNumericDate(time.Now()), jwt.UnsafeAllowNoneSignatureType},
		{"wrong secret", jwt.SigningMethodHS256, time.Now().Add(time.Hour), jwt.NewNumericDate(time.Now()), []byte(strings.Repeat("x", 32))},
		{"missing issued", jwt.SigningMethodHS256, time.Now().Add(time.Hour), nil, []byte(secret)},
		{"future issued", jwt.SigningMethodHS256, time.Now().Add(2 * time.Hour), jwt.NewNumericDate(time.Now().Add(time.Hour)), []byte(secret)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Claims{Roles: []string{"client"}, RegisteredClaims: jwt.RegisteredClaims{Subject: user.ID, ID: uuid.NewString(), IssuedAt: tc.issued, ExpiresAt: jwt.NewNumericDate(tc.expires)}}
			raw, err := jwt.NewWithClaims(tc.method, c).SignedString(tc.key)
			if err != nil {
				t.Fatal("test token signing failed")
			}
			if _, err = tokens.Validate(raw); err == nil {
				t.Fatal("unsafe token accepted")
			}
		})
	}
	missingExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{Roles: []string{"client"}, RegisteredClaims: jwt.RegisteredClaims{Subject: user.ID, ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(time.Now())}}).SignedString([]byte(secret))
	if _, err := tokens.Validate(missingExp); err == nil {
		t.Fatal("missing exp accepted")
	}
}

func TestRoleMiddleware(t *testing.T) {
	tokens := NewTokens(strings.Repeat("s", 32), time.Minute)
	for _, tc := range []struct {
		roles  []string
		status int
	}{{[]string{"client"}, 403}, {[]string{"client", "admin"}, 204}} {
		raw, _ := tokens.Access(domain.User{ID: uuid.NewString(), Roles: tc.roles})
		handler := tokens.Authenticate(RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
		r := httptest.NewRequest("GET", "/protected", nil)
		r.Header.Set("Authorization", "Bearer "+raw)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatal("incorrect role decision")
		}
	}
}
