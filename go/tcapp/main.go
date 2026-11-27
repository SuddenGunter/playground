// Command tcapp runs the tcapp HTTP service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SuddenGunter/playground/go/tcapp/app"
	"github.com/SuddenGunter/playground/go/tcapp/app/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("app failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// run holds everything main would do, so deferred calls execute before os.Exit.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.New(logger, cfg).Run(ctx); err != nil {
		return fmt.Errorf("run app: %w", err)
	}

	return nil
}
