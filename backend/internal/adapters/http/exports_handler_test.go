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

type fakeExportUC struct {
	created app.CreateExportInput
	job     domain.ExportJob
	pdf     []byte
	err     error
}

func (f *fakeExportUC) Create(_ context.Context, in app.CreateExportInput) (domain.ExportJob, error) {
	f.created = in
	return f.job, f.err
}
func (f *fakeExportUC) Get(context.Context, string, string, string) (domain.ExportJob, error) {
	return f.job, f.err
}
func (f *fakeExportUC) List(context.Context, string, string) ([]domain.ExportJob, error) {
	return []domain.ExportJob{f.job}, f.err
}
func (f *fakeExportUC) Open(context.Context, string, string, string) ([]byte, error) {
	return f.pdf, f.err
}
func (f *fakeExportUC) Settings(context.Context, string, string) (domain.ReportSettings, error) {
	return domain.DefaultReportSettings(), f.err
}

func exportsEngine(t *testing.T, uc *fakeExportUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterExports(e.Group("/", withPrincipal(adminP)), uc)
	return e
}

func TestCreateExportParsesFiltersAndSettings(t *testing.T) {
	exp := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	uc := &fakeExportUC{job: domain.ExportJob{ID: "j1", Status: domain.ExportQueued, ExpiresAt: exp}}
	rec := do(exportsEngine(t, uc), http.MethodPost, "/documents/d1/exports",
		`{"query":"impact=high&impact=critical&status=done&from=2026-01-01&to=2026-12-31&page=3","goals_this_year":"ship","section_map":{"x":"Projects"}}`)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	require.Equal(t, "d1", uc.created.DocumentID)
	require.Equal(t, "u1", uc.created.UserID)
	require.Equal(t, "Acme", uc.created.TenantName)
	require.Equal(t, []string{"high", "critical"}, uc.created.Filter.Impacts)
	require.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), *uc.created.Filter.To, "to is inclusive in the URL")
	require.Equal(t, "ship", uc.created.Settings.GoalsThisYear)
	var body ExportJobResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "j1", body.ID)

	rec = do(exportsEngine(t, uc), http.MethodPost, "/documents/d1/exports", `{"query":"from=nope"}`)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	rec = do(exportsEngine(t, uc), http.MethodPost, "/documents/d1/exports", `{"query":"%zz"}`)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = do(exportsEngine(t, uc), http.MethodPost, "/documents/d1/exports", `{"query":"?impact=low"}`)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, []string{"low"}, uc.created.Filter.Impacts, "leading ? from location.search")
}

func TestExportFileServesPDF(t *testing.T) {
	uc := &fakeExportUC{pdf: []byte("%PDF-1.7")}
	rec := do(exportsEngine(t, uc), http.MethodGet, "/documents/d1/exports/j1/file", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "%PDF-1.7", rec.Body.String())

	uc.err = domain.ErrNotFound
	rec = do(exportsEngine(t, uc), http.MethodGet, "/documents/d1/exports/j1/file", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExportStatusAndHistoryAndSettings(t *testing.T) {
	uc := &fakeExportUC{job: domain.ExportJob{ID: "j1", Status: domain.ExportRunning, Progress: 40}}
	e := exportsEngine(t, uc)
	rec := do(e, http.MethodGet, "/documents/d1/exports/j1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"progress":40`)
	rec = do(e, http.MethodGet, "/documents/d1/exports", "")
	require.Contains(t, rec.Body.String(), `"items":[`)
	rec = do(e, http.MethodGet, "/documents/d1/report-settings", "")
	require.Contains(t, rec.Body.String(), `"project":"Projects"`)
}
