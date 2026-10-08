package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestTenantsListMembers(t *testing.T) {
	admin := domain.User{ID: "u1", TenantID: "t1", Role: domain.RoleAdmin}
	member := domain.User{ID: "u2", TenantID: "t1", Role: domain.RoleMember}
	f := newFakeTenants()
	f.members["u1"], f.members["u2"] = admin, member
	s := NewTenants(f)

	_, err := s.ListMembers(context.Background(), member)
	require.ErrorIs(t, err, domain.ErrForbidden)

	got, err := s.ListMembers(context.Background(), admin)
	require.NoError(t, err)
	require.Len(t, got, 2)
}
