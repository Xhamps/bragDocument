# PDF Report Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Any member of a document exports a review-ready PDF (PRD-0006): cover, goals, summary table, logs grouped into the article's sections, clickable links; generated asynchronously and downloadable for 24 h.

**Architecture:** `POST /documents/:id/exports` checks `PermRead`, counts matches (≤ 2,000), saves the per-document report settings, and inserts a `queued` row in `export_jobs`. The `worker` subcommand claims jobs with `FOR UPDATE SKIP LOCKED` under `app.provisioning`, then runs inside the job's tenant: re-check access → page the logs → `domain.NewReport` → `html/template` → Gotenberg (`generateDocumentOutline=true`, which implies a tagged PDF) → AES-GCM encrypt → file under `EXPORT_DIR/<tenant>/<job>.pdf` → `done`, `expires_at = now + 24h`. Progress goes to Redis through `Degrading`. Each worker tick deletes expired jobs and their files. The frontend polls the job and downloads with an authenticated fetch.

**Tech Stack:** Go 1.27, Gin, pgx/sqlc, PostgreSQL RLS, Redis (`ports.Cache`), Gotenberg 8, stdlib `crypto/aes`+`crypto/cipher`, `os.Root`, `html/template`, `embed`; React 19, react-query 5, Vitest.

**Design:** `docs/plans/2026-10-09-pdf-report-design.md`. **Branch:** `feat/pdf-report` (already created from `main`). Read `.claude/skills/backend-endpoint/SKILL.md` before backend tasks.

**Commands** (backend from `backend/`, frontend from `frontend/`):
- Unit: `go test ./...` · Integration (Docker): `go test -tags integration ./internal/adapters/...` · Lint: `make lint` · sqlc: `make sqlc`
- Frontend: `npm test -w @bragdoc/app` · `npm run typecheck -w @bragdoc/app` · `npx eslint packages/app/src` · `npx prettier --check packages/app/src` · `npm run build -w @bragdoc/app`

**Commit trailer:** end every commit message with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Simplifications against the design (deliberate):**
- No `PUT /report-settings`: `POST /exports` saves the settings it was sent. `GET /report-settings` prefills the dialog.
- The job routes nest under the document (`/documents/:id/exports/:jobId`), so `access()` runs exactly like every other document route. A job is visible only to the user who requested it, which gives the dialog history.
- `export_jobs` has no FK to `documents`, so deleting a document never orphans files. Its jobs fail on access and are cleaned up at expiry like any other job.
- A queued job also has `expires_at` (created + 24 h), so cleanup covers stuck and failed jobs too.
- A `running` job older than 5 minutes is claimed again, which covers a worker killed mid-job.

---

### Task 1: Docs: accept PRD-0006, amend ADR-0010

**Files:** `docs/prd/0006-pdf-report.md`, `docs/adr/0010-pdf-generation-with-gotenberg.md`, `docs/README.md`

**Step 1:** In PRD-0006, set `status: accepted`. Mark the §12 question answered by changing its row to `| Does the summary table belong on page 2 or at the end? | Design | answered 2026-10-09: page 2 |`. Then append to §13:

```markdown
| 2026-10-09 | Summary table on page 2, after the cover and goals | The overview comes before the detail, as in promotion packets |
| 2026-10-09 | A log appears once, in its first matching section in template order | Section counts add up to the summary |
| 2026-10-09 | The tag→section mapping and the goals are saved per document | The next export starts where the last one ended |
| 2026-10-09 | Cover shows the tenant name; no logo in v1 | Tenants have no logo yet |
| 2026-10-09 | Example logs are never in the report | They are starter content, not work |
```

**Step 2:** In ADR-0010, set `status: accepted`. Replace the "Flow:" paragraph with:

```markdown
Flow: the API inserts a `queued` row in `export_jobs` (PostgreSQL). The `worker` claims it with `FOR UPDATE SKIP LOCKED` under `app.provisioning`, then works inside the job's tenant: it renders `report.html` from the template with the filtered logs, POSTs it to Gotenberg with `generateDocumentOutline=true` (which implies a tagged PDF), encrypts the result with AES-256-GCM (`EXPORT_KEY`), and writes it under `EXPORT_DIR/<tenant>/<job>.pdf` (local: a Docker volume shared by api and worker; production: the same through a mounted volume until an S3 store is needed). The job is then `done` with a 24 h expiry. Progress (0–100) lives in Redis through the `Degrading` cache; a Redis outage hides the bar and nothing else. Each worker tick deletes expired jobs and their files.
```

Replace the "Bad, because tagged (accessible) PDF…" bullet with:

```markdown
* Neutral, because tagged (accessible) output comes from Chromium's `generateTaggedPdf`, enabled through `generateDocumentOutline`; no post-processing step.
```

**Step 3:** In `docs/README.md`, set PRD 0006 and ADR 0010 to `accepted`.

**Step 4: Commit**

```bash
git add docs
git commit -m "docs: accept PRD-0006 and ADR-0010 with report decisions"
```

---

### Task 2: Domain: report, settings, export job

**Files:**
- Create: `backend/internal/domain/report.go`
- Test: `backend/internal/domain/report_test.go`

**Step 1: Write the failing tests**

```go
package domain

import (
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
```

**Step 2:** Run `go test ./internal/domain/ -run 'Report|ExportJob'`. Expected: FAIL (undefined: NewReport).

**Step 3: Implement**

```go
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
			fields["section_map"] = "unknown section " + fmt.Sprintf("%q", sec)
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

// ReportSection is one heading of the report and its logs, oldest first.
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
			if sec, ok := s.SectionMap[t]; ok && rank[sec] < best {
				best = rank[sec]
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
```

Also add a test that `DefaultSectionMap`'s keys equal `SuggestedTags`:

```go
func TestDefaultSectionMapCoversSuggestedTags(t *testing.T) {
	require.ElementsMatch(t, SuggestedTags, slices.Collect(maps.Keys(DefaultSectionMap)))
}
```

(add `"maps"` and `"slices"` to the test imports).

**Step 4:** Run `go test ./internal/domain/`. Expected: PASS. Then run `make lint`.

**Step 5: Commit**

```bash
git add backend/internal/domain/report.go backend/internal/domain/report_test.go
git commit -m "feat(domain): report sections, settings, and export job"
```

---

### Task 3: Migration and queries

**Files:**
- Create: `backend/migrations/0006_exports.up.sql`, `backend/migrations/0006_exports.down.sql`, `backend/queries/exports.sql`
- Generated: `backend/internal/adapters/postgres/sqlcgen/*`

**Step 1: Up migration**

```sql
-- PRD-0006 export jobs and per-document report settings. RLS per ADR-0007.
-- The worker claims and expires jobs before it knows the tenant, under
-- app.provisioning (ADR-0010); the rows hold no secrets beyond their params.
-- No FK to documents: a deleted document's jobs fail on access and expire
-- with their files, so no file is ever orphaned.
CREATE TABLE export_jobs (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    document_id  uuid NOT NULL,
    requested_by uuid NOT NULL, -- provenance, like granted_by
    params       jsonb NOT NULL,
    status       text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
    error        text NOT NULL DEFAULT '',
    file_key     text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    started_at   timestamptz,
    finished_at  timestamptz,
    expires_at   timestamptz NOT NULL DEFAULT now() + interval '24 hours' -- FR-5, NFR-2
);
CREATE INDEX export_jobs_queue_idx ON export_jobs (created_at) WHERE status IN ('queued', 'running');
CREATE INDEX export_jobs_expires_idx ON export_jobs (expires_at);
CREATE INDEX export_jobs_history_idx ON export_jobs (document_id, requested_by, created_at DESC);
CREATE INDEX export_jobs_tenant_id_idx ON export_jobs (tenant_id);

CREATE TABLE document_report_settings (
    document_id     uuid PRIMARY KEY,
    tenant_id       uuid NOT NULL REFERENCES tenants (id),
    goals_this_year text NOT NULL DEFAULT '',
    goals_next_year text NOT NULL DEFAULT '',
    section_map     jsonb NOT NULL DEFAULT '{}',
    updated_at      timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX document_report_settings_tenant_id_idx ON document_report_settings (tenant_id);

ALTER TABLE export_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON export_jobs
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());

ALTER TABLE document_report_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_report_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_report_settings
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

-- The default privilege from 0002 already covers these; explicit for readers.
GRANT SELECT, INSERT, UPDATE, DELETE ON export_jobs, document_report_settings TO bragdoc_app;
```

**Step 2: Down migration**

```sql
DROP TABLE IF EXISTS document_report_settings;
DROP TABLE IF EXISTS export_jobs;
```

**Step 3: Queries** (`backend/queries/exports.sql`)

```sql
-- name: CreateExportJob :one
INSERT INTO export_jobs (tenant_id, document_id, requested_by, params)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetExportJob :one
SELECT * FROM export_jobs WHERE id = $1 AND document_id = $2;

-- The dialog's history: the caller's live jobs on the document.
-- name: ListExportJobs :many
SELECT * FROM export_jobs
WHERE document_id = $1 AND requested_by = $2 AND expires_at > now()
ORDER BY created_at DESC
LIMIT 10;

-- Under app.provisioning. A running job not finished in 5 minutes had its worker die; take it again.
-- name: ClaimExportJob :one
UPDATE export_jobs SET status = 'running', started_at = now()
WHERE id = (
    SELECT id FROM export_jobs
    WHERE status = 'queued' OR (status = 'running' AND started_at < now() - interval '5 minutes')
    ORDER BY created_at
    LIMIT 1
    FOR UPDATE SKIP LOCKED)
RETURNING *;

-- name: FinishExportJob :exec
UPDATE export_jobs
SET status = 'done', file_key = $2, error = '', finished_at = now(), expires_at = now() + interval '24 hours'
WHERE id = $1;

-- name: FailExportJob :exec
UPDATE export_jobs SET status = 'failed', error = $2, finished_at = now() WHERE id = $1;

-- Under app.provisioning.
-- name: ListExpiredExportJobs :many
SELECT * FROM export_jobs WHERE expires_at <= now() ORDER BY expires_at LIMIT 100;

-- Under app.provisioning.
-- name: DeleteExportJob :exec
DELETE FROM export_jobs WHERE id = $1;

-- name: GetReportSettings :one
SELECT * FROM document_report_settings WHERE document_id = $1;

-- name: UpsertReportSettings :exec
INSERT INTO document_report_settings (document_id, tenant_id, goals_this_year, goals_next_year, section_map)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (document_id) DO UPDATE SET
    goals_this_year = EXCLUDED.goals_this_year,
    goals_next_year = EXCLUDED.goals_next_year,
    section_map     = EXCLUDED.section_map,
    updated_at      = now();
```

