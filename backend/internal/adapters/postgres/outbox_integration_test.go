//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// count returns the tenant's rows in table.
func count(t *testing.T, db *DB, tenantID, table string) int {
	t.Helper()
	var n int
	run := func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n)
	}
	require.NoError(t, db.WithTenant(context.Background(), tenantID, run))
	return n
}

// TestOutboxRelayAndStore: messages commit with the action, relay is
// all-or-nothing, Store is idempotent (ADR-0015).
func TestOutboxRelayAndStore(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, audits, outbox := NewUserRepo(db), NewDocumentRepo(db), NewAuditRepo(db), NewOutboxRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)

	_, err = docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil, docCreated(ada.ID))
	require.NoError(t, err)
	require.Equal(t, 1, count(t, db, ta.ID, "outbox"))
	require.Zero(t, count(t, db, ta.ID, "audit_entries"), "nothing stored before the relay")

	// A failed publish rolls back: the message stays unpublished.
	n, err := outbox.Relay(context.Background(), 100, func(context.Context, []domain.OutboxMessage) error { return errors.New("redis down") })
	require.Error(t, err)
	require.Zero(t, n)

	var got []domain.OutboxMessage
	capture := func(_ context.Context, ms []domain.OutboxMessage) error { got = append(got, ms...); return nil }
	n, err = outbox.Relay(context.Background(), 100, capture)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Len(t, got, 1)
	m := got[0]
	require.Equal(t, domain.TopicAudit, m.Topic)
	require.Equal(t, ta.ID, m.TenantID)

	// Redelivery stores one entry.
	require.NoError(t, audits.Store(context.Background(), m))
	require.NoError(t, audits.Store(context.Background(), m))
	page, err := audits.List(ctxA, domain.AuditFilter{Limit: 50})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	e := page.Entries[0]
	require.Equal(t, domain.AuditDocumentCreated, e.Action)
	require.Equal(t, ada.ID, e.ActorID)
	require.Equal(t, ada.Email, e.ActorEmail)
	require.Equal(t, "2026", e.DocumentTitle)
	require.WithinDuration(t, time.Now(), e.At, time.Minute)
	var outboxID int64
	require.NoError(t, db.WithTenant(ctxA, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT outbox_id FROM audit_entries").Scan(&outboxID)
	}))
	require.Equal(t, m.ID, outboxID)

	n, err = outbox.Relay(context.Background(), 100, capture)
	require.NoError(t, err)
	require.Zero(t, n, "already published")

	// A rolled-back action leaves no message.
	_, err = docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "x"}, nil, docCreated(uuid.NewString()))
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Equal(t, 1, count(t, db, ta.ID, "outbox"))

	// Purge keeps recent published rows.
	require.NoError(t, outbox.Purge(context.Background()))
	require.Equal(t, 1, count(t, db, ta.ID, "outbox"))
}

// TestWrite covers write[T]: snapshot before fn survives a delete, ids filled
// by fn reach the message, fn errors and unknown ids roll back, errNoChange
// commits without a message.
func TestWrite(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, audits := NewUserRepo(db), NewDocumentRepo(db), NewAuditRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil, docCreated(ada.ID))
	require.NoError(t, err)
	drain(t, db)
	before := count(t, db, ta.ID, "outbox")

	// Snapshot before fn: the title survives the delete (FR-12).
	ok, err := write(ctxA, db, docDeleted(ada.ID, doc.ID), func(ctx context.Context, q *sqlcgen.Queries, _ *domain.AuditEntry) (bool, error) {
		did, _ := parseID(doc.ID)
		n, err := q.DeleteDocument(ctx, did)
		return n == 1, wrap(err)
	})
	require.NoError(t, err)
	require.True(t, ok, "fn's result is returned")

	// Snapshot after fn: fn fills the id.
	_, err = write(ctxA, db, tgEntry(ada.ID, domain.AuditTelegramLinked), func(_ context.Context, _ *sqlcgen.Queries, a *domain.AuditEntry) (struct{}, error) {
		a.TargetID = "filled"
		return struct{}{}, nil
	})
	require.NoError(t, err)

	// fn error, unknown actor, and errNoChange leave no message.
	boom := errors.New("boom")
	_, err = write(ctxA, db, tgEntry(ada.ID, domain.AuditTelegramLinked), func(context.Context, *sqlcgen.Queries, *domain.AuditEntry) (int, error) { return 0, boom })
	require.ErrorIs(t, err, boom)
	_, err = write(ctxA, db, tgEntry(uuid.NewString(), domain.AuditTelegramLinked), func(context.Context, *sqlcgen.Queries, *domain.AuditEntry) (int, error) { return 0, nil })
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = write(ctxA, db, docDeleted(ada.ID, doc.ID), func(context.Context, *sqlcgen.Queries, *domain.AuditEntry) (int, error) { return 0, nil })
	require.ErrorIs(t, err, domain.ErrNotFound, "the document is gone")
	n, err := write(ctxA, db, tgEntry(ada.ID, domain.AuditTelegramUnlinked), func(context.Context, *sqlcgen.Queries, *domain.AuditEntry) (int, error) { return 7, errNoChange })
	require.NoError(t, err)
	require.Equal(t, 7, n)

	require.Equal(t, before+2, count(t, db, ta.ID, "outbox"))
	drain(t, db)
	page, err := audits.List(ctxA, domain.AuditFilter{Limit: 2})
	require.NoError(t, err)
	linked, deleted := page.Entries[0], page.Entries[1]
	require.Equal(t, domain.AuditDocumentDeleted, deleted.Action)
	require.Equal(t, "2026", deleted.DocumentTitle)
	require.Equal(t, ada.Email, deleted.ActorEmail)
	require.Equal(t, domain.AuditTelegramLinked, linked.Action)
	require.Equal(t, "filled", linked.TargetID)
	require.Equal(t, ada.Email, linked.ActorEmail)
}
