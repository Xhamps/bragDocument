package app

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type exportsFixture struct {
	uc    *Exports
	docs  *fakeDocs
	logs  *fakeLogs
	jobs  *fakeExports
	pdf   *fakeRenderer
	files *fakeFiles
	cache *fakeCache
	scope *string // the tenant the worker scoped its context to
}

func newExportsFixture(t *testing.T) exportsFixture {
	t.Helper()
	fd := newFakeDocs()
	fd.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: "u1", OwnerName: "Ada", Title: "2026", State: domain.DocumentActive}
	fd.grant("d1", "u2", domain.RoleViewer)
	fd.grant("d1", "u3", domain.RoleEditor)
	f := exportsFixture{docs: fd, logs: newFakeLogs(), jobs: newFakeExports(), pdf: &fakeRenderer{}, files: newFakeFiles(),
		cache: newFakeCache(), scope: new(string)}
	f.uc = NewExports(fd, f.logs, f.jobs, f.pdf, f.files, f.cache, func(ctx context.Context, tenantID string) context.Context {
		*f.scope = tenantID
		return ctx
	})
	return f
}

func (f exportsFixture) addLog(t *testing.T, name string, tags ...string) {
	t.Helper()
	_, err := f.logs.Create(context.Background(), domain.Log{DocumentID: "d1", Name: name, Impact: "high", Status: domain.StatusDone, Tags: tags})
	require.NoError(t, err)
}

func TestExportCreateQueuesAndSavesSettings(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a", "project")
	in := CreateExportInput{DocumentID: "d1", UserID: "u3", TenantName: "Acme",
		Settings: domain.ReportSettings{GoalsThisYear: "ship", SectionMap: map[string]string{"Project": "Projects"}}}
	j, err := f.uc.Create(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.ExportQueued, j.Status)
	require.True(t, j.Params.Filter.HideExamples, "examples never in the report")
	require.Equal(t, "Acme", j.Params.TenantName)
	s, err := f.uc.Settings(context.Background(), "d1", "u1")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"project": "Projects"}, s.SectionMap, "saved normalized")
}

func TestExportCreateLimits(t *testing.T) {
	f := newExportsFixture(t)
	var ve *domain.ValidationError
	_, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.ErrorAs(t, err, &ve, "nothing to export")
	require.Contains(t, ve.Fields, "filters")

	for i := range domain.MaxReportLogs + 1 {
		f.addLog(t, "l"+strconv.Itoa(i))
	}
	_, err = f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.ErrorAs(t, err, &ve, "FR-7")
	require.Contains(t, ve.Fields["filters"], "2001")
}

func TestExportSettingsDefault(t *testing.T) {
	f := newExportsFixture(t)
	s, err := f.uc.Settings(context.Background(), "d1", "u1")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultSectionMap, s.SectionMap)
}

func TestExportVisibleOnlyToRequester(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	j, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u2"})
	require.NoError(t, err)
	_, err = f.uc.Get(context.Background(), "d1", j.ID, "u1")
	require.ErrorIs(t, err, domain.ErrNotFound, "the owner does not see a viewer's job")
	list, err := f.uc.List(context.Background(), "d1", "u2")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestExportRunNextRendersAndStores(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a", "project")
	f.addLog(t, "b", "learning")
	j, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u2", TenantName: "Acme",
		Settings: domain.DefaultReportSettings()})
	require.NoError(t, err)

	ran, err := f.uc.RunNext(context.Background())
	require.NoError(t, err)
	require.True(t, ran)
	require.Equal(t, "2026", f.pdf.got.Title)
	require.Equal(t, "Ada", f.pdf.got.Author)
	require.Equal(t, "Acme", f.pdf.got.Tenant)
	require.Equal(t, "t1", *f.scope, "worker runs inside the job's tenant")
	require.Equal(t, "created_at", f.logs.filter.Sort)
	require.False(t, f.logs.filter.Desc)
	require.True(t, f.logs.filter.HideExamples)
	require.Equal(t, 2, f.pdf.got.Total)

	done, err := f.uc.Get(context.Background(), "d1", j.ID, "u2")
	require.NoError(t, err)
	require.Equal(t, domain.ExportDone, done.Status)
	require.Equal(t, 100, done.Progress)
	require.Equal(t, "t1/"+j.ID+".pdf", done.FileKey, "per tenant (NFR-2)")
	pdf, err := f.uc.Open(context.Background(), "d1", j.ID, "u2")
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(pdf))

	ran, err = f.uc.RunNext(context.Background())
	require.NoError(t, err)
	require.False(t, ran, "queue empty")
}

func TestExportRunNextPagesPastOneHundred(t *testing.T) {
	f := newExportsFixture(t)
	for i := range 250 {
		f.addLog(t, "l"+strconv.Itoa(i))
	}
	_, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.NoError(t, err)
	_, err = f.uc.RunNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, 250, f.pdf.got.Total)
}