**Step 4:** Run `make sqlc` and then `go build ./...`. Expected: the generated `exports.sql.go`, and models `ExportJob` and `DocumentReportSetting`, with `Params`/`SectionMap` as `[]byte`.

**Step 5:** In `backend/internal/adapters/postgres/db.go`, add `ExportRepo.Claim, Expired, and Delete` to the list of `WithProvisioning` callers in its doc comment. Add the sentence: "export_jobs opens fully under the flag: the worker claims across tenants."

**Step 6: Commit**

```bash
git add backend/migrations backend/queries/exports.sql backend/internal/adapters/postgres/sqlcgen backend/internal/adapters/postgres/db.go
git commit -m "feat(postgres): export_jobs and document_report_settings with RLS"
```

---

### Task 4: Ports

**Files:** Create `backend/internal/ports/exports.go`

```go
package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ExportRepo stores export jobs and report settings (PRD-0006).
type ExportRepo interface {
	Create(ctx context.Context, j domain.ExportJob) (domain.ExportJob, error)
	// Get returns domain.ErrNotFound when the job is not on the document.
	Get(ctx context.Context, documentID, id string) (domain.ExportJob, error)
	// List returns the user's unexpired jobs on the document, newest first, at most 10.
	List(ctx context.Context, documentID, userID string) ([]domain.ExportJob, error)
	// Claim marks the oldest runnable job running, across tenants. domain.ErrNotFound: queue empty.
	Claim(ctx context.Context) (domain.ExportJob, error)
	Finish(ctx context.Context, id, fileKey string) error
	Fail(ctx context.Context, id, reason string) error
	// Expired and Delete look across tenants (worker cleanup).
	Expired(ctx context.Context) ([]domain.ExportJob, error)
	Delete(ctx context.Context, id string) error
	// Settings returns domain.ErrNotFound before the first export.
	Settings(ctx context.Context, documentID string) (domain.ReportSettings, error)
	SaveSettings(ctx context.Context, tenantID, documentID string, s domain.ReportSettings) error
}

// ReportRenderer turns a report into PDF bytes (ADR-0010). An unreachable
// renderer returns domain.ErrUnavailable.
type ReportRenderer interface {
	Render(ctx context.Context, r domain.Report) ([]byte, error)
}

// FileStore keeps export files, encrypted at rest (PRD-0006 NFR-2).
// Get returns domain.ErrNotFound for a missing file; Delete is idempotent.
type FileStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
```

Run `go build ./...` and commit with the next task.

---

### Task 5: Postgres ExportRepo

**Files:**
- Create: `backend/internal/adapters/postgres/export_repo.go`
- Test: `backend/internal/adapters/postgres/exports_integration_test.go`

**Step 1: Write the failing integration test** (build tag `integration`; same setup as `dashboard_integration_test.go`)

```go
//go:build integration

package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestExports(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, exports := NewUserRepo(db), NewDocumentRepo(db), NewExportRepo(db)
	admin, ta := provisionTenant(t, users, "A", "a@example.com")
	_, tb := provisionTenant(t, users, "B", "b@example.com")
	ctx := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: admin.ID, Title: "2026"}, nil)
	require.NoError(t, err)

	// Settings round trip.
	_, err = exports.Settings(ctx, doc.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	s := domain.ReportSettings{GoalsThisYear: "ship", SectionMap: map[string]string{"x": "Projects"}}
	require.NoError(t, exports.SaveSettings(ctx, ta.ID, doc.ID, s))
	require.NoError(t, exports.SaveSettings(ctx, ta.ID, doc.ID, s)) // upsert
	got, err := exports.Settings(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, s, got)

	// Create, get, list; tenant B sees nothing.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	params := domain.ExportParams{Filter: domain.LogFilter{From: &from, Impacts: []string{"high"}}, Settings: s, TenantName: "A"}
	j, err := exports.Create(ctx, domain.ExportJob{TenantID: ta.ID, DocumentID: doc.ID, RequestedBy: admin.ID, Params: params})
	require.NoError(t, err)
	require.Equal(t, domain.ExportQueued, j.Status)
	require.WithinDuration(t, time.Now().Add(24*time.Hour), j.ExpiresAt, time.Minute)
	j2, err := exports.Get(ctx, doc.ID, j.ID)
	require.NoError(t, err)
	require.Equal(t, params.Filter.Impacts, j2.Params.Filter.Impacts)
	require.True(t, from.Equal(*j2.Params.Filter.From))
	list, err := exports.List(ctx, doc.ID, admin.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	_, err = exports.Get(ctxB, doc.ID, j.ID)
	require.ErrorIs(t, err, domain.ErrNotFound, "RLS")

	// Two concurrent claims take the job once.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			_, err := exports.Claim(context.Background())
			results <- err
		})
	}
	wg.Wait()
	close(results)
	var ok, empty int
	for err := range results {
		if err == nil {
			ok++
		} else {
			require.ErrorIs(t, err, domain.ErrNotFound)
			empty++
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, empty)

	require.NoError(t, exports.Finish(ctx, j.ID, ta.ID+"/"+j.ID+".pdf"))
	done, err := exports.Get(ctx, doc.ID, j.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ExportDone, done.Status)
	require.Equal(t, ta.ID+"/"+j.ID+".pdf", done.FileKey)

	// Expiry: force it into the past through the owner connection, then clean up.
	owner, err := Connect(context.Background(), ownerURL, 5*time.Second)
	require.NoError(t, err)
	defer owner.Close()
	_, err = owner.Pool.Exec(context.Background(), "UPDATE export_jobs SET expires_at = now() - interval '1 second'")
	require.NoError(t, err)
	expired, err := exports.Expired(context.Background())
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, done.FileKey, expired[0].FileKey)
	require.NoError(t, exports.Delete(context.Background(), j.ID))
	_, err = exports.Get(ctx, doc.ID, j.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
```

(If `sync.WaitGroup.Go` is not available, use `wg.Add(1); go func(){ defer wg.Done(); ... }()`.)

**Step 2:** Run `go test -tags integration ./internal/adapters/postgres/ -run TestExports`. Expected: FAIL (undefined: NewExportRepo).

**Step 3: Implement**

```go
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ExportRepo implements ports.ExportRepo.
type ExportRepo struct{ db *DB }

// NewExportRepo wires the repository to the pool.
func NewExportRepo(db *DB) *ExportRepo { return &ExportRepo{db: db} }

func toExportJob(j sqlcgen.ExportJob) (domain.ExportJob, error) {
	out := domain.ExportJob{ID: j.ID.String(), TenantID: j.TenantID.String(), DocumentID: j.DocumentID.String(),
		RequestedBy: j.RequestedBy.String(), Status: j.Status, Error: j.Error, FileKey: j.FileKey,
		CreatedAt: j.CreatedAt, ExpiresAt: j.ExpiresAt}
	if err := json.Unmarshal(j.Params, &out.Params); err != nil {
		return domain.ExportJob{}, fmt.Errorf("postgres: export params: %w", err)
	}
	return out, nil
}

func (r *ExportRepo) Create(ctx context.Context, j domain.ExportJob) (domain.ExportJob, error) {
	tid, err := parseID(j.TenantID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	did, err := parseID(j.DocumentID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	uid, err := parseID(j.RequestedBy)
	if err != nil {
		return domain.ExportJob{}, err
	}
	params, err := json.Marshal(j.Params)
	if err != nil {
		return domain.ExportJob{}, err
	}
	var out domain.ExportJob
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateExportJob(ctx, sqlcgen.CreateExportJobParams{TenantID: tid, DocumentID: did, RequestedBy: uid, Params: params})
		if err != nil {
			return wrap(err)
		}
		out, err = toExportJob(row)
		return err
	})
	return out, err
}

func (r *ExportRepo) Get(ctx context.Context, documentID, id string) (domain.ExportJob, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	jid, err := parseID(id)
	if err != nil {
		return domain.ExportJob{}, err
	}
	var out domain.ExportJob
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetExportJob(ctx, sqlcgen.GetExportJobParams{ID: jid, DocumentID: did})
		if err != nil {
			return wrap(err)
		}
		out, err = toExportJob(row)
		return err
	})
	return out, err
}

func (r *ExportRepo) List(ctx context.Context, documentID, userID string) ([]domain.ExportJob, error) {
	did, err := parseID(documentID)
	if err != nil {
		return nil, err
	}
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	out := []domain.ExportJob{}
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListExportJobs(ctx, sqlcgen.ListExportJobsParams{DocumentID: did, RequestedBy: uid})
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			j, err := toExportJob(row)
			if err != nil {
				return err
			}
			out = append(out, j)
		}
		return nil
	})
	return out, err
}

// Claim runs under the provisioning flag: the tenant is what we are about to learn.
func (r *ExportRepo) Claim(ctx context.Context) (domain.ExportJob, error) {
	var out domain.ExportJob
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row, err := sqlcgen.New(tx).ClaimExportJob(ctx)
		if err != nil {
			return wrap(err) // no row: domain.ErrNotFound
		}
		out, err = toExportJob(row)
		return err
	})
	return out, err
}

func (r *ExportRepo) Finish(ctx context.Context, id, fileKey string) error {
	jid, err := parseID(id)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.FinishExportJob(ctx, sqlcgen.FinishExportJobParams{ID: jid, FileKey: fileKey}))
	})
}

func (r *ExportRepo) Fail(ctx context.Context, id, reason string) error {
	jid, err := parseID(id)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.FailExportJob(ctx, sqlcgen.FailExportJobParams{ID: jid, Error: reason}))
	})
}

// Expired runs under the provisioning flag: cleanup spans tenants.
func (r *ExportRepo) Expired(ctx context.Context) ([]domain.ExportJob, error) {
	out := []domain.ExportJob{}
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListExpiredExportJobs(ctx)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			j, err := toExportJob(row)
			if err != nil {
				return err
			}
			out = append(out, j)
		}
		return nil
	})
	return out, err
}

// Delete runs under the provisioning flag, after Expired.
func (r *ExportRepo) Delete(ctx context.Context, id string) error {
	jid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).DeleteExportJob(ctx, jid))
	})
}

func (r *ExportRepo) Settings(ctx context.Context, documentID string) (domain.ReportSettings, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.ReportSettings{}, err
	}
	var out domain.ReportSettings
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetReportSettings(ctx, did)
		if err != nil {
			return wrap(err)
		}
		out = domain.ReportSettings{GoalsThisYear: row.GoalsThisYear, GoalsNextYear: row.GoalsNextYear}
		return json.Unmarshal(row.SectionMap, &out.SectionMap)
	})
	return out, err
}

func (r *ExportRepo) SaveSettings(ctx context.Context, tenantID, documentID string, s domain.ReportSettings) error {
	tid, err := parseID(tenantID)
	if err != nil {
		return err
	}
	did, err := parseID(documentID)
	if err != nil {
		return err
	}
	m, err := json.Marshal(orEmptyMap(s.SectionMap))
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.UpsertReportSettings(ctx, sqlcgen.UpsertReportSettingsParams{DocumentID: did, TenantID: tid,
			GoalsThisYear: s.GoalsThisYear, GoalsNextYear: s.GoalsNextYear, SectionMap: m}))
	})
}

// orEmptyMap keeps jsonb a JSON object: nil marshals to null.
func orEmptyMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
```

