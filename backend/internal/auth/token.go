package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/domain"
)

type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokens(secret string, accessTTL time.Duration) *Tokens {
	return &Tokens{secret: []byte(secret), accessTTL: accessTTL}
}

func (t *Tokens) Access(user domain.User) (string, error) {
	now := time.Now()
	claims := Claims{Roles: user.Roles, RegisteredClaims: jwt.RegisteredClaims{
		Subject: user.ID, ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *Tokens) Validate(raw string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, apperrors.ErrUnauthorized
		}
		return t.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !parsed.Valid || claims.IssuedAt == nil || claims.ExpiresAt == nil || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		return nil, apperrors.ErrUnauthorized
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, apperrors.ErrUnauthorized
	}
	if _, err := uuid.Parse(claims.ID); err != nil {
		return nil, apperrors.ErrUnauthorized
	}
	if len(claims.Roles) == 0 {
		return nil, apperrors.ErrUnauthorized
	}
	for _, role := range claims.Roles {
		if role != "client" && role != "driver" && role != "admin" {
			return nil, apperrors.ErrUnauthorized
		}
	}
	return claims, nil
}

func OpaqueToken() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random[:]), nil
}

func HashToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
