package main

import (
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/files"
	"github.com/xhamps/bragdocument/backend/internal/adapters/gotenberg"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/config"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

const workerPoll = 2 * time.Second

func newExports(cfg config.Config, db *postgres.DB, cache ports.Cache) (*app.Exports, error) {
	key, err := cfg.ExportKeyBytes()
	if err != nil {
		return nil, err
	}
	store, err := files.New(cfg.ExportDir, key)
	if err != nil {
		return nil, err
	}
	return app.NewExports(postgres.NewDocumentRepo(db), postgres.NewLogRepo(db), postgres.NewExportRepo(db),
		gotenberg.New(cfg.GotenbergURL, cfg.GotenbergTimeout), store, cache, telemetry.WithTenantID), nil
}

// workerCmd runs PDF exports (PRD-0006, ADR-0010): claim, render, store; expire old files.
func workerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run the export worker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := boot()
			if err != nil {
				return err
			}
			db, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBTimeout)
			if err != nil {
				return err
			}
			defer db.Close()
			rc, err := redis.Connect(ctx, cfg.RedisURL, cfg.CacheTimeout)
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()
			reg := telemetry.NewRegistry() // ponytail: not served; the worker has no /metrics endpoint yet
			uc, err := newExports(cfg, db, redis.NewDegrading(rc, reg))
			if err != nil {
				return err
			}

			slog.InfoContext(ctx, "worker started")
			tick := time.NewTicker(workerPoll)
			defer tick.Stop()
			for {
				if err := uc.Cleanup(ctx); err != nil && ctx.Err() == nil {
					slog.WarnContext(ctx, "export cleanup failed", slog.Any("err", err))
				}
				// Drain the queue, then wait. A job cut by shutdown stays running and is reclaimed after 5 minutes.
				for ctx.Err() == nil {
					ran, err := uc.RunNext(ctx)
					if err != nil && ctx.Err() == nil {
						slog.ErrorContext(ctx, "export worker", slog.Any("err", err))
					}
					if !ran || err != nil {
						break
					}
				}
				select {
				case <-ctx.Done():
					slog.InfoContext(ctx, "worker stopped")
					return nil
				case <-tick.C:
				}
			}
		},
	}
}
