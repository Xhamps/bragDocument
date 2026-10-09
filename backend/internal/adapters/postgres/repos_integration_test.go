//go:build integration

package postgres

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func appRoleURL(t *testing.T, ownerURL string) string {
	t.Helper()
	u, err := url.Parse(ownerURL)
	require.NoError(t, err)
	u.User = url.UserPassword("bragdoc_app", "bragdoc_app")
	return u.String()
}

// docCreated and docDeleted are the audit entries the app passes for a document.
func docCreated(ownerID string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: ownerID, Source: domain.SourceWeb, Action: domain.AuditDocumentCreated}
}

func docDeleted(actorID, docID string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actorID, Source: domain.SourceWeb, Action: domain.AuditDocumentDeleted, DocumentID: docID}
}

// logEntry is the audit entry the app passes for a log write in docID.
func logEntry(actorID, docID, action string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actorID, Source: domain.SourceWeb, Action: action, DocumentID: docID, TargetType: domain.TargetLog}
}

// tgEntry is the audit entry the app passes for a Telegram link or unlink.
func tgEntry(actorID, action string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actorID, Source: domain.SourceTelegram, Action: action}
}

// drain relays every pending outbox message straight into audit_entries, as the
// worker does through Redis (ADR-0015), so Postgres-only tests need no Redis.
func drain(t *testing.T, db *DB) {
	t.Helper()
	audits, outbox := NewAuditRepo(db), NewOutboxRepo(db)
	for {
		n, err := outbox.Relay(context.Background(), 100, func(ctx context.Context, ms []domain.OutboxMessage) error {
			for _, m := range ms {
				if err := audits.Store(ctx, m); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err)
		if n == 0 {
			return
		}
	}
}

// auditActions drains the outbox and lists the tenant's audit actions, oldest first.
func auditActions(t *testing.T, db *DB, tenantID string) []string {
	t.Helper()
	drain(t, db)
	var out []string
	require.NoError(t, db.WithTenant(context.Background(), tenantID, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT action FROM audit_entries ORDER BY id")
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, pgx.RowTo[string])
		return err
	}))
	return out
}

// provisionTenant creates a tenant with one admin through the real provisioning path.
func provisionTenant(t *testing.T, users *UserRepo, name, email string) (domain.User, domain.Tenant) {
	t.Helper()
	var u domain.User
	var tn domain.Tenant
	err := users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		var err error
		if tn, err = tx.CreateTenant(ctx, name); err != nil {
			return err
		}
		u, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: tn.ID, Email: email, Role: domain.RoleAdmin})
		return err
	})
	require.NoError(t, err)
	return u, tn
}

func TestTenantIsolationAsAppRole(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))

	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, tenants, docs := NewUserRepo(db), NewTenantRepo(db), NewDocumentRepo(db)
	adminA, tenantA := provisionTenant(t, users, "A", "a@example.com")
	_, tenantB := provisionTenant(t, users, "B", "b@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), tenantA.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tenantB.ID)

	doc, err := docs.Create(ctxA, domain.Document{TenantID: tenantA.ID, OwnerID: adminA.ID, Title: "2026"}, nil, docCreated(adminA.ID))
	require.NoError(t, err)

	// Tenant B sees nothing of A, by id or by list, on every table.
	_, err = docs.GetForUser(ctxB, doc.ID, adminA.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	list, err := docs.ListByOwner(ctxB, adminA.ID)
	require.NoError(t, err)
	require.Empty(t, list)
	_, err = docs.Update(ctxB, domain.Document{ID: doc.ID, Title: "hijack", State: domain.DocumentActive}, domain.AuditEntry{ActorID: adminA.ID, Source: domain.SourceWeb, Action: domain.AuditDocumentRenamed, DocumentID: doc.ID})
	require.ErrorIs(t, err, domain.ErrNotFound)
	members, err := tenants.ListMembers(ctxB)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.NotEqual(t, adminA.ID, members[0].ID)

	// Inserting into another tenant is rejected by WITH CHECK.
	_, err = docs.Create(ctxB, domain.Document{TenantID: tenantA.ID, OwnerID: adminA.ID, Title: "x"}, nil, docCreated(adminA.ID))
	require.ErrorIs(t, err, domain.ErrForbidden)

	// Provisioning reads users and tenants across tenants, but documents stays closed.
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		u, err := tx.GetUser(ctx, adminA.ID)
		require.NoError(t, err)
		require.Equal(t, adminA.ID, u.ID)
		tn, err := tx.GetTenant(ctx, tenantA.ID)
		require.NoError(t, err)
		require.Equal(t, tenantA.ID, tn.ID)
		return nil
	}))
	err = db.WithProvisioning(context.Background(), func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO documents (tenant_id, owner_id, title) VALUES ($1, $2, 'x')", tenantA.ID, adminA.ID)
		return wrap(err)
	})
	require.ErrorIs(t, err, domain.ErrForbidden)

	// Tenant A sees its own document and can delete it.
	got, err := docs.GetForUser(ctxA, doc.ID, adminA.ID)
	require.NoError(t, err)
	require.Equal(t, "2026", got.Title)

	// A member who owns documents cannot be removed; after delete they can.
	require.ErrorIs(t, tenants.DeleteMember(ctxA, adminA.ID), domain.ErrConflict)
	require.NoError(t, docs.Delete(ctxA, doc.ID, docDeleted(adminA.ID, doc.ID)))
	require.ErrorIs(t, docs.Delete(ctxA, doc.ID, docDeleted(adminA.ID, doc.ID)), domain.ErrNotFound)

	// Invitations: create, duplicate conflicts, provisioning finds it across tenants.
	inv, err := tenants.CreateInvitation(ctxA, domain.Invitation{TenantID: tenantA.ID, Email: "c@example.com", CreatedBy: adminA.ID})
	require.NoError(t, err)
	_, err = tenants.CreateInvitation(ctxA, domain.Invitation{TenantID: tenantA.ID, Email: "c@example.com", CreatedBy: adminA.ID})
	require.ErrorIs(t, err, domain.ErrConflict)
	invsA, err := tenants.ListInvitations(ctxA)
	require.NoError(t, err)
	require.Len(t, invsA, 1)
	invsB, err := tenants.ListInvitations(ctxB)
	require.NoError(t, err)
	require.Empty(t, invsB)
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		found, err := tx.FindInvitationByEmail(ctx, "c@example.com")
		if err != nil {
			return err
		}
		require.Equal(t, inv.ID, found.ID)
		require.False(t, found.ForDocument)
		return nil
	}))
	require.NoError(t, tenants.DeleteInvitation(ctxA, inv.ID))
	invs, err := tenants.ListInvitations(ctxA)
	require.NoError(t, err)
	require.Empty(t, invs)
}
