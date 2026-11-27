// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration for the app.
type Config struct {
	// Port the HTTP server listens on. Use 0 to pick a random free port (handy in tests).
	Port int `env:"PORT,required"`
	// LogLevel is one of debug, info, warn, error.
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`
	// ShutdownTimeout bounds how long graceful shutdown may take.
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
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
