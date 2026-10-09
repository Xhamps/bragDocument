// Package config loads the process configuration from environment variables.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is the full configuration for every subcommand. Unused fields for a
// given subcommand are harmless.
type Config struct {
	Env       string `env:"APP_ENV" envDefault:"development"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`
	HTTPAddr  string `env:"HTTP_ADDR" envDefault:":8080"`

	DatabaseURL      string `env:"DATABASE_URL,required,notEmpty"`
	DatabaseOwnerURL string `env:"DATABASE_OWNER_URL"` // migrations; defaults to DatabaseURL
	SupabaseURL      string `env:"SUPABASE_URL"`       // required by api; JWKS at /auth/v1/.well-known/jwks.json
	RedisURL         string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
	GotenbergURL     string `env:"GOTENBERG_URL" envDefault:"http://localhost:3000"`

	GotenbergTimeout time.Duration `env:"GOTENBERG_TIMEOUT" envDefault:"60s"`
	ExportDir        string        `env:"EXPORT_DIR" envDefault:"data/exports"`
	ExportKey        string        `env:"EXPORT_KEY"` // base64 of 32 bytes; required by api and worker (PRD-0006 NFR-2)

	TelegramToken       string `env:"TELEGRAM_BOT_TOKEN"`
	TelegramMode        string `env:"TELEGRAM_MODE" envDefault:"polling"`
	TelegramBotUsername string `env:"TELEGRAM_BOT_USERNAME"`                      // for the t.me link in Settings; optional
	AppURL              string `env:"APP_URL" envDefault:"http://localhost:5173"` // bot deep links; "" disables them

	OpenAIAPIKey string        `env:"OPENAI_API_KEY"` // empty: impact extraction disabled (PRD-0007)
	OpenAIModel  string        `env:"OPENAI_MODEL" envDefault:"gpt-4.1-mini"`
	LLMTimeout   time.Duration `env:"LLM_TIMEOUT" envDefault:"5s"`

	ResendAPIKey string        `env:"RESEND_API_KEY"` // empty: share emails disabled (PRD-0004 FR-6, ADR-0014)
	MailFrom     string        `env:"MAIL_FROM" envDefault:"Brag Document <onboarding@resend.dev>"`
	MailTimeout  time.Duration `env:"MAIL_TIMEOUT" envDefault:"5s"`

	DBTimeout       time.Duration `env:"DB_TIMEOUT" envDefault:"5s"`
	CacheTimeout    time.Duration `env:"CACHE_TIMEOUT" envDefault:"200ms"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

// Load reads and validates the configuration. It is called once per process.
func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if c.LogFormat != "json" && c.LogFormat != "text" {
		return Config{}, fmt.Errorf("config: LOG_FORMAT must be json or text, got %q", c.LogFormat)
	}
	if c.LLMTimeout <= 0 {
		return Config{}, fmt.Errorf("config: LLM_TIMEOUT must be positive, got %s", c.LLMTimeout)
	}
	c.AppURL = strings.TrimRight(c.AppURL, "/")
	if c.DatabaseOwnerURL == "" {
		c.DatabaseOwnerURL = c.DatabaseURL
	}
	return c, nil
}

// ExportKeyBytes decodes EXPORT_KEY. api and worker call it; the bot never needs it.
func (c Config) ExportKeyBytes() ([]byte, error) {
	k, err := base64.StdEncoding.DecodeString(c.ExportKey)
	if err != nil || len(k) != 32 {
		return nil, errors.New("config: EXPORT_KEY must be base64 of 32 bytes (openssl rand -base64 32)")
	}
	return k, nil
}
