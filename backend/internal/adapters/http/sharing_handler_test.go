package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type fakeSharingUC struct {
	calls []string
	share app.ShareInput
	err   error
}

func (f *fakeSharingUC) rec(s string) error { f.calls = append(f.calls, s); return f.err }

func (f *fakeSharingUC) Get(_ context.Context, a domain.User, doc string) (domain.Sharing, error) {
	return domain.Sharing{
		Grants:      []domain.Grant{{UserID: "u2", Email: "bob@acme.com", Role: domain.RoleViewer, GrantedAt: time.Unix(0, 0).UTC()}},
		Invitations: []domain.DocumentInvitation{{ID: "i1", Email: "new@acme.com", Role: domain.RoleEditor}},
	}, f.rec("get " + a.ID + " " + doc)
}
func (f *fakeSharingUC) Share(_ context.Context, in app.ShareInput) (string, error) {
	f.share = in
	return app.ShareInvited, f.rec("share")
}
func (f *fakeSharingUC) ChangeRole(_ context.Context, a domain.User, doc, user string, r domain.Role) error {
	return f.rec("role " + doc + " " + user + " " + string(r))
}
func (f *fakeSharingUC) Revoke(_ context.Context, a domain.User, doc, user string) error {
	return f.rec("revoke " + doc + " " + user)
}
func (f *fakeSharingUC) CancelInvitation(_ context.Context, a domain.User, doc, inv string) error {
	return f.rec("cancel " + doc + " " + inv)
}
func (f *fakeSharingUC) Transfer(_ context.Context, a domain.User, doc, to string) error {
	return f.rec("transfer " + doc + " " + to)
}
func (f *fakeSharingUC) DocumentAudit(_ context.Context, a domain.User, doc string) ([]domain.AuditEntry, error) {
	return []domain.AuditEntry{{ID: 7, Action: domain.AuditGrant, DocumentTitle: "2026", Target: "bob@acme.com"}}, f.rec("audit " + doc)
}
func (f *fakeSharingUC) TenantAudit(_ context.Context, a domain.User) ([]domain.AuditEntry, error) {
	return []domain.AuditEntry{}, f.rec("tenant audit " + a.ID)
}

func sharingEngine(t *testing.T, uc *fakeSharingUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterSharing(e.Group("/", withPrincipal(adminP)), uc)
	return e
}

func TestSharingRoutes(t *testing.T) {
	uc := &fakeSharingUC{}
	e := sharingEngine(t, uc)

	rec := do(e, http.MethodGet, "/documents/d1/sharing", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"grants":[{"user_id":"u2","email":"bob@acme.com","display_name":"","role":"viewer"`)
	require.Contains(t, rec.Body.String(), `"invitations":[{"id":"i1","email":"new@acme.com","role":"editor"`)

	rec = do(e, http.MethodPost, "/documents/d1/sharing", `{"email":"new@acme.com","role":"editor"}`)
	require.Equal(t, 201, rec.Code)
	require.JSONEq(t, `{"kind":"invitation"}`, rec.Body.String())
	require.Equal(t, app.ShareInput{Actor: adminP.User, DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleEditor}, uc.share)

	require.Equal(t, 204, do(e, http.MethodPatch, "/documents/d1/grants/u2", `{"role":"editor"}`).Code)
	require.Equal(t, 204, do(e, http.MethodDelete, "/documents/d1/grants/u2", "").Code)
	require.Equal(t, 204, do(e, http.MethodDelete, "/documents/d1/invitations/i1", "").Code)
	require.Equal(t, 204, do(e, http.MethodPost, "/documents/d1/transfer", `{"user_id":"u2"}`).Code)

	rec = do(e, http.MethodGet, "/documents/d1/audit", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"document_title":"2026"`)
	require.Equal(t, 200, do(e, http.MethodGet, "/tenant/audit", "").Code)

	require.Equal(t, []string{"get u1 d1", "share", "role d1 u2 editor", "revoke d1 u2", "cancel d1 i1",
		"transfer d1 u2", "audit d1", "tenant audit u1"}, uc.calls)
}

func TestSharingErrorsAndBadJSON(t *testing.T) {
	uc := &fakeSharingUC{err: &domain.AccessError{Role: domain.RoleEditor, Perm: domain.PermShare}}
	e := sharingEngine(t, uc)
	rec := do(e, http.MethodDelete, "/documents/d1/grants/u2", "")
	require.Equal(t, 403, rec.Code)
	require.Contains(t, rec.Body.String(), "you are editor on this document")

	uc = &fakeSharingUC{}
	require.Equal(t, 422, do(sharingEngine(t, uc), http.MethodPost, "/documents/d1/sharing", `{bad`).Code)
	require.Empty(t, uc.calls)
}
