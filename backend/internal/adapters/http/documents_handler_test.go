package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type fakeDocUC struct {
	created   app.CreateDocumentInput
	updated   app.UpdateDocumentInput
	deletedID string
	deletedBy string
	getID     string
	getBy     string
	doc       domain.Document
	err       error
}

func (f *fakeDocUC) List(context.Context, string) (app.DocumentListOutput, error) {
	return app.DocumentListOutput{Owned: []domain.Document{f.doc}, Shared: []domain.Document{}}, f.err
}

func (f *fakeDocUC) Create(_ context.Context, in app.CreateDocumentInput) (domain.Document, error) {
	f.created = in
	return f.doc, f.err
}

func (f *fakeDocUC) Update(_ context.Context, in app.UpdateDocumentInput) (domain.Document, error) {
	f.updated = in
	return f.doc, f.err
}

func (f *fakeDocUC) Delete(_ context.Context, id, userID string) error {
	f.deletedID, f.deletedBy = id, userID
	return f.err
}

func (f *fakeDocUC) Get(_ context.Context, id, userID string) (domain.Document, error) {
	f.getID, f.getBy = id, userID
	return f.doc, f.err
}

func docsEngine(t *testing.T, uc *fakeDocUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterDocuments(e.Group("/", withPrincipal(adminP)), uc)
	return e
}

func TestDocumentsList(t *testing.T) {
	ts := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	uc := &fakeDocUC{doc: domain.Document{ID: "d1", OwnerID: "u1", Title: "Q3", LogCount: 2, LastLogAt: &ts}}
	rec := do(docsEngine(t, uc), http.MethodGet, "/documents", "")
	require.Equal(t, 200, rec.Code)
	var body DocumentListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Owned, 1)
	require.Equal(t, "d1", body.Owned[0].ID)
	require.Equal(t, 2, body.Owned[0].LogCount)
	require.Contains(t, rec.Body.String(), `"last_log_at":"2026-03-01T00:00:00Z"`)
	require.Contains(t, rec.Body.String(), `"shared":[]`)
}

func TestDocumentsCreate(t *testing.T) {
	uc := &fakeDocUC{doc: domain.Document{ID: "d1", OwnerID: "u1", Title: "Q3"}}
	rec := do(docsEngine(t, uc), http.MethodPost, "/documents", `{"title":"Q3","description":"wins"}`)
	require.Equal(t, 201, rec.Code)
	require.Equal(t, app.CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "Q3", Description: "wins"}, uc.created)
	require.Contains(t, rec.Body.String(), `"id":"d1"`)
}

func TestDocumentsCreateInvalidJSON(t *testing.T) {
	uc := &fakeDocUC{}
	rec := do(docsEngine(t, uc), http.MethodPost, "/documents", `{not json`)
	require.Equal(t, 422, rec.Code)
	require.Empty(t, uc.created.Title, "use case never called")
}

func TestDocumentsUpdate(t *testing.T) {
	uc := &fakeDocUC{doc: domain.Document{ID: "d1", OwnerID: "u1", Title: "Q4"}}
	rec := do(docsEngine(t, uc), http.MethodPatch, "/documents/d1", `{"title":"Q4","state":"archived"}`)
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "d1", uc.updated.ID)
	require.Equal(t, "u1", uc.updated.UserID)
	require.Equal(t, "Q4", *uc.updated.Title)
	require.Equal(t, "archived", *uc.updated.State)
	require.Nil(t, uc.updated.Description)
}

func TestDocumentsDelete(t *testing.T) {
	uc := &fakeDocUC{}
	rec := do(docsEngine(t, uc), http.MethodDelete, "/documents/d1", "")
	require.Equal(t, 204, rec.Code)
	require.Equal(t, "d1", uc.deletedID)
	require.Equal(t, "u1", uc.deletedBy)
}

func TestDocumentsDeleteForbidden(t *testing.T) {
	rec := do(docsEngine(t, &fakeDocUC{err: domain.ErrForbidden}), http.MethodDelete, "/documents/d1", "")
	require.Equal(t, 403, rec.Code)
}

func TestDocumentsUpdateInvalidJSON(t *testing.T) {
	uc := &fakeDocUC{}
	rec := do(docsEngine(t, uc), http.MethodPatch, "/documents/d1", `{not json`)
	require.Equal(t, 422, rec.Code)
	require.Empty(t, uc.updated.ID, "use case never called")
}

func TestDocumentsGet(t *testing.T) {
	uc := &fakeDocUC{doc: domain.Document{ID: "d1", OwnerID: "u9", Title: "Q3", Role: domain.RoleViewer, OwnerName: "Bob", IsNew: true}}
	rec := do(docsEngine(t, uc), http.MethodGet, "/documents/d1", "")
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "d1", uc.getID)
	require.Equal(t, "u1", uc.getBy)
	require.Contains(t, rec.Body.String(), `"role":"viewer"`)
	require.Contains(t, rec.Body.String(), `"owner_name":"Bob"`)
	require.Contains(t, rec.Body.String(), `"is_new":true`)
}
