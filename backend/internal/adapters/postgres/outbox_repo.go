package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Delivery confirmation (ADR-0015).
const (
	// redeliverAfter: a published row the consumer has not confirmed
	// (MarkDelivered) is published again this long after its last publish,
	// which covers entries lost with Redis (restart, failover, flush).
	redeliverAfter = 10 * time.Minute
	// maxAttempts caps publishes per row. ponytail: a row that reaches it
	// undelivered stays in the outbox for an operator (ADR-0015, Operations);
	// no alert yet, add one with worker metrics.
	maxAttempts = 5
)

// OutboxRepo relays outbox messages (ADR-0015).
type OutboxRepo struct{ db *DB }

// NewOutboxRepo wires the repository to the pool.
func NewOutboxRepo(db *DB) *OutboxRepo { return &OutboxRepo{db: db} }

// Relay claims up to limit undelivered messages across tenants (never
// published, or published over redeliverAfter ago and fewer than maxAttempts
// times), hands them to publish, and marks them published in the same
// transaction. A failed publish
// rolls back: the messages stay for the next call. A crash after publish and
// before commit publishes them again (at-least-once). It returns how many it relayed.
func (r *OutboxRepo) Relay(ctx context.Context, limit int, publish func(context.Context, []domain.OutboxMessage) error) (int, error) {
	var n int
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		rows, err := q.ClaimOutbox(ctx, sqlcgen.ClaimOutboxParams{
			MaxAttempts: int32(maxAttempts), RedeliverSecs: redeliverAfter.Seconds(), Lim: int32(limit)})
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

// MarkDelivered records that the consumer handled message id, which stops its
// redelivery. Idempotent.
func (r *OutboxRepo) MarkDelivered(ctx context.Context, id int64) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).MarkOutboxDelivered(ctx, id))
	})
}

// Purge deletes messages delivered more than 7 days ago.
func (r *OutboxRepo) Purge(ctx context.Context) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).PurgeOutbox(ctx))
	})
}
