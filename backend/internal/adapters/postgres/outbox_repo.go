package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// OutboxRepo relays outbox messages (ADR-0015).
type OutboxRepo struct{ db *DB }

// NewOutboxRepo wires the repository to the pool.
func NewOutboxRepo(db *DB) *OutboxRepo { return &OutboxRepo{db: db} }

// Relay claims up to limit unpublished messages across tenants, hands them to
// publish, and marks them published in the same transaction. A failed publish
// rolls back: the messages stay for the next call. A crash after publish and
// before commit publishes them again (at-least-once). It returns how many it relayed.
func (r *OutboxRepo) Relay(ctx context.Context, limit int, publish func(context.Context, []domain.OutboxMessage) error) (int, error) {
	var n int
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		rows, err := q.ClaimOutbox(ctx, int32(limit))
		if err != nil || len(rows) == 0 {
			return wrap(err)
		}
		msgs := make([]domain.OutboxMessage, len(rows))
		ids := make([]int64, len(rows))
		for i, row := range rows {
			msgs[i] = domain.OutboxMessage{ID: row.ID, TenantID: row.TenantID.String(), Topic: row.Topic, Payload: row.Payload}
			ids[i] = row.ID
		}
		if err := publish(ctx, msgs); err != nil {
			return err
		}
		if err := q.MarkOutboxPublished(ctx, ids); err != nil {
			return wrap(err)
		}
		n = len(msgs)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Purge deletes messages published more than 7 days ago.
func (r *OutboxRepo) Purge(ctx context.Context) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).PurgeOutbox(ctx))
	})
}
