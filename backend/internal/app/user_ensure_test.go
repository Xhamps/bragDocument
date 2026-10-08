package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestUserEnsureExistingUser(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.users["u1"] = domain.User{ID: "u1", TenantID: "t1", Email: "a@acme.com", Role: domain.RoleMember}

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u1", Email: "A@acme.com"})
	require.NoError(t, err)
	require.Equal(t, "u1", p.User.ID)
	require.Equal(t, "Acme", p.Tenant.Name)
}

func TestUserEnsureJoinsInvitedTenant(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.invitations["new@acme.com"] = domain.Invitation{ID: "i1", TenantID: "t1", Email: "new@acme.com"}

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u2", Email: "New@Acme.com", DisplayName: "New"})
	require.NoError(t, err)
	require.Equal(t, "t1", p.User.TenantID)
	require.Equal(t, domain.RoleMember, p.User.Role)
	require.Equal(t, "new@acme.com", p.User.Email)
	require.Empty(t, f.invitations, "invitation consumed")
}

func TestUserEnsureCreatesTenantForNewUser(t *testing.T) {
	f := newFakeUsers()

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u3", Email: "solo@example.com"})
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, p.User.Role)
	require.Equal(t, "solo", p.Tenant.Name, "tenant named after the email local part when no display name")
	require.Equal(t, p.Tenant.ID, p.User.TenantID)
}

func TestUserEnsureRetriesOnceOnConflict(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t9"] = domain.Tenant{ID: "t9", Name: "x"}
	// First pass: not found → create → conflict; the fake inserts the concurrent winner's row.
	f.createUserErrOnce = domain.ErrConflict

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u4", Email: "r@example.com"})
	require.NoError(t, err)
	require.Equal(t, "t9", p.Tenant.ID)
}

func TestUserEnsureRejectsBadEmail(t *testing.T) {
	_, err := NewUserEnsure(newFakeUsers()).Execute(context.Background(), EnsureUserInput{ID: "u5", Email: "nope"})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
