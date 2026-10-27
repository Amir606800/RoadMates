package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestPIIEncryptionKey(t *testing.T) {
	for key, value := range map[string]string{"DATABASE_URL": "postgres://test@localhost/test", "REDIS_URL": "redis://localhost:6379/0", "JWT_SECRET": strings.Repeat("s", 32), "ALLOWED_ORIGIN": "http://localhost:8081", "ADMIN_EMAIL": "", "ADMIN_PASSWORD": "", "PORT": "8080", "MIN_SUPPORTED_APP_VERSION": "0.0.0", "API_V1_DEPRECATED": "false", "COOKIE_SECURE": "false", "TRUST_PROXY_HEADERS": "false", "ACCESS_TOKEN_TTL": "15m", "REFRESH_TOKEN_TTL": "30d", "LOG_LEVEL": "info"} {
		t.Setenv(key, value)
	}
	for _, size := range []int{0, 31, 32, 33} {
		t.Setenv("PII_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", size))))
		cfg, err := Load()
		if size == 32 {
			if err != nil || len(cfg.PIIEncryptionKey) != 32 {
				t.Fatal("valid PII key rejected")
			}
		} else if err == nil {
			t.Fatal("invalid PII key length accepted")
		}
	}
	t.Setenv("PII_ENCRYPTION_KEY", "not-base64")
	if _, err := Load(); err == nil {
		t.Fatal("invalid base64 key accepted")
	}
}
