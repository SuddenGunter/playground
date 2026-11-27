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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// client fails requests quickly instead of hanging the test suite.
var client = &http.Client{Timeout: 5 * time.Second}

// startApp creates a fresh database, runs the real app against it on a random port,
// and stops it (checking that Run returned nil) when the test ends.
func startApp(t *testing.T) testApp {
	t.Helper()

	dsn := pg.NewDatabase(t)
	cfg := config.Config{Port: 0, DatabaseURL: dsn, LogLevel: slog.LevelDebug}
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: cfg.LogLevel}))
	a := app.New(logger, cfg)

	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)

	go func() { runErr <- a.Run(ctx) }()

	select {
	case <-a.Ready():
	case err := <-runErr:
		cancel()
		require.FailNow(t, "app exited before becoming ready", "Run error: %v", err)
	case <-time.After(30 * time.Second):
		cancel()
		require.FailNow(t, "app did not become ready in time")
	}

	t.Cleanup(func() {
		cancel()

		select {
		case err := <-runErr:
			assert.NoError(t, err, "Run returned error on shutdown")
		case <-time.After(15 * time.Second): // longer than the app's internal shutdown timeout
			assert.Fail(t, "app did not shut down in time")
		}
	})

	tcpAddr, ok := a.Addr().(*net.TCPAddr)
	require.True(t, ok, "unexpected addr type %T", a.Addr())

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
		_, err := ta.db.Exec(t.Context(), "INSERT INTO mock_data (name, value) VALUES ($1, $2)", r[0], r[1])
		require.NoError(t, err, "seed %v", r)
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
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err, "%s %s", method, path)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if out != nil {
		require.NoError(t, json.Unmarshal(raw, out), "decode %s %s body %q", method, path, raw)
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

			assert.Equal(t, tt.wantStatus, ta.call(t, tt.method, tt.path, "", nil))
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
		require.Equal(t, http.StatusOK, ta.call(t, http.MethodGet, "/mock-data", "", &items))
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, names(items))
	})

	t.Run("list with limit", func(t *testing.T) {
		t.Parallel()

		var items []mockdata.Item
		require.Equal(t, http.StatusOK, ta.call(t, http.MethodGet, "/mock-data?limit=2", "", &items))
		assert.Equal(t, []string{"alpha", "beta"}, names(items))
	})

	t.Run("get by id", func(t *testing.T) {
		t.Parallel()

		var item mockdata.Item
		require.Equal(t, http.StatusOK, ta.call(t, http.MethodGet, "/mock-data/2", "", &item))
		assert.Equal(t, int64(2), item.ID)
		assert.Equal(t, "beta", item.Name)
		assert.Equal(t, "2", item.Value)
		assert.False(t, item.CreatedAt.IsZero(), "created_at not set")
	})

	t.Run("get missing id", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusNotFound, ta.call(t, http.MethodGet, "/mock-data/999", "", nil))
	})
}

func TestApp_CreateThenQuery(t *testing.T) {
	t.Parallel()

	ta := startApp(t)
	ta.seed(t, [2]string{"seeded", "x"})

	var created mockdata.Item
	require.Equal(t, http.StatusCreated, ta.call(t, http.MethodPost, "/mock-data", `{"name":"fresh","value":"y"}`, &created))
	assert.Equal(t, int64(2), created.ID)
	assert.Equal(t, "fresh", created.Name)
	assert.Equal(t, "y", created.Value)

	var fetched mockdata.Item
	require.Equal(t, http.StatusOK, ta.call(t, http.MethodGet, "/mock-data/"+strconv.FormatInt(created.ID, 10), "", &fetched))
	assert.Equal(t, created, fetched)

	var items []mockdata.Item
	require.Equal(t, http.StatusOK, ta.call(t, http.MethodGet, "/mock-data", "", &items))
	assert.Equal(t, []string{"seeded", "fresh"}, names(items))

	// The row really is in the database, not just in the response.
	var count int
	require.NoError(t, ta.db.QueryRow(t.Context(), "SELECT count(*) FROM mock_data WHERE name = 'fresh'").Scan(&count))
	assert.Equal(t, 1, count)

	assert.Equal(t, http.StatusBadRequest, ta.call(t, http.MethodPost, "/mock-data", `{"value":"no name"}`, nil))
}

func TestApp_RunFailsOnBusyPort(t *testing.T) {
	t.Parallel()

	ta := startApp(t)

	cfg := config.Config{Port: ta.port, DatabaseURL: pg.NewDatabase(t)}
	logger := slog.New(slog.DiscardHandler)

	// Bounded so the test fails instead of hanging if Run ever binds the port.
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	err := app.New(logger, cfg).Run(ctx)
	require.Error(t, err, "expected Run to fail on a port that is already in use")
	assert.ErrorContains(t, err, "listen")
}

func TestApp_RunFailsWithUnreachableDatabase(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:        0,
		DatabaseURL: "postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=2",
	}
	logger := slog.New(slog.DiscardHandler)

	err := app.New(logger, cfg).Run(t.Context())
	require.Error(t, err, "expected Run to fail when the database is unreachable")
	assert.ErrorContains(t, err, "ping db")
}

func names(items []mockdata.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}

	return out
}
