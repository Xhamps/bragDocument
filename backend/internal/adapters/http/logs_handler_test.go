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

type fakeLogUC struct {
	filter  domain.LogFilter
	listDoc string
	created app.CreateLogInput
	updated app.UpdateLogInput
	deleted []string // docID, id, userID
	got     []string // docID, id, userID
	exDoc   string
	log     domain.Log
	err     error

	dashFrom, dashTo *time.Time
}

func (f *fakeLogUC) List(_ context.Context, docID, _ string, fl domain.LogFilter) (domain.LogPage, error) {
	f.listDoc, f.filter = docID, fl
	return domain.LogPage{Items: []domain.Log{f.log}, Total: 7}, f.err
}
func (f *fakeLogUC) Create(_ context.Context, in app.CreateLogInput) (domain.Log, error) {
	f.created = in
	return f.log, f.err
}
func (f *fakeLogUC) Update(_ context.Context, in app.UpdateLogInput) (domain.Log, error) {
	f.updated = in
	return f.log, f.err
}
func (f *fakeLogUC) Get(_ context.Context, docID, id, userID string) (domain.Log, error) {
	f.got = []string{docID, id, userID}
	return f.log, f.err
}
func (f *fakeLogUC) Delete(_ context.Context, docID, id, userID string) error {
	f.deleted = []string{docID, id, userID}
	return f.err
}
func (f *fakeLogUC) DeleteExamples(_ context.Context, docID, _ string) error {
	f.exDoc = docID
	return f.err
}
func (f *fakeLogUC) Tags(context.Context) ([]string, error) { return []string{"project"}, f.err }
func (f *fakeLogUC) Dashboard(_ context.Context, _, _ string, from, to *time.Time) (domain.Dashboard, error) {
	f.dashFrom, f.dashTo = from, to
	return domain.Dashboard{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Total: 7, InPeriod: 3, Months: []domain.Bucket{{Key: "2026-01", Count: 3}}}, f.err
}

func logsEngine(t *testing.T, uc *fakeLogUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterLogs(e.Group("/", withPrincipal(adminP)), uc)
	return e
}

func TestLogsListParsesQuery(t *testing.T) {
	uc := &fakeLogUC{log: domain.Log{ID: "l1", Name: "X", Tags: []string{}, Links: []domain.Link{}}}
	rec := do(logsEngine(t, uc), http.MethodGet,
		"/documents/d1/logs?q=mig&tag=a&tag=b&status=done&impact=high&impact=low&from=2026-01-01&to=2026-01-31&domain=github.com&sort=-impact&page=2&per_page=10", "")
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Equal(t, "d1", uc.listDoc)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC) // "to" is inclusive: next midnight, exclusive
	require.Equal(t, domain.LogFilter{Query: "mig", Tags: []string{"a", "b"}, Statuses: []string{"done"},
		Impacts: []string{"high", "low"}, Domain: "github.com", From: &from, To: &to,
		Sort: "impact", Desc: true, Page: 2, PerPage: 10}, uc.filter)

	var body LogListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 7, body.Total)
	require.Contains(t, rec.Body.String(), `"impact_statement":null`)
	require.Contains(t, rec.Body.String(), `"tags":[]`)
}

func TestLogsListRejectsBadQuery(t *testing.T) {
	for _, qs := range []string{"page=x", "per_page=-", "from=01/02/2026", "to=tomorrow"} {
		uc := &fakeLogUC{}
		rec := do(logsEngine(t, uc), http.MethodGet, "/documents/d1/logs?"+qs, "")
		require.Equal(t, 422, rec.Code, qs)
		require.Empty(t, uc.listDoc, "use case never called")
	}
}

