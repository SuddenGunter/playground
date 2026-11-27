// Package app wires dependencies together and runs the service.
// It is kept separate from main so integration/e2e tests can run the real app in-process.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SuddenGunter/playground/go/tcapp/api/health"
	apimockdata "github.com/SuddenGunter/playground/go/tcapp/api/mockdata"
	"github.com/SuddenGunter/playground/go/tcapp/app/config"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

// App is the service. Create it with New and start it with Run.
type App struct {
	log *slog.Logger
	cfg config.Config

	ready     chan struct{}
	readyOnce sync.Once
	addr      net.Addr
}

// New creates an App. It does not start anything.
func New(l *slog.Logger, cfg config.Config) *App {
	return &App{
		log:   l,
		cfg:   cfg,
		ready: make(chan struct{}),
	}
}

// Ready is closed once the HTTP server is listening. After that Addr returns the bound address.
func (a *App) Ready() <-chan struct{} {
	return a.ready
}

// Addr returns the address the server listens on, or nil if it is not listening yet.
// Useful when the app is started with port 0.
func (a *App) Addr() net.Addr {
	select {
	case <-a.ready:
		return a.addr
	default:
		return nil
	}
}

// Run connects to the database, applies the schema, starts the HTTP server and blocks
// until ctx is cancelled or the server fails. It returns nil on graceful shutdown.
func (a *App) Run(ctx context.Context) error {
	pool, err := connectDB(ctx, a.cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := mockdata.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	a.log.Info("database ready")

	var lc net.ListenConfig

	ln, err := lc.Listen(ctx, "tcp", ":"+strconv.Itoa(a.cfg.Port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{
		Handler:           a.routes(pool),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(a.log.Handler(), slog.LevelError),
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	a.addr = ln.Addr()
	a.readyOnce.Do(func() { close(a.ready) })
	a.log.Info("http server listening", slog.String("addr", a.addr.String()))

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- srv.Serve(ln)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	a.log.Info("shutting down", slog.String("timeout", a.cfg.ShutdownTimeout.String()))

	// ctx is already cancelled, so derive the shutdown deadline from a fresh context.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	a.log.Info("shutdown complete")

	return nil
}

func (a *App) routes(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", health.Handler())
	apimockdata.New(a.log, mockdata.NewStore(pool)).Register(mux)

	return mux
}

func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
