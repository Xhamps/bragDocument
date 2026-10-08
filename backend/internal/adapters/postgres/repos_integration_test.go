//go:build integration

package postgres

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
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

	doc, err := docs.Create(ctxA, domain.Document{TenantID: tenantA.ID, OwnerID: adminA.ID, Title: "2026"})
	require.NoError(t, err)

	// Tenant B sees nothing of A, by id or by list, on every table.
	_, err = docs.Get(ctxB, doc.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	list, err := docs.ListByOwner(ctxB, adminA.ID)
	require.NoError(t, err)
	require.Empty(t, list)
	members, err := tenants.ListMembers(ctxB)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.NotEqual(t, adminA.ID, members[0].ID)

	// Inserting into another tenant is rejected by WITH CHECK.
	_, err = docs.Create(ctxB, domain.Document{TenantID: tenantA.ID, OwnerID: adminA.ID, Title: "x"})
	require.Error(t, err)

	// Tenant A sees its own document and can delete it.
	got, err := docs.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Equal(t, "2026", got.Title)

	// A member who owns documents cannot be removed; after delete they can.
	require.ErrorIs(t, tenants.DeleteMember(ctxA, adminA.ID), domain.ErrConflict)
	require.NoError(t, docs.Delete(ctxA, doc.ID))
	require.ErrorIs(t, docs.Delete(ctxA, doc.ID), domain.ErrNotFound)

	// Invitations: create, duplicate conflicts, provisioning finds it across tenants.
	inv, err := tenants.CreateInvitation(ctxA, domain.Invitation{TenantID: tenantA.ID, Email: "c@example.com", CreatedBy: adminA.ID})
	require.NoError(t, err)
	_, err = tenants.CreateInvitation(ctxA, domain.Invitation{TenantID: tenantA.ID, Email: "c@example.com", CreatedBy: adminA.ID})
	require.ErrorIs(t, err, domain.ErrConflict)
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		found, err := tx.FindInvitationByEmail(ctx, "c@example.com")
		if err != nil {
			return err
		}
		require.Equal(t, inv.ID, found.ID)
		return tx.DeleteInvitation(ctx, found.ID)
	}))
	invs, err := tenants.ListInvitations(ctxA)
	require.NoError(t, err)
	require.Empty(t, invs)
}
