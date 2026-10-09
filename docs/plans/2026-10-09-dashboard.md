# Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A per-document dashboard (PRD-0005): stat tiles, logs per month, top tags, status, impact, and coverage of the article's sections for a chosen period; every number links to the matching filtered logs list.

**Architecture:** `GET /documents/:id/dashboard?from&to` runs `access(PermRead)`, then a Redis cache-aside lookup keyed on a per-document version, then two SQL queries (totals with `FILTER`, one `UNION ALL` of `GROUP BY`s). `domain.Dashboard.Normalize` fills zero buckets and picks the top 10 tags. Log writes bump the version key. The frontend draws bars with Recharts and offers a table of real links.

**Tech Stack:** Go 1.27, Gin, pgx/sqlc, PostgreSQL, Redis via `ports.Cache` (`Degrading`), React 19, react-query 5, Recharts 3, Vitest.

**Design:** `docs/plans/2026-10-09-dashboard-design.md`. **Branch:** `feat/dashboard` (already created from `main`). Read `.claude/skills/backend-endpoint/SKILL.md` before backend tasks.

**Commands** (backend from `backend/`, frontend from `frontend/`):
- Unit: `go test ./...` · Integration (Docker): `go test -tags integration ./internal/adapters/postgres/...` · Lint: `make lint` · sqlc: `make sqlc`
- Frontend: `npm test -w @bragdoc/app` · `npm run typecheck -w @bragdoc/app` · `npx eslint packages/app/src` · `npx prettier --check packages/app/src` · `npm run build -w @bragdoc/app`

**Commit trailer:** end every commit message with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

---

### Task 1: Docs: accept PRD-0005

**Files:** `docs/prd/0005-dashboard.md`, `docs/README.md`

**Step 1:** In PRD-0005 set `status: accepted` and append to §13:

```markdown
| 2026-10-09 | Example logs are excluded; the logs list gains `examples=false` so chart links match (FR-6) | Starter examples would distort the shape of the year |
| 2026-10-09 | Everything except "Total" follows the period; default last 12 months | Reviews look at a window; Total anchors the whole document |
| 2026-10-09 | Aggregates cached 60 s in Redis behind a per-document version key bumped on every log write | NFR-1 and ADR-0006; access is checked before the cache, so revocation stays immediate |
| 2026-10-09 | Recharts with a table view of links for every chart | Charts that click through (FR-3) and an accessible alternative (NFR-2) |
```

**Step 2:** In `docs/README.md` set PRD 0005 to `accepted`.

**Step 3: Commit**

```bash
git add docs
git commit -m "docs: accept PRD-0005 with dashboard decisions"
```

---

### Task 2: Domain: period, dashboard, examples filter

**Files:**
- Create: `backend/internal/domain/dashboard.go`, `backend/internal/domain/dashboard_test.go`
- Modify: `backend/internal/domain/log_filter.go`

**Step 1: Failing test** `backend/internal/domain/dashboard_test.go`:

```go
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
```

**Step 2: Run, expect FAIL** — `cd backend && go test ./internal/domain/ -run 'Period|Normalize'` → `undefined: NewPeriod`.

**Step 3: Implement** `backend/internal/domain/dashboard.go`:

```go
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
	end := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, 1)
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
	From, To                               time.Time
	Total, InPeriod, HighImpact, InProgress int
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
```

**Step 4: Examples filter.** In `log_filter.go` add to `LogFilter` after `To`:

```go
	HideExamples bool // examples=false: matches the dashboard's numbers (PRD-0005 FR-6)
```

**Step 5: Run** `go test ./internal/domain/` → PASS. **Commit:**

```bash
git add backend/internal/domain
git commit -m "feat(domain): dashboard period, buckets, and normalization"
```

---

### Task 3: Queries and repository

**Files:**
- Modify: `backend/queries/logs.sql`, `backend/internal/ports/logs.go`, `backend/internal/adapters/postgres/log_repo.go`, `backend/internal/app/fakes_test.go`
- Create: `backend/internal/adapters/postgres/dashboard_integration_test.go`
- Regenerate: sqlcgen

**Step 1: SQL.** In `ListLogs`, add after the `to_at` line:

```sql
  AND (NOT @hide_examples::bool OR NOT l.is_example)
```

Append:

```sql
-- name: DashboardTotals :one
-- PRD-0005: examples never count. Total is all-time, the rest is [from, to).
SELECT count(*)::int AS total,
       count(*) FILTER (WHERE created_at >= @from_at AND created_at < @to_at)::int AS in_period,
       count(*) FILTER (WHERE created_at >= @from_at AND created_at < @to_at
                         AND impact IN ('high', 'critical'))::int AS high_impact,
       count(*) FILTER (WHERE created_at >= @from_at AND created_at < @to_at
                         AND status = 'in_progress')::int AS in_progress
FROM logs
WHERE document_id = @document_id AND NOT is_example;

-- name: DashboardBuckets :many
-- Non-zero counts only; domain.Dashboard.Normalize fills the gaps. Months are UTC.
WITH p AS (
    SELECT id, created_at, status, impact FROM logs
    WHERE document_id = @document_id AND NOT is_example
      AND created_at >= @from_at AND created_at < @to_at
)
SELECT 'month'::text AS kind, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM')::text AS key, count(*)::int AS count
FROM p GROUP BY 2
UNION ALL
SELECT 'status', status, count(*)::int FROM p GROUP BY 2
UNION ALL
SELECT 'impact', impact, count(*)::int FROM p GROUP BY 2
UNION ALL
SELECT 'tag', t.tag_name, count(*)::int FROM p JOIN log_tags t ON t.log_id = p.id GROUP BY 2;
```

Run `cd backend && make sqlc`. Expect `ListLogsParams.HideExamples bool`, `DashboardTotalsParams{DocumentID uuid.UUID; FromAt, ToAt time.Time}` with row `{Total, InPeriod, HighImpact, InProgress int32}`, and `DashboardBucketsRow{Kind, Key string; Count int32}`. If sqlc infers other types, fix with casts in SQL.

