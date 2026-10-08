// Package postgres is the pgx adapter: connection pool, tenant-scoped
// transactions, migrations, and repository implementations.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

const connectAttempts = 5

// DB wraps the pool and the per-call timeout.
type DB struct {
	Pool    *pgxpool.Pool
	timeout time.Duration
}

// Connect opens the pool and verifies it with a bounded number of retries.
// Postgres is a required dependency: callers exit when this fails.
func Connect(ctx context.Context, url string, timeout time.Duration) (*DB, error) {
	var lastErr error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return &DB{Pool: pool, timeout: timeout}, nil
			}
			pool.Close()
		}
		lastErr = err
		backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond // ponytail: quadratic backoff, waits 0.5s, 2s, 4.5s, 8s (15s total, no sleep after the last attempt); jitter if herds appear
		slog.WarnContext(ctx, "postgres not ready", slog.Int("attempt", attempt), slog.Any("err", err))
		if attempt == connectAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("postgres: connect after %d attempts: %w", connectAttempts, lastErr)
}

// Ping reports pool health; used by /readyz.
func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	return d.Pool.Ping(ctx)
}

// Close releases the pool.
func (d *DB) Close() { d.Pool.Close() }

// WithTenant runs fn inside a transaction whose app.tenant_id setting is set
// for the duration of the transaction. Row-level-security policies read that
// setting. Every tenant-scoped repository call goes through here.
func (d *DB) WithTenant(ctx context.Context, tenantID string, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if tenantID == "" {
		return fmt.Errorf("postgres: %w: empty tenant id", domain.ErrForbidden)
	}
	return d.inTx(ctx, "app.tenant_id", tenantID, fn)
}

// WithProvisioning runs fn inside a transaction flagged app.provisioning = '1'.
// RLS policies on tenants, users, and tenant_invitations open up under that
// flag. Only UserRepo.Provision (the sign-in path) calls this.
func (d *DB) WithProvisioning(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return d.inTx(ctx, "app.provisioning", "1", fn)
}

func (d *DB) inTx(ctx context.Context, setting, value string, fn func(ctx context.Context, tx pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin: %v", domain.ErrUnavailable, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL cannot take bind parameters; set_config with is_local=true is the equivalent.
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", setting, value); err != nil {
		return fmt.Errorf("postgres: set %s: %w", setting, wrap(err))
	}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", wrap(err))
	}
	return nil
}
