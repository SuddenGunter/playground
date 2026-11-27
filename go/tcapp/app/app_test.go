package app_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SuddenGunter/playground/go/tcapp/app"
	"github.com/SuddenGunter/playground/go/tcapp/app/config"
)

// client fails requests quickly instead of hanging the test suite.
var client = &http.Client{Timeout: 5 * time.Second}

// startApp runs the real app on a random port and returns the port it bound.
// The app is shut down (and its Run error checked) when the test ends.
func startApp(t *testing.T) int {
	t.Helper()

	cfg := config.Config{Port: 0, LogLevel: slog.LevelDebug}
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: cfg.LogLevel}))
	a := app.New(logger, cfg)

	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)

	go func() { runErr <- a.Run(ctx) }()

	select {
	case <-a.Ready():
	case err := <-runErr:
		cancel()
		t.Fatalf("app exited before becoming ready: %v", err)
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("app did not become ready in time")
	}

	t.Cleanup(func() {
		cancel()

		select {
		case err := <-runErr:
			if err != nil {
				t.Errorf("Run returned error on shutdown: %v", err)
			}
		case <-time.After(15 * time.Second): // longer than the app's internal shutdown timeout
			t.Error("app did not shut down in time")
		}
	})

	tcpAddr, ok := a.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected addr type %T", a.Addr())
	}

	return tcpAddr.Port
}

func TestApp(t *testing.T) {
	t.Parallel()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", startApp(t))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "health ok", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`},
		{name: "health wrong method", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed},
		{name: "unknown path", method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), tt.method, baseURL+tt.path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantBody == "" {
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			if got := strings.TrimSpace(string(body)); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestApp_RunFailsOnBusyPort(t *testing.T) {
	t.Parallel()

	port := startApp(t)

	cfg, err := config.LoadFrom(map[string]string{"PORT": strconv.Itoa(port)})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Bounded so the test fails instead of hanging if Run ever binds the port.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := app.New(logger, cfg).Run(ctx); err == nil {
		t.Fatal("expected Run to fail on a port that is already in use")
	}
}