func TestLogsCreate(t *testing.T) {
	st := "Cut p95"
	uc := &fakeLogUC{log: domain.Log{ID: "l1", ImpactStatement: &st}}
	rec := do(logsEngine(t, uc), http.MethodPost, "/documents/d1/logs",
		`{"name":"X","description":"d","impact":"high","status":"idea","tags":["a"],"links":[{"url":"https://a.io","label":"PR"}],"created_at":"2026-01-01T00:00:00Z"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	back := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, app.CreateLogInput{DocumentID: "d1", UserID: "u1", Name: "X", Description: "d", Impact: "high",
		Status: "idea", Tags: []string{"a"}, Links: []domain.Link{{URL: "https://a.io", Label: "PR"}}, CreatedAt: &back}, uc.created)
	require.Contains(t, rec.Body.String(), `"impact_statement":"Cut p95"`)
}

func TestLogsUpdateDistinguishesAbsentFromEmpty(t *testing.T) {
	uc := &fakeLogUC{}
	rec := do(logsEngine(t, uc), http.MethodPatch, "/documents/d1/logs/l1", `{"status":"done","tags":[]}`)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Equal(t, "l1", uc.updated.ID)
	require.Equal(t, "d1", uc.updated.DocumentID)
	require.Equal(t, "done", *uc.updated.Status)
	require.NotNil(t, uc.updated.Tags)
	require.Empty(t, *uc.updated.Tags, "[] clears the tags")
	require.Nil(t, uc.updated.Links, "absent links are unchanged")
	require.Nil(t, uc.updated.Name)
}

func TestLogsGet(t *testing.T) {
	uc := &fakeLogUC{log: domain.Log{ID: "l1", Name: "X"}}
	rec := do(logsEngine(t, uc), http.MethodGet, "/documents/d1/logs/l1", "")
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Equal(t, []string{"d1", "l1", "u1"}, uc.got)
	require.Contains(t, rec.Body.String(), `"name":"X"`)
}

func TestLogsDelete(t *testing.T) {
	uc := &fakeLogUC{}
	e := logsEngine(t, uc)
	rec := do(e, http.MethodDelete, "/documents/d1/logs/l1", "")
	require.Equal(t, 204, rec.Code)
	require.Equal(t, []string{"d1", "l1", "u1"}, uc.deleted)
	rec = do(e, http.MethodDelete, "/documents/d1/example-logs", "")
	require.Equal(t, 204, rec.Code)
	require.Equal(t, "d1", uc.exDoc)
}

func TestLogsArchivedConflict(t *testing.T) {
	rec := do(logsEngine(t, &fakeLogUC{err: domain.ErrConflict}), http.MethodPost, "/documents/d1/logs", `{"name":"X","impact":"low"}`)
	require.Equal(t, 409, rec.Code)
}

func TestTagsList(t *testing.T) {
	rec := do(logsEngine(t, &fakeLogUC{}), http.MethodGet, "/tags", "")
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `{"tags":["project"]}`, rec.Body.String())
}

func TestLogsDashboard(t *testing.T) {
	uc := &fakeLogUC{}
	rec := do(logsEngine(t, uc), http.MethodGet, "/documents/d1/dashboard?from=2026-01-01&to=2026-03-31", "")
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), *uc.dashFrom)
	require.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), *uc.dashTo, "to is inclusive")
	body := rec.Body.String()
	require.Contains(t, body, `"from":"2026-01-01","to":"2026-03-31"`)
	require.Contains(t, body, `"total":7,"in_period":3`)
	require.Contains(t, body, `"months":[{"key":"2026-01","count":3}]`)
	require.Contains(t, body, `"tags":[]`)

	rec = do(logsEngine(t, uc), http.MethodGet, "/documents/d1/dashboard?from=jan", "")
	require.Equal(t, 422, rec.Code)
}

func TestLogsListHidesExamples(t *testing.T) {
	uc := &fakeLogUC{}
	do(logsEngine(t, uc), http.MethodGet, "/documents/d1/logs?examples=false", "")
	require.True(t, uc.filter.HideExamples)
	do(logsEngine(t, uc), http.MethodGet, "/documents/d1/logs", "")
	require.False(t, uc.filter.HideExamples)
}
