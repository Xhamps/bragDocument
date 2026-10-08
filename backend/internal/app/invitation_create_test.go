package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestTenantsInvite(t *testing.T) {
	admin := domain.User{ID: "u1", TenantID: "t1", Role: domain.RoleAdmin}
	member := domain.User{ID: "u2", TenantID: "t1", Email: "member@acme.com", Role: domain.RoleMember}
	f := newFakeTenants()
	f.members["u1"], f.members["u2"] = admin, member
	s := NewTenants(f)

	_, err := s.Invite(context.Background(), member, "new@acme.com")
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = s.Invite(context.Background(), admin, "nope")
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)

	_, err = s.Invite(context.Background(), admin, "Member@Acme.com")
	require.ErrorIs(t, err, domain.ErrConflict, "existing member")

	inv, err := s.Invite(context.Background(), admin, " New@Acme.com ")
	require.NoError(t, err)
	require.Equal(t, "new@acme.com", inv.Email)
	require.Equal(t, "t1", inv.TenantID)
	require.Equal(t, "u1", inv.CreatedBy)

	_, err = s.Invite(context.Background(), admin, "new@acme.com")
	require.ErrorIs(t, err, domain.ErrConflict, "duplicate invitation")
}