**Step 2: Port.** In `ports/logs.go` add to `LogRepo`:

```go
	// Dashboard returns sparse aggregates of the document's non-example logs
	// for p; the caller runs Normalize.
	Dashboard(ctx context.Context, documentID string, p domain.Period) (domain.Dashboard, error)
```

**Step 3: Keep app compiling.** In `app/fakes_test.go` add a `dashCalls int` field to `fakeLogs` and:

```go
func (f *fakeLogs) Dashboard(_ context.Context, documentID string, p domain.Period) (domain.Dashboard, error) {
	f.dashCalls++
	d := domain.Dashboard{From: p.From, To: p.To}
	for _, l := range f.logs {
		if l.DocumentID == documentID && !l.IsExample {
			d.Total++
		}
	}
	return d, nil
}
```

**Step 4: Failing integration test** `backend/internal/adapters/postgres/dashboard_integration_test.go`:

```go
//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestDashboard(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db)
	admin, ta := provisionTenant(t, users, "A", "a@example.com")
	_, tb := provisionTenant(t, users, "B", "b@example.com")
	ctx := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)

	examples := domain.ExampleLogs(time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC), admin.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: admin.ID, Title: "2026"}, examples)
	require.NoError(t, err)

	mk := func(name, impact, status string, at time.Time, tags ...string) {
		l := domain.Log{TenantID: ta.ID, DocumentID: doc.ID, Name: name, Impact: impact, Status: status,
			Tags: tags, Links: []domain.Link{}, CreatedAt: at, CreatedBy: admin.ID, UpdatedBy: admin.ID}
		require.NoError(t, l.Validate())
		_, err := logs.Create(ctx, l)
		require.NoError(t, err)
	}
	at := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 12, 0, 0, 0, time.UTC) }
	mk("a", "high", "done", at(1, 15), "project", "mentorship")
	mk("b", "critical", "in_progress", at(3, 2), "project")
	mk("c", "low", "idea", at(3, 31), "learning")
	mk("d", "medium", "done", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "project") // before the period
	mk("e", "high", "in_progress", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))      // the exclusive end

	p := domain.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)}
	d, err := logs.Dashboard(ctx, doc.ID, p)
	require.NoError(t, err)
	require.Equal(t, 5, d.Total, "all-time, examples excluded")
	require.Equal(t, 3, d.InPeriod)
	require.Equal(t, 2, d.HighImpact)
	require.Equal(t, 1, d.InProgress)

	d.Normalize()
	require.Equal(t, []domain.Bucket{{"2026-01", 1}, {"2026-02", 0}, {"2026-03", 2}}, d.Months, "the example in February does not count")
	require.Equal(t, []domain.Bucket{{"project", 2}, {"learning", 1}, {"mentorship", 1}}, d.Tags)
	require.Equal(t, []domain.Bucket{{"idea", 1}, {"in_progress", 1}, {"done", 1}, {"dropped", 0}}, d.Statuses)
	require.Equal(t, []domain.Bucket{{"low", 1}, {"medium", 0}, {"high", 1}, {"critical", 1}}, d.Impacts)
	require.Equal(t, domain.Bucket{"mentorship", 1}, d.Coverage[2])

	// FR-6: every slice equals the list's total for the same filter.
	count := func(f domain.LogFilter) int {
		f.From, f.To, f.HideExamples = &p.From, &p.To, true
		require.NoError(t, f.Validate())
		page, err := logs.List(ctx, doc.ID, f)
		require.NoError(t, err)
		return page.Total
	}
	require.Equal(t, d.InPeriod, count(domain.LogFilter{}))
	require.Equal(t, d.Tags[0].Count, count(domain.LogFilter{Tags: []string{"project"}}))
	require.Equal(t, d.HighImpact, count(domain.LogFilter{Impacts: []string{"high", "critical"}}))
	require.Equal(t, d.InProgress, count(domain.LogFilter{Statuses: []string{"in_progress"}}))
	all := domain.LogFilter{}
	require.NoError(t, all.Validate())
	page, err := logs.List(ctx, doc.ID, all)
	require.NoError(t, err)
	require.Equal(t, 5+len(examples), page.Total, "without examples=false the list still shows examples")

	// Other tenants see nothing.
	d, err = logs.Dashboard(ctxB, doc.ID, p)
	require.NoError(t, err)
	require.Zero(t, d.Total)
}
```

