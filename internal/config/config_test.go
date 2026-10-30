package config

import (
	"os"
	"testing"
)

func TestLoad_EnvOverridesAndFillsMissingKeys(t *testing.T) {
	dir := t.TempDir()
	env := "APP_TIMEZONE=UTC\nAPP_PORT=8000\nJWT_HMAC_KEY=0123456789abcdef0123456789abcdef\n"
	if err := os.WriteFile(dir+"/.env", []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("APP_PORT", "9000")                         // in .env, overridden
	t.Setenv("MAIL_FROM_ADDRESS", "noreply@example.com") // only in the environment

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppPort != "9000" {
		t.Errorf("AppPort = %q, want env override 9000", cfg.AppPort)
	}
	if cfg.MailFromAddress != "noreply@example.com" {
		t.Errorf("MailFromAddress = %q, want value from env", cfg.MailFromAddress)
	}
	if cfg.JWTHmacKey != "0123456789abcdef0123456789abcdef" {
		t.Errorf("JWTHmacKey not read from .env")
	}
}
