//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// TestAuditInsert covers the nullable actor and document branches of audit().
func TestAuditInsert(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs := NewUserRepo(db), NewDocumentRepo(db)
	_, ta := provisionTenant(t, users, "A", "ada@example.com")
	var bob domain.User
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		bob, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: ta.ID, Email: "bob@example.com", DisplayName: "Bob", Role: domain.RoleMember})
		return err
	}))
	ctx := telemetry.WithTenantID(context.Background(), ta.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: bob.ID, Title: "2026"}, nil, docCreated(bob.ID))
	require.NoError(t, err)

	write := func(a domain.AuditEntry) error {
		return db.WithTenant(ctx, ta.ID, func(ctx context.Context, tx pgx.Tx) error { return audit(ctx, sqlcgen.New(tx), a) })
	}
	require.NoError(t, write(domain.AuditEntry{ActorID: bob.ID, Source: domain.SourceTelegram, Action: domain.AuditTelegramLinked}))
	require.NoError(t, write(domain.AuditEntry{Source: domain.SourceSystem, Action: domain.AuditExportRequested, DocumentID: doc.ID}))
	require.ErrorIs(t, write(domain.AuditEntry{ActorID: bob.ID, Source: domain.SourceWeb, Action: domain.AuditExportRequested, DocumentID: uuid.NewString()}), domain.ErrNotFound)
	require.ErrorIs(t, write(domain.AuditEntry{ActorID: uuid.NewString(), Source: domain.SourceWeb, Action: domain.AuditTelegramLinked}), domain.ErrNotFound)

	type row struct {
		actorID, docID     *string
		name, email, title *string
	}
	var rows []row
	require.NoError(t, db.WithTenant(ctx, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
		rs, err := tx.Query(ctx, "SELECT actor_id::text, actor_name, actor_email, document_id::text, document_title FROM audit_entries ORDER BY id")
		if err != nil {
			return err
		}
		defer rs.Close()
		for rs.Next() {
			var r row
			if err := rs.Scan(&r.actorID, &r.name, &r.email, &r.docID, &r.title); err != nil {
				return err
			}
			rows = append(rows, r)
		}
		return rs.Err()
	}))
	require.Len(t, rows, 3, "document.created plus two; rejected entries wrote nothing")

	linked, export := rows[1], rows[2]
	require.Equal(t, bob.ID, *linked.actorID)
	require.Equal(t, "Bob", *linked.name, "copied from users")
	require.Equal(t, "bob@example.com", *linked.email)
	require.Nil(t, linked.docID)
	require.Nil(t, linked.title)

	require.Nil(t, export.actorID, "the system acted")
	require.Equal(t, "", *export.name)
	require.Equal(t, doc.ID, *export.docID)
	require.Equal(t, "2026", *export.title)
}

// TestAuditRepo: filters, keyset paging, owner scope, tenant isolation, deleted titles, append-only.
func TestAuditRepo(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs, audits := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db), NewAuditRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	zed, tb := provisionTenant(t, users, "B", "zed@example.com")
	bob := addMember(t, users, ta, "bob@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)

	newDoc := func(ctx context.Context, tenantID, ownerID, title string) domain.Document {
		d, err := docs.Create(ctx, domain.Document{TenantID: tenantID, OwnerID: ownerID, Title: title}, nil, docCreated(ownerID))
		require.NoError(t, err)
		return d
	}
	doc1, doc2, doc3 := newDoc(ctxA, ta.ID, ada.ID, "2026"), newDoc(ctxA, ta.ID, ada.ID, "2027"), newDoc(ctxA, ta.ID, bob.ID, "Bob's")
	newDoc(ctxB, tb.ID, zed.ID, "Zed's")
	for _, w := range []struct{ user, doc string }{{ada.ID, doc1.ID}, {ada.ID, doc1.ID}, {ada.ID, doc1.ID}, {bob.ID, doc3.ID}} {
		_, err := logs.Create(ctxA, testLog(ta.ID, w.doc, w.user), logEntry(w.user, w.doc, domain.AuditLogCreated))
		require.NoError(t, err)
	}
	require.NoError(t, docs.Delete(ctxA, doc2.ID, docDeleted(ada.ID, doc2.ID)))

	// newest first, paging by cursor
	f := domain.AuditFilter{Limit: 2}
	p1, err := audits.List(ctxA, f)
	require.NoError(t, err)
	require.Len(t, p1.Entries, 2)
	require.NotZero(t, p1.NextBefore)
	require.Greater(t, p1.Entries[0].ID, p1.Entries[1].ID)
	f.Before = p1.NextBefore
	p2, err := audits.List(ctxA, f)
	require.NoError(t, err)
	require.Less(t, p2.Entries[0].ID, p1.Entries[1].ID)

	// filters combine
	got, err := audits.List(ctxA, domain.AuditFilter{ActorID: ada.ID, DocumentID: doc1.ID, Action: domain.AuditLogCreated, Limit: 50})
	require.NoError(t, err)
	require.Len(t, got.Entries, 3)
	require.Zero(t, got.NextBefore)
	from := time.Now().Add(-time.Hour)
	got, err = audits.List(ctxA, domain.AuditFilter{From: &from, Limit: 50})
	require.NoError(t, err)
	require.Len(t, got.Entries, 8)
	got, err = audits.List(ctxA, domain.AuditFilter{To: &from, Limit: 50})
	require.NoError(t, err)
	require.Empty(t, got.Entries)

	// owner scope: ada sees doc1 entries, not bob's doc3 nor the deleted doc2
	got, err = audits.List(ctxA, domain.AuditFilter{OwnerID: ada.ID, Limit: 50})
	require.NoError(t, err)
	require.Len(t, got.Entries, 4)
	for _, e := range got.Entries {
		require.Equal(t, doc1.ID, e.DocumentID)
	}

	// deleted document keeps its title (FR-12)
	got, err = audits.List(ctxA, domain.AuditFilter{DocumentID: doc2.ID, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, domain.AuditDocumentDeleted, got.Entries[0].Action)
	require.Equal(t, doc2.Title, got.Entries[0].DocumentTitle)

	// tenant isolation (NFR-1)
	got, err = audits.List(ctxB, domain.AuditFilter{Limit: 50})
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)

	// pickers
	actors, documents, err := audits.Filters(ctxA, ada.ID)
	require.NoError(t, err)
	require.Equal(t, []string{ada.ID}, actorIDs(actors))
	require.Len(t, documents, 1)
	actors, documents, err = audits.Filters(ctxA, "")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{ada.ID, bob.ID}, actorIDs(actors))
	require.Len(t, documents, 3, "the deleted document stays pickable")

	// append-only (FR-4): the app role cannot rewrite history
	for _, stmt := range []string{"UPDATE audit_entries SET action = 'x'", "DELETE FROM audit_entries"} {
		err = db.WithTenant(ctxA, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, stmt)
			return err
		})
		require.ErrorContains(t, err, "permission denied", stmt)
	}
}

// testLog is a valid log in docID written by userID.
func testLog(tenantID, docID, userID string) domain.Log {
	return domain.Log{TenantID: tenantID, DocumentID: docID, Name: "Shipped", Impact: "high", Status: "done",
		CreatedAt: time.Now().UTC(), CreatedBy: userID, UpdatedBy: userID}
}

func actorIDs(as []domain.AuditActor) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.ID)
	}
	return out
}
