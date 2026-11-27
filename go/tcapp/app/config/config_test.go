package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/SuddenGunter/playground/go/tcapp/app/config"
)

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
			environ: map[string]string{"PORT": "8080"},
			want: config.Config{
				Port:            8080,
				LogLevel:        slog.LevelInfo,
				ShutdownTimeout: 10 * time.Second,
			},
		},
		{
			name: "all values set",
			environ: map[string]string{
				"PORT":             "9000",
				"LOG_LEVEL":        "debug",
				"SHUTDOWN_TIMEOUT": "3s",
			},
			want: config.Config{
				Port:            9000,
				LogLevel:        slog.LevelDebug,
				ShutdownTimeout: 3 * time.Second,
			},
		},
		{
			name:    "port zero allowed",
			environ: map[string]string{"PORT": "0"},
			want: config.Config{
				Port:            0,
				LogLevel:        slog.LevelInfo,
				ShutdownTimeout: 10 * time.Second,
			},
		},
		{
			name:    "missing port",
			environ: map[string]string{},
			wantErr: true,
		},
		{
			name:    "non-numeric port",
			environ: map[string]string{"PORT": "http"},
			wantErr: true,
		},
		{
			name:    "port out of range",
			environ: map[string]string{"PORT": "70000"},
			wantErr: true,
		},
		{
			name:    "invalid log level",
			environ: map[string]string{"PORT": "8080", "LOG_LEVEL": "loud"},
			wantErr: true,
		},
		{
			name:    "invalid shutdown timeout",
			environ: map[string]string{"PORT": "8080", "SHUTDOWN_TIMEOUT": "soon"},
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
