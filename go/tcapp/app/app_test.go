//go:build integration

package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SuddenGunter/playground/go/tcapp/app"
	"github.com/SuddenGunter/playground/go/tcapp/app/config"
	"github.com/SuddenGunter/playground/go/tcapp/internal/testenv"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

// testApp is a running app instance backed by its own fresh database.
type testApp struct {
	baseURL string
	port    int
	db      *pgxpool.Pool // direct DB access for seeding and assertions
}

// startApp creates a fresh database, runs the real app against it on a random port,
// and stops it (checking that Run returned nil) when the test ends.
func startApp(t *testing.T) testApp {
	t.Helper()

	dsn := pg.NewDatabase(t)
	cfg := config.Config{Port: 0, DatabaseURL: dsn, LogLevel: slog.LevelDebug, ShutdownTimeout: 5 * time.Second}
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
	case <-time.After(30 * time.Second):
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
		case <-time.After(10 * time.Second):
			t.Error("app did not shut down in time")
		}
	})

	tcpAddr, ok := a.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected addr type %T", a.Addr())
	}

	return testApp{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port),
		port:    tcpAddr.Port,
		db:      testenv.Pool(t, dsn),
	}
}

// seed inserts rows directly into the database, bypassing the API.
func (ta testApp) seed(t *testing.T, rows ...[2]string) {
	t.Helper()

	for _, r := range rows {
		if _, err := ta.db.Exec(t.Context(), "INSERT INTO mock_data (name, value) VALUES ($1, $2)", r[0], r[1]); err != nil {
			t.Fatalf("seed %v: %v", r, err)
		}
	}
}

// call sends a request and decodes a JSON response into out (if non-nil). It returns the status code.
func (ta testApp) call(t *testing.T, method, path, body string, out any) int {
	t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, ta.baseURL+path, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("decode %s %s body %q: %v", method, path, raw, err)
		}
	}

	return resp.StatusCode
}

func TestApp_Health(t *testing.T) {
	t.Parallel()

	ta := startApp(t)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "health ok", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK},
		{name: "health wrong method", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed},
		{name: "unknown path", method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ta.call(t, tt.method, tt.path, "", nil); got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
		})
	}
}

func TestApp_QuerySeededData(t *testing.T) {
	t.Parallel()

	ta := startApp(t)
	ta.seed(t, [2]string{"alpha", "1"}, [2]string{"beta", "2"}, [2]string{"gamma", "3"})

	t.Run("list all", func(t *testing.T) {
		t.Parallel()

		var items []mockdata.Item
		if status := ta.call(t, http.MethodGet, "/mock-data", "", &items); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		if got := names(items); got != "alpha,beta,gamma" {
			t.Errorf("names = %s, want alpha,beta,gamma", got)
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		t.Parallel()

		var items []mockdata.Item
		if status := ta.call(t, http.MethodGet, "/mock-data?limit=2", "", &items); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		if got := names(items); got != "alpha,beta" {
			t.Errorf("names = %s, want alpha,beta", got)
		}
	})

	t.Run("get by id", func(t *testing.T) {
		t.Parallel()

		var item mockdata.Item
		if status := ta.call(t, http.MethodGet, "/mock-data/2", "", &item); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		if item.ID != 2 || item.Name != "beta" || item.Value != "2" || item.CreatedAt.IsZero() {
			t.Errorf("unexpected item %+v", item)
		}
	})

	t.Run("get missing id", func(t *testing.T) {
		t.Parallel()

		if status := ta.call(t, http.MethodGet, "/mock-data/999", "", nil); status != http.StatusNotFound {
			t.Errorf("status = %d, want 404", status)
		}
	})
}

func TestApp_CreateThenQuery(t *testing.T) {
	t.Parallel()

	ta := startApp(t)
	ta.seed(t, [2]string{"seeded", "x"})

	var created mockdata.Item
	if status := ta.call(t, http.MethodPost, "/mock-data", `{"name":"fresh","value":"y"}`, &created); status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", status)
	}

	if created.ID != 2 || created.Name != "fresh" || created.Value != "y" {
		t.Fatalf("unexpected created item %+v", created)
	}

	var fetched mockdata.Item
	if status := ta.call(t, http.MethodGet, "/mock-data/"+strconv.FormatInt(created.ID, 10), "", &fetched); status != http.StatusOK {
		t.Fatalf("get status = %d", status)
	}

	if fetched != created {
		t.Errorf("fetched %+v, want %+v", fetched, created)
	}

	var items []mockdata.Item
	ta.call(t, http.MethodGet, "/mock-data", "", &items)

	if got := names(items); got != "seeded,fresh" {
		t.Errorf("names = %s, want seeded,fresh", got)
	}

	// The row really is in the database, not just in the response.
	var count int
	if err := ta.db.QueryRow(t.Context(), "SELECT count(*) FROM mock_data WHERE name = 'fresh'").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}

	if count != 1 {
		t.Errorf("rows named fresh = %d, want 1", count)
	}

	if status := ta.call(t, http.MethodPost, "/mock-data", `{"value":"no name"}`, nil); status != http.StatusBadRequest {
		t.Errorf("invalid create status = %d, want 400", status)
	}
}

func TestApp_RunFailsOnBusyPort(t *testing.T) {
	t.Parallel()

	ta := startApp(t)

	cfg := config.Config{Port: ta.port, DatabaseURL: pg.NewDatabase(t), ShutdownTimeout: time.Second}
	logger := slog.New(slog.DiscardHandler)

	if err := app.New(logger, cfg).Run(t.Context()); err == nil {
		t.Fatal("expected Run to fail on a port that is already in use")
	}
}

func TestApp_RunFailsWithUnreachableDatabase(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:            0,
		DatabaseURL:     "postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=2",
		ShutdownTimeout: time.Second,
	}
	logger := slog.New(slog.DiscardHandler)

	if err := app.New(logger, cfg).Run(t.Context()); err == nil {
		t.Fatal("expected Run to fail when the database is unreachable")
	}
}

func names(items []mockdata.Item) string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}

	return strings.Join(out, ",")
}
