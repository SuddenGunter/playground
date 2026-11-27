// Package app wires dependencies together and runs the service.
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
	mockdatacreate "github.com/SuddenGunter/playground/go/tcapp/api/mockdata/create"
	mockdataget "github.com/SuddenGunter/playground/go/tcapp/api/mockdata/get"
	mockdatalist "github.com/SuddenGunter/playground/go/tcapp/api/mockdata/list"
	"github.com/SuddenGunter/playground/go/tcapp/app/config"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

// shutdownTimeout bounds how long graceful shutdown may take before connections are force-closed.
const shutdownTimeout = 10 * time.Second

// App is the service.
type App struct {
	log *slog.Logger
	cfg config.Config

	ready     chan struct{}
	readyOnce sync.Once
	addr      net.Addr
}

// New creates an App.
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
// Call it at most once per App.
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
		// Request contexts must outlive ctx: cancelling ctx starts the shutdown, and Shutdown
		// then waits for in-flight requests, which must not be cancelled out from under them.
		BaseContext: func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
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

	a.log.Info("shutting down", slog.String("timeout", shutdownTimeout.String()))

	// ctx is already cancelled, so derive the shutdown deadline from a fresh context.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		// Drain didn't finish in time: drop remaining connections so Run doesn't return
		// while handlers are still running.
		errs := []error{err, srv.Close()}
		if serr := <-serveErr; !errors.Is(serr, http.ErrServerClosed) {
			errs = append(errs, serr)
		}

		return fmt.Errorf("shutdown: %w", errors.Join(errs...))
	}

	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	a.log.Info("shutdown complete")

	return nil
}

func (a *App) routes(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	store := mockdata.NewStore(pool)

	mux.Handle("GET /health", health.Handler())
	mux.Handle("POST /mock-data", mockdatacreate.New(a.log, store))
	mux.Handle("GET /mock-data", mockdatalist.New(a.log, store))
	mux.Handle("GET /mock-data/{id}", mockdataget.New(a.log, store))

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
