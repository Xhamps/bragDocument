package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type fakeAuditUC struct {
	f     domain.AuditFilter
	actor domain.User
	page  *domain.AuditPage // nil: one log.edited entry
	err   error
}

func (u *fakeAuditUC) List(_ context.Context, a domain.User, f domain.AuditFilter) (domain.AuditPage, error) {
	u.actor, u.f = a, f
	if u.page != nil {
		return *u.page, u.err
	}
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	return domain.AuditPage{NextBefore: 41, Entries: []domain.AuditEntry{{
		ID: 42, At: at, Source: domain.SourceWeb, Action: domain.AuditLogEdited,
		ActorID: "u1", ActorName: "Ana", ActorEmail: "ana@acme.com",
		DocumentID: "d1", DocumentTitle: "2026",
		TargetType: domain.TargetLog, TargetID: "l1", Target: "Migrated billing",
		ChangedFields: []string{"name"},
	}}}, u.err
}
func (u *fakeAuditUC) Filters(_ context.Context, a domain.User) ([]domain.AuditActor, []domain.AuditDocument, error) {
	u.actor = a
	return []domain.AuditActor{{ID: "u1", Name: "Ana", Email: "ana@acme.com"}}, []domain.AuditDocument{{ID: "d1", Title: "2026"}}, u.err
}

func auditEngine(t *testing.T, uc *fakeAuditUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	g := e.Group("/", withPrincipal(adminP))
	RegisterSharing(g, &fakeSharingUC{}) // /documents/:id/audit must coexist with the other /documents/:id routes
	RegisterAudit(g, uc)
	return e
}

func TestAuditRoutes(t *testing.T) {
	uc := &fakeAuditUC{}
	e := auditEngine(t, uc)

	const u1, d1 = "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	rec := do(e, http.MethodGet, "/audit?actor="+u1+"&document="+d1+"&action=log.edited&from=2026-10-01&to=2026-10-09&before=99&limit=20", "")
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `{"next_before":41,"entries":[{"id":42,"at":"2026-10-09T12:00:00Z","source":"web","action":"log.edited",
		"actor":{"id":"u1","name":"Ana","email":"ana@acme.com"},"document":{"id":"d1","title":"2026"},
		"target":{"type":"log","id":"l1","name":"Migrated billing"},"role":"","changed_fields":["name"]}]}`, rec.Body.String())
	require.Equal(t, adminP.User, uc.actor)
	require.Equal(t, u1, uc.f.ActorID)
	require.Equal(t, d1, uc.f.DocumentID)
	require.Equal(t, domain.AuditLogEdited, uc.f.Action)
	require.Equal(t, int64(99), uc.f.Before)
	require.Equal(t, 20, uc.f.Limit)
	require.Equal(t, "2026-10-01", uc.f.From.Format(time.DateOnly))
	require.Equal(t, "2026-10-10", uc.f.To.Format(time.DateOnly), "to is inclusive in the URL")

	rec = do(e, http.MethodGet, "/documents/d9/audit?document="+d1+"&limit=10", "")
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "d9", uc.f.DocumentID, "the path wins")
	require.Equal(t, 10, uc.f.Limit)

	rec = do(e, http.MethodGet, "/audit/filters", "")
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `{"actors":[{"id":"u1","name":"Ana","email":"ana@acme.com"}],"documents":[{"id":"d1","title":"2026"}]}`, rec.Body.String())

	require.Equal(t, 422, do(e, http.MethodGet, "/audit?before=abc", "").Code)
	require.Equal(t, 422, do(e, http.MethodGet, "/audit?limit=x", "").Code)
	require.Equal(t, 422, do(e, http.MethodGet, "/audit?from=yesterday", "").Code)
	rec = do(e, http.MethodGet, "/audit?actor=u1&document=d1", "")
	require.Equal(t, 422, rec.Code)
	require.Contains(t, rec.Body.String(), `"actor":"must be a uuid"`)
	require.Contains(t, rec.Body.String(), `"document":"must be a uuid"`)
}

func TestAuditNullActorAndDocument(t *testing.T) {
	uc := &fakeAuditUC{page: &domain.AuditPage{Entries: []domain.AuditEntry{{
		ID: 1, At: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), Source: domain.SourceSystem, Action: domain.AuditTelegramUnlinked,
	}}}}
	rec := do(auditEngine(t, uc), http.MethodGet, "/audit", "")
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `{"next_before":null,"entries":[{"id":1,"at":"2026-10-09T00:00:00Z","source":"system","action":"telegram.unlinked",
		"actor":{"id":null,"name":"","email":""},"document":null,
		"target":{"type":"","id":"","name":""},"role":"","changed_fields":null}]}`, rec.Body.String())
}

func TestAuditForbidden(t *testing.T) {
	e := auditEngine(t, &fakeAuditUC{err: domain.ErrForbidden})
	require.Equal(t, 403, do(e, http.MethodGet, "/audit", "").Code)
	require.Equal(t, 403, do(e, http.MethodGet, "/audit/filters", "").Code)
}
