package config

import (
	"os"
	"testing"
)

func TestLoad_EnvOverridesAndFillsMissingKeys(t *testing.T) {
	dir := t.TempDir()
	env := "APP_TIMEZONE=UTC\nAPP_PORT=8000\nMAIL_USER=relay-user\nJWT_HMAC_KEY=0123456789abcdef0123456789abcdef\n"
	if err := os.WriteFile(dir+"/.env", []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("APP_PORT", "9000")                         // in .env, overridden
	t.Setenv("MAIL_FROM_ADDRESS", "noreply@example.com") // only in the environment
	t.Setenv("MAIL_USER", "")

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
	if cfg.MailUser != "" {
		t.Errorf("MailUser = %q, want empty env override", cfg.MailUser)
	}
	if cfg.JWTHmacKey != "0123456789abcdef0123456789abcdef" {
		t.Errorf("JWTHmacKey not read from .env")
	}
}

func TestLoad_EnvironmentOnly(t *testing.T) {
	for _, tc := range []struct {
		name         string
		createDotenv bool
	}{
		{name: "missing .env"},
		{name: "empty .env", createDotenv: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tc.createDotenv {
				if err := os.WriteFile(".env", nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("APP_ENV", "production")
			t.Setenv("APP_PORT", "9000")
			t.Setenv("APP_TIMEZONE", "UTC")
			t.Setenv("JWT_HMAC_KEY", "0123456789abcdef0123456789abcdef")

			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AppEnv != Production || cfg.AppPort != "9000" || cfg.AppTimezone != "UTC" {
				t.Errorf("AppEnv = %d, AppPort = %q, AppTimezone = %q, want production, 9000, UTC",
					cfg.AppEnv, cfg.AppPort, cfg.AppTimezone)
			}
		})
	}
}
