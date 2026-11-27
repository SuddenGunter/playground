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
	// Set up JSON logging before anything else, so even config errors are logged as JSON.
	// The level starts at Info and is updated once config is loaded.
	var level slog.LevelVar

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &level}))
	slog.SetDefault(logger)

	if err := run(logger, &level); err != nil {
		logger.Error("app failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// run holds everything main would do, so deferred calls execute before os.Exit.
func run(logger *slog.Logger, level *slog.LevelVar) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	level.Set(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.New(logger, cfg).Run(ctx); err != nil {
		return fmt.Errorf("run app: %w", err)
	}

	return nil
}
