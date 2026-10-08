package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestTenantsRemoveMember(t *testing.T) {
	admin := domain.User{ID: "u1", TenantID: "t1", Role: domain.RoleAdmin}
	member := domain.User{ID: "u2", TenantID: "t1", Role: domain.RoleMember}
	f := newFakeTenants()
	f.members["u1"], f.members["u2"] = admin, member
	s := NewTenants(f)

	require.ErrorIs(t, s.RemoveMember(context.Background(), member, "u1"), domain.ErrForbidden)

	var ve *domain.ValidationError
	require.ErrorAs(t, s.RemoveMember(context.Background(), admin, "u1"), &ve)

	require.NoError(t, s.RemoveMember(context.Background(), admin, "u2"))
	require.NotContains(t, f.members, "u2")

	require.ErrorIs(t, s.RemoveMember(context.Background(), admin, "missing"), domain.ErrNotFound)
}
