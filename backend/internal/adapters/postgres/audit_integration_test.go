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

// TestEveryActionWritesOneEntry performs each audited action once through the
// real repos with the entry the app builds, then expects exactly one entry per
// action: a new action without a case fails here (PRD-0009 success metric).
func TestEveryActionWritesOneEntry(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs, sharing := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db), NewSharingRepo(db)
	links, exports, audits := NewTelegramLinkRepo(db), NewExportRepo(db), NewAuditRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	bob := addMember(t, users, ta, "bob@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)

	var doc domain.Document
	var lg domain.Log
	var invID, newID string
	share := func(action, targetType, targetID, target string, role domain.Role) domain.AuditEntry {
		return domain.AuditEntry{ActorID: ada.ID, Source: domain.SourceWeb, Action: action, DocumentID: doc.ID,
			TargetType: targetType, TargetID: targetID, Target: target, Role: role}
	}
	updateDoc := func(change func(*domain.Document)) func() error {
		return func() error {
			next := doc
			change(&next)
			action, fields := domain.DocumentChange(doc, next)
			a := docCreated(ada.ID)
			a.Action, a.DocumentID, a.ChangedFields = action, doc.ID, fields
			var err error
			doc, err = docs.Update(ctxA, next, a)
			return err
		}
	}
	updateLog := func(change func(*domain.Log)) func() error {
		return func() error {
			next := lg
			change(&next)
			action, fields := domain.LogChange(lg, next)
			a := logEntry(ada.ID, doc.ID, action)
			a.TargetID, a.Target, a.ChangedFields = lg.ID, next.Name, fields
			var err error
			lg, err = logs.Update(ctxA, next, a)
			return err
		}
	}

	cases := []struct {
		action string
		do     func() error
	}{
		{domain.AuditDocumentCreated, func() error {
			doc, err = docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil, docCreated(ada.ID))
			return err
		}},
		{domain.AuditDocumentRenamed, updateDoc(func(d *domain.Document) { d.Title = "2026 brag" })},
		{domain.AuditDocumentEdited, updateDoc(func(d *domain.Document) { d.Description = "the year" })},
		{domain.AuditDocumentArchived, updateDoc(func(d *domain.Document) { d.State = domain.DocumentArchived })},
		{domain.AuditDocumentUnarchived, updateDoc(func(d *domain.Document) { d.State = domain.DocumentActive })},

		{domain.AuditLogCreated, func() error {
			a := logEntry(ada.ID, doc.ID, domain.AuditLogCreated)
			a.Target = "Shipped"
			lg, err = logs.Create(ctxA, testLog(ta.ID, doc.ID, ada.ID), a)
			return err
		}},
		{domain.AuditLogEdited, updateLog(func(l *domain.Log) { l.Name = "Shipped v2" })},
		{domain.AuditLogStatusChanged, updateLog(func(l *domain.Log) { l.Status = "in_progress" })},
		{domain.AuditLogDeleted, func() error {
			a := logEntry(ada.ID, doc.ID, domain.AuditLogDeleted)
			a.TargetID, a.Target = lg.ID, lg.Name
			return logs.Delete(ctxA, doc.ID, lg.ID, a)
		}},

		{domain.AuditGrant, func() error {
			return sharing.Grant(ctxA, domain.Grant{DocumentID: doc.ID, UserID: bob.ID, Role: domain.RoleViewer, GrantedBy: ada.ID},
				share(domain.AuditGrant, domain.TargetUser, bob.ID, bob.Email, domain.RoleViewer))
		}},
		{domain.AuditRoleChange, func() error {
			return sharing.SetRole(ctxA, doc.ID, bob.ID, domain.RoleEditor, share(domain.AuditRoleChange, domain.TargetUser, bob.ID, bob.Email, domain.RoleEditor))
		}},
		{domain.AuditInvite, func() error {
			inv, err := sharing.Invite(ctxA, domain.DocumentInvitation{DocumentID: doc.ID, Email: "new@example.com", Role: domain.RoleViewer, InvitedBy: ada.ID},
				share(domain.AuditInvite, domain.TargetInvitation, "", "new@example.com", domain.RoleViewer))
			invID = inv.ID
			return err
		}},
		{domain.AuditInviteCancel, func() error {
			return sharing.CancelInvitation(ctxA, doc.ID, invID, share(domain.AuditInviteCancel, domain.TargetInvitation, invID, "new@example.com", domain.RoleViewer))
		}},
		{domain.AuditInviteAccept, func() error {
			if _, err := sharing.Invite(ctxA, domain.DocumentInvitation{DocumentID: doc.ID, Email: "new@example.com", Role: domain.RoleViewer, InvitedBy: ada.ID},
				share(domain.AuditInvite, domain.TargetInvitation, "", "new@example.com", domain.RoleViewer)); err != nil {
				return err
			}
			newID = uuid.NewString()
			return users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
				u, err := tx.CreateUser(ctx, domain.User{ID: newID, TenantID: ta.ID, Email: "new@example.com", Role: domain.RoleMember})
				if err != nil {
					return err
				}
				return tx.AcceptInvitations(ctx, u)
			})
		}},
		{domain.AuditTransfer, func() error {
			return sharing.Transfer(ctxA, doc.ID, ada.ID, bob.ID, share(domain.AuditTransfer, domain.TargetUser, bob.ID, bob.Email, domain.RoleOwner))
		}},
		{domain.AuditRevoke, func() error {
			return sharing.Revoke(ctxA, doc.ID, newID, share(domain.AuditRevoke, domain.TargetUser, newID, "new@example.com", domain.RoleViewer))
		}},

		{domain.AuditTelegramLinked, func() error {
			return links.Link(context.Background(), domain.TelegramLink{UserID: ada.ID, TenantID: ta.ID, TelegramUserID: 42, LinkedAt: time.Now()},
				tgEntry(ada.ID, domain.AuditTelegramLinked))
		}},
		{domain.AuditTelegramUnlinked, func() error { return links.Delete(ctxA, ada.ID, tgEntry(ada.ID, domain.AuditTelegramUnlinked)) }},

		{domain.AuditExportRequested, func() error {
			a := docCreated(ada.ID)
			a.Action, a.DocumentID = domain.AuditExportRequested, doc.ID
			_, err := exports.Create(ctxA, domain.ExportJob{TenantID: ta.ID, DocumentID: doc.ID, RequestedBy: ada.ID}, a)
			return err
		}},
		{domain.AuditDocumentDeleted, func() error { return docs.Delete(ctxA, doc.ID, docDeleted(ada.ID, doc.ID)) }},
	}
	for _, c := range cases {
		require.NoError(t, c.do(), c.action)
	}
	for _, action := range domain.AuditActions {
		want := 1
		if action == domain.AuditInvite {
			want = 2 // sent twice: once to cancel, once to accept
		}
		page, err := audits.List(ctxA, domain.AuditFilter{Action: action, Limit: 50})
		require.NoError(t, err)
		require.Len(t, page.Entries, want, action)
	}
}

// TestFailedAuditRollsBackTheAction: an entry whose DocumentID matches nothing
// makes audit() return ErrNotFound; the action in the same transaction must not
// persist (FR-1).
func TestFailedAuditRollsBackTheAction(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil, docCreated(ada.ID))
	require.NoError(t, err)

	_, err = logs.Create(ctxA, testLog(ta.ID, doc.ID, ada.ID), logEntry(ada.ID, uuid.NewString(), domain.AuditLogCreated))
	require.ErrorIs(t, err, domain.ErrNotFound)
	page, err := logs.List(ctxA, doc.ID, domain.LogFilter{Sort: "created_at", Page: 1, PerPage: 50})
	require.NoError(t, err)
	require.Zero(t, page.Total, "the log was rolled back with its entry")
	require.Equal(t, []string{domain.AuditDocumentCreated}, auditActions(t, db, ta.ID))
}