Adjust the sqlc param struct names to whatever `make sqlc` generated (e.g. `GetExportJobParams`). If sqlc generates `ClaimExportJob(ctx)` with no params, keep the call above as it is.

**Step 4:** Run `go test -tags integration ./internal/adapters/postgres/ -run TestExports`. Expected: PASS. Then run `make lint`.

**Step 5: Commit**

```bash
git add backend/internal/ports/exports.go backend/internal/adapters/postgres/export_repo.go backend/internal/adapters/postgres/exports_integration_test.go
git commit -m "feat(postgres): export repo with cross-tenant claim and cleanup"
```

---

### Task 6: Encrypted file store

**Files:**
- Create: `backend/internal/adapters/files/store.go`
- Test: `backend/internal/adapters/files/store_test.go`

**Step 1: Write the failing test**

```go
package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func key32() []byte { return []byte("0123456789abcdef0123456789abcdef") }

func TestStoreRoundTripEncrypted(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, key32())
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "t1/j1.pdf", []byte("%PDF-1.7 hello")))

	raw, err := os.ReadFile(filepath.Join(dir, "t1", "j1.pdf"))
	require.NoError(t, err)
	require.NotContains(t, string(raw), "hello", "encrypted at rest")

	got, err := s.Get(ctx, "t1/j1.pdf")
	require.NoError(t, err)
	require.Equal(t, "%PDF-1.7 hello", string(got))

	require.NoError(t, s.Delete(ctx, "t1/j1.pdf"))
	require.NoError(t, s.Delete(ctx, "t1/j1.pdf"), "idempotent")
	_, err = s.Get(ctx, "t1/j1.pdf")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStoreRejectsTamperingAndSwaps(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, key32())
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "t1/a.pdf", []byte("A")))
	require.NoError(t, s.Put(ctx, "t2/b.pdf", []byte("B")))

	// A file copied under another key does not decrypt: the key is the AAD.
	raw, _ := os.ReadFile(filepath.Join(dir, "t1", "a.pdf"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "t2", "b.pdf"), raw, 0o600))
	_, err = s.Get(ctx, "t2/b.pdf")
	require.Error(t, err)

	raw[len(raw)-1] ^= 1
	require.NoError(t, os.WriteFile(filepath.Join(dir, "t1", "a.pdf"), raw, 0o600))
	_, err = s.Get(ctx, "t1/a.pdf")
	require.Error(t, err)
}

func TestStoreRejectsBadKeys(t *testing.T) {
	_, err := New(t.TempDir(), []byte("short"))
	require.Error(t, err)
	s, err := New(t.TempDir(), key32())
	require.NoError(t, err)
	require.Error(t, s.Put(context.Background(), "../escape.pdf", []byte("x")))
}
```

**Step 2:** Run `go test ./internal/adapters/files/`. Expected: FAIL.

**Step 3: Implement**

```go
// Package files is the local-disk FileStore: AES-256-GCM at rest (PRD-0006
// NFR-2), confined to one directory with os.Root.
package files

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Store implements ports.FileStore. The file key is the GCM additional data,
// so a file moved under another key fails to open.
type Store struct {
	root *os.Root
	aead cipher.AEAD
}

// New opens dir (created if missing) with a 32-byte key.
func New(dir string, key []byte) (*Store, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("files: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	return &Store{root: root, aead: aead}, nil
}

// ponytail: whole file in memory; a 2,000-log report is a few MB. Stream (chunked GCM) if reports grow.
func (s *Store) Put(_ context.Context, key string, data []byte) error {
	nonce := make([]byte, s.aead.NonceSize())
	_, _ = rand.Read(nonce)
	sealed := s.aead.Seal(nonce, nonce, data, []byte(key))
	if err := s.root.MkdirAll(path.Dir(key), 0o700); err != nil {
		return fmt.Errorf("files: %w", err)
	}
	tmp := key + ".tmp"
	if err := s.root.WriteFile(tmp, sealed, 0o600); err != nil {
		return fmt.Errorf("files: %w", err)
	}
	if err := s.root.Rename(tmp, key); err != nil {
		return fmt.Errorf("files: %w", err)
	}
	return nil
}

func (s *Store) Get(_ context.Context, key string) ([]byte, error) {
	sealed, err := s.root.ReadFile(key)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	n := s.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("files: truncated file")
	}
	data, err := s.aead.Open(nil, sealed[:n], sealed[n:], []byte(key))
	if err != nil {
		return nil, fmt.Errorf("files: %s: %w", key, err)
	}
	return data, nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	if err := s.root.Remove(key); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("files: %w", err)
	}
	return nil
}
```

**Step 4:** Run `go test ./internal/adapters/files/`. Expected: PASS. Run `make lint`. If depguard needs an entry for the new adapter package, mirror the rule for `adapters/email` in `.golangci.yml`.

**Step 5: Commit**

```bash
git add backend/internal/adapters/files backend/.golangci.yml
git commit -m "feat(files): AES-GCM file store confined with os.Root"
```

---

### Task 7: Report template and Gotenberg client

**Files:**
- Create: `backend/internal/adapters/gotenberg/client.go`, `backend/internal/adapters/gotenberg/report.html`, `backend/internal/adapters/gotenberg/footer.html`
- Test: `backend/internal/adapters/gotenberg/client_test.go`, `backend/internal/adapters/gotenberg/client_integration_test.go`

**Step 1: Write the failing tests** (`client_test.go`)

```go
package gotenberg

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func fixture() domain.Report {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	logs := []domain.Log{
		{Name: "Shipped SSO", Description: "Line one\n<script>alert(1)</script>", Impact: "high", Status: "done",
			Tags: []string{"project"}, Links: []domain.Link{{URL: "https://github.com/acme/app/pull/12", Label: "PR"}},
			CreatedAt: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
		{Name: "Read DDIA", Impact: "low", Status: "in_progress", Tags: []string{"learning"},
			CreatedAt: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
	}
	r := domain.NewReport(logs, domain.ReportSettings{GoalsThisYear: "Lead the auth rewrite", SectionMap: domain.DefaultSectionMap})
	r.Title, r.Author, r.Tenant, r.From, r.To = "2026", "Ada", "Acme", &from, &to
	r.GeneratedAt = time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	return r
}

func TestRenderHTML(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, RenderHTML(&b, fixture()))
	html := b.String()
	for _, want := range []string{"2026", "Ada", "Acme", "2026-01-01 – 2026-12-31", "Generated 2026-10-09",
		"Goals for this year", "Lead the auth rewrite", "Summary",
		`href="https://github.com/acme/app/pull/12"`, "In progress"} {
		require.Contains(t, html, want)
	}
	require.NotContains(t, html, "Goals for next year", "empty goals are omitted")
	require.NotContains(t, html, "<script>alert", "descriptions are escaped")
	require.Less(t, strings.Index(html, "<h2>Projects</h2>"), strings.Index(html, "<h2>What you learned</h2>"))
	require.Less(t, strings.Index(html, "Summary"), strings.Index(html, "<h2>Projects</h2>"), "summary on page 2, before sections")
}

func TestRenderPostsToGotenberg(t *testing.T) {
	var fields = map[string]string{}
	var files []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/forms/chromium/convert/html", r.URL.Path)
		require.NoError(t, r.ParseMultipartForm(10<<20))
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
		for _, fh := range r.MultipartForm.File["files"] {
			files = append(files, fh.Filename)
		}
		_, _ = io.WriteString(w, "%PDF-1.7")
	}))
	defer srv.Close()
	pdf, err := New(srv.URL, 5*time.Second).Render(context.Background(), fixture())
	require.NoError(t, err)
	require.Equal(t, "%PDF-1.7", string(pdf))
	require.ElementsMatch(t, []string{"index.html", "footer.html"}, files)
	require.Equal(t, "true", fields["generateDocumentOutline"], "implies tagged PDF (NFR-3)")
}

func TestRenderUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err := New(srv.URL, time.Second).Render(context.Background(), fixture())
	require.ErrorIs(t, err, domain.ErrUnavailable)

	_, err = New("http://127.0.0.1:1", time.Second).Render(context.Background(), fixture())
	require.ErrorIs(t, err, domain.ErrUnavailable)
}
```

**Step 2:** Run `go test ./internal/adapters/gotenberg/`. Expected: FAIL.

**Step 3: Implement the template** (`report.html`)

