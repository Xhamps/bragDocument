// bragdoc is the single backend binary. Subcommands select the mode.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/config"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func main() {
	// JSON before config loads, so a config error is JSON too (FR-8); boot replaces it.
	slog.SetDefault(telemetry.NewLogger("info", "json", os.Stderr))
	root := &cobra.Command{
		Use:           "bragdoc",
		Short:         "Brag Document backend",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(apiCmd(), migrateCmd(), botCmd(), workerCmd(), allCmd())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// A second SIGINT/SIGTERM during shutdown terminates the process with Go's default handling instead of being swallowed.
	context.AfterFunc(ctx, stop)
	defer stop()

	if err := root.ExecuteContext(ctx); err != nil {
		slog.Error("exit", slog.Any("err", err))
		os.Exit(1)
	}
}

// boot loads config and installs the process logger. Every subcommand starts here.
func boot() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	slog.SetDefault(telemetry.NewLogger(cfg.LogLevel, cfg.LogFormat, os.Stderr))
	return cfg, nil
}
