package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"roadmates/backend/internal/appversion"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	RedisURL               string
	MinSupportedAppVersion string
	LogLevel               slog.Level
	V1Deprecated           bool
	V1DeprecationDate      time.Time
	V1SunsetDate           time.Time
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	CookieSecure           bool
	AllowedOrigin          string
	AdminEmail             string
	AdminPassword          string
	TrustProxyHeaders      bool
	PIIEncryptionKey       []byte
}

func Load() (Config, error) {
	c := Config{Port: env("PORT", "8080"), DatabaseURL: os.Getenv("DATABASE_URL"), RedisURL: os.Getenv("REDIS_URL"), MinSupportedAppVersion: env("MIN_SUPPORTED_APP_VERSION", "0.0.0")}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return c, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if c.DatabaseURL == "" || c.RedisURL == "" {
		return c, fmt.Errorf("DATABASE_URL and REDIS_URL are required")
	}
	if !appversion.Valid(c.MinSupportedAppVersion) {
		return c, fmt.Errorf("MIN_SUPPORTED_APP_VERSION must be a full semantic version")
	}
	if err := c.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		return c, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	c.V1Deprecated, err = strconv.ParseBool(env("API_V1_DEPRECATED", "false"))
	if err != nil {
		return c, fmt.Errorf("API_V1_DEPRECATED must be a boolean")
	}
	if c.V1Deprecated {
		c.V1DeprecationDate, err = time.Parse(time.RFC3339, os.Getenv("API_V1_DEPRECATION_DATE"))
		if err != nil {
			return c, fmt.Errorf("API_V1_DEPRECATION_DATE must be RFC3339 when v1 is deprecated")
		}
		c.V1SunsetDate, err = time.Parse(time.RFC3339, os.Getenv("API_V1_SUNSET_DATE"))
		if err != nil || !c.V1SunsetDate.After(c.V1DeprecationDate) {
			return c, fmt.Errorf("API_V1_SUNSET_DATE must be RFC3339 and after the deprecation date")
		}
	}
	c.JWTSecret = os.Getenv("JWT_SECRET")
	if len(c.JWTSecret) < 32 || strings.HasPrefix(c.JWTSecret, "replace-") {
		return c, fmt.Errorf("JWT_SECRET must be a non-placeholder secret of at least 32 bytes")
	}
	c.AccessTokenTTL, err = time.ParseDuration(env("ACCESS_TOKEN_TTL", "15m"))
	if err != nil || c.AccessTokenTTL < time.Second || c.AccessTokenTTL > 24*time.Hour {
		return c, fmt.Errorf("ACCESS_TOKEN_TTL must be between 1s and 24h")
	}
	refreshValue := env("REFRESH_TOKEN_TTL", "30d")
	if strings.HasSuffix(refreshValue, "d") {
		days, parseErr := strconv.Atoi(strings.TrimSuffix(refreshValue, "d"))
		if parseErr != nil || days < 1 || days > 365 {
			return c, fmt.Errorf("REFRESH_TOKEN_TTL days must be between 1d and 365d")
		}
		c.RefreshTokenTTL = time.Duration(days) * 24 * time.Hour
	} else {
		c.RefreshTokenTTL, err = time.ParseDuration(refreshValue)
		if err != nil || c.RefreshTokenTTL < time.Second || c.RefreshTokenTTL > 365*24*time.Hour {
			return c, fmt.Errorf("REFRESH_TOKEN_TTL must be a positive duration up to 365d")
		}
	}
	c.CookieSecure, err = strconv.ParseBool(env("COOKIE_SECURE", "false"))
	if err != nil {
		return c, fmt.Errorf("COOKIE_SECURE must be a boolean")
	}
	c.AllowedOrigin = os.Getenv("ALLOWED_ORIGIN")
	origin, err := url.Parse(c.AllowedOrigin)
	if err != nil || origin.Host == "" || (origin.Scheme != "http" && origin.Scheme != "https") || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return c, fmt.Errorf("ALLOWED_ORIGIN must be one exact http(s) origin without a trailing slash")
	}
	c.AdminEmail, c.AdminPassword = strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))), os.Getenv("ADMIN_PASSWORD")
	if (c.AdminEmail == "") != (c.AdminPassword == "") {
		return c, fmt.Errorf("ADMIN_EMAIL and ADMIN_PASSWORD must both be set or both empty")
	}
	c.TrustProxyHeaders, err = strconv.ParseBool(env("TRUST_PROXY_HEADERS", "false"))
	if err != nil {
		return c, fmt.Errorf("TRUST_PROXY_HEADERS must be a boolean")
	}
	c.PIIEncryptionKey, err = base64.StdEncoding.Strict().DecodeString(os.Getenv("PII_ENCRYPTION_KEY"))
	if err != nil || len(c.PIIEncryptionKey) != 32 {
		return c, fmt.Errorf("PII_ENCRYPTION_KEY must be base64 encoding of exactly 32 bytes")
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