```html
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<style>
  @page { size: A4; margin: 20mm 18mm 22mm; }
  body { font: 11pt/1.45 -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; color: #111; }
  h1 { font-size: 26pt; margin: 0 0 8mm; }
  h2 { font-size: 16pt; margin: 10mm 0 4mm; border-bottom: 1px solid #ccc; padding-bottom: 1mm; }
  h3 { font-size: 12pt; margin: 0 0 1mm; }
  .cover { height: 230mm; display: flex; flex-direction: column; justify-content: center; break-after: page; }
  .cover p { margin: 1mm 0; color: #444; }
  .page2 { break-after: page; }
  .goals { white-space: pre-wrap; }
  table { border-collapse: collapse; width: 100%; font-size: 10pt; }
  th, td { border: 1px solid #ccc; padding: 1.5mm 2mm; text-align: right; }
  th:first-child, td:first-child { text-align: left; }
  article { break-inside: avoid; margin: 0 0 6mm; }
  .meta, .tags { color: #555; font-size: 9.5pt; margin: 0 0 1.5mm; }
  .desc { white-space: pre-wrap; margin: 0 0 1.5mm; }
  ul.links { margin: 0; padding-left: 5mm; font-size: 9.5pt; }
  a { color: #0645ad; word-break: break-all; }
</style>
</head>
<body>
<section class="cover">
  <h1>{{.Title}}</h1>
  <p>{{.Author}} · {{.Tenant}}</p>
  <p>{{period .From .To}}</p>
  <p>Generated {{date .GeneratedAt}}</p>
</section>

<section class="page2">
  {{with .GoalsThisYear}}<h2>Goals for this year</h2><p class="goals">{{.}}</p>{{end}}
  {{with .GoalsNextYear}}<h2>Goals for next year</h2><p class="goals">{{.}}</p>{{end}}
  <h2>Summary</h2>
  <table>
    <thead><tr><th scope="col">Impact</th>{{range statuses}}<th scope="col">{{statusLabel .}}</th>{{end}}<th scope="col">Total</th></tr></thead>
    <tbody>
    {{range .Summary}}<tr><th scope="row">{{.Impact}}</th>{{range .Counts}}<td>{{.}}</td>{{end}}<td>{{.Total}}</td></tr>
    {{end}}</tbody>
    <tfoot><tr><th scope="row">Total</th>{{range .StatusTotals}}<td>{{.}}</td>{{end}}<td>{{.Total}}</td></tr></tfoot>
  </table>
</section>

{{range .Sections}}
<section>
  <h2>{{.Name}}</h2>
  {{range .Logs}}
  <article>
    <h3>{{.Name}}</h3>
    <p class="meta">{{date .CreatedAt}} · Impact: {{.Impact}} · {{statusLabel .Status}}</p>
    {{with .Description}}<p class="desc">{{.}}</p>{{end}}
    {{with .Tags}}<p class="tags">Tags: {{join . ", "}}</p>{{end}}
    {{with .Links}}<ul class="links">{{range .}}<li><a href="{{.URL}}">{{or .Label .URL}}</a></li>{{end}}</ul>{{end}}
  </article>
  {{end}}
</section>
{{end}}
</body>
</html>
```

