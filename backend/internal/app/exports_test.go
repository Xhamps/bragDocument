package app

import (
	"context"
	"strconv"
	"testing"

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
}

func newExportsFixture(t *testing.T) exportsFixture {
	t.Helper()
	fd := newFakeDocs()
	fd.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: "u1", OwnerName: "Ada", Title: "2026", State: domain.DocumentActive}
	fd.grant("d1", "u2", domain.RoleViewer)
	f := exportsFixture{docs: fd, logs: newFakeLogs(), jobs: newFakeExports(), pdf: &fakeRenderer{}, files: newFakeFiles(), cache: newFakeCache()}
	f.uc = NewExports(fd, f.logs, f.jobs, f.pdf, f.files, f.cache, func(ctx context.Context, _ string) context.Context { return ctx })
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
	in := CreateExportInput{DocumentID: "d1", UserID: "u2", TenantName: "Acme",
		Settings: domain.ReportSettings{GoalsThisYear: "ship", SectionMap: map[string]string{"Project": "Projects"}}}
	j, err := f.uc.Create(context.Background(), in)
	require.NoError(t, err, "viewers can export (PermRead)")
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
