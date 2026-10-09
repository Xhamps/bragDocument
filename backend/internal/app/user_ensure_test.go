package app

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

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

func TestUserEnsureRetriesOnConflictIntoInvitedTenant(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.invitations["r@acme.com"] = domain.Invitation{ID: "i1", TenantID: "t1", Email: "r@acme.com"}
	f.createUserErrOnce = domain.ErrConflict

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u6", Email: "r@acme.com"})
	require.NoError(t, err)
	require.Equal(t, "t1", p.Tenant.ID)
	require.Equal(t, "t1", p.User.TenantID)
	require.Equal(t, 1, f.createUserCalls, "second pass finds the winner's row")
}

func TestUserEnsureRetryConflictIsWrapped(t *testing.T) {
	f := newFakeUsers()
	f.createUserErrAlways = domain.ErrConflict // both passes miss and conflict

	_, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u7", Email: "c@example.com"})
	require.ErrorIs(t, err, domain.ErrConflict)
	require.Contains(t, err.Error(), "u7")
	require.Equal(t, 2, f.createUserCalls)
}

func TestUserEnsureDoesNotRetryNonConflict(t *testing.T) {
	f := newFakeUsers()
	f.createUserErrOnce = domain.ErrUnavailable

	_, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u8", Email: "d@example.com"})
	require.ErrorIs(t, err, domain.ErrUnavailable)
	require.Equal(t, 1, f.createUserCalls)
}

func TestUserEnsureTruncatesLongNames(t *testing.T) {
	f := newFakeUsers()
	long := strings.Repeat("é", 250)

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u9", Email: "l@example.com", DisplayName: long})
	require.NoError(t, err)
	require.Equal(t, 200, utf8.RuneCountInString(p.User.DisplayName))
	require.Equal(t, 200, utf8.RuneCountInString(p.Tenant.Name))
}

func TestUserEnsureRejectsBadEmail(t *testing.T) {
	_, err := NewUserEnsure(newFakeUsers()).Execute(context.Background(), EnsureUserInput{ID: "u5", Email: "nope"})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestUserEnsureAcceptsDocumentInvitation(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.invitations["new@acme.com"] = domain.Invitation{ID: "di1", TenantID: "t1", Email: "new@acme.com", ForDocument: true}

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u2", Email: "new@acme.com"})
	require.NoError(t, err)
	require.Equal(t, "t1", p.User.TenantID, "a document invitation joins its tenant")
	require.Equal(t, domain.RoleMember, p.User.Role)
	require.Equal(t, []string{"u2"}, f.accepted)
}
