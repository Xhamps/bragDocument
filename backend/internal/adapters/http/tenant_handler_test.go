package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// fakeTenantUC mirrors the real use cases' admin check so memberP gets 403 everywhere.
type fakeTenantUC struct {
	members   []domain.User
	invites   []domain.Invitation
	invited   string
	removedID string
	uninvited string
}

func (f *fakeTenantUC) ListMembers(_ context.Context, actor domain.User) ([]domain.User, error) {
	if !actor.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	return f.members, nil
}

func (f *fakeTenantUC) RemoveMember(_ context.Context, actor domain.User, id string) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	f.removedID = id
	return nil
}

func (f *fakeTenantUC) ListInvitations(_ context.Context, actor domain.User) ([]domain.Invitation, error) {
	if !actor.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	return f.invites, nil
}

func (f *fakeTenantUC) Invite(_ context.Context, actor domain.User, email string) (domain.Invitation, error) {
	if !actor.IsAdmin() {
		return domain.Invitation{}, domain.ErrForbidden
	}
	f.invited = email
	return domain.Invitation{ID: "i1", Email: email}, nil
}

func (f *fakeTenantUC) Uninvite(_ context.Context, actor domain.User, id string) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	f.uninvited = id
	return nil
}

func tenantEngine(t *testing.T, p app.Principal, uc *fakeTenantUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterTenant(e.Group("/", withPrincipal(p)), uc)
	return e
}

func TestTenantMemberIsForbidden(t *testing.T) {
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/tenant/members", ""},
		{http.MethodDelete, "/tenant/members/u1", ""},
		{http.MethodGet, "/tenant/invitations", ""},
		{http.MethodPost, "/tenant/invitations", `{"email":"x@acme.com"}`},
		{http.MethodDelete, "/tenant/invitations/i1", ""},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			rec := do(tenantEngine(t, memberP, &fakeTenantUC{}), r.method, r.path, r.body)
			require.Equal(t, 403, rec.Code)
		})
	}
}

func TestTenantListMembers(t *testing.T) {
	uc := &fakeTenantUC{members: []domain.User{{ID: "u1", Email: "a@acme.com", Role: domain.RoleAdmin}}}
	rec := do(tenantEngine(t, adminP, uc), http.MethodGet, "/tenant/members", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"id":"u1"`)
	require.Contains(t, rec.Body.String(), `"role":"admin"`)
}

func TestTenantEmptyListsAreArrays(t *testing.T) {
	e := tenantEngine(t, adminP, &fakeTenantUC{})
	for _, path := range []string{"/tenant/members", "/tenant/invitations"} {
		rec := do(e, http.MethodGet, path, "")
		require.Equal(t, 200, rec.Code)
		require.Equal(t, "[]", rec.Body.String(), path)
	}
}

func TestTenantInvite(t *testing.T) {
	uc := &fakeTenantUC{}
	rec := do(tenantEngine(t, adminP, uc), http.MethodPost, "/tenant/invitations", `{"email":"new@acme.com"}`)
	require.Equal(t, 201, rec.Code)
	require.Equal(t, "new@acme.com", uc.invited)
	require.Contains(t, rec.Body.String(), `"email":"new@acme.com"`)
}

func TestTenantRemoveMember(t *testing.T) {
	uc := &fakeTenantUC{}
	rec := do(tenantEngine(t, adminP, uc), http.MethodDelete, "/tenant/members/u9", "")
	require.Equal(t, 204, rec.Code)
	require.Equal(t, "u9", uc.removedID)
}

func TestTenantListInvitations(t *testing.T) {
	uc := &fakeTenantUC{invites: []domain.Invitation{{ID: "i1", Email: "new@acme.com"}}}
	rec := do(tenantEngine(t, adminP, uc), http.MethodGet, "/tenant/invitations", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"id":"i1"`)
	require.Contains(t, rec.Body.String(), `"email":"new@acme.com"`)
}

func TestTenantUninvite(t *testing.T) {
	uc := &fakeTenantUC{}
	rec := do(tenantEngine(t, adminP, uc), http.MethodDelete, "/tenant/invitations/i9", "")
	require.Equal(t, 204, rec.Code)
	require.Equal(t, "i9", uc.uninvited)
}
