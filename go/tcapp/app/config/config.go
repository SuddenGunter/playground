// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration for the app.
type Config struct {
	// Port the HTTP server listens on. Use 0 to pick a random free port (handy in tests).
	Port int `env:"PORT,required,notEmpty"`
	// DatabaseURL is a Postgres connection string, e.g. postgres://user:pass@host:5432/db?sslmode=disable.
	DatabaseURL string `env:"DATABASE_URL,required,notEmpty"`
	// LogLevel is one of debug, info, warn, error.
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(env.ToMap(os.Environ()))
}

// LoadFrom reads configuration from the given environment map.
// Tests use it to avoid touching the process environment.
func LoadFrom(environ map[string]string) (Config, error) {
	cfg, err := env.ParseAsWithOptions[Config](env.Options{Environment: environ})
	if err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}

	if cfg.Port < 0 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT %d: must be in range 0-65535", cfg.Port)
	}

	return cfg, nil
}
