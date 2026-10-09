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

	// Tenants may only append: UPDATE and DELETE match nothing under RLS, and
	// tenant B sees none of A's messages.
	for _, stmt := range []string{"UPDATE outbox SET published_at = NULL", "DELETE FROM outbox"} {
		require.NoError(t, db.WithTenant(ctxA, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, stmt)
			require.Zero(t, tag.RowsAffected(), stmt)
			return err
		}))
	}
	_, tb := provisionTenant(t, users, "B", "zed@example.com")
	require.Zero(t, count(t, db, tb.ID, "outbox"))

	// Store takes audit.entry messages only.
	other := m
	other.Topic = "other.topic"
	require.Error(t, audits.Store(context.Background(), other))

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

// TestWriteSnapshotsBeforeAndEnqueuesAfter runs the repos through write[T]:
// titles are snapshotted before fn when the document is known, ids assigned by
// fn reach the entry, and no-ops and failures leave no message.
func TestWriteSnapshotsBeforeAndEnqueuesAfter(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs, links, audits := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db), NewTelegramLinkRepo(db), NewAuditRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	latest := func() domain.AuditEntry {
		t.Helper()
		drain(t, db)
		page, err := audits.List(ctxA, domain.AuditFilter{Limit: 1})
		require.NoError(t, err)
		require.Len(t, page.Entries, 1)
		return page.Entries[0]
	}

	// Snapshot after fn: Create sets the document id.
	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil, docCreated(ada.ID))
	require.NoError(t, err)
	e := latest()
	require.Equal(t, doc.ID, e.DocumentID)
	require.Equal(t, "2026", e.DocumentTitle)

	// Enqueued after fn: the new log's id is the target.
	lg, err := logs.Create(ctxA, testLog(ta.ID, doc.ID, ada.ID), logEntry(ada.ID, doc.ID, domain.AuditLogCreated))
	require.NoError(t, err)
	require.Equal(t, lg.ID, latest().TargetID)

	// Snapshot before fn: a rename names the title as it was.
	next := doc
	next.Title = "2026 brag"
	_, err = docs.Update(ctxA, next, domain.AuditEntry{ActorID: ada.ID, Source: domain.SourceWeb,
		Action: domain.AuditDocumentRenamed, DocumentID: doc.ID, ChangedFields: []string{"title"}})
	require.NoError(t, err)
	require.Equal(t, "2026", latest().DocumentTitle)

	// No-ops and failures leave no message.
	before := count(t, db, ta.ID, "outbox")
	require.NoError(t, links.Delete(ctxA, ada.ID, tgEntry(ada.ID, domain.AuditTelegramUnlinked)), "not linked: errNoChange")
	require.NoError(t, logs.DeleteExamples(ctxA, doc.ID, logEntry(ada.ID, doc.ID, domain.AuditLogDeleted)), "no examples: errNoChange")
	require.ErrorIs(t, logs.Delete(ctxA, doc.ID, uuid.NewString(), logEntry(ada.ID, doc.ID, domain.AuditLogDeleted)), domain.ErrNotFound)
	require.Equal(t, before, count(t, db, ta.ID, "outbox"))

	// Removing examples that exist writes exactly one message.
	ex := testLog(ta.ID, "", ada.ID)
	ex.IsExample = true
	withEx, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "examples"}, []domain.Log{ex}, docCreated(ada.ID))
	require.NoError(t, err)
	before = count(t, db, ta.ID, "outbox")
	require.NoError(t, logs.DeleteExamples(ctxA, withEx.ID, logEntry(ada.ID, withEx.ID, domain.AuditLogDeleted)))
	require.Equal(t, before+1, count(t, db, ta.ID, "outbox"))

	// Snapshot before fn: the deleted document keeps its title (FR-12).
	require.NoError(t, docs.Delete(ctxA, doc.ID, docDeleted(ada.ID, doc.ID)))
	e = latest()
	require.Equal(t, domain.AuditDocumentDeleted, e.Action)
	require.Equal(t, "2026 brag", e.DocumentTitle)
}

// TestOutboxDeliveryConfirmation: a published row is published again after
// redeliverAfter until the consumer confirms it, at most maxAttempts times;
// Purge only removes rows delivered over 7 days ago (ADR-0015). Time passing
// is simulated by backdating published_at and delivered_at.
func TestOutboxDeliveryConfirmation(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	users, docs, audits, outbox := NewUserRepo(db), NewDocumentRepo(db), NewAuditRepo(db), NewOutboxRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	ctxA := telemetry.WithTenantID(ctx, ta.ID)
	exec := func(sql string, args ...any) {
		t.Helper()
		require.NoError(t, db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql, args...)
			return err
		}))
	}
	// age moves every undelivered row's last publish just past redeliverAfter.
	age := func() {
		exec("UPDATE outbox SET published_at = published_at - $1::float8 * interval '1 second' WHERE delivered_at IS NULL",
			redeliverAfter.Seconds()+1)
	}
	var got []domain.OutboxMessage
	lose := func(_ context.Context, ms []domain.OutboxMessage) error { got = append(got, ms...); return nil } // Redis loses them
	relay := func() int {
		t.Helper()
		got = nil
		n, err := outbox.Relay(ctx, 100, lose)
		require.NoError(t, err)
		return n
	}

	_, err = docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil, docCreated(ada.ID))
	require.NoError(t, err)
	require.Equal(t, 1, relay())
	require.Zero(t, relay(), "not republished before redeliverAfter")

	// Lost in Redis: republished after redeliverAfter, then stored and confirmed.
	age()
	require.Equal(t, 1, relay())
	require.NoError(t, audits.Store(ctx, got[0]))
	require.NoError(t, outbox.MarkDelivered(ctx, got[0].ID))
	require.NoError(t, outbox.MarkDelivered(ctx, got[0].ID), "idempotent")
	page, err := audits.List(ctxA, domain.AuditFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	age()
	require.Zero(t, relay(), "delivered rows are never republished")

	// Never confirmed: published maxAttempts times, then left for an operator.
	_, err = docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "stuck"}, nil, docCreated(ada.ID))
	require.NoError(t, err)
	for i := range maxAttempts {
		require.Equal(t, 1, relay(), "attempt %d", i+1)
		age()
	}
	require.Zero(t, relay(), "attempt cap reached")
	var attempts int
	require.NoError(t, db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT attempts FROM outbox WHERE delivered_at IS NULL").Scan(&attempts)
	}))
	require.Equal(t, maxAttempts, attempts)
	exec("UPDATE outbox SET attempts = 0 WHERE delivered_at IS NULL") // the operator's retry (ADR-0015, Operations)
	require.Equal(t, 1, relay())

	// Purge removes delivered rows older than 7 days only; undelivered rows stay however old.
	_, err = docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "recent"}, nil, docCreated(ada.ID))
	require.NoError(t, err)
	drain(t, db)
	exec("UPDATE outbox SET published_at = now() - interval '30 days' WHERE delivered_at IS NULL")
	exec("UPDATE outbox SET delivered_at = now() - interval '8 days' WHERE id = (SELECT min(id) FROM outbox)")
	require.Equal(t, 3, count(t, db, ta.ID, "outbox"))
	require.NoError(t, outbox.Purge(ctx))
	require.Equal(t, 2, count(t, db, ta.ID, "outbox"), "only the old delivered row is gone")
}
