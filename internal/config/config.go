package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Env = int

const (
	Production Env = iota
	Staging
	Local
)

type Config struct {
	AppEnv      Env
	AppPort     string `mapstructure:"APP_PORT"`
	AppTimezone string `mapstructure:"APP_TIMEZONE"`
	AppDebug    bool   `mapstructure:"APP_DEBUG"`

	DBHost string `mapstructure:"DB_HOST"`
	DBPort int    `mapstructure:"DB_PORT"`
	DBUser string `mapstructure:"DB_USER"`
	DBPass string `mapstructure:"DB_PASS"`
	DBName string `mapstructure:"DB_NAME"`

	HashingDriver string `mapstructure:"HASHING_DRIVER"`

	LogLevel  string `mapstructure:"LOG_LEVEL"`  // "debug" | "info" | "warn" | "error"
	LogFormat string `mapstructure:"LOG_FORMAT"` // "text" | "json"
	LogPath   string `mapstructure:"LOG_PATH"`   // "./logs/app.log" | "/var/log/<app_name>/app.log"
	LogOutput string `mapstructure:"LOG_OUTPUT"` // "stdout" (default) | "file"

	JWTHmacKey string `mapstructure:"JWT_HMAC_KEY"`

	MailDriver        string `mapstructure:"MAIL_DRIVER"`
	MailHost          string `mapstructure:"MAIL_HOST"`
	MailPort          int    `mapstructure:"MAIL_PORT"`
	MailUser          string `mapstructure:"MAIL_USER"`
	MailPass          string `mapstructure:"MAIL_PASS"`
	MailSkipTLSVerify bool   `mapstructure:"MAIL_SKIP_TLS_VERIFY"`
	MailFromName      string `mapstructure:"MAIL_FROM_NAME"`
	MailFromAddress   string `mapstructure:"MAIL_FROM_ADDRESS"`
}

func Load() (*Config, error) {
	// BindStruct makes Unmarshal read env vars for every field, not only keys that also appear in .env
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading .env: %w", err)
	}

	if len(v.AllKeys()) == 0 {
		return nil, fmt.Errorf("no .env variables found")
	}

	// allow real ENV to override
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	switch v.GetString("APP_ENV") {
	case "production":
		cfg.AppEnv = Production
	case "staging":
		cfg.AppEnv = Staging
	default:
		cfg.AppEnv = Local
	}

	// HS256 keys shorter than the 256-bit hash output are brute-forceable (RFC 7518 §3.2)
	if len(cfg.JWTHmacKey) < 32 {
		return nil, fmt.Errorf("JWT_HMAC_KEY must be at least 32 bytes")
	}

	// set app timezone
	loc, err := time.LoadLocation(cfg.AppTimezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}
	time.Local = loc

	return &cfg, nil
}
