package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/email"
	httpadapter "github.com/xhamps/bragdocument/backend/internal/adapters/http"
	"github.com/xhamps/bragdocument/backend/internal/adapters/llm"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/config"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func apiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Run the HTTP API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			return runAPI(cmd.Context(), cfg)
		},
	}
}

// runAPI serves HTTP until ctx is done, then shuts down within cfg.ShutdownTimeout.
func runAPI(ctx context.Context, cfg config.Config) error {
	ctx = telemetry.WithService(ctx, "api")
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
	// in a state where every token is a 401. The fetch is bounded by
	// keyfunc's default 1-minute HTTP timeout.
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
	var impact ports.ImpactExtractor = llm.Disabled{}
	if cfg.OpenAIAPIKey == "" {
		slog.WarnContext(ctx, "OPENAI_API_KEY not set; impact extraction disabled")
	} else {
		impact = llm.NewOpenAIExtractor(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.LLMTimeout, reg)
	}
	var mailer ports.Mailer = email.Disabled{}
	if cfg.ResendAPIKey == "" {
		slog.WarnContext(ctx, "RESEND_API_KEY not set; share emails disabled")
	} else {
		mailer = email.NewResend(cfg.ResendAPIKey, cfg.MailFrom, cfg.MailTimeout, reg)
	}
	cache := redis.NewDegrading(rc, reg)

	engine := httpadapter.NewEngine(reg)
	httpadapter.RegisterHealth(engine, []httpadapter.Check{
		{Name: "postgres", Required: true, Ping: db.Ping},
		{Name: "cache", Ping: rc.Ping},
	})
	authed := engine.Group("/", httpadapter.Auth(jwks.Keyfunc, app.NewUserEnsure(postgres.NewUserRepo(db)), reg))
	httpadapter.RegisterMe(authed)
	docRepo := postgres.NewDocumentRepo(db)
	httpadapter.RegisterDocuments(authed, app.NewDocuments(docRepo))
	logs := app.NewLogs(docRepo, postgres.NewLogRepo(db), impact, cache)
	httpadapter.RegisterLogs(authed, logs)
	if cfg.ExportKey == "" {
		slog.WarnContext(ctx, "EXPORT_KEY not set; PDF export disabled")
	} else {
		exports, err := newExports(cfg, db, cache)
		if err != nil {
			return err
		}
		httpadapter.RegisterExports(authed, exports)
	}
	// codes: raw rc so a Redis outage surfaces as 503; undo: Degrading (a miss is harmless).
	tgUC := app.NewTelegram(postgres.NewTelegramLinkRepo(db), docRepo, logs, rc, cache, cfg.AppURL, telemetry.WithTenantID)
	httpadapter.RegisterTelegram(authed, tgUC, cfg.TelegramBotUsername)
	httpadapter.RegisterTenant(authed, app.NewTenants(postgres.NewTenantRepo(db)))
	httpadapter.RegisterSharing(authed, app.NewSharing(docRepo, postgres.NewSharingRepo(db), mailer, cfg.AppURL))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		// Requests inherit service (FR-7) but not cancellation: Shutdown drains them.
		BaseContext: func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
		// net/http logs without a ctx; tag its lines too.
		ErrorLog: slog.NewLogLogger(slog.Default().With(slog.String("service", "api")).Handler(), slog.LevelError),
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
}
