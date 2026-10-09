package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"

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
	outboxPurge = time.Hour
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

	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker" // a consumer name is required
	}
	// host:pid keeps two workers on one host (or `all` plus `worker`) apart.
	consumer := fmt.Sprintf("%s:%d", host, os.Getpid())
	var wg sync.WaitGroup
	wg.Go(func() { runOutbox(ctx, db, redis.NewStream(rc, "bragdoc", consumer)) })
	slog.InfoContext(ctx, "worker started")
	if uc != nil {
		runExports(ctx, uc)
	} else {
		slog.WarnContext(ctx, "EXPORT_KEY not set; exports disabled")
		<-ctx.Done()
	}
	wg.Wait()
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
	// Store, then confirm delivery. If the confirmation fails the message stays
	// pending and is handled again; Store is idempotent.
	store := func(ctx context.Context, m domain.OutboxMessage) error {
		if err := audits.Store(ctx, m); err != nil {
			return err
		}
		return outbox.MarkDelivered(ctx, m.ID)
	}
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Go(func() { _ = stream.Consume(ctx, domain.TopicAudit, store) }) // always nil: it logs and retries
	tick := time.NewTicker(outboxPoll)
	defer tick.Stop()
	var lastRelayWarn, lastPurgeWarn, lastPurge time.Time
	for {
		for ctx.Err() == nil { // drain
			n, err := outbox.Relay(ctx, outboxBatch, stream.Publish)
			if err != nil && ctx.Err() == nil && time.Since(lastRelayWarn) > time.Minute {
				lastRelayWarn = time.Now()
				slog.WarnContext(ctx, "outbox relay failed; messages stay queued", slog.Any("err", err))
			}
			if err != nil || n == 0 {
				break
			}
		}
		if time.Since(lastPurge) >= outboxPurge {
			err := outbox.Purge(ctx)
			if err == nil {
				lastPurge = time.Now()
			} else if ctx.Err() == nil && time.Since(lastPurgeWarn) > time.Minute {
				lastPurgeWarn = time.Now()
				slog.WarnContext(ctx, "outbox purge failed", slog.Any("err", err))
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
