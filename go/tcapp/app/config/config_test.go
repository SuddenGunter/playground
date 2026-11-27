package config_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
				Port:     8080,
				LogLevel: slog.LevelInfo,
			},
		},
		{
			name: "all values set",
			environ: map[string]string{
				"PORT":      "9000",
				"LOG_LEVEL": "debug",
			},
			want: config.Config{
				Port:     9000,
				LogLevel: slog.LevelDebug,
			},
		},
		{
			name:    "port zero allowed",
			environ: map[string]string{"PORT": "0"},
			want: config.Config{
				Port:     0,
				LogLevel: slog.LevelInfo,
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
			name:    "empty port",
			environ: map[string]string{"PORT": ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.LoadFrom(tt.environ)
			if tt.wantErr {
				require.Error(t, err, "got config %+v", got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