`footer.html` (Chromium fills `pageNumber`/`totalPages`; the footer has no access to the page's CSS):

```html
<html><head><style>body{font:8pt sans-serif;color:#777;width:100%;text-align:center;margin:0 0 8mm}</style></head>
<body><span class="pageNumber"></span> / <span class="totalPages"></span></body></html>
```

**Step 4: Implement the client** (`client.go`)

```go
// Package gotenberg renders the PDF report (ADR-0010): html/template to
// HTML, Gotenberg's Chromium route to PDF.
package gotenberg

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

//go:embed report.html footer.html
var assets embed.FS

var statusLabels = map[string]string{"idea": "Idea", "in_progress": "In progress", "done": "Done", "dropped": "Dropped"}

var tmpl = template.Must(template.New("report.html").Funcs(template.FuncMap{
	"date":        func(t time.Time) string { return t.UTC().Format(time.DateOnly) },
	"join":        strings.Join,
	"statuses":    func() []string { return domain.Statuses },
	"statusLabel": func(s string) string { return statusLabels[s] },
	// period prints the inclusive range; To is exclusive in the domain.
	"period": func(from, to *time.Time) string {
		switch {
		case from == nil && to == nil:
			return "All time"
		case to == nil:
			return "Since " + from.UTC().Format(time.DateOnly)
		case from == nil:
			return "Until " + to.AddDate(0, 0, -1).UTC().Format(time.DateOnly)
		}
		return from.UTC().Format(time.DateOnly) + " – " + to.AddDate(0, 0, -1).UTC().Format(time.DateOnly)
	},
}).ParseFS(assets, "report.html"))

const maxPDF = 100 << 20

// Client implements ports.ReportRenderer.
type Client struct {
	url  string
	http *http.Client
}

// New points the client at Gotenberg's base URL.
func New(url string, timeout time.Duration) *Client {
	return &Client{url: strings.TrimRight(url, "/"), http: &http.Client{Timeout: timeout}}
}

// RenderHTML writes the report page. html/template escapes every field, so
// log text never reaches Chromium as markup.
func RenderHTML(w io.Writer, r domain.Report) error { return tmpl.Execute(w, r) }

// Render returns the PDF. Gotenberg down or failing is domain.ErrUnavailable.
func (c *Client) Render(ctx context.Context, r domain.Report) ([]byte, error) {
	var page bytes.Buffer
	if err := RenderHTML(&page, r); err != nil {
		return nil, fmt.Errorf("gotenberg: template: %w", err)
	}
	footer, _ := assets.ReadFile("footer.html")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for name, data := range map[string][]byte{"index.html": page.Bytes(), "footer.html": footer} {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(data); err != nil {
			return nil, err
		}
	}
	for k, v := range map[string]string{
		"generateDocumentOutline": "true", // implies generateTaggedPdf (NFR-3)
		"printBackground":         "true",
		"preferCssPageSize":       "true",
	} {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/forms/chromium/convert/html", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: gotenberg: %v", domain.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("%w: gotenberg %d: %s", domain.ErrUnavailable, resp.StatusCode, msg)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxPDF))
}
```

The footer margin comes from `@page` (`preferCssPageSize`). If the page number overlaps the content, raise the `@page` bottom margin.

**Step 5: Integration test** (`client_integration_test.go`, build tag `integration`). It needs a running Gotenberg (`docker compose up gotenberg`) and is skipped when Gotenberg is unreachable.

```go
//go:build integration

package gotenberg

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func gotenbergURL(t *testing.T) string {
	u := os.Getenv("GOTENBERG_URL")
	if u == "" {
		u = "http://localhost:3000"
	}
	if resp, err := http.Get(u + "/health"); err != nil || resp.StatusCode != 200 {
		t.Skip("gotenberg not reachable at " + u)
	}
	return u
}

// ADR-0010 confirmation: a real PDF with the sections, under NFR-1 for 500 logs.
func TestRenderRealPDF(t *testing.T) {
	c := New(gotenbergURL(t), 60*time.Second)
	logs := make([]domain.Log, 500)
	for i := range logs {
		logs[i] = domain.Log{Name: fmt.Sprintf("Log %d", i), Description: "Did a thing that mattered.",
			Impact: domain.Impacts[i%4], Status: domain.Statuses[i%4], Tags: []string{domain.SuggestedTags[i%8]},
			Links: []domain.Link{{URL: "https://example.com/" + fmt.Sprint(i)}}, CreatedAt: time.Now()}
	}
	r := domain.NewReport(logs, domain.DefaultReportSettings())
	r.Title, r.Author, r.Tenant, r.GeneratedAt = "2026", "Ada", "Acme", time.Now()
	start := time.Now()
	pdf, err := c.Render(context.Background(), r)
	t.Logf("500 logs rendered in %s (NFR-1: < 15s)", time.Since(start))
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")))
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed; skipping text checks")
	}
	f := t.TempDir() + "/r.pdf"
	require.NoError(t, os.WriteFile(f, pdf, 0o600))
	out, err := exec.Command("pdftotext", f, "-").Output()
	require.NoError(t, err)
	for _, h := range []string{"Summary", "Projects", "Collaboration & mentorship", "What you learned"} {
		require.Contains(t, string(out), h)
	}
	info, err := exec.Command("pdfinfo", f).Output()
	if err == nil {
		require.Contains(t, string(info), "Tagged:         yes")
	}
}
```

**Step 6:** Run `go test ./internal/adapters/gotenberg/`. Expected: PASS. Then run `docker compose up -d gotenberg && go test -tags integration ./internal/adapters/gotenberg/ -v`. Expected: PASS, with the duration logged. Then `make lint` (add a depguard entry for the package like the other adapters if needed).

**Step 7: Commit**

```bash
git add backend/internal/adapters/gotenberg backend/.golangci.yml
git commit -m "feat(gotenberg): report template and Chromium HTML-to-PDF client"
```

---

### Task 8: Use cases: Create, Get, List, Open, Settings

**Files:**
- Create: `backend/internal/app/exports.go`, `export_create.go`, `export_get.go`, `export_settings.go`
- Test: `backend/internal/app/exports_test.go`; modify `fakes_test.go` and `access_matrix_test.go`

**Step 1: Fakes** (append to `fakes_test.go`)

```go
type fakeExports struct {
	jobs     map[string]domain.ExportJob
	order    []string
	settings map[string]domain.ReportSettings
	seq      int
}

func newFakeExports() *fakeExports {
	return &fakeExports{jobs: map[string]domain.ExportJob{}, settings: map[string]domain.ReportSettings{}}
}

func (f *fakeExports) Create(_ context.Context, j domain.ExportJob) (domain.ExportJob, error) {
	f.seq++
	j.ID, j.Status, j.CreatedAt = "j"+strconv.Itoa(f.seq), domain.ExportQueued, time.Now()
	j.ExpiresAt = j.CreatedAt.Add(24 * time.Hour)
	f.jobs[j.ID] = j
	f.order = append(f.order, j.ID)
	return j, nil
}
func (f *fakeExports) Get(_ context.Context, docID, id string) (domain.ExportJob, error) {
	j, ok := f.jobs[id]
	if !ok || j.DocumentID != docID {
		return domain.ExportJob{}, domain.ErrNotFound
	}
	return j, nil
}
func (f *fakeExports) List(_ context.Context, docID, userID string) ([]domain.ExportJob, error) {
	out := []domain.ExportJob{}
	for _, id := range slices.Backward(f.order) {
		if j := f.jobs[id]; j.DocumentID == docID && j.RequestedBy == userID {
			out = append(out, j)
		}
	}
	return out, nil
}
func (f *fakeExports) Claim(context.Context) (domain.ExportJob, error) {
	for _, id := range f.order {
		if j := f.jobs[id]; j.Status == domain.ExportQueued {
			j.Status = domain.ExportRunning
			f.jobs[id] = j
			return j, nil
		}
	}
	return domain.ExportJob{}, domain.ErrNotFound
}
func (f *fakeExports) Finish(_ context.Context, id, key string) error {
	j := f.jobs[id]
	j.Status, j.FileKey, j.ExpiresAt = domain.ExportDone, key, time.Now().Add(24*time.Hour)
	f.jobs[id] = j
	return nil
}
func (f *fakeExports) Fail(_ context.Context, id, reason string) error {
	j := f.jobs[id]
	j.Status, j.Error = domain.ExportFailed, reason
	f.jobs[id] = j
	return nil
}
func (f *fakeExports) Expired(context.Context) ([]domain.ExportJob, error) {
	out := []domain.ExportJob{}
	for _, id := range f.order {
		if j, ok := f.jobs[id]; ok && !time.Now().Before(j.ExpiresAt) {
			out = append(out, j)
		}
	}
	return out, nil
}
func (f *fakeExports) Delete(_ context.Context, id string) error { delete(f.jobs, id); return nil }
func (f *fakeExports) Settings(_ context.Context, docID string) (domain.ReportSettings, error) {
	s, ok := f.settings[docID]
	if !ok {
		return domain.ReportSettings{}, domain.ErrNotFound
	}
	return s, nil
}
func (f *fakeExports) SaveSettings(_ context.Context, _, docID string, s domain.ReportSettings) error {
	f.settings[docID] = s
	return nil
}

type fakeRenderer struct {
	got domain.Report
	err error
}

func (f *fakeRenderer) Render(_ context.Context, r domain.Report) ([]byte, error) {
	f.got = r
	return []byte("%PDF"), f.err
}

type fakeFiles struct{ files map[string][]byte }

func newFakeFiles() *fakeFiles { return &fakeFiles{files: map[string][]byte{}} }
func (f *fakeFiles) Put(_ context.Context, k string, b []byte) error { f.files[k] = b; return nil }
func (f *fakeFiles) Get(_ context.Context, k string) ([]byte, error) {
	b, ok := f.files[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return b, nil
}
func (f *fakeFiles) Delete(_ context.Context, k string) error { delete(f.files, k); return nil }
```

Check that `fakeLogs.List` honours `Total`, `Page`, and `PerPage`. If it ignores paging, extend it so that `Total` is the number of matches and `Items` is the requested page. The worker test below depends on that.

**Step 2: Write the failing tests** (`exports_test.go`)

```go
package app

import (
	"context"
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
```

(add `"strconv"` to the imports).

**Step 3:** Run `go test ./internal/app/ -run Export`. Expected: FAIL (undefined: NewExports).

**Step 4: Implement**

`exports.go`:

```go
package app

import (
	"context"
	"strconv"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Exports is the PDF report (PRD-0006): request, status, download, and the
// worker's run and cleanup. Everything needs PermRead (FR-6).
type Exports struct {
	docs  ports.DocumentRepo
	logs  ports.LogRepo
	jobs  ports.ExportRepo
	pdf   ports.ReportRenderer
	files ports.FileStore
	cache ports.Cache // Degrading: progress is cosmetic
	// scope puts the tenant in the context (telemetry.WithTenantID); injected
	// because app may not import telemetry.
	scope func(ctx context.Context, tenantID string) context.Context
	now   func() time.Time
}

// NewExports wires the use cases.
func NewExports(docs ports.DocumentRepo, logs ports.LogRepo, jobs ports.ExportRepo, pdf ports.ReportRenderer,
	files ports.FileStore, cache ports.Cache, scope func(context.Context, string) context.Context) *Exports {
	return &Exports{docs: docs, logs: logs, jobs: jobs, pdf: pdf, files: files, cache: cache, scope: scope, now: time.Now}
}

const progressTTL = time.Hour

func progressKey(jobID string) string { return "export:" + jobID + ":progress" }

func (s *Exports) setProgress(ctx context.Context, jobID string, pct int) {
	_ = s.cache.Set(ctx, progressKey(jobID), []byte(strconv.Itoa(pct)), progressTTL)
}

func (s *Exports) progress(ctx context.Context, j domain.ExportJob) int {
	switch j.Status {
	case domain.ExportDone:
		return 100
	case domain.ExportRunning:
		if b, ok, _ := s.cache.Get(ctx, progressKey(j.ID)); ok {
			n, _ := strconv.Atoi(string(b))
			return n
		}
	}
	return 0
}
```

`export_create.go`:

```go
package app

import (
	"context"
	"fmt"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CreateExportInput is one "Generate" click. TenantName goes on the cover.
type CreateExportInput struct {
	DocumentID, UserID, TenantName string
	Filter                         domain.LogFilter
	Settings                       domain.ReportSettings
}

// Create checks the size (FR-7), saves the settings for next time, and queues the job.
func (s *Exports) Create(ctx context.Context, in CreateExportInput) (domain.ExportJob, error) {
	d, err := access(ctx, s.docs, in.DocumentID, in.UserID, domain.PermRead)
	if err != nil {
		return domain.ExportJob{}, err
	}
	f := in.Filter
	f.HideExamples, f.Page, f.PerPage = true, 1, 1
	if err := f.Validate(); err != nil {
		return domain.ExportJob{}, err
	}
	if err := in.Settings.Validate(); err != nil {
		return domain.ExportJob{}, err
	}
	page, err := s.logs.List(ctx, d.ID, f)
	if err != nil {
		return domain.ExportJob{}, err
	}
	switch {
	case page.Total == 0:
		return domain.ExportJob{}, domain.NewValidationError(map[string]string{"filters": "no logs match these filters"})
	case page.Total > domain.MaxReportLogs:
		return domain.ExportJob{}, domain.NewValidationError(map[string]string{"filters": fmt.Sprintf(
			"%d logs match; narrow the filters to at most %d", page.Total, domain.MaxReportLogs)})
	}
	if err := s.jobs.SaveSettings(ctx, d.TenantID, d.ID, in.Settings); err != nil {
		return domain.ExportJob{}, err
	}
	return s.jobs.Create(ctx, domain.ExportJob{TenantID: d.TenantID, DocumentID: d.ID, RequestedBy: in.UserID,
		Params: domain.ExportParams{Filter: f, Settings: in.Settings, TenantName: in.TenantName}})
}
```

`export_get.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns one of the caller's jobs with its progress. Another user's job is not found.
func (s *Exports) Get(ctx context.Context, docID, jobID, userID string) (domain.ExportJob, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return domain.ExportJob{}, err
	}
	j, err := s.jobs.Get(ctx, docID, jobID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	if j.RequestedBy != userID {
		return domain.ExportJob{}, domain.ErrNotFound
	}
	j.Progress = s.progress(ctx, j)
	return j, nil
}

// List is the dialog's history: the caller's live jobs, newest first.
func (s *Exports) List(ctx context.Context, docID, userID string) ([]domain.ExportJob, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return nil, err
	}
	jobs, err := s.jobs.List(ctx, docID, userID)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		jobs[i].Progress = s.progress(ctx, jobs[i])
	}
	return jobs, nil
}

// Open returns the decrypted PDF while it is downloadable (FR-5). Access is
// checked again, so a revoked reader cannot download an old report.
func (s *Exports) Open(ctx context.Context, docID, jobID, userID string) ([]byte, error) {
	j, err := s.Get(ctx, docID, jobID, userID)
	if err != nil {
		return nil, err
	}
	if !j.Downloadable(s.now()) {
		return nil, domain.ErrNotFound
	}
	return s.files.Get(ctx, j.FileKey)
}
```

`export_settings.go`:

```go
package app

import (
	"context"
	"errors"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Settings prefills the export dialog: the last export's goals and mapping, or the defaults.
func (s *Exports) Settings(ctx context.Context, docID, userID string) (domain.ReportSettings, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return domain.ReportSettings{}, err
	}
	st, err := s.jobs.Settings(ctx, docID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.DefaultReportSettings(), nil
	}
	return st, err
}
```

**Step 5: Access matrix.** Add `exports *Exports` to `matrixFixture`, building it as `NewExports(fd, fl, newFakeExports(), &fakeRenderer{}, newFakeFiles(), newFakeCache(), func(ctx context.Context, _ string) context.Context { return ctx })`. Note that `fl` must be the same fake that holds the log. Then add these actions:

```go
		{"export pdf", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.exports.Create(ctx, CreateExportInput{DocumentID: "d1", UserID: u.ID})
			return err
		}},
		{"export settings", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.exports.Settings(ctx, "d1", u.ID)
			return err
		}},
		{"export history", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.exports.List(ctx, "d1", u.ID)
			return err
		}},
```

**Step 6:** Run `go test ./internal/app/`. Expected: PASS.

**Step 7: Commit**

```bash
git add backend/internal/app
git commit -m "feat(app): export create, status, history, download, and settings"
```

---

### Task 9: Use cases: worker run and cleanup

**Files:**
- Create: `backend/internal/app/export_run.go`
- Test: append to `backend/internal/app/exports_test.go`

**Step 1: Write the failing tests**

```go
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
```

(add `"time"` to the imports).

**Step 2:** Run `go test ./internal/app/ -run Export`. Expected: FAIL (undefined: RunNext).

**Step 3: Implement** (`export_run.go`)

```go
package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

const reportPageSize = 100 // the list's maximum page

// RunNext claims and runs one job. It returns false when the queue is empty.
// A job that fails is marked failed; only repository errors come back.
func (s *Exports) RunNext(ctx context.Context) (bool, error) {
	j, err := s.jobs.Claim(ctx)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	ctx = s.scope(ctx, j.TenantID)
	key, err := s.run(ctx, j)
	if err != nil {
		slog.WarnContext(ctx, "export failed", slog.String("job_id", j.ID), slog.Any("err", err))
		return true, s.jobs.Fail(ctx, j.ID, failReason(err))
	}
	slog.InfoContext(ctx, "export done", slog.String("job_id", j.ID))
	return true, s.jobs.Finish(ctx, j.ID, key)
}

func (s *Exports) run(ctx context.Context, j domain.ExportJob) (string, error) {
	// FR-6: the requester's access as of now, not as of the request.
	d, err := access(ctx, s.docs, j.DocumentID, j.RequestedBy, domain.PermRead)
	if err != nil {
		return "", err
	}
	f := j.Params.Filter
	f.Sort, f.Desc, f.PerPage = "created_at", false, reportPageSize
	var logs []domain.Log
	for f.Page = 1; len(logs) < domain.MaxReportLogs; f.Page++ {
		page, err := s.logs.List(ctx, d.ID, f)
		if err != nil {
			return "", err
		}
		logs = append(logs, page.Items...)
		s.setProgress(ctx, j.ID, 80*len(logs)/max(page.Total, 1))
		if len(page.Items) < reportPageSize {
			break
		}
	}
	r := domain.NewReport(logs, j.Params.Settings)
	r.Title, r.Author, r.Tenant, r.From, r.To, r.GeneratedAt = d.Title, d.OwnerName, j.Params.TenantName, f.From, f.To, s.now()
	pdf, err := s.pdf.Render(ctx, r)
	if err != nil {
		return "", err
	}
	s.setProgress(ctx, j.ID, 95)
	key := j.TenantID + "/" + j.ID + ".pdf"
	return key, s.files.Put(ctx, key, pdf)
}

// failReason is what the user sees in the dialog.
func failReason(err error) string {
	var ae *domain.AccessError
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.As(err, &ae):
		return "You no longer have access to this document."
	case errors.Is(err, domain.ErrUnavailable):
		return "The PDF service is unavailable. Try again in a minute."
	}
	return "Generating the report failed. Try again."
}

// Cleanup deletes expired jobs and their files (FR-5, NFR-2). The file goes
// first: a crash in between leaves a row the next tick retries, never an orphan file.
func (s *Exports) Cleanup(ctx context.Context) error {
	jobs, err := s.jobs.Expired(ctx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.FileKey != "" {
			if err := s.files.Delete(ctx, j.FileKey); err != nil {
				return err
			}
		}
		if err := s.jobs.Delete(ctx, j.ID); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 4:** Run `go test ./internal/app/` and `make lint`. Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/app
git commit -m "feat(app): export worker run with access re-check, and expiry cleanup"
```

---

### Task 10: HTTP handlers

**Files:**
- Modify: `backend/internal/adapters/http/logs_handler.go` (split the query parsing out of `logFilter`)
- Create: `backend/internal/adapters/http/exports_handler.go`
- Test: `backend/internal/adapters/http/exports_handler_test.go`

**Step 1: Refactor.** In `logs_handler.go`, change `logFilter(c *gin.Context)` to `logFilter(q url.Values)` and make the body start from `q` instead of `c.Request.URL.Query()`. The `/logs` handler then calls `logFilter(c.Request.URL.Query())`. Run `go test ./internal/adapters/http/`. Expected: still PASS.

**Step 2: Write the failing tests**

```go
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
func (f *fakeExportUC) Get(context.Context, string, string, string) (domain.ExportJob, error) { return f.job, f.err }
func (f *fakeExportUC) List(context.Context, string, string) ([]domain.ExportJob, error) {
	return []domain.ExportJob{f.job}, f.err
}
func (f *fakeExportUC) Open(context.Context, string, string, string) ([]byte, error) { return f.pdf, f.err }
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
}

func TestExportFileServesPDF(t *testing.T) {
	uc := &fakeExportUC{pdf: []byte("%PDF-1.7")}
	rec := do(exportsEngine(t, uc), http.MethodGet, "/documents/d1/exports/j1/file", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
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
```

**Step 3:** Run `go test ./internal/adapters/http/ -run Export`. Expected: FAIL.

**Step 4: Implement** (`exports_handler.go`)

```go
package http

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ExportUseCases is the slice of app.Exports the handlers need.
type ExportUseCases interface {
	Create(ctx context.Context, in app.CreateExportInput) (domain.ExportJob, error)
	Get(ctx context.Context, docID, jobID, userID string) (domain.ExportJob, error)
	List(ctx context.Context, docID, userID string) ([]domain.ExportJob, error)
	Open(ctx context.Context, docID, jobID, userID string) ([]byte, error)
	Settings(ctx context.Context, docID, userID string) (domain.ReportSettings, error)
}

// ReportSettingsDTO is the dialog's goals and tag→section mapping.
type ReportSettingsDTO struct {
	GoalsThisYear string            `json:"goals_this_year"`
	GoalsNextYear string            `json:"goals_next_year"`
	SectionMap    map[string]string `json:"section_map"`
}

// CreateExportRequest carries the logs list's URL query string verbatim (FR-1) plus the settings.
type CreateExportRequest struct {
	Query string `json:"query"`
	ReportSettingsDTO
}

// ExportJobResponse is one job; progress is 0–100.
type ExportJobResponse struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	Error        string    `json:"error"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Downloadable bool      `json:"downloadable"`
}

