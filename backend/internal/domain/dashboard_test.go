package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestNewPeriodDefaultsAndLimits(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 4, 0, 0, time.UTC)

	p, err := NewPeriod(nil, nil, now)
	require.NoError(t, err)
	require.Equal(t, Period{From: day(2025, 10, 10), To: day(2026, 10, 10)}, p, "last 12 months, today included")

	to := day(2026, 4, 1)
	p, err = NewPeriod(nil, &to, now)
	require.NoError(t, err)
	require.Equal(t, day(2025, 4, 1), p.From, "a lone to looks back 12 months")

	from := day(2026, 4, 1)
	_, err = NewPeriod(&from, &to, now)
	var ve *ValidationError
	require.ErrorAs(t, err, &ve, "empty period")

	from = day(2020, 1, 1)
	_, err = NewPeriod(&from, &to, now)
	require.ErrorAs(t, err, &ve, "more than 5 years")

	from = day(2021, 4, 1)
	_, err = NewPeriod(&from, &to, now)
	require.NoError(t, err, "exactly 5 years is fine")
}

func TestNormalizeFillsSeries(t *testing.T) {
	d := Dashboard{
		From: day(2025, 11, 15), To: day(2026, 2, 1),
		Months:   []Bucket{{"2026-01", 3}},
		Statuses: []Bucket{{"done", 2}},
		Impacts:  []Bucket{{"high", 1}},
		Tags: []Bucket{
			{"zeta", 1}, {"project", 5}, {"alpha", 1}, {"mentorship", 2},
			{"t1", 1}, {"t2", 1}, {"t3", 1}, {"t4", 1}, {"t5", 1}, {"t6", 1}, {"t7", 1},
		},
	}
	d.Normalize()

	require.Equal(t, []Bucket{{"2025-11", 0}, {"2025-12", 0}, {"2026-01", 3}}, d.Months, "every month that overlaps, across the year")
	require.Equal(t, []Bucket{{"idea", 0}, {"in_progress", 0}, {"done", 2}, {"dropped", 0}}, d.Statuses)
	require.Equal(t, []Bucket{{"low", 0}, {"medium", 0}, {"high", 1}, {"critical", 0}}, d.Impacts)
	require.Len(t, d.Tags, 10)
	require.Equal(t, Bucket{"project", 5}, d.Tags[0])
	require.Equal(t, Bucket{"mentorship", 2}, d.Tags[1])
	require.Equal(t, Bucket{"alpha", 1}, d.Tags[2], "ties by name")
	require.Len(t, d.Coverage, len(SuggestedTags))
	require.Equal(t, Bucket{"project", 5}, d.Coverage[0])
	require.Equal(t, Bucket{"collaboration", 0}, d.Coverage[1])
	require.Equal(t, Bucket{"mentorship", 2}, d.Coverage[2])
}

func TestNormalizeEmpty(t *testing.T) {
	d := Dashboard{From: day(2026, 3, 1), To: day(2026, 4, 1)}
	d.Normalize()
	require.Equal(t, []Bucket{{"2026-03", 0}}, d.Months)
	require.NotNil(t, d.Tags)
	require.Empty(t, d.Tags)
}

func TestNormalizeCoverageCountsCutTags(t *testing.T) {
	d := Dashboard{From: day(2026, 3, 1), To: day(2026, 4, 1), Tags: []Bucket{{"project", 1}}}
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		d.Tags = append(d.Tags, Bucket{k, 2})
	}
	d.Normalize()
	require.Len(t, d.Tags, 10)
	require.NotContains(t, d.Tags, Bucket{"project", 1}, "ranked 11th, cut from the top tags")
	require.Equal(t, Bucket{"project", 1}, d.Coverage[0], "still counted in coverage")
}