func TestExportRunNextFailures(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	j, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u2"})
	require.NoError(t, err)
	f.pdf.err = domain.ErrUnavailable
	_, err = f.uc.RunNext(context.Background())
	require.NoError(t, err, "a failed job is not a worker error")
	require.Equal(t, domain.ExportFailed, f.jobs.jobs[j.ID].Status)
	require.Contains(t, f.jobs.jobs[j.ID].Error, "PDF service")

	// Access revoked between request and run (FR-6).
	f.pdf.err = nil
	j2, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u2"})
	require.NoError(t, err)
	delete(f.docs.grants["d1"], "u2")
	_, err = f.uc.RunNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, domain.ExportFailed, f.jobs.jobs[j2.ID].Status)
	require.Contains(t, f.jobs.jobs[j2.ID].Error, "access")
	require.Empty(t, f.files.files)
}

func TestExportCleanupAndExpiredDownload(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	j, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.NoError(t, err)
	_, err = f.uc.RunNext(context.Background())
	require.NoError(t, err)

	f.uc.now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	_, err = f.uc.Open(context.Background(), "d1", j.ID, "u1")
	require.ErrorIs(t, err, domain.ErrNotFound, "past 24 h")

	job := f.jobs.jobs[j.ID]
	job.ExpiresAt = time.Now().Add(-time.Second)
	f.jobs.jobs[j.ID] = job
	require.NoError(t, f.uc.Cleanup(context.Background()))
	require.Empty(t, f.files.files)
	require.Empty(t, f.jobs.jobs)
}

func TestExportRunNextSuperseded(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	_, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.NoError(t, err)
	f.jobs.finishErr = domain.ErrNotFound // reclaimed by another worker
	ran, err := f.uc.RunNext(context.Background())
	require.NoError(t, err, "a superseded job is not a worker error")
	require.True(t, ran)
}

func TestExportCreateSettingsOnlyForWriters(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	ctx := context.Background()
	mine := domain.ReportSettings{GoalsThisYear: "owner goals", SectionMap: map[string]string{"project": "Projects"}}
	_, err := f.uc.Create(ctx, CreateExportInput{DocumentID: "d1", UserID: "u1", Settings: mine})
	require.NoError(t, err)

	_, err = f.uc.Create(ctx, CreateExportInput{DocumentID: "d1", UserID: "u2",
		Settings: domain.ReportSettings{GoalsThisYear: "viewer goals"}})
	require.NoError(t, err, "viewers can export (PermRead)")
	s, err := f.uc.Settings(ctx, "d1", "u1")
	require.NoError(t, err)
	require.Equal(t, "owner goals", s.GoalsThisYear, "a viewer does not overwrite shared settings")

	_, err = f.uc.Create(ctx, CreateExportInput{DocumentID: "d1", UserID: "u3",
		Settings: domain.ReportSettings{GoalsThisYear: "editor goals"}})
	require.NoError(t, err)
	s, err = f.uc.Settings(ctx, "d1", "u1")
	require.NoError(t, err)
	require.Equal(t, "editor goals", s.GoalsThisYear)
}

func TestExportRunNextTooManyLogsNow(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	j, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.NoError(t, err)
	for i := range domain.MaxReportLogs {
		f.addLog(t, "l"+strconv.Itoa(i))
	}
	_, err = f.uc.RunNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, domain.ExportFailed, f.jobs.jobs[j.ID].Status)
	require.Contains(t, f.jobs.jobs[j.ID].Error, "More than 2,000")
	require.Empty(t, f.files.files)
}

func TestExportRunNextShutdownLeavesJobRunning(t *testing.T) {
	f := newExportsFixture(t)
	f.addLog(t, "a")
	j, err := f.uc.Create(context.Background(), CreateExportInput{DocumentID: "d1", UserID: "u1"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f.pdf.err = context.Canceled
	ran, err := f.uc.RunNext(ctx)
	require.True(t, ran)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, domain.ExportRunning, f.jobs.jobs[j.ID].Status, "reclaimed later, not failed")
}

func TestExportCleanupContinuesPastFileDeleteError(t *testing.T) {
	f := newExportsFixture(t)
	past := time.Now().Add(-time.Second)
	for _, id := range []string{"j1", "j2"} {
		f.jobs.jobs[id] = domain.ExportJob{ID: id, Status: domain.ExportDone, FileKey: "t1/" + id + ".pdf", ExpiresAt: past}
		f.jobs.order = append(f.jobs.order, id)
		f.files.files["t1/"+id+".pdf"] = []byte("x")
	}
	f.files.deleteErr = map[string]error{"t1/j1.pdf": errors.New("disk")}

	require.NoError(t, f.uc.Cleanup(context.Background()))
	require.Contains(t, f.jobs.jobs, "j1", "row kept so the next tick retries")
	require.NotContains(t, f.jobs.jobs, "j2", "a failing job does not block the next")
	require.NotContains(t, f.files.files, "t1/j2.pdf")
}