// ExportListResponse is the dialog's history.
type ExportListResponse struct {
	Items []ExportJobResponse `json:"items"`
}

func toExportJob(j domain.ExportJob) ExportJobResponse {
	return ExportJobResponse{ID: j.ID, Status: j.Status, Progress: j.Progress, Error: j.Error,
		CreatedAt: j.CreatedAt, ExpiresAt: j.ExpiresAt, Downloadable: j.Downloadable(time.Now())}
}

// RegisterExports adds /documents/:id/exports and /documents/:id/report-settings.
func RegisterExports(r gin.IRouter, uc ExportUseCases) {
	g := r.Group("/documents/:id")
	g.GET("/report-settings", func(c *gin.Context) {
		s, err := uc.Settings(c.Request.Context(), c.Param("id"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, ReportSettingsDTO{GoalsThisYear: s.GoalsThisYear, GoalsNextYear: s.GoalsNextYear, SectionMap: s.SectionMap})
	})
	g.POST("/exports", func(c *gin.Context) {
		var req CreateExportRequest
		if !bindJSON(c, &req) {
			return
		}
		q, err := url.ParseQuery(req.Query)
		if err != nil {
			RespondError(c, domain.NewValidationError(map[string]string{"query": "must be a URL query string"}))
			return
		}
		f, err := logFilter(q)
		if err != nil {
			RespondError(c, err)
			return
		}
		p := principal(c)
		j, err := uc.Create(c.Request.Context(), app.CreateExportInput{DocumentID: c.Param("id"), UserID: p.User.ID,
			TenantName: p.Tenant.Name, Filter: f, Settings: domain.ReportSettings{GoalsThisYear: req.GoalsThisYear,
				GoalsNextYear: req.GoalsNextYear, SectionMap: req.SectionMap}})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, toExportJob(j))
	})
	g.GET("/exports", func(c *gin.Context) {
		jobs, err := uc.List(c.Request.Context(), c.Param("id"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		items := make([]ExportJobResponse, 0, len(jobs))
		for _, j := range jobs {
			items = append(items, toExportJob(j))
		}
		c.JSON(http.StatusOK, ExportListResponse{Items: items})
	})
	g.GET("/exports/:jobId", func(c *gin.Context) {
		j, err := uc.Get(c.Request.Context(), c.Param("id"), c.Param("jobId"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toExportJob(j))
	})
	g.GET("/exports/:jobId/file", func(c *gin.Context) {
		pdf, err := uc.Open(c.Request.Context(), c.Param("id"), c.Param("jobId"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.Header("Content-Disposition", `attachment; filename="brag-report.pdf"`)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "application/pdf", pdf)
	})
}
```

**Step 5:** Run `go test ./internal/adapters/http/` and `make lint`. Expected: PASS.

**Step 6: Commit**

```bash
git add backend/internal/adapters/http
git commit -m "feat(api): export endpoints and report settings"
```

---

### Task 11: Config, wiring, worker command, compose

**Files:**
- Modify: `backend/internal/config/config.go`, `backend/internal/config/config_test.go`, `backend/cmd/bragdoc/api.go`, `backend/cmd/bragdoc/main.go`
- Create: `backend/cmd/bragdoc/worker.go`; delete `backend/cmd/bragdoc/daemon.go`
- Modify: `docker-compose.yml`, `.env.example`

**Step 1: Config.** Add these fields:

```go
	GotenbergTimeout time.Duration `env:"GOTENBERG_TIMEOUT" envDefault:"60s"`
	ExportDir        string        `env:"EXPORT_DIR" envDefault:"data/exports"`
	ExportKey        string        `env:"EXPORT_KEY"` // base64 of 32 bytes; required by api and worker (PRD-0006 NFR-2)
```

Add a helper:

```go
// ExportKeyBytes decodes EXPORT_KEY. api and worker call it; the bot never needs it.
func (c Config) ExportKeyBytes() ([]byte, error) {
	k, err := base64.StdEncoding.DecodeString(c.ExportKey)
	if err != nil || len(k) != 32 {
		return nil, errors.New("config: EXPORT_KEY must be base64 of 32 bytes (openssl rand -base64 32)")
	}
	return k, nil
}
```

Write a test: a valid key decodes; an empty key and a short key fail.

**Step 2: Shared constructor.** In `cmd/bragdoc/worker.go`, add `newExports(cfg, db, cache)`, which both commands call:

```go
package main

import (
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/files"
	"github.com/xhamps/bragdocument/backend/internal/adapters/gotenberg"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/config"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

const workerPoll = 2 * time.Second

func newExports(cfg config.Config, db *postgres.DB, cache ports.Cache) (*app.Exports, error) {
	key, err := cfg.ExportKeyBytes()
	if err != nil {
		return nil, err
	}
	store, err := files.New(cfg.ExportDir, key)
	if err != nil {
		return nil, err
	}
	return app.NewExports(postgres.NewDocumentRepo(db), postgres.NewLogRepo(db), postgres.NewExportRepo(db),
		gotenberg.New(cfg.GotenbergURL, cfg.GotenbergTimeout), store, cache, telemetry.WithTenantID), nil
}

// workerCmd runs PDF exports (PRD-0006, ADR-0010): claim, render, store; expire old files.
func workerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run the export worker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := boot()
			if err != nil {
				return err
			}
			db, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBTimeout)
			if err != nil {
				return err
			}
			defer db.Close()
			rc, err := redis.Connect(ctx, cfg.RedisURL, cfg.CacheTimeout)
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()
			reg := telemetry.NewRegistry() // ponytail: not served; the worker has no /metrics endpoint yet
			uc, err := newExports(cfg, db, redis.NewDegrading(rc, reg))
			if err != nil {
				return err
			}

			slog.InfoContext(ctx, "worker started")
			tick := time.NewTicker(workerPoll)
			defer tick.Stop()
			for {
				if err := uc.Cleanup(ctx); err != nil && ctx.Err() == nil {
					slog.WarnContext(ctx, "export cleanup failed", slog.Any("err", err))
				}
				// Drain the queue, then wait. A job cut by shutdown stays running and is reclaimed after 5 minutes.
				for ctx.Err() == nil {
					ran, err := uc.RunNext(ctx)
					if err != nil && ctx.Err() == nil {
						slog.ErrorContext(ctx, "export worker", slog.Any("err", err))
					}
					if !ran || err != nil {
						break
					}
				}
				select {
				case <-ctx.Done():
					slog.InfoContext(ctx, "worker stopped")
					return nil
				case <-tick.C:
				}
			}
		},
	}
}
```

In `main.go`, replace `daemonCmd("worker")` with `workerCmd()` and delete `daemon.go`.

**Step 3: API.** In `api.go`, after `logs := ...`:

```go
			exports, err := newExports(cfg, db, cache)
			if err != nil {
				return err
			}
			httpadapter.RegisterExports(authed, exports)
```

**Step 4: Compose.** In `docker-compose.yml`:
- api: add `EXPORT_DIR: /data/exports` under environment, plus `volumes: [exports:/data/exports]`.
- worker: add `EXPORT_DIR: /data/exports`. Add `migrate: condition: service_completed_successfully` to its `depends_on`, because it reads `export_jobs`.

`EXPORT_KEY` comes from `.env` through `env_file`.

**Step 5: `.env.example`.** After the Gotenberg block, add:

```
# PDF export (PRD-0006). Files are AES-256-GCM encrypted with this key and deleted after 24 h.
# Generate: openssl rand -base64 32. Required by the api and the worker.
EXPORT_KEY=
# EXPORT_DIR=data/exports
# GOTENBERG_TIMEOUT=60s
```

Also add `backend/data/` to `.gitignore`.

**Step 6:** Run `go build ./... && go test ./... && make lint`. Expected: PASS. Then smoke test:

```bash
export EXPORT_KEY=$(openssl rand -base64 32)
docker compose up -d postgres redis gotenberg && make migrate && go run ./cmd/bragdoc worker
```

Expected: `worker started`, then nothing more. Stop it with Ctrl-C and expect `worker stopped`.

**Step 7: Commit**

```bash
git add backend docker-compose.yml .env.example .gitignore
git commit -m "feat(worker): export worker loop; wire exports into the api and compose"
```

---

### Task 12: OpenAPI

**Files:** `backend/api/openapi.yaml`

Add the four paths under `/documents/{id}`: `report-settings` (GET), `exports` (GET, POST), `exports/{jobId}` (GET), and `exports/{jobId}/file` (GET, `application/pdf`). Add the schemas `ReportSettings`, `CreateExportRequest`, `ExportJob`, and `ExportList`, following the style of the `/documents/{id}/dashboard` entry. Document these points:
- POST returns 202.
- 422 `fields.filters` when nothing matches or more than 2,000 logs match (FR-7).
- The file returns 404 once expired.
- `query` is the logs list's query string.

Validate the YAML with `npx @redocly/cli lint backend/api/openapi.yaml` if that worked for earlier features. Otherwise check that it parses, for example with `python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' backend/api/openapi.yaml`.

```bash
git add backend/api/openapi.yaml
git commit -m "docs(api): export and report-settings endpoints"
```

---

### Task 13: Frontend data layer

**Files:**
- Modify: `frontend/packages/app/src/lib/types.ts`, `frontend/packages/app/src/lib/api.ts`
- Create: `frontend/packages/app/src/exports/useExports.ts`, `frontend/packages/app/src/exports/sections.ts`
- Test: `frontend/packages/app/src/lib/api.test.ts` (extend)

**Step 1: Types** (`types.ts`)

```ts
export type ExportStatus = "queued" | "running" | "done" | "failed";
export type ExportJob = {
  id: string;
  status: ExportStatus;
  progress: number;
  error: string;
  created_at: string;
  expires_at: string;
  downloadable: boolean;
};
export type ReportSettings = {
  goals_this_year: string;
  goals_next_year: string;
  section_map: Record<string, string>;
};
```

**Step 2: Download helper** (`api.ts`). Factor the header building into `authHeaders()` and reuse it in `api()`:

```ts
async function authHeaders(init?: HeadersInit) {
  const { data } = await supabase.auth.getSession();
  const headers = new Headers(init);
  const token = data.session?.access_token;
  if (token) headers.set("Authorization", `Bearer ${token}`);
  return headers;
}

/** Saves an authenticated binary response: a plain link cannot carry the bearer token. */
export async function download(path: string, filename: string) {
  const res = await fetch(`${env.apiUrl}${path}`, { headers: await authHeaders() });
  if (!res.ok) throw new ApiError(res.status, res.statusText);
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
```

Add a test in `api.test.ts`. Stub `fetch` to return `new Response("pdf")`. Stub `URL.createObjectURL` (it's missing in jsdom) with `vi.fn(() => "blob:x")`, and `URL.revokeObjectURL` with `vi.fn()`. Spy on `HTMLAnchorElement.prototype.click`. Call `download("/documents/d1/exports/j1/file", "r.pdf")`. Assert the Authorization header was sent and that `click` was called.

**Step 3: Sections** (`sections.ts`). It mirrors `domain.ReportSections`, so add a sync comment like the one on `logs/constants.ts`:

```ts
/** Mirrors backend domain.ReportSections (PRD-0006 FR-2). */
export const SECTIONS = [
  "Projects",
  "Collaboration & mentorship",
  "Design & documentation",
  "Company building",
  "What you learned",
  "Outside of work",
  "Other",
] as const;
```

**Step 4: Hooks** (`useExports.ts`)

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { ExportJob, ReportSettings } from "../lib/types";

export function useReportSettings(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["report-settings", docId],
    queryFn: () => api<ReportSettings>(`/documents/${docId}/report-settings`),
    enabled,
  });
}

