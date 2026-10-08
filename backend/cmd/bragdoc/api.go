package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	httpadapter "github.com/xhamps/bragdocument/backend/internal/adapters/http"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func apiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Run the HTTP API",
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
			defer func() { _ = rc.Close() }() // deferred after db.Close, so runs first: Redis closes before the pool

			reg := telemetry.NewRegistry()
			_ = redis.NewDegrading(rc, reg) // ponytail: wired now so the counter exists; use cases take it in the next pass

			engine := httpadapter.NewEngine(reg)
			httpadapter.RegisterHealth(engine, []httpadapter.Check{
				{Name: "postgres", Required: true, Ping: db.Ping},
				{Name: "cache", Ping: rc.Ping},
			})

			srv := &http.Server{
				Addr:              cfg.HTTPAddr,
				Handler:           engine,
				ReadHeaderTimeout: 5 * time.Second,
			}
			errCh := make(chan error, 1)
			go func() { errCh <- srv.ListenAndServe() }()
			slog.InfoContext(ctx, "api listening", slog.String("addr", cfg.HTTPAddr))

			select {
			case <-ctx.Done():
			case err := <-errCh:
				if !errors.Is(err, http.ErrServerClosed) {
					return err
				}
			}

			slog.InfoContext(ctx, "shutting down", slog.Duration("timeout", cfg.ShutdownTimeout))
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	}
}
