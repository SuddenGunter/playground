package config_test

import (
	"log/slog"
	"testing"

	"github.com/SuddenGunter/playground/go/tcapp/app/config"
)

const dsn = "postgres://u:p@localhost:5432/db"

func TestLoadFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		environ map[string]string
		want    config.Config
		wantErr bool
	}{
		{
			name:    "defaults applied",
			environ: map[string]string{"PORT": "8080", "DATABASE_URL": dsn},
			want: config.Config{
				Port:        8080,
				DatabaseURL: dsn,
				LogLevel:    slog.LevelInfo,
			},
		},
		{
			name: "all values set",
			environ: map[string]string{
				"PORT":         "9000",
				"DATABASE_URL": dsn,
				"LOG_LEVEL":    "debug",
			},
			want: config.Config{
				Port:        9000,
				DatabaseURL: dsn,
				LogLevel:    slog.LevelDebug,
			},
		},
		{
			name:    "port zero allowed",
			environ: map[string]string{"PORT": "0", "DATABASE_URL": dsn},
			want: config.Config{
				Port:        0,
				DatabaseURL: dsn,
				LogLevel:    slog.LevelInfo,
			},
		},
		{
			name:    "missing port",
			environ: map[string]string{"DATABASE_URL": dsn},
			wantErr: true,
		},
		{
			name:    "missing database url",
			environ: map[string]string{"PORT": "8080"},
			wantErr: true,
		},
		{
			name:    "non-numeric port",
			environ: map[string]string{"PORT": "http", "DATABASE_URL": dsn},
			wantErr: true,
		},
		{
			name:    "port out of range",
			environ: map[string]string{"PORT": "70000", "DATABASE_URL": dsn},
			wantErr: true,
		},
		{
			name:    "invalid log level",
			environ: map[string]string{"PORT": "8080", "DATABASE_URL": dsn, "LOG_LEVEL": "loud"},
			wantErr: true,
		},
		{
			name:    "empty port",
			environ: map[string]string{"PORT": "", "DATABASE_URL": dsn},
			wantErr: true,
		},
		{
			name:    "empty database url",
			environ: map[string]string{"PORT": "8080", "DATABASE_URL": ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.LoadFrom(tt.environ)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got config %+v", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