export function useExportHistory(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["exports", docId],
    queryFn: () => api<{ items: ExportJob[] }>(`/documents/${docId}/exports`),
    select: (d) => d.items,
    enabled,
  });
}

const settled = (j?: ExportJob) => j?.status === "done" || j?.status === "failed";

/** Polls one job every second until it is done or failed (FR-5). */
export function useExportJob(docId: string, jobId: string | null) {
  const qc = useQueryClient();
  return useQuery({
    queryKey: ["exports", docId, jobId],
    queryFn: async () => {
      const j = await api<ExportJob>(`/documents/${docId}/exports/${jobId}`);
      if (settled(j)) void qc.invalidateQueries({ queryKey: ["exports", docId], exact: true });
      return j;
    },
    enabled: !!jobId,
    refetchInterval: (q) => (settled(q.state.data) ? false : 1000),
  });
}

export function useCreateExport(docId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ReportSettings & { query: string }) =>
      api<ExportJob>(`/documents/${docId}/exports`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["report-settings", docId] });
      void qc.invalidateQueries({ queryKey: ["exports", docId], exact: true });
    },
  });
}
```

**Step 5:** Run `npm run typecheck -w @bragdoc/app && npm test -w @bragdoc/app`. Expected: PASS.

**Step 6: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): export types, hooks, and authenticated download"
```

---

### Task 14: Export button, dialog, and status notice

**Files:**
- Create: `frontend/packages/app/src/exports/ExportButton.tsx`, `frontend/packages/app/src/exports/ExportDialog.tsx`
- Modify: `frontend/packages/app/src/routes/DocumentLogs.tsx`, `frontend/packages/app/src/routes/Dashboard.tsx`
- Test: `frontend/packages/app/src/routes/Export.test.tsx`

**Step 1: Write the failing tests** (`Export.test.tsx`). Use `mockFetch`/`renderAt` from `test/mocks`, and the `doc` fixture shape from `Dashboard.test.tsx`.

```tsx
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Document, ExportJob } from "../lib/types";

const doc: Document = {
  id: "d1", owner_id: "u1", title: "2026", description: "", state: "active",
  created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-02T00:00:00Z",
  log_count: 4, last_log_at: null, role: "viewer", owner_name: "Bob", is_new: false,
};
const job = (over: Partial<ExportJob> = {}): ExportJob => ({
  id: "j1", status: "queued", progress: 0, error: "",
  created_at: "2026-10-09T10:00:00Z", expires_at: "2026-10-10T10:00:00Z", downloadable: false, ...over,
});
const base = {
  "GET /me": me,
  "GET /documents/d1": doc,
  "GET /documents/d1/logs": { items: [], total: 0 },
  "GET /tags": { tags: ["project", "misc"] },
  "GET /documents/d1/report-settings": { goals_this_year: "ship", goals_next_year: "", section_map: { project: "Projects" } },
  "GET /documents/d1/exports": { items: [] },
};

test("export carries the list's filters, goals, and mapping", async () => {
  const calls = mockFetch({ ...base, "POST /documents/d1/exports": { status: 202, body: job() } });
  renderAt("/documents/d1?impact=high&status=done&page=2");
  await userEvent.click(await screen.findByRole("button", { name: /export pdf/i }));
  const dialog = await screen.findByRole("dialog");
  expect(within(dialog).getByLabelText(/goals for this year/i)).toHaveValue("ship");
  expect(within(dialog).getByLabelText("Section for misc")).toHaveValue("Other");
  await userEvent.selectOptions(within(dialog).getByLabelText("Section for misc"), "Company building");
  await userEvent.click(within(dialog).getByRole("button", { name: /generate/i }));
  const post = calls.find((c) => c.method === "POST");
  expect(post?.body).toMatchObject({
    query: "impact=high&status=done",
    goals_this_year: "ship",
    section_map: { project: "Projects", misc: "Company building" },
  });
});

test("too many logs shows the narrowing message", async () => {
  mockFetch({ ...base, "POST /documents/d1/exports": { status: 422, body: { message: "validation failed",
    fields: { filters: "2400 logs match; narrow the filters to at most 2000" } } } });
  renderAt("/documents/d1");
  await userEvent.click(await screen.findByRole("button", { name: /export pdf/i }));
  await userEvent.click(within(await screen.findByRole("dialog")).getByRole("button", { name: /generate/i }));
  expect(await screen.findByText(/narrow the filters/i)).toBeInTheDocument();
});

test("polls to done and offers the download outside the dialog", async () => {
  let n = 0;
  mockFetch({
    ...base,
    "POST /documents/d1/exports": { status: 202, body: job() },
    "GET /documents/d1/exports/j1": () => (++n < 2 ? job({ status: "running", progress: 40 })
      : job({ status: "done", progress: 100, downloadable: true })),
  });
  renderAt("/documents/d1");
  await userEvent.click(await screen.findByRole("button", { name: /export pdf/i }));
  await userEvent.click(within(await screen.findByRole("dialog")).getByRole("button", { name: /generate/i }));
  await userEvent.keyboard("{Escape}");
  const status = await screen.findByRole("status", {}, { timeout: 3000 });
  expect(await within(status).findByRole("button", { name: /download/i }, { timeout: 3000 })).toBeInTheDocument();
});

test("dashboard export carries the period", async () => {
  const calls = mockFetch({
    ...base,
    "GET /documents/d1/dashboard": { from: "2026-01-01", to: "2026-03-31", total: 0, in_period: 0, high_impact: 0,
      in_progress: 0, months: [], tags: [], statuses: [], impacts: [], coverage: [] },
    "POST /documents/d1/exports": { status: 202, body: job() },
  });
  renderAt("/documents/d1/dashboard?period=custom&from=2026-01-01&to=2026-03-31");
  await userEvent.click(await screen.findByRole("button", { name: /export pdf/i }));
  await userEvent.click(within(await screen.findByRole("dialog")).getByRole("button", { name: /generate/i }));
  expect(calls.find((c) => c.method === "POST")?.body).toMatchObject({ query: "from=2026-01-01&to=2026-03-31" });
});
```

