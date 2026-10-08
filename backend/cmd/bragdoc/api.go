package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/spf13/cobra"

	httpadapter "github.com/xhamps/bragdocument/backend/internal/adapters/http"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/app"
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

			if cfg.SupabaseURL == "" {
				return errors.New("SUPABASE_URL is required for api")
			}
			// Fetches the JWKS now, then refreshes hourly and on unknown kid. The
			// default storage only logs a failed first fetch, so check we actually
			// got keys: Supabase is a required tier and the api must not start
			// in a state where every token is a 401.
			jwksURL := strings.TrimRight(cfg.SupabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
			jwks, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
			if err != nil {
				return fmt.Errorf("jwks: %w", err)
			}
			ks, err := jwks.VerificationKeySet(ctx)
			if err == nil && len(ks.Keys) == 0 {
				err = errors.New("empty key set")
			}
			if err != nil {
				return fmt.Errorf("jwks: %s: %w", jwksURL, err)
			}

			reg := telemetry.NewRegistry()
			_ = redis.NewDegrading(rc, reg) // ponytail: wired now so the counter exists; use cases take it in the next pass

			engine := httpadapter.NewEngine(reg)
			httpadapter.RegisterHealth(engine, []httpadapter.Check{
				{Name: "postgres", Required: true, Ping: db.Ping},
				{Name: "cache", Ping: rc.Ping},
			})
			authed := engine.Group("/", httpadapter.Auth(jwks.Keyfunc, app.NewUserEnsure(postgres.NewUserRepo(db)), reg))
			httpadapter.RegisterMe(authed)
			httpadapter.RegisterDocuments(authed, app.NewDocuments(postgres.NewDocumentRepo(db)))
			httpadapter.RegisterTenant(authed, app.NewTenants(postgres.NewTenantRepo(db)))

			srv := &http.Server{
				Addr:              cfg.HTTPAddr,
				Handler:           engine,
				ReadHeaderTimeout: 5 * time.Second,
				MaxHeaderBytes:    1 << 20,
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
