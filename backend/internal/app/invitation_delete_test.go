package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestTenantsUninvite(t *testing.T) {
	admin := domain.User{ID: "u1", TenantID: "t1", Role: domain.RoleAdmin}
	member := domain.User{ID: "u2", TenantID: "t1", Role: domain.RoleMember}
	f := newFakeTenants()
	f.invitations["i1"] = domain.Invitation{ID: "i1", TenantID: "t1", Email: "x@acme.com"}
	s := NewTenants(f)

	require.ErrorIs(t, s.Uninvite(context.Background(), member, "i1"), domain.ErrForbidden)
	require.NoError(t, s.Uninvite(context.Background(), admin, "i1"))
	require.Empty(t, f.invitations)
	require.ErrorIs(t, s.Uninvite(context.Background(), admin, "i1"), domain.ErrNotFound)
}