**Step 2:** Run `npm test -w @bragdoc/app -- Export`. Expected: FAIL.

**Step 3: `ExportDialog.tsx`**

```tsx
import { useState } from "react";
import {
  Button, Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, Label,
} from "@bragdoc/ui";
import { ApiError, download } from "../lib/api";
import type { ExportJob, ReportSettings } from "../lib/types";
import { FIELD } from "../logs/constants";
import { useTags } from "../logs/useLogs";
import { SECTIONS } from "./sections";
import { useCreateExport, useExportHistory, useReportSettings } from "./useExports";

/** Filters only: paging, sorting, and the dashboard's preset name do not change the report. */
export function exportQuery(params: URLSearchParams) {
  const q = new URLSearchParams(params);
  for (const k of ["page", "per_page", "sort", "period", "edit"]) q.delete(k);
  return q.toString();
}

export function downloadJob(docId: string, j: ExportJob) {
  return download(`/documents/${docId}/exports/${j.id}/file`, `brag-report-${j.created_at.slice(0, 10)}.pdf`);
}

export function ExportDialog({
  docId, params, open, onOpenChange, onStarted,
}: {
  docId: string;
  params: URLSearchParams;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onStarted: (j: ExportJob) => void;
}) {
  const settings = useReportSettings(docId, open);
  const history = useExportHistory(docId, open);
  const tags = useTags();
  const create = useCreateExport(docId);
  const query = exportQuery(params);
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Export PDF</DialogTitle>
          <DialogDescription>
            {query ? `Filters: ${decodeURIComponent(query).replaceAll("&", " · ")}` : "All logs"}. Example logs are left out.
          </DialogDescription>
        </DialogHeader>
        {settings.data && (
          <SettingsForm
            key={docId}
            initial={settings.data}
            tags={tags.data ?? []}
            pending={create.isPending}
            error={create.error instanceof ApiError ? (create.error.fields?.filters ?? create.error.message) : undefined}
            onSubmit={(s) =>
              create.mutate({ ...s, query }, { onSuccess: (j) => { onStarted(j); onOpenChange(false); } })
            }
          />
        )}
        {!!history.data?.length && (
          <section aria-label="Recent exports" className="flex flex-col gap-1 text-sm">
            <h3 className="font-medium">Recent exports</h3>
            {history.data.map((j) => (
              <div key={j.id} className="flex items-center gap-2">
                <span>{new Date(j.created_at).toLocaleString()}</span>
                <span className="text-muted-foreground">{j.status === "failed" ? j.error : j.status}</span>
                {j.downloadable && (
                  <Button size="sm" variant="link" onClick={() => void downloadJob(docId, j)}>Download</Button>
                )}
              </div>
            ))}
          </section>
        )}
      </DialogContent>
    </Dialog>
  );
}

function SettingsForm({
  initial, tags, pending, error, onSubmit,
}: {
  initial: ReportSettings;
  tags: string[];
  pending: boolean;
  error?: string;
  onSubmit: (s: ReportSettings) => void;
}) {
  const [goalsThis, setGoalsThis] = useState(initial.goals_this_year);
  const [goalsNext, setGoalsNext] = useState(initial.goals_next_year);
  const [map, setMap] = useState(initial.section_map);
  const rows = [...new Set([...Object.keys(map), ...tags])].sort();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ goals_this_year: goalsThis, goals_next_year: goalsNext, section_map: map });
      }}
    >
      <Label className="flex flex-col items-start gap-1">
        Goals for this year
        <textarea className={`${FIELD} min-h-20 w-full`} value={goalsThis} onChange={(e) => setGoalsThis(e.target.value)} />
      </Label>
      <Label className="flex flex-col items-start gap-1">
        Goals for next year
        <textarea className={`${FIELD} min-h-20 w-full`} value={goalsNext} onChange={(e) => setGoalsNext(e.target.value)} />
      </Label>
      <table className="text-sm">
        <caption className="text-left font-medium">Sections by tag (untagged and unmapped logs go to Other)</caption>
        <tbody>
          {rows.map((t) => (
            <tr key={t}>
              <th scope="row" className="py-1 pr-4 text-left font-normal">{t}</th>
              <td>
                <select
                  aria-label={`Section for ${t}`}
                  className={FIELD}
                  value={map[t] ?? "Other"}
                  onChange={(e) => setMap({ ...map, [t]: e.target.value })}
                >
                  {SECTIONS.map((s) => <option key={s}>{s}</option>)}
                </select>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <DialogFooter>
        <Button type="submit" disabled={pending}>{pending ? "Starting…" : "Generate"}</Button>
      </DialogFooter>
    </form>
  );
}
```

**Step 4: `ExportButton.tsx`**. The notice lives outside the dialog so it survives closing it:

```tsx
import { useState } from "react";
import { Button } from "@bragdoc/ui";
import type { ExportJob } from "../lib/types";
import { ExportDialog, downloadJob } from "./ExportDialog";
import { useExportJob } from "./useExports";

/** "Export PDF" for the document header (FR-1) plus a live status of the last export. */
export function ExportButton({ docId, params }: { docId: string; params: URLSearchParams }) {
  const [open, setOpen] = useState(false);
  const [jobId, setJobId] = useState<string | null>(null);
  const job = useExportJob(docId, jobId).data;
  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>Export PDF</Button>
      <ExportDialog docId={docId} params={params} open={open} onOpenChange={setOpen}
        onStarted={(j: ExportJob) => setJobId(j.id)} />
      {job && (
        <span role="status" className="flex items-center gap-2 text-sm">
          {job.status === "done" ? (
            <>
              Report ready
              <Button size="sm" variant="link" onClick={() => void downloadJob(docId, job)}>Download</Button>
            </>
          ) : job.status === "failed" ? (
            <span className="text-destructive">{job.error}</span>
          ) : (
            <>
              Generating report
              <progress max={100} value={job.progress} aria-label="Report progress" />
            </>
          )}
        </span>
      )}
    </>
  );
}
```

**Step 5: Wire it in.**
- In `DocumentLogs.tsx`, inside the `ml-auto` button group, add `<ExportButton docId={doc.id} params={params} />` before the Share button. It's visible to every role.
- In `Dashboard.tsx`, add `<ExportButton docId={id} params={params} />` inside the `ml-auto` group, using the page's `useSearchParams` value. When the period comes from a preset rather than the URL, build the params from the active `from`/`to` range: `new URLSearchParams({ from, to })`.

**Step 6:** Run `npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npx eslint packages/app/src && npx prettier --write packages/app/src && npm run build -w @bragdoc/app`. Expected: PASS.

**Step 7: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): export dialog with goals, tag mapping, history, and progress"
```

---

### Task 15: Verify end to end and open the PR

**Step 1: All checks.** Each must pass, with the output read:

```bash
cd backend && go test ./... && go test -tags integration ./internal/adapters/... && make lint
cd ../frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npx eslint packages/app/src && npx prettier --check packages/app/src && npm run build -w @bragdoc/app
```

**Step 2: Manual run.** Start `docker compose --profile app up --build` with `EXPORT_KEY` set in `.env`. Then:
- Open a document, apply filters, export, and watch the progress.
- Download the PDF. Check the cover, page 2 (goals and summary), the sections, and that the links are clickable.
- Run `pdfinfo` on it and expect `Tagged: yes`.
- Run `xxd` on the file in the `exports` volume and confirm there's no `%PDF` header, i.e. it's encrypted.

**Step 3: `docs/README.md`.** Add the plan and design to the plans list if the README indexes plans.

**Step 4: Push and open the PR**

```bash
git push -u origin feat/pdf-report
gh pr create --base main --title "feat: PDF report export (PRD-0006)" --body "$(cat <<'EOF'
## Summary
- Async PDF export of a document (PRD-0006): cover, goals, summary table (page 2), the article's sections by tag, clickable links.
- `export_jobs` queue in PostgreSQL claimed by the `worker` with `FOR UPDATE SKIP LOCKED`; progress in Redis via `Degrading`.
- Gotenberg renders an embedded `html/template` with `generateDocumentOutline` (tagged PDF, NFR-3).
- Files AES-256-GCM encrypted (`EXPORT_KEY`), stored per tenant, deleted after 24 h (NFR-2).
- Access re-checked when the job runs and on download (FR-6); ≤ 2,000 logs (FR-7).
- Export dialog on the logs list and the dashboard, carrying the active filters (FR-1), with goals and an editable tag→section mapping saved per document (FR-4).

Implements PRD-0006 under ADR-0010 (amended). Design: `docs/plans/2026-10-09-pdf-report-design.md`.

## Test plan
- [ ] `go test ./...`, `go test -tags integration ./internal/adapters/...`, `make lint`
- [ ] Frontend tests, typecheck, eslint, prettier, build
- [ ] Manual: export with filters, download, `pdfinfo` shows Tagged: yes, stored file is encrypted

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
