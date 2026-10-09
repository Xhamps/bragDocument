package domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// SectionOther collects logs no mapped tag places elsewhere (PRD-0006 FR-4).
const SectionOther = "Other"

// ReportSections are the article's template sections in report order (PRD-0006 FR-2).
var ReportSections = []string{"Projects", "Collaboration & mentorship", "Design & documentation",
	"Company building", "What you learned", "Outside of work", SectionOther}

// DefaultSectionMap maps SuggestedTags to sections; any other tag goes to Other.
var DefaultSectionMap = map[string]string{
	"project":          "Projects",
	"collaboration":    "Collaboration & mentorship",
	"mentorship":       "Collaboration & mentorship",
	"design":           "Design & documentation",
	"documentation":    "Design & documentation",
	"company-building": "Company building",
	"learning":         "What you learned",
	"outside-of-work":  "Outside of work",
}

// Export job statuses (PRD-0006 §10).
const (
	ExportQueued  = "queued"
	ExportRunning = "running"
	ExportDone    = "done"
	ExportFailed  = "failed"
)

const (
	MaxReportLogs = 2000 // PRD-0006 FR-7
	maxGoalsLen   = 5000
	maxSectionMap = 500
)

// ReportSettings are the per-document export inputs, saved so they need no re-typing.
// SectionMap is tag → section; a tag not in it goes to Other. It replaces
// DefaultSectionMap entirely once saved.
type ReportSettings struct {
	GoalsThisYear string            `json:"goals_this_year"`
	GoalsNextYear string            `json:"goals_next_year"`
	SectionMap    map[string]string `json:"section_map"`
}

// Validate lowercases tags, drops entries mapped to Other (the fallback anyway),
// and rejects unknown sections and over-long goals. It mutates the receiver.
func (s *ReportSettings) Validate() error {
	fields := map[string]string{}
	for k, v := range map[string]string{"goals_this_year": s.GoalsThisYear, "goals_next_year": s.GoalsNextYear} {
		if utf8.RuneCountInString(v) > maxGoalsLen {
			fields[k] = fmt.Sprintf("must be at most %d characters", maxGoalsLen)
		}
	}
	if len(s.SectionMap) > maxSectionMap {
		fields["section_map"] = fmt.Sprintf("must have at most %d tags", maxSectionMap)
	}
	m := make(map[string]string, len(s.SectionMap))
	for tag, sec := range s.SectionMap {
		if !slices.Contains(ReportSections, sec) {
			fields["section_map"] = fmt.Sprintf("unknown section %q", sec)
			continue
		}
		if tag = strings.ToLower(strings.TrimSpace(tag)); tag != "" && sec != SectionOther {
			m[tag] = sec
		}
	}
	s.SectionMap = m
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}

// DefaultReportSettings is what the dialog shows before the first export.
func DefaultReportSettings() ReportSettings {
	return ReportSettings{SectionMap: maps.Clone(DefaultSectionMap)}
}

// ExportParams is everything a job renders, captured when it was requested.
type ExportParams struct {
	Filter     LogFilter
	Settings   ReportSettings
	TenantName string
}

// ExportJob is one report generation (PRD-0006 FR-5).
type ExportJob struct {
	ID          string
	TenantID    string
	DocumentID  string
	RequestedBy string
	Params      ExportParams
	Status      string
	Error       string // user-facing reason when failed
	FileKey     string
	Progress    int // 0–100; from the cache, not stored
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// Downloadable reports whether the file may be served at now.
func (j ExportJob) Downloadable(now time.Time) bool {
	return j.Status == ExportDone && now.Before(j.ExpiresAt)
}

// ReportSection is one heading of the report and its logs in input order (the worker passes them oldest first).
type ReportSection struct {
	Name string
	Logs []Log
}

// SummaryRow counts one impact level; Counts follow Statuses.
type SummaryRow struct {
	Impact string
	Counts []int
	Total  int
}

// Report is what the template renders (PRD-0006 FR-2, FR-3).
type Report struct {
	Title, Author, Tenant        string
	From, To                     *time.Time // To is exclusive; nil means open-ended
	GeneratedAt                  time.Time
	GoalsThisYear, GoalsNextYear string
	Summary                      []SummaryRow // one per Impacts, zeros included
	StatusTotals                 []int        // follow Statuses
	Total                        int
	Sections                     []ReportSection // template order, empty ones omitted
}

// NewReport places each log in its first matching section in template order
// (so no log appears twice) and counts impact × status. The caller sets the cover fields.
func NewReport(logs []Log, s ReportSettings) Report {
	r := Report{GoalsThisYear: s.GoalsThisYear, GoalsNextYear: s.GoalsNextYear, Total: len(logs),
		StatusTotals: make([]int, len(Statuses))}
	rank := map[string]int{}
	for i, name := range ReportSections {
		rank[name] = i
	}
	buckets := make([][]Log, len(ReportSections))
	counts := map[string][]int{}
	for _, imp := range Impacts {
		counts[imp] = make([]int, len(Statuses))
	}
	for _, l := range logs {
		best := len(ReportSections) - 1 // Other
		for _, t := range l.Tags {
			if i, ok := rank[s.SectionMap[t]]; ok && i < best {
				best = i
			}
		}
		buckets[best] = append(buckets[best], l)
		if si := slices.Index(Statuses, l.Status); si >= 0 && counts[l.Impact] != nil {
			counts[l.Impact][si]++
			r.StatusTotals[si]++
		}
	}
	for i, b := range buckets {
		if len(b) > 0 {
			r.Sections = append(r.Sections, ReportSection{Name: ReportSections[i], Logs: b})
		}
	}
	for _, imp := range Impacts {
		row := SummaryRow{Impact: imp, Counts: counts[imp]}
		for _, n := range row.Counts {
			row.Total += n
		}
		r.Summary = append(r.Summary, row)
	}
	return r
}
