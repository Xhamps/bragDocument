package domain

import (
	"slices"
	"strings"
	"time"
)

// LogSorts are the keys the list sorts by (PRD-0002 FR-7).
var LogSorts = []string{"created_at", "name", "impact", "status"}

const (
	defaultLogsPerPage = 50
	maxLogsPerPage     = 100
	maxLogsPage        = 100000 // keeps the offset well inside int32
)

// LogFilter is one list query (PRD-0002 FR-5). Zero values mean "no filter".
type LogFilter struct {
	Query        string     `json:"query"` // substring of name or description
	Tags         []string   `json:"tags"`  // any-of
	Statuses     []string   `json:"statuses"`
	Impacts      []string   `json:"impacts"`
	Domain       string     `json:"domain"`        // link host or a subdomain of it
	From         *time.Time `json:"from"`          // created_at >= From
	To           *time.Time `json:"to"`            // created_at < To
	HideExamples bool       `json:"hide_examples"` // examples=false: matches the dashboard's numbers (PRD-0005 FR-6)
	Sort         string     `json:"sort"`
	Desc         bool       `json:"desc"`
	Page         int        `json:"page"` // 1-based
	PerPage      int        `json:"per_page"`
}

// LogPage is one page of matches and the count of all matches.
type LogPage struct {
	Items []Log
	Total int
}

// Validate applies defaults, normalizes values, and rejects unknown enums.
// It mutates the receiver.
func (f *LogFilter) Validate() error {
	fields := map[string]string{}
	f.Query = strings.TrimSpace(f.Query)
	f.Domain = NormalizeHost(f.Domain)
	if strings.ContainsAny(f.Domain, `%_\`) {
		fields["domain"] = "must be a host name, e.g. github.com"
	}
	// Not normalizeTags: its limits would drop tags, widening the filter.
	// An over-long tag is kept and simply matches nothing.
	tags := []string{}
	for _, t := range f.Tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}
	f.Tags = tags
	for _, s := range f.Statuses {
		if !slices.Contains(Statuses, s) {
			fields["status"] = "must be one of " + strings.Join(Statuses, ", ")
		}
	}
	for _, i := range f.Impacts {
		if !slices.Contains(Impacts, i) {
			fields["impact"] = "must be one of " + strings.Join(Impacts, ", ")
		}
	}
	if f.Sort == "" {
		f.Sort, f.Desc = "created_at", true
	}
	if !slices.Contains(LogSorts, f.Sort) {
		fields["sort"] = "must be one of " + strings.Join(LogSorts, ", ")
	}
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		fields["to"] = "must be after from"
	}
	f.Page = min(max(f.Page, 1), maxLogsPage)
	if f.PerPage < 1 {
		f.PerPage = defaultLogsPerPage
	}
	f.PerPage = min(f.PerPage, maxLogsPerPage)
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}
