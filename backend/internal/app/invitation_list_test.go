package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestTenantsListInvitations(t *testing.T) {
	admin := domain.User{ID: "u1", TenantID: "t1", Role: domain.RoleAdmin}
	member := domain.User{ID: "u2", TenantID: "t1", Role: domain.RoleMember}
	f := newFakeTenants()
	f.invitations["i1"] = domain.Invitation{ID: "i1", TenantID: "t1", Email: "x@acme.com"}
	s := NewTenants(f)

	_, err := s.ListInvitations(context.Background(), member)
	require.ErrorIs(t, err, domain.ErrForbidden)

	got, err := s.ListInvitations(context.Background(), admin)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "i1", got[0].ID)
}
