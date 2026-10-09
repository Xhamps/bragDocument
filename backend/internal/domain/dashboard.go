package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// SuggestedTags are the article's sections (PRD-0002 FR-10). The dashboard's
// coverage panel counts them (PRD-0005 FR-4). The frontend keeps a copy in
// logs/constants.ts.
var SuggestedTags = []string{"project", "collaboration", "mentorship", "design",
	"documentation", "company-building", "learning", "outside-of-work"}

const (
	maxDashboardTags  = 10 // PRD-0005 FR-1
	maxDashboardYears = 5  // bounds the month series
)

// Period is [From, To) in UTC.
type Period struct{ From, To time.Time }

// NewPeriod applies defaults and limits. No dates means the last 12 months
// including today; a lone "to" looks back 12 months. Dates come from the
// HTTP layer already in UTC with "to" exclusive.
func NewPeriod(from, to *time.Time, now time.Time) (Period, error) {
	end := now.UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	p := Period{To: end}
	if to != nil {
		p.To = *to
	}
	p.From = p.To.AddDate(-1, 0, 0)
	if from != nil {
		p.From = *from
	}
	switch {
	case !p.From.Before(p.To):
		return Period{}, NewValidationError(map[string]string{"to": "must be after from"})
	case p.From.AddDate(maxDashboardYears, 0, 0).Before(p.To):
		return Period{}, NewValidationError(map[string]string{"from": "period is at most 5 years"})
	}
	return p, nil
}

// Bucket is one bar: a month ("2026-03"), tag, status, or impact and its count.
type Bucket struct {
	Key   string
	Count int
}

// Dashboard aggregates one document's logs, examples excluded (PRD-0005).
// Total is all-time; everything else is within [From, To).
type Dashboard struct {
	From, To                                  time.Time
	Total, InPeriod, HighImpact, InProgress   int
	Months, Tags, Statuses, Impacts, Coverage []Bucket
}

// Normalize turns the repository's sparse rows into the series the charts
// draw: every month of the period and every status and impact (zeros
// included), the top 10 tags, and the suggested tags' counts.
func (d *Dashboard) Normalize() {
	d.Months = fill(monthKeys(d.From, d.To), d.Months)
	d.Statuses = fill(Statuses, d.Statuses)
	d.Impacts = fill(Impacts, d.Impacts)
	d.Coverage = fill(SuggestedTags, d.Tags)
	tags := append([]Bucket{}, d.Tags...)
	slices.SortFunc(tags, func(a, b Bucket) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Key, b.Key))
	})
	d.Tags = tags[:min(len(tags), maxDashboardTags)]
}

// monthKeys lists "YYYY-MM" for every month overlapping [from, to).
func monthKeys(from, to time.Time) []string {
	keys := []string{}
	for m := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC); m.Before(to); m = m.AddDate(0, 1, 0) {
		keys = append(keys, m.Format("2006-01"))
	}
	return keys
}

// fill returns one bucket per key, in key order, with counts from got.
func fill(keys []string, got []Bucket) []Bucket {
	out := make([]Bucket, len(keys))
	for i, k := range keys {
		out[i].Key = k
		if j := slices.IndexFunc(got, func(b Bucket) bool { return b.Key == k }); j >= 0 {
			out[i].Count = got[j].Count
		}
	}
	return out
}
