package domain

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func names(s ReportSection) []string {
	out := []string{}
	for _, l := range s.Logs {
		out = append(out, l.Name)
	}
	return out
}

func TestNewReportFirstMatchAndOther(t *testing.T) {
	logs := []Log{
		{Name: "a", Impact: "high", Status: "done", Tags: []string{"mentorship", "project"}},
		{Name: "b", Impact: "low", Status: "idea", Tags: []string{"misc"}},
		{Name: "c", Impact: "critical", Status: "done", Tags: []string{}},
		{Name: "d", Impact: "high", Status: "in_progress", Tags: []string{"learning"}},
	}
	r := NewReport(logs, ReportSettings{GoalsThisYear: "ship", SectionMap: DefaultSectionMap})
	got := []string{}
	for _, s := range r.Sections {
		got = append(got, s.Name)
	}
	require.Equal(t, []string{"Projects", "What you learned", "Other"}, got, "template order, empty sections omitted")
	require.Equal(t, []string{"a"}, names(r.Sections[0]), "first section in template order, not first tag")
	require.Equal(t, []string{"b", "c"}, names(r.Sections[2]), "unmapped and untagged go to Other")
	require.Equal(t, "ship", r.GoalsThisYear)
	require.Equal(t, 4, r.Total)
	require.Len(t, r.Summary, len(Impacts))
	require.Equal(t, SummaryRow{Impact: "high", Counts: []int{0, 1, 1, 0}, Total: 2}, r.Summary[2])
	require.Equal(t, []int{1, 1, 2, 0}, r.StatusTotals)
}

func TestNewReportCustomMapReplacesDefault(t *testing.T) {
	logs := []Log{
		{Name: "a", Impact: "low", Status: "done", Tags: []string{"project"}},
		{Name: "b", Impact: "low", Status: "done", Tags: []string{"misc"}},
	}
	r := NewReport(logs, ReportSettings{SectionMap: map[string]string{"misc": "Company building"}})
	require.Equal(t, "Company building", r.Sections[0].Name)
	require.Equal(t, []string{"b"}, names(r.Sections[0]))
	require.Equal(t, "Other", r.Sections[1].Name)
	require.Equal(t, []string{"a"}, names(r.Sections[1]))

	r = NewReport(logs[:1], ReportSettings{SectionMap: map[string]string{"project": "Nope"}})
	require.Equal(t, "Other", r.Sections[0].Name, "unknown section falls back to Other")
}

func TestReportSettingsValidate(t *testing.T) {
	s := ReportSettings{SectionMap: map[string]string{" Project ": "Projects", "misc": "Other", "": "Projects"}}
	require.NoError(t, s.Validate())
	require.Equal(t, map[string]string{"project": "Projects"}, s.SectionMap, "normalized; Other and empty tags dropped")

	bad := ReportSettings{SectionMap: map[string]string{"x": "Nope"}, GoalsNextYear: string(make([]byte, maxGoalsLen+1))}
	var ve *ValidationError
	require.ErrorAs(t, bad.Validate(), &ve)
	require.Contains(t, ve.Fields, "section_map")
	require.Contains(t, ve.Fields, "goals_next_year")

	runes := ReportSettings{GoalsThisYear: strings.Repeat("é", maxGoalsLen)}
	require.NoError(t, runes.Validate(), "limit counts runes, not bytes")

	empty := ReportSettings{}
	require.NoError(t, empty.Validate())
	require.NotNil(t, empty.SectionMap, "never nil: stored as jsonb")
}

func TestExportJobDownloadable(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	j := ExportJob{Status: ExportDone, ExpiresAt: now.Add(time.Hour)}
	require.True(t, j.Downloadable(now))
	require.False(t, j.Downloadable(now.Add(2*time.Hour)), "expired")
	j.Status = ExportRunning
	require.False(t, j.Downloadable(now))
}

func TestDefaultSectionMapCoversSuggestedTags(t *testing.T) {
	require.ElementsMatch(t, SuggestedTags, slices.Collect(maps.Keys(DefaultSectionMap)))
}