Also add a timing test following `TestLogListTiming` in `logs_integration_test.go` (read it first and reuse its seeding approach for 10,000 logs): call `logs.Dashboard` for a 12-month period, `t.Logf` the duration, and `require.Less(t, elapsed, 2*time.Second)` (generous; NFR-1's 500 ms p95 includes the cache).

**Step 5: Run, expect FAIL** — `go vet -tags integration ./internal/adapters/postgres/` → `logs.Dashboard undefined`.

**Step 6: Implement.** In `log_repo.go` `listParams` set `HideExamples: f.HideExamples`. Add:

```go
// Dashboard returns the sparse aggregates for p in one transaction.
func (r *LogRepo) Dashboard(ctx context.Context, documentID string, p domain.Period) (domain.Dashboard, error) {
	did, err := parseID(documentID)
	if err != nil {
		return domain.Dashboard{}, err
	}
	d := domain.Dashboard{From: p.From, To: p.To}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		t, err := q.DashboardTotals(ctx, sqlcgen.DashboardTotalsParams{DocumentID: did, FromAt: p.From, ToAt: p.To})
		if err != nil {
			return wrap(err)
		}
		d.Total, d.InPeriod, d.HighImpact, d.InProgress = int(t.Total), int(t.InPeriod), int(t.HighImpact), int(t.InProgress)
		rows, err := q.DashboardBuckets(ctx, sqlcgen.DashboardBucketsParams{DocumentID: did, FromAt: p.From, ToAt: p.To})
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			b := domain.Bucket{Key: row.Key, Count: int(row.Count)}
			switch row.Kind {
			case "month":
				d.Months = append(d.Months, b)
			case "status":
				d.Statuses = append(d.Statuses, b)
			case "impact":
				d.Impacts = append(d.Impacts, b)
			case "tag":
				d.Tags = append(d.Tags, b)
			}
		}
		return nil
	})
	return d, err
}
```

**Step 7: Run** `go build ./... && go test ./... && go test -count=1 -tags integration ./internal/adapters/postgres/...` → PASS. **Commit:**

```bash
git add backend/queries backend/internal
git commit -m "feat(postgres): dashboard aggregates and the examples=false list filter"
```

---

### Task 4: App: cached dashboard, invalidation on writes

**Files:**
- Create: `backend/internal/app/log_dashboard.go`, `backend/internal/app/log_dashboard_test.go`
- Modify: `backend/internal/app/logs.go`, `log_create.go`, `log_update.go`, `log_delete.go`, `access_matrix_test.go`, `logs_test.go`, every other `NewLogs(` call (`grep -rn "NewLogs(" backend`), `backend/cmd/bragdoc/api.go`, `backend/cmd/bragdoc/bot.go`

**Step 1: Constructor.** `Logs` gains `cache ports.Cache`; `NewLogs(docs, logs, impact, cache ports.Cache)`. Update every caller: tests pass `newFakeCache()` (keep a handle where a test needs it, e.g. `logsFixture.cache`); `api.go` passes `cache` (the `redis.NewDegrading` value it already builds; move that line above `NewLogs` if needed); `bot.go` builds `redis.NewDegrading(rc, reg)` once into a variable and passes it to both `NewLogs` and `NewTelegram`.

**Step 2: Failing tests** `backend/internal/app/log_dashboard_test.go`:

```go
package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDashboardCachesUntilAWrite(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()

	d, err := f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Len(t, d.Months, 12, "normalized: the default period spans 12 months")
	_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, f.logs.dashCalls, "second read is a hit")

	_, err = f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	d, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, f.logs.dashCalls, "a write invalidates")
	require.Equal(t, 1, d.Total)

	_, err = f.s.Dashboard(ctx, "d3", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 3, f.logs.dashCalls, "other documents keep their own entries")
}

func TestDashboardInvalidatedByEveryWrite(t *testing.T) {
	ctx := context.Background()
	writes := map[string]func(f logsFixture, logID string) error{
		"update": func(f logsFixture, id string) error {
			n := "renamed"
			_, err := f.s.Update(ctx, UpdateLogInput{ID: id, DocumentID: "d1", UserID: "u1", Name: &n})
			return err
		},
		"delete":          func(f logsFixture, id string) error { return f.s.Delete(ctx, "d1", id, "u1") },
		"delete examples": func(f logsFixture, string) error { return f.s.DeleteExamples(ctx, "d1", "u1") },
	}
	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			f := newLogsFixture()
			l, err := f.s.Create(ctx, createIn("d1"))
			require.NoError(t, err)
			_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
			require.NoError(t, err)
			require.NoError(t, write(f, l.ID))
			_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
			require.NoError(t, err)
			require.Equal(t, 2, f.logs.dashCalls)
		})
	}
}

func TestDashboardWithoutCache(t *testing.T) {
	f := newLogsFixture()
	f.cache.err = errors.New("redis down")
	for range 2 {
		_, err := f.s.Dashboard(context.Background(), "d1", "u1", nil, nil)
		require.NoError(t, err)
	}
	require.Equal(t, 2, f.logs.dashCalls, "every read goes to Postgres")
}

func TestDashboardCorruptEntryIsAMiss(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()
	_, err := f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	for k := range f.cache.data {
		if strings.HasPrefix(k, "dash:") {
			f.cache.data[k] = []byte("{")
		}
	}
	_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, f.logs.dashCalls)
}

func TestDashboardRules(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()
	_, err := f.s.Dashboard(ctx, "d1", "u9", nil, nil)
	require.ErrorIs(t, err, domain.ErrNotFound, "no grant")
	_, err = f.s.Dashboard(ctx, "d1", "u2", nil, nil)
	require.NoError(t, err, "viewers read the dashboard (FR-5)")

	from, to := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err = f.s.Dashboard(ctx, "d1", "u1", &from, &to)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	require.Zero(t, f.logs.dashCalls, "access and validation run before any read")
}
```

(`newLogsFixture` grants `u2` viewer on `d1` since PRD-0004; add `cache *fakeCache` to `logsFixture`. Read `log_update.go` to confirm `UpdateLogInput` field names.)

Also add to the actions in `access_matrix_test.go`:

```go
		{"dashboard", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.Dashboard(ctx, "d1", u.ID, nil, nil)
			return err
		}},
```

**Step 3: Run, expect FAIL** — `go test ./internal/app/` → `f.s.Dashboard undefined`.

**Step 4: Implement.** In `logs.go` add:

```go
const (
	dashboardTTL  = 60 * time.Second // PRD-0005 NFR-1, ADR-0006
	docVersionTTL = 24 * time.Hour   // outlives every entry keyed on it
)

func docVersionKey(docID string) string { return "doc:" + docID + ":v" }

// touch invalidates the document's cached aggregates (ADR-0006): a fresh
// version makes every older entry unreachable. The cache is Degrading, so a
// failed write costs at most one TTL of staleness, never the request.
func (s *Logs) touch(ctx context.Context, docID string) {
	_ = s.cache.Set(ctx, docVersionKey(docID), []byte(rand.Text()), docVersionTTL)
}
```

(`crypto/rand`; `rand.Text` is Go 1.24+.) Call `s.touch(ctx, docID)` after each successful repository write:
- `log_create.go`: `out, err := s.logs.Create(ctx, l); if err == nil { s.touch(ctx, d.ID) }; return out, err`
- `log_update.go`: same around `s.logs.Update`, with `in.DocumentID`.
- `log_delete.go`: `Delete` and `DeleteExamples`, same pattern.

Create `backend/internal/app/log_dashboard.go`:

```go
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Dashboard aggregates the document's logs for a period (PRD-0005). Access is
// checked before the cache so a revoked reader is refused at once. Cache
// errors are misses: the numbers then come from Postgres.
func (s *Logs) Dashboard(ctx context.Context, docID, userID string, from, to *time.Time) (domain.Dashboard, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return domain.Dashboard{}, err
	}
	p, err := domain.NewPeriod(from, to, s.now())
	if err != nil {
		return domain.Dashboard{}, err
	}
	ver := "0"
	if v, ok, _ := s.cache.Get(ctx, docVersionKey(docID)); ok {
		ver = string(v)
	}
	key := fmt.Sprintf("dash:%s:v%s:%d:%d", docID, ver, p.From.Unix(), p.To.Unix())
	if b, ok, _ := s.cache.Get(ctx, key); ok {
		var d domain.Dashboard
		if json.Unmarshal(b, &d) == nil {
			return d, nil
		}
	}
	d, err := s.logs.Dashboard(ctx, docID, p)
	if err != nil {
		return domain.Dashboard{}, err
	}
	d.Normalize()
	if b, err := json.Marshal(d); err == nil {
		_ = s.cache.Set(ctx, key, b, dashboardTTL)
	}
	return d, nil
}
```

**Step 5: Run** `go build ./... && go test ./... && make lint` → PASS (matrix now 18 actions × 4). **Commit:**

```bash
git add backend/internal/app backend/cmd
git commit -m "feat(app): cached dashboard; log writes bump the document's cache version"
```

---

### Task 5: HTTP route and OpenAPI

**Files:**
- Modify: `backend/internal/adapters/http/logs_handler.go`, `logs_handler_test.go`, `backend/api/openapi.yaml`

**Step 1: Failing tests.** In `logs_handler_test.go` add to `fakeLogUC` fields `dashFrom, dashTo *time.Time` and:

```go
func (f *fakeLogUC) Dashboard(_ context.Context, docID, _ string, from, to *time.Time) (domain.Dashboard, error) {
	f.dashFrom, f.dashTo = from, to
	return domain.Dashboard{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Total: 7, InPeriod: 3, Months: []domain.Bucket{{Key: "2026-01", Count: 3}}}, f.err
}
```

and tests (use the file's existing engine helper; read the top of the file):

```go
func TestLogsDashboard(t *testing.T) {
	uc := &fakeLogUC{}
	rec := do(logsEngine(t, uc), http.MethodGet, "/documents/d1/dashboard?from=2026-01-01&to=2026-03-31", "")
	require.Equal(t, 200, rec.Code)
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
```

(Adjust `logsEngine` / `uc.filter` to the names the file actually uses.)

**Step 2: Run, expect FAIL** — `go test ./internal/adapters/http/`.

**Step 3: Implement** in `logs_handler.go`:
- Add `Dashboard(ctx context.Context, docID, userID string, from, to *time.Time) (domain.Dashboard, error)` to `LogUseCases`.
- Extract the date loop of `logFilter` into:

```go
// dateRange reads from/to as YYYY-MM-DD in UTC; "to" is inclusive in the URL
// and returned exclusive. Bad values are added to fields.
func dateRange(q url.Values, fields map[string]string) (from, to *time.Time) {
	for key, dst := range map[string]**time.Time{"from": &from, "to": &to} {
		if v := q.Get(key); v != "" {
			d, err := time.Parse(time.DateOnly, v)
			if err != nil {
				fields[key] = "must be YYYY-MM-DD"
				continue
			}
			if key == "to" {
				d = d.AddDate(0, 0, 1)
			}
			*dst = &d
		}
	}
	return from, to
}
```

  and use it in `logFilter` (`f.From, f.To = dateRange(q, fields)`); also set `f.HideExamples = q.Get("examples") == "false"`.
- DTOs and route:

```go
// BucketResponse is one bar of a dashboard chart.
type BucketResponse struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// DashboardResponse is PRD-0005's aggregate; dates are YYYY-MM-DD, "to" inclusive.
type DashboardResponse struct {
	From       string           `json:"from"`
	To         string           `json:"to"`
	Total      int              `json:"total"`
	InPeriod   int              `json:"in_period"`
	HighImpact int              `json:"high_impact"`
	InProgress int              `json:"in_progress"`
	Months     []BucketResponse `json:"months"`
	Tags       []BucketResponse `json:"tags"`
	Statuses   []BucketResponse `json:"statuses"`
	Impacts    []BucketResponse `json:"impacts"`
	Coverage   []BucketResponse `json:"coverage"`
}

func toBuckets(bs []domain.Bucket) []BucketResponse {
	out := make([]BucketResponse, 0, len(bs))
	for _, b := range bs {
		out = append(out, BucketResponse{Key: b.Key, Count: b.Count})
	}
	return out
}
```

```go
	g.GET("/dashboard", func(c *gin.Context) {
		fields := map[string]string{}
		from, to := dateRange(c.Request.URL.Query(), fields)
		if len(fields) > 0 {
			RespondError(c, domain.NewValidationError(fields))
			return
		}
		d, err := uc.Dashboard(c.Request.Context(), c.Param("id"), principal(c).User.ID, from, to)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, DashboardResponse{
			From: d.From.Format(time.DateOnly), To: d.To.AddDate(0, 0, -1).Format(time.DateOnly),
			Total: d.Total, InPeriod: d.InPeriod, HighImpact: d.HighImpact, InProgress: d.InProgress,
			Months: toBuckets(d.Months), Tags: toBuckets(d.Tags), Statuses: toBuckets(d.Statuses),
			Impacts: toBuckets(d.Impacts), Coverage: toBuckets(d.Coverage),
		})
	})
```

Update the `RegisterLogs` doc comment to mention `/documents/:id/dashboard`.

**Step 4: OpenAPI.** On `GET /documents/{id}/logs` add the query parameter:

```yaml
        - { name: examples, in: query, schema: { type: string, enum: ["false"] }, description: "false hides example logs (matches the dashboard, PRD-0005 FR-6)" }
```

Add the path and schemas:

```yaml
  /documents/{id}/dashboard:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
    get:
      summary: Log aggregates for a period, examples excluded (PRD-0005); read access
      parameters:
        - { name: from, in: query, schema: { type: string, format: date }, description: "UTC; default 12 months before to" }
        - { name: to, in: query, schema: { type: string, format: date }, description: "UTC, inclusive; default today" }
      responses:
        "200": { description: OK, content: { application/json: { schema: { $ref: "#/components/schemas/Dashboard" } } } }
        "404": { description: Not found, or no role on the document }
        "422": { description: Bad date, empty period, or longer than 5 years }
```

```yaml
    Bucket:
      type: object
      properties:
        key: { type: string, description: "Month (YYYY-MM), tag, status, or impact" }
        count: { type: integer }
    Dashboard:
      type: object
      properties:
        from: { type: string, format: date }
        to: { type: string, format: date, description: Inclusive }
        total: { type: integer, description: All-time }
        in_period: { type: integer }
        high_impact: { type: integer, description: high + critical in period }
        in_progress: { type: integer, description: status in_progress in period }
        months: { type: array, items: { $ref: "#/components/schemas/Bucket" }, description: Every month of the period, zeros included }
        tags: { type: array, items: { $ref: "#/components/schemas/Bucket" }, description: Top 10 by count }
        statuses: { type: array, items: { $ref: "#/components/schemas/Bucket" } }
        impacts: { type: array, items: { $ref: "#/components/schemas/Bucket" } }
        coverage: { type: array, items: { $ref: "#/components/schemas/Bucket" }, description: The article's suggested tags, zeros included }
```

Validate: `python3 -c "import yaml; yaml.safe_load(open('api/openapi.yaml'))"`.

**Step 5: Run** `go test ./... && make lint` → PASS. **Commit:**

```bash
git add backend/internal/adapters/http backend/api/openapi.yaml
git commit -m "feat(api): GET /documents/:id/dashboard and examples=false on the list"
```

---

### Task 6: Frontend: dashboard page

**Files:**
- Modify: `frontend/packages/app/package.json` (via npm), `src/lib/types.ts`, `src/router.tsx`, `src/routes/DocumentLogs.tsx`, `src/logs/LogFilters.tsx`, `src/test/setup.ts`
- Create: `src/documents/DocumentTabs.tsx`, `src/dashboard/periods.ts`, `src/dashboard/periods.test.ts`, `src/dashboard/useDashboard.ts`, `src/dashboard/BarList.tsx`, `src/routes/Dashboard.tsx`, `src/routes/Dashboard.test.tsx`

**Step 1: Dependency.** `cd frontend && npm install recharts@^3 react-is -w @bragdoc/app`.

**Step 2: jsdom.** Recharts' `ResponsiveContainer` needs `ResizeObserver`. Append to `src/test/setup.ts`:

```ts
// jsdom has no ResizeObserver; Recharts' ResponsiveContainer needs one.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
```

**Step 3: Failing unit tests** `src/dashboard/periods.test.ts`:

```ts
import { logsHref, monthRange, presetRange } from "./periods";

const today = new Date(Date.UTC(2026, 9, 9, 15)); // 2026-10-09

test("presets are inclusive UTC ranges", () => {
  expect(presetRange("30d", today)).toEqual({ from: "2026-09-10", to: "2026-10-09" });
  expect(presetRange("12m", today)).toEqual({ from: "2025-10-10", to: "2026-10-09" });
  expect(presetRange("year", today)).toEqual({ from: "2026-01-01", to: "2026-12-31" });
  expect(presetRange("quarter", today)).toEqual({ from: "2026-07-01", to: "2026-09-30" });
  expect(presetRange("quarter", new Date(Date.UTC(2026, 1, 3)))).toEqual({
    from: "2025-10-01",
    to: "2025-12-31",
  });
});

test("a month bar narrows to the month, clipped to the period", () => {
  expect(monthRange("2026-03", "2026-01-01", "2026-12-31")).toEqual({ from: "2026-03-01", to: "2026-03-31" });
  expect(monthRange("2026-03", "2026-03-15", "2026-12-31")).toEqual({ from: "2026-03-15", to: "2026-03-31" });
  expect(monthRange("2024-02", "2024-01-01", "2024-02-10")).toEqual({ from: "2024-02-01", to: "2024-02-10" });
});

test("links hide examples so counts match", () => {
  expect(logsHref("d1", [["tag", "project"], ["from", "2026-01-01"]])).toBe(
    "/documents/d1?tag=project&from=2026-01-01&examples=false",
  );
});
```

**Step 4: Implement** `src/dashboard/periods.ts`:

```ts
export type Preset = "30d" | "quarter" | "12m" | "year";
export type Range = { from: string; to: string };

export const PRESETS: { value: Preset | "custom"; label: string }[] = [
  { value: "30d", label: "Last 30 days" },
  { value: "quarter", label: "Last quarter" },
  { value: "12m", label: "Last 12 months" },
  { value: "year", label: "This year" },
  { value: "custom", label: "Custom range" },
];

const iso = (d: Date) => d.toISOString().slice(0, 10);
const utc = (y: number, m: number, d: number) => new Date(Date.UTC(y, m, d));

/** Inclusive YYYY-MM-DD range in UTC; the API's dates are UTC (PRD-0005 FR-2). */
export function presetRange(p: Preset, today = new Date()): Range {
  const y = today.getUTCFullYear();
  const m = today.getUTCMonth();
  const d = today.getUTCDate();
  switch (p) {
    case "30d":
      return { from: iso(utc(y, m, d - 29)), to: iso(utc(y, m, d)) };
    case "12m":
      return { from: iso(utc(y - 1, m, d + 1)), to: iso(utc(y, m, d)) };
    case "year":
      return { from: `${y}-01-01`, to: `${y}-12-31` };
    case "quarter": {
      const q = Math.floor(m / 3) * 3; // first month of this quarter
      return { from: iso(utc(y, q - 3, 1)), to: iso(utc(y, q, 0)) };
    }
  }
}

/** The days of month "YYYY-MM" that fall inside [from, to]. */
export function monthRange(key: string, from: string, to: string): Range {
  const [y, m] = key.split("-").map(Number);
  const start = iso(utc(y, m - 1, 1));
  const end = iso(utc(y, m, 0));
  return { from: start > from ? start : from, to: end < to ? end : to };
}

/** The logs list filtered to a dashboard slice; examples hidden like the dashboard (FR-3, FR-6). */
export function logsHref(docId: string, pairs: [string, string][]) {
  const p = new URLSearchParams(pairs);
  p.set("examples", "false");
  return `/documents/${docId}?${p}`;
}
```

Run `npm test -w @bragdoc/app -- periods` → PASS.

**Step 5: Types and hook.** In `types.ts`:

```ts
export type Bucket = { key: string; count: number };

export type Dashboard = {
  from: string;
  to: string;
  total: number;
  in_period: number;
  high_impact: number;
  in_progress: number;
  months: Bucket[];
  tags: Bucket[];
  statuses: Bucket[];
  impacts: Bucket[];
  coverage: Bucket[];
};
```

`src/dashboard/useDashboard.ts`:

```ts
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Dashboard } from "../lib/types";
import type { Range } from "./periods";

export function useDashboard(docId: string, { from, to }: Range) {
  return useQuery({
    queryKey: ["dashboard", docId, from, to],
    queryFn: () => api<Dashboard>(`/documents/${docId}/dashboard?from=${from}&to=${to}`),
    placeholderData: keepPreviousData,
  });
}
```

**Step 6: Tabs** `src/documents/DocumentTabs.tsx`:

```tsx
import { Link } from "react-router";

const base = "border-b-2 pb-1 text-sm";

/** Logs | Dashboard switch for one document. */
export function DocumentTabs({ id, current }: { id: string; current: "logs" | "dashboard" }) {
  const tab = (to: string, label: string, key: typeof current) => (
    <Link
      to={to}
      aria-current={current === key ? "page" : undefined}
      className={current === key ? `${base} border-foreground font-medium` : `${base} border-transparent text-muted-foreground hover:text-foreground`}
    >
      {label}
    </Link>
  );
  return (
    <nav aria-label="Document views" className="flex gap-4">
      {tab(`/documents/${id}`, "Logs", "logs")}
      {tab(`/documents/${id}/dashboard`, "Dashboard", "dashboard")}
    </nav>
  );
}
```

In `DocumentLogs.tsx` render `<DocumentTabs id={doc.id} current="logs" />` right after the `<h2>` title. In `router.tsx` add `{ path: "documents/:id/dashboard", lazy: () => import("./routes/Dashboard") }` after the `documents/:id` route.

**Step 7: Examples chip.** In `LogFilters.tsx` add `examples: "Examples"` to `LABELS`, and render its value as `hidden`: change the chip text to `{LABELS[k]}: {k === "status" ? STATUS_LABEL[v as LogStatus] : k === "examples" ? "hidden" : v}`.

**Step 8: Chart component** `src/dashboard/BarList.tsx`:

```tsx
import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Button } from "@bragdoc/ui";
import type { Bucket } from "../lib/types";

type Props = {
  title: string;
  buckets: Bucket[];
  hrefFor: (b: Bucket) => string;
  label?: (key: string) => string;
  /** columns: one bar per x value (months); rows: horizontal bars (tags, status, impact). */
  layout: "columns" | "rows";
  color?: string;
};

/** A bar chart whose bars link to the filtered logs (FR-3), with a table of links as its text alternative (NFR-2). */
export function BarList({ title, buckets, hrefFor, label = (k) => k, layout, color = "var(--chart-1)" }: Props) {
  const [table, setTable] = useState(false);
  const navigate = useNavigate();
  const data = buckets.map((b) => ({ ...b, name: label(b.key) }));
  const rows = layout === "rows";
  return (
    <section aria-label={title} className="flex flex-col gap-2 rounded-xl p-4 ring-1 ring-foreground/10">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">{title}</h3>
        <Button variant="link" size="sm" onClick={() => setTable(!table)}>
          {table ? "Show as chart" : "Show as table"}
        </Button>
      </div>
      {buckets.length === 0 ? (
        <p className="text-sm text-muted-foreground">No logs in this period.</p>
      ) : table ? (
        <table className="text-sm">
          <thead>
            <tr className="text-muted-foreground">
              <th className="text-left font-normal">{title}</th>
              <th className="text-right font-normal">Logs</th>
            </tr>
          </thead>
          <tbody>
            {data.map((b) => (
              <tr key={b.key}>
                <td>
                  <Link className="hover:underline" to={hrefFor(b)}>
                    {b.name}
                  </Link>
                </td>
                <td className="text-right tabular-nums">{b.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div style={{ height: rows ? Math.max(120, data.length * 28) : 220 }}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data} layout={rows ? "vertical" : "horizontal"} accessibilityLayer>
              <XAxis type={rows ? "number" : "category"} dataKey={rows ? undefined : "name"} hide={rows}
                allowDecimals={false} tickLine={false} axisLine={false} />
              <YAxis type={rows ? "category" : "number"} dataKey={rows ? "name" : undefined} width={rows ? 128 : 32}
                allowDecimals={false} tickLine={false} axisLine={false} />
              <Tooltip cursor={{ fill: "var(--muted)" }} />
              <Bar dataKey="count" name="Logs" fill={color} radius={4} cursor="pointer"
                onClick={(_, i) => navigate(hrefFor(buckets[i]))} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </section>
  );
}
```

(If Recharts 3's `onClick` typing differs, keep the behaviour: navigate to `hrefFor(buckets[index])`.)

**Step 9: Failing page tests** `src/routes/Dashboard.test.tsx`:

```tsx
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Dashboard, Document } from "../lib/types";

const doc: Document = {
  id: "d1", owner_id: "u2", title: "2026", description: "", state: "active",
  created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-02T00:00:00Z",
  log_count: 4, last_log_at: null, role: "viewer", owner_name: "Bob", is_new: false,
};

const dash: Dashboard = {
  from: "2026-01-01", to: "2026-03-31",
  total: 9, in_period: 4, high_impact: 2, in_progress: 1,
  months: [{ key: "2026-01", count: 1 }, { key: "2026-02", count: 0 }, { key: "2026-03", count: 3 }],
  tags: [{ key: "project", count: 3 }],
  statuses: [{ key: "idea", count: 0 }, { key: "in_progress", count: 1 }, { key: "done", count: 3 }, { key: "dropped", count: 0 }],
  impacts: [{ key: "low", count: 1 }, { key: "medium", count: 1 }, { key: "high", count: 1 }, { key: "critical", count: 1 }],
  coverage: [{ key: "project", count: 3 }, { key: "mentorship", count: 0 }],
};

const url = "/documents/d1/dashboard?period=custom&from=2026-01-01&to=2026-03-31";
const routes = { "GET /me": me, "GET /documents/d1": doc, "GET /documents/d1/dashboard": dash };
const q = "from=2026-01-01&to=2026-03-31&examples=false";

test("tiles show the numbers and link to the list", async () => {
  const calls = mockFetch(routes);
  renderAt(url);
  const total = await screen.findByRole("link", { name: /total logs\s*9/i });
  expect(total).toHaveAttribute("href", "/documents/d1?examples=false");
  expect(screen.getByRole("link", { name: /in period\s*4/i })).toHaveAttribute("href", `/documents/d1?${q}`);
  expect(screen.getByRole("link", { name: /high or critical\s*2/i })).toHaveAttribute(
    "href", `/documents/d1?impact=high&impact=critical&${q}`);
  expect(screen.getByRole("link", { name: /in progress\s*1/i })).toHaveAttribute(
    "href", `/documents/d1?status=in_progress&${q}`);
  expect(calls.find((c) => c.path === "/documents/d1/dashboard")?.search).toBe("?from=2026-01-01&to=2026-03-31");
});

test("table views are real links to each slice", async () => {
  mockFetch(routes);
  renderAt(url);
  const months = await screen.findByRole("region", { name: "Logs per month" });
  await userEvent.click(within(months).getByRole("button", { name: /show as table/i }));
  expect(within(months).getByRole("link", { name: "Mar 2026" })).toHaveAttribute(
    "href", "/documents/d1?from=2026-03-01&to=2026-03-31&examples=false");

  const tags = screen.getByRole("region", { name: "Top tags" });
  await userEvent.click(within(tags).getByRole("button", { name: /show as table/i }));
  expect(within(tags).getByRole("link", { name: "project" })).toHaveAttribute("href", `/documents/d1?tag=project&${q}`);
});

test("coverage highlights tags never used", async () => {
  mockFetch(routes);
  renderAt(url);
  const cov = await screen.findByRole("region", { name: /coverage/i });
  expect(within(cov).getByRole("link", { name: /mentorship\s*0/i })).toHaveAttribute("data-zero", "true");
  expect(within(cov).getByRole("link", { name: /project\s*3/i })).not.toHaveAttribute("data-zero");
});

test("period select drives the URL and the request", async () => {
  const calls = mockFetch(routes);
  const { router } = renderAt(url);
  await screen.findByRole("link", { name: /total logs/i });
  await userEvent.selectOptions(screen.getByLabelText(/period/i), "year");
  await vi.waitFor(() => expect(router.state.location.search).toBe("?period=year"));
  const y = new Date().getUTCFullYear();
  await vi.waitFor(() =>
    expect(calls.at(-1)?.search).toBe(`?from=${y}-01-01&to=${y}-12-31`),
  );
  expect(screen.queryByLabelText(/^from$/i)).not.toBeInTheDocument();
});

test("viewers get the dashboard; tabs switch views", async () => {
  mockFetch(routes);
  renderAt(url);
  expect(await screen.findByRole("link", { name: "Dashboard" })).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", { name: "Logs" })).toHaveAttribute("href", "/documents/d1");
});
```

Run → FAIL (no route).

**Step 10: Page** `src/routes/Dashboard.tsx`:

```tsx
import { Link, useParams, useSearchParams } from "react-router";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { Bucket } from "../lib/types";
import { DocumentTabs } from "../documents/DocumentTabs";
import { useDocument } from "../documents/useDocuments";
import { STATUS_LABEL } from "../logs/constants";
import { BarList } from "../dashboard/BarList";
import { logsHref, monthRange, PRESETS, presetRange, type Preset, type Range } from "../dashboard/periods";
import { useDashboard } from "../dashboard/useDashboard";
import { FIELD } from "../logs/constants";
import type { LogStatus } from "../lib/types";

const monthLabel = (key: string) =>
  new Date(`${key}-01T00:00:00Z`).toLocaleDateString("en-US", { month: "short", year: "numeric", timeZone: "UTC" });

export function Component() {
  const { id = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const period = (params.get("period") ?? "12m") as Preset | "custom";
  const fallback = presetRange("12m");
  const range: Range =
    period === "custom"
      ? { from: params.get("from") ?? fallback.from, to: params.get("to") ?? fallback.to }
      : presetRange(period);
  const doc = useDocument(id);
  const dash = useDashboard(id, range);

  if (doc.isPending) return <p className="text-muted-foreground">Loading…</p>;
  if (!doc.data)
    return (
      <p role="alert" className="text-destructive">
        {doc.error instanceof ApiError && doc.error.status === 404 ? "Document not found." : errorText(doc.error)}
      </p>
    );

  const inPeriod: [string, string][] = [["from", range.from], ["to", range.to]];
  const href = (pairs: [string, string][]) => logsHref(id, [...pairs, ...inPeriod]);
  const d = dash.data;

  const tile = (label: string, value: number | undefined, to: string) => (
    <Link to={to} className="flex flex-col gap-1 rounded-xl p-4 ring-1 ring-foreground/10 hover:bg-muted">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="text-2xl font-semibold tabular-nums">{value ?? "–"}</span>
    </Link>
  );

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-4">
        <Link to="/" className="text-sm text-muted-foreground hover:underline">← Documents</Link>
        <h2 className="text-xl font-semibold">{doc.data.title}</h2>
        <DocumentTabs id={id} current="dashboard" />
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <label htmlFor="period" className="text-sm text-muted-foreground">Period</label>
          <select
            id="period"
            className={FIELD}
            value={period}
            onChange={(e) => {
              const v = e.target.value;
              setParams(v === "custom" ? { period: v, from: range.from, to: range.to } : { period: v });
            }}
          >
            {PRESETS.map((p) => (
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
          </select>
          {period === "custom" && (
            <>
              <input aria-label="From" type="date" className={FIELD} value={range.from}
                onChange={(e) => e.target.value && setParams({ period, from: e.target.value, to: range.to })} />
              <input aria-label="To" type="date" className={FIELD} value={range.to}
                onChange={(e) => e.target.value && setParams({ period, from: range.from, to: e.target.value })} />
            </>
          )}
        </div>
      </div>

      {dash.error && (
        <p role="alert" className="text-destructive">{errorText(dash.error)}</p>
      )}

      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {tile("Total logs", d?.total, logsHref(id, []))}
        {tile("In period", d?.in_period, href([]))}
        {tile("High or critical", d?.high_impact, href([["impact", "high"], ["impact", "critical"]]))}
        {tile("In progress", d?.in_progress, href([["status", "in_progress"]]))}
      </div>

      {d && (
        <>
          <BarList title="Logs per month" layout="columns" buckets={d.months} label={monthLabel}
            hrefFor={(b) => { const r = monthRange(b.key, range.from, range.to); return logsHref(id, [["from", r.from], ["to", r.to]]); }} />
          <div className="grid gap-4 lg:grid-cols-3">
            <div className="flex flex-col gap-4 lg:col-span-2">
              <BarList title="Top tags" layout="rows" color="var(--chart-2)" buckets={d.tags}
                hrefFor={(b) => href([["tag", b.key]])} />
              <BarList title="Status" layout="rows" color="var(--chart-3)" buckets={d.statuses}
                label={(k) => STATUS_LABEL[k as LogStatus]} hrefFor={(b) => href([["status", b.key]])} />
              <BarList title="Impact" layout="rows" color="var(--chart-4)" buckets={d.impacts}
                hrefFor={(b) => href([["impact", b.key]])} />
            </div>
            <section aria-label="Coverage of the article's sections" className="flex flex-col gap-2 rounded-xl p-4 ring-1 ring-foreground/10">
              <h3 className="text-sm font-medium">Coverage</h3>
              <p className="text-xs text-muted-foreground">The brag document article's sections. Zero means nothing logged there this period.</p>
              <ul className="flex flex-col gap-1 text-sm">
                {d.coverage.map((b: Bucket) => (
                  <li key={b.key}>
                    <Link to={href([["tag", b.key]])} data-zero={b.count === 0 ? "true" : undefined}
                      className={`flex justify-between hover:underline ${b.count === 0 ? "font-medium text-destructive" : ""}`}>
                      <span>{b.key}</span>
                      <span className="tabular-nums">{b.count}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          </div>
        </>
      )}
    </div>
  );
}
```

(Merge the two `../logs/constants` imports into one when writing the file.)

**Step 11: Run** `npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npx eslint packages/app/src && npx prettier --write packages/app/src && npm run build -w @bragdoc/app` → all PASS. If a test's accessible name does not match (e.g. tile text spacing), fix the query, not the behaviour. Confirm existing `DocumentLogs` tests still pass with the tabs.

**Step 12: Commit**

```bash
git add frontend
git commit -m "feat(frontend): document dashboard with linked charts, coverage, and period selector"
```

---

### Task 7: Verify and open the PR

**Step 1:** Full verification (@superpowers:verification-before-completion):

```bash
cd backend && go build ./... && go test ./... && make lint && go test -count=1 -tags integration ./...
cd ../frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npx eslint packages/app/src && npx prettier --check packages/app/src && npm run build -w @bragdoc/app
```

**Step 2:** Review the branch with @superpowers:requesting-code-review; fix findings.

**Step 3:** Push and open the PR:

```bash
git push -u origin feat/dashboard
gh pr create --base main --title "feat: document dashboard (PRD-0005)" --body "$(cat <<'EOF'
## Summary
- `GET /documents/:id/dashboard?from&to`: totals, logs per month, top 10 tags, status, impact, and coverage of the article's sections. Examples excluded; everything except Total follows the period (default last 12 months, max 5 years).
- Read access only (viewers included, PRD-0004); access is checked before the cache.
- Redis cache-aside, 60 s, keyed on a per-document version that every log write bumps (web and bot); Redis down falls back to Postgres (ADR-0006).
- Logs list gains `examples=false`; every chart, tile, and table row links to the list with the same filter, so the numbers match (FR-6, asserted in an integration test).
- Frontend: Logs | Dashboard tabs, period selector, Recharts bars with a "Show as table" view of real links, coverage panel highlighting zero sections.

Implements [PRD-0005](docs/prd/0005-dashboard.md) under [ADR-0006](docs/adr/0006-redis-as-cache.md). Design: `docs/plans/2026-10-09-dashboard-design.md`.

## Test plan
- [ ] `go test ./...`, `make lint`
- [ ] `go test -tags integration ./...` (aggregates, FR-6 equality with the list, tenant isolation, 10k-log timing)
- [ ] Frontend: Vitest, typecheck, eslint, prettier, build
- [ ] Manual: add a log, reopen the dashboard, see it counted at once; click a bar and land on a list with the same count

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Report the PR URL.
