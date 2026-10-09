package main

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/xhamps/bragdocument/backend/internal/adapters/files"
	"github.com/xhamps/bragdocument/backend/internal/adapters/gotenberg"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/config"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

const (
	workerPoll  = 2 * time.Second
	outboxPoll  = time.Second
	outboxBatch = 100
)

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

// workerCmd relays the audit outbox (ADR-0015) and runs PDF exports (PRD-0006, ADR-0010).
func workerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run the outbox and export worker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			return runWorker(cmd.Context(), cfg)
		},
	}
}

// runWorker relays the outbox, stores audit entries and, when EXPORT_KEY is
// set, claims export jobs and expires old files, until ctx is done.
func runWorker(ctx context.Context, cfg config.Config) error {
	ctx = domain.WithSource(telemetry.WithService(ctx, "worker"), domain.SourceSystem)
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
	var uc *app.Exports
	if cfg.ExportKey != "" {
		reg := telemetry.NewRegistry() // ponytail: not served; the worker has no /metrics endpoint yet
		if uc, err = newExports(cfg, db, redis.NewDegrading(rc, reg)); err != nil {
			return err
		}
	}

	host, _ := os.Hostname()
	var g errgroup.Group
	g.Go(func() error { runOutbox(ctx, db, redis.NewStream(rc, "bragdoc", host)); return nil })
	slog.InfoContext(ctx, "worker started")
	if uc != nil {
		runExports(ctx, uc)
	} else {
		slog.WarnContext(ctx, "EXPORT_KEY not set; exports disabled")
		<-ctx.Done()
	}
	_ = g.Wait()
	slog.InfoContext(ctx, "worker stopped")
	return nil
}

// runExports claims export jobs and expires old files until ctx is done.
func runExports(ctx context.Context, uc *app.Exports) {
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
			return
		case <-tick.C:
		}
	}
}

// runOutbox relays outbox messages to the stream and stores audit entries
// from it until ctx is done (ADR-0015). The relay runs as the app role, so
// cfg.DatabaseURL works as it does for exports.
func runOutbox(ctx context.Context, db *postgres.DB, stream *redis.Stream) {
	outbox, audits := postgres.NewOutboxRepo(db), postgres.NewAuditRepo(db)
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Go(func() {
		if err := stream.Consume(ctx, domain.TopicAudit, audits.Store); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "audit consumer stopped", slog.Any("err", err))
		}
	})
	tick := time.NewTicker(outboxPoll)
	defer tick.Stop()
	var lastWarn time.Time
	for {
		for ctx.Err() == nil { // drain
			n, err := outbox.Relay(ctx, outboxBatch, stream.Publish)
			if err != nil && ctx.Err() == nil && time.Since(lastWarn) > time.Minute {
				slog.WarnContext(ctx, "outbox relay failed; messages stay queued", slog.Any("err", err))
				lastWarn = time.Now()
			}
			if err != nil || n == 0 {
				break
			}
		}
		if err := outbox.Purge(ctx); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "outbox purge failed", slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
