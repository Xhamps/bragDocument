//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func addMember(t *testing.T, users *UserRepo, tn domain.Tenant, email string) domain.User {
	t.Helper()
	var u domain.User
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		var err error
		u, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: tn.ID, Email: email, Role: domain.RoleMember})
		return err
	}))
	return u
}

func TestSharing(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, sharing, tenants := NewUserRepo(db), NewDocumentRepo(db), NewSharingRepo(db), NewTenantRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	zed, tb := provisionTenant(t, users, "B", "zed@example.com")
	bob := addMember(t, users, ta, "bob@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)

	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil)
	require.NoError(t, err)
	audit := func(action, target string, role domain.Role) domain.AuditEntry {
		return domain.AuditEntry{ActorID: ada.ID, ActorEmail: ada.Email, Action: action, DocumentID: doc.ID, Target: target, Role: role}
	}

	// Roles: owner; an ungranted member sees nothing.
	got, err := docs.GetForUser(ctxA, doc.ID, ada.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, got.Role)
	require.False(t, got.IsNew)
	_, err = docs.GetForUser(ctxA, doc.ID, bob.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Members resolve by email only inside the tenant.
	m, err := sharing.MemberByEmail(ctxA, "bob@example.com")
	require.NoError(t, err)
	require.Equal(t, bob.ID, m.ID)
	_, err = sharing.MemberByEmail(ctxB, "bob@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Grant as viewer: New until seen; not writable.
	g := domain.Grant{DocumentID: doc.ID, UserID: bob.ID, Role: domain.RoleViewer, GrantedBy: ada.ID}
	require.NoError(t, sharing.Grant(ctxA, g, audit(domain.AuditGrant, bob.Email, domain.RoleViewer)))
	require.ErrorIs(t, sharing.Grant(ctxA, g, audit(domain.AuditGrant, bob.Email, domain.RoleViewer)), domain.ErrConflict)
	// Another tenant's user can be neither granted nor resolved.
	require.ErrorIs(t, sharing.Grant(ctxA, domain.Grant{DocumentID: doc.ID, UserID: zed.ID, Role: domain.RoleViewer, GrantedBy: ada.ID},
		audit(domain.AuditGrant, zed.Email, domain.RoleViewer)), domain.ErrConflict)
	entries, err := sharing.Audit(ctxA, doc.ID)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the failed grant wrote no audit row")
	_, err = sharing.Member(ctxA, zed.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = sharing.Member(ctxA, bob.ID)
	require.NoError(t, err)
	got, err = docs.GetForUser(ctxA, doc.ID, bob.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleViewer, got.Role)
	require.True(t, got.IsNew)
	require.Equal(t, "ada@example.com", got.OwnerName)
	shared, err := docs.ListShared(ctxA, bob.ID)
	require.NoError(t, err)
	require.Len(t, shared, 1)
	require.True(t, shared[0].IsNew)
	require.NoError(t, docs.MarkSeen(ctxA, doc.ID, bob.ID))
	shared, err = docs.ListShared(ctxA, bob.ID)
	require.NoError(t, err)
	require.False(t, shared[0].IsNew)
	writable, err := docs.ListWritable(ctxA, bob.ID)
	require.NoError(t, err)
	require.Empty(t, writable)

	// Editor can write.
	require.NoError(t, sharing.SetRole(ctxA, doc.ID, bob.ID, domain.RoleEditor, audit(domain.AuditRoleChange, bob.Email, domain.RoleEditor)))
	writable, err = docs.ListWritable(ctxA, bob.ID)
	require.NoError(t, err)
	require.Len(t, writable, 1)

	// Tenant B sees none of it.
	_, err = docs.GetForUser(ctxB, doc.ID, bob.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	entriesB, err := sharing.Audit(ctxB, "")
	require.NoError(t, err)
	require.Empty(t, entriesB)

	// Invitation for an unknown email; admins see it; sign-in turns it into a grant.
	inv := domain.DocumentInvitation{DocumentID: doc.ID, Email: "new@example.com", Role: domain.RoleEditor, InvitedBy: ada.ID}
	created, err := sharing.Invite(ctxA, inv, audit(domain.AuditInvite, inv.Email, domain.RoleEditor))
	require.NoError(t, err)
	_, err = sharing.Invite(ctxA, inv, audit(domain.AuditInvite, inv.Email, domain.RoleEditor))
	require.ErrorIs(t, err, domain.ErrConflict)
	sh, err := sharing.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Len(t, sh.Grants, 1)
	require.Equal(t, "bob@example.com", sh.Grants[0].Email)
	require.Len(t, sh.Invitations, 1)
	invs, err := tenants.ListInvitations(ctxA)
	require.NoError(t, err)
	require.Len(t, invs, 1)
	require.True(t, invs[0].ForDocument)
	require.Equal(t, "2026", invs[0].DocumentTitle)

	newID := uuid.NewString()
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		found, err := tx.FindInvitationByEmail(ctx, "new@example.com")
		require.NoError(t, err)
		require.True(t, found.ForDocument)
		require.Equal(t, ta.ID, found.TenantID)
		u, err := tx.CreateUser(ctx, domain.User{ID: newID, TenantID: found.TenantID, Email: "new@example.com", Role: domain.RoleMember})
		if err != nil {
			return err
		}
		return tx.AcceptInvitations(ctx, u)
	}))
	got, err = docs.GetForUser(ctxA, doc.ID, newID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleEditor, got.Role)
	sh, err = sharing.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Empty(t, sh.Invitations)
	require.ErrorIs(t, sharing.CancelInvitation(ctxA, doc.ID, created.ID, audit(domain.AuditInviteCancel, inv.Email, inv.Role)), domain.ErrNotFound, "accepted, no longer pending")

	// Cancel a pending invitation.
	other, err := sharing.Invite(ctxA, domain.DocumentInvitation{DocumentID: doc.ID, Email: "other@example.com", Role: domain.RoleViewer, InvitedBy: ada.ID},
		audit(domain.AuditInvite, "other@example.com", domain.RoleViewer))
	require.NoError(t, err)
	require.NoError(t, sharing.CancelInvitation(ctxA, doc.ID, other.ID, audit(domain.AuditInviteCancel, other.Email, other.Role)))
	sh, err = sharing.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Empty(t, sh.Invitations)

	// A transfer to another tenant's user rolls back whole.
	before, err := sharing.Audit(ctxA, doc.ID)
	require.NoError(t, err)
	require.ErrorIs(t, sharing.Transfer(ctxA, doc.ID, ada.ID, zed.ID, audit(domain.AuditTransfer, zed.Email, domain.RoleOwner)), domain.ErrConflict)
	got, err = docs.GetForUser(ctxA, doc.ID, ada.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, got.Role)
	after, err := sharing.Audit(ctxA, doc.ID)
	require.NoError(t, err)
	require.Len(t, after, len(before))

	// Transfer: bob owns it, ada edits it.
	require.NoError(t, sharing.Transfer(ctxA, doc.ID, ada.ID, bob.ID, audit(domain.AuditTransfer, bob.Email, domain.RoleOwner)))
	got, err = docs.GetForUser(ctxA, doc.ID, bob.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, got.Role)
	got, err = docs.GetForUser(ctxA, doc.ID, ada.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleEditor, got.Role)

	require.ErrorIs(t, sharing.Transfer(ctxA, doc.ID, ada.ID, newID, audit(domain.AuditTransfer, "new@example.com", domain.RoleOwner)),
		domain.ErrNotFound, "ada no longer owns it")

	// Revocation takes effect at once.
	require.NoError(t, sharing.Revoke(ctxA, doc.ID, newID, audit(domain.AuditRevoke, "new@example.com", domain.RoleEditor)))
	_, err = docs.GetForUser(ctxA, doc.ID, newID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, sharing.Revoke(ctxA, doc.ID, newID, audit(domain.AuditRevoke, "x", "")), domain.ErrNotFound)

	// Audit: newest first, copies the title, survives deletion, cannot be rewritten.
	entries, err = sharing.Audit(ctxA, doc.ID)
	require.NoError(t, err)
	require.Contains(t, actions(entries), domain.AuditInviteCancel)
	require.Equal(t, domain.AuditRevoke, entries[0].Action)
	require.Equal(t, domain.AuditGrant, entries[len(entries)-1].Action)
	require.Equal(t, "2026", entries[0].DocumentTitle)
	require.Contains(t, actions(entries), domain.AuditInviteAccept)
	require.NoError(t, docs.Delete(ctxA, doc.ID))
	all, err := sharing.Audit(ctxA, "")
	require.NoError(t, err)
	require.Len(t, all, len(entries))
	err = db.WithTenant(ctxA, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM audit_entries")
		return wrap(err)
	})
	require.ErrorIs(t, err, domain.ErrForbidden)

	// Removing a member cascades their grants.
	doc2, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2027"}, nil)
	require.NoError(t, err)
	require.NoError(t, sharing.Grant(ctxA, domain.Grant{DocumentID: doc2.ID, UserID: bob.ID, Role: domain.RoleViewer, GrantedBy: ada.ID},
		domain.AuditEntry{ActorID: ada.ID, ActorEmail: ada.Email, Action: domain.AuditGrant, DocumentID: doc2.ID, Target: bob.Email, Role: domain.RoleViewer}))
	require.NoError(t, tenants.DeleteMember(ctxA, bob.ID))
	sh, err = sharing.Get(ctxA, doc2.ID)
	require.NoError(t, err)
	require.Empty(t, sh.Grants)
}

func actions(es []domain.AuditEntry) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.Action)
	}
	return out
}
